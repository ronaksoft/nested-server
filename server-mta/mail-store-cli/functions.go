package main

import (
    "io"
    "github.com/jhillyerd/enmime"
    "bytes"
    "fmt"
    "net/mail"
    "strings"
    "git.ronaksoftware.com/nested/server/model"
    "crypto/md5"
    "encoding/hex"
    "net/http"
    "sync"
    "io/ioutil"
    "encoding/base64"
    "git.ronaksoftware.com/nested/server/server-gateway/client"
)

func Dispatch(sender string, recipients []string, body io.Reader) error {
    var mailEnvelope *enmime.Envelope

    buf := new(bytes.Buffer)
    io.Copy(buf, body)

    // Parse message
    if env, err := enmime.ReadEnvelope(bytes.NewReader(buf.Bytes())); err != nil {
        _Log.Debugf("Read message error: %s", err.Error())
        return err
    } else {
        mailEnvelope = env
    }

    subject := mailEnvelope.GetHeader("Subject")
    if v := mailEnvelope.GetHeader("From"); len(v) != 0 {
        sender = mailEnvelope.GetHeader("From")
    }
    _Log.Debugf("Sender: %s", sender)

    switch sender {
    case _Config.GetString("MAILER_DAEMON"):
        _Log.Debugf("System Message: %s", subject)
        pmSubject := mailEnvelope.GetHeader("Postmaster-Subject")

        switch pmSubject {
        case "Postmaster Copy: Undelivered Mail":
            _Log.Debug("Undelivered Mail")
            // FIXME: Notify Sender About Delivery Report

        case "Postmaster Warning: Delayed Mail":
            _Log.Debug("Undelivered Mail")

        }

        switch subject {
        case "Successful Mail Delivery Report":
            _Log.Debug("Successful Mail Delivery Report")

        case "Mail Delivery Status Report":
            _Log.Debug("Mail Delivery Status Report")

        }

        if from := mailEnvelope.GetHeader("From"); len(from) > 0 {
            sender = from
        } else {
            sender = fmt.Sprintf("%s@%s", sender, _Config.GetString("DOMAIN"))
        }
    }

    return store(sender, recipients, mailEnvelope, bytes.NewReader(buf.Bytes()))
}

func store(sender string, recipients []string, mailEnvelop *enmime.Envelope, rawBody io.Reader) error {
    bodyHtml := mailEnvelop.HTML
    bodyPlain := mailEnvelop.Text

    messageID := mailEnvelop.GetHeader("Message-ID")
    inReplyTo := mailEnvelop.GetHeader("In-Reply-To")
    subject := mailEnvelop.GetHeader("Subject")
    from := mailEnvelop.GetHeader("From")

    var senderID, senderName string
    if addr, err := mail.ParseAddress(sender); err != nil {
        _Log.Error("Parse sender address error:", err)
        senderID = sender
    } else {
        senderID = addr.Address
        senderName = addr.Name
    }

    if addr, err := mail.ParseAddress(from); err != nil {
        _Log.Error("Parse from address error:", err)
    } else {
        if addr.Address == senderID && senderName == "" {
            senderName = addr.Name
        }
    }

    var replyTo string
    if replyToHeader := strings.TrimSpace(mailEnvelop.GetHeader("Reply-To")); 0 == len(replyToHeader) {
        // No Reply-To has been set
    } else if addr, err := mail.ParseAddress(replyToHeader); err != nil {
        _Log.Error("Reply-to address is invalid:", replyToHeader)
    } else {
        replyTo = addr.Address
    }

    // --Process Recipients
    recipientGroup := NewRecipientGroup(mailEnvelop)

    var nonBlindPlaceIDs []string
    var blindPlaceIDs []string
    for _, rcpt := range recipients {
        _, isTo := recipientGroup.ToMap[rcpt]
        _, isCc := recipientGroup.CcMap[rcpt]
        _, isBcc := recipientGroup.BccMap[rcpt]

        req := strings.Split(rcpt, "@")
        // TODO: Check alias
        mailbox := req[0]

        if isTo || isCc {
            nonBlindPlaceIDs = append(nonBlindPlaceIDs, mailbox)
        } else if isBcc {
            blindPlaceIDs = append(blindPlaceIDs, mailbox)
        }
    }

    nonBlindTargets := nonBlindPlaceIDs
    for _, addr := range recipientGroup.GetAllNonBlind() {
        isInRcpt := false
        for _, rcpt := range recipients {
            if addr == rcpt {
                isInRcpt = true

                break
            }
        }

        if !isInRcpt {
            nonBlindTargets = append(nonBlindTargets, addr)
        }
    }

    _Log.Info("Got email to: To, CC: %v; BCC: %v", nonBlindPlaceIDs, blindPlaceIDs)
    // --/Process Recipients

    attachmentOwners := append(nonBlindPlaceIDs, blindPlaceIDs...)

    var rawMsgFileID nested.UniversalID
    if finfo, err := uploadFile(fmt.Sprintf("%s-%s.eml", messageID, subject), senderID, nested.FILE_STATUS_ATTACHED, attachmentOwners, rawBody); err != nil {
        // TODO: Retry
        _Log.Error("Unable to upload raw message file")
    } else {
        _Log.Debugf("File %s uploaded", fmt.Sprintf("%s-%s.eml", messageID, subject))
        rawMsgFileID = finfo.UniversalId
    }

    // --Get Sender Picture From Gravatar
    var senderPicture nested.Picture
    encoder := md5.New()
    encoder.Write([]byte(senderID))
    senderIdHash := hex.EncodeToString(encoder.Sum(nil))
    senderPictureUrl := fmt.Sprintf("http://www.gravatar.com/avatar/%s?size=%d&rating=g&default=404", senderIdHash, 512)
    if req, err := http.NewRequest(http.MethodGet, senderPictureUrl, nil); err != nil {
        _Log.Error("Unable to create gravatar http request:", err)
    } else if res, err := http.DefaultClient.Do(req); err != nil {
        _Log.Error("Unable to do gravatar http request:", err)
    } else if res.StatusCode != http.StatusOK {
        _Log.Error("Gravatar not found")
    } else {
        if finfo, err := uploadFile(fmt.Sprintf("%s.jpg", senderIdHash), senderID, nested.FILE_STATUS_PUBLIC, []string{}, res.Body); err != nil {
            _Log.Error("Unable set sender profile picture:", err)
        } else {
            senderPicture = finfo.Thumbs
        }
        res.Body.Close()
    }
    // --/Get Sender Picture From Gravatar

    // --Save Attachments
    wg := sync.WaitGroup{}

    type multipartAttachment struct {
        finfo     nestedGateway.UploadedFile
        content   string
        contentId string
    }

    // --Inline Attachments
    _Log.Debug("Going to save inline attachments")
    chInlineAttachments := make(chan multipartAttachment, len(mailEnvelop.Inlines))
    for k, att := range mailEnvelop.Inlines {
        wg.Add(1)
        go func(att *enmime.Part, index int) {
            defer wg.Done()

            // Upload File
            var cid string
            if c := att.Header.Get("Content-Id"); len(c) > 0 {
                cid = c[1 : len(c)-1]
                _Log.Debugf("CID: %s --> %s", c, cid)
            }

            filename := att.FileName
            if 0 == len(filename) {
                filename = fmt.Sprintf("inline_attachment_%d", index)
            }

            _Log.Debugf("Uploading Inline: %s, %s, %s, %v", att.ContentType, filename, cid, att.Header)

            attachmentContent, _ := ioutil.ReadAll(att)
            if finfo, err := uploadFile(filename, senderID, nested.FILE_STATUS_ATTACHED, attachmentOwners, bytes.NewReader(attachmentContent)); err != nil {
                _Log.Error("Error adding inline attachment:", filename, err)
            } else {
                _Log.Debug("Gonna create file token for %s", finfo.UniversalId)

                if tk, err := _Model.Token.CreateFileToken(finfo.UniversalId, "", ""); err != nil {
                    _Log.Errorf("Error creating file token for inline attachment: %s, %s, %s", filename, cid, err)

                    benc := make([]byte, base64.StdEncoding.EncodedLen(len(attachmentContent)))
                    base64.StdEncoding.Encode(benc, attachmentContent)

                    chInlineAttachments <- multipartAttachment{
                        contentId: cid,
                        content:   fmt.Sprintf("data:image/png;base64, %s", string(benc)),
                        finfo:     *finfo,
                    }
                } else {
                    chInlineAttachments <- multipartAttachment{
                        contentId: cid,
                        content:   fmt.Sprintf("%s/file/view/%s", _Config.GetString("CYRUS_URL"), tk),
                        finfo:     *finfo,
                    }
                }
                _Log.Debugf("Uploaded Inline: %s, %s", filename, cid)
            }
        }(att, k)
    }
    // --/Inline Attachments

    // --Attachments
    _Log.Debug("Going to save attachments")
    chAttachments := make(chan multipartAttachment, len(mailEnvelop.Attachments))
    for k, att := range mailEnvelop.Attachments {
        wg.Add(1)
        go func(att *enmime.Part, index int) {
            defer wg.Done()

            // Upload File
            var cid string
            if c := att.Header.Get("Content-Id"); len(c) > 0 {
                cid = c[1 : len(c)-1]
                _Log.Debugf("CID: %s --> %s", c, cid)
            }

            filename := att.FileName
            if 0 == len(filename) {
                filename = fmt.Sprintf("attachment_%d", index)
            }

            _Log.Debugf("Uploading: %s, %s, %s, %v", att.ContentType, filename, cid, att.Header)
            attachmentContent, _ := ioutil.ReadAll(att)
            if finfo, err := uploadFile(att.FileName, senderID, nested.FILE_STATUS_ATTACHED, attachmentOwners, bytes.NewReader(attachmentContent)); err != nil {
                _Log.Errorf("Error adding attachment: %s, %s", att.FileName, err)
            } else {
                _Log.Debug("Gonna create file token for %s", finfo.UniversalId)

                if tk, err := _Model.Token.CreateFileToken(finfo.UniversalId, "", ""); err != nil {
                    _Log.Errorf("Error creating file token for attachment: %s, %s, %s", att.FileName, cid, err)

                    benc := make([]byte, base64.StdEncoding.EncodedLen(len(attachmentContent)))
                    base64.StdEncoding.Encode(benc, attachmentContent)

                    chAttachments <- multipartAttachment{
                        contentId: cid,
                        content:   fmt.Sprintf("data:image/png;base64, %s", string(benc)),
                        finfo:     *finfo,
                    }
                } else {
                    chAttachments <- multipartAttachment{
                        contentId: cid,
                        content:   fmt.Sprintf("%s/file/view/%s", _Config.GetString("CYRUS_URL"), tk),
                        finfo:     *finfo,
                    }
                }
                _Log.Debugf("Uploaded: %s", att.FileName)
            }
        }(att, k)
    }

    // Wait for files to be saved
    wg.Wait()
    _Log.Debug("All attachments jobs have been done")
    close(chAttachments)
    close(chInlineAttachments)

    inlineAttachments := make(map[string]string, len(mailEnvelop.Attachments)+len(mailEnvelop.Inlines))
    attachments := make(map[string]nested.FileInfo, len(mailEnvelop.Attachments)+len(mailEnvelop.Inlines))
    for att := range chInlineAttachments {
        if len(att.contentId) > 0 && strings.Count(bodyHtml, att.contentId)+strings.Count(bodyPlain, att.contentId) > 0 {
            _Log.Debugf("Found %s in body: %d", att.contentId, strings.Count(bodyHtml, att.contentId), strings.Count(bodyPlain, att.contentId))
            inlineAttachments[att.contentId] = att.content
        } else {
            _Log.Debugf("Not found %s in body", att.contentId)
            if att.finfo.Size > 0 {
                attachments[string(att.finfo.UniversalId)] = nested.FileInfo{
                    Size:     att.finfo.Size,
                    Filename: att.finfo.Name,
                }
            }
        }
    }
    for att := range chAttachments {
        if len(att.contentId) > 0 && strings.Count(bodyHtml, att.contentId)+strings.Count(bodyPlain, att.contentId) > 0 {
            _Log.Debugf("Found %s in body: %d", att.contentId, strings.Count(bodyHtml, att.contentId), strings.Count(bodyPlain, att.contentId))
            inlineAttachments[att.contentId] = att.content
        } else {
            _Log.Debugf("Not found %s in body", att.contentId)
        }

        if att.finfo.Size > 0 {
            attachments[string(att.finfo.UniversalId)] = nested.FileInfo{
                Size:     att.finfo.Size,
                Filename: att.finfo.Name,
            }
        }
    }
    // --/Attachments
    // --/Save Attachments

    // --Prepare Body
    for k, v := range inlineAttachments {
        bodyHtml = strings.Replace(bodyHtml, fmt.Sprintf("\"cid:%s\"", k), fmt.Sprintf("\"%s\"", v), -1)
        bodyPlain = strings.Replace(bodyPlain, fmt.Sprintf("\"cid:%s\"", k), fmt.Sprintf("\"%s\"", v), -1)
    }
    _Log.Debugf("HTML Body: %s", bodyHtml)
    _Log.Debugf("Plain Body: %s", bodyPlain)
    // --/Prepare Body

    postCreate := func(targets []string) error {
        postCreateReq := nested.PostCreateRequest{
            SenderID: senderID,
            Subject:  subject,
            EmailMetadata: nested.EmailMetadata{
                Name:           senderName,
                RawMessageFile: rawMsgFileID,
                MessageID:      messageID,
                InReplyTo:      inReplyTo,
                ReplyTo:        replyTo,
                Picture:        senderPicture,
            },
            PlaceIDs:   []string{},
            Recipients: []string{},
        }
        // TODO:: update the code
        //if "" != inReplyTo {
        //	if postId, domain := msgapi.EmailMessageIdDecode(inReplyTo); 0 == len(postId) || domain != _Config.GetString("DOMAIN") {
        //		// Query Nothing
        //	} else if repliedToPost := _Model.Post.GetPostByID(bson.ObjectIdHex(postId)); nil == repliedToPost {
        //		_Log.Debugf("In-Reply-to post not exists: %s", postId)
        //	} else {
        //		postCreateReq.ReplyTo = repliedToPost.ID
        //	}
        //}
        postAttachmentIDs := make([]nested.UniversalID, 0, len(attachments))
        postAttachmentSizes := make([]int64, 0, len(attachments))
        for attachID, info := range attachments {
            postAttachmentIDs = append(postAttachmentIDs, nested.UniversalID(attachID))
            postAttachmentSizes = append(postAttachmentSizes, int64(info.Size))
        }
        postCreateReq.AttachmentIDs = postAttachmentIDs
        postCreateReq.AttachmentSizes = postAttachmentSizes

        if len(bodyHtml) > 0 {
            postCreateReq.ContentType = nested.CONTENT_TYPE_TEXT_HTML
            postCreateReq.Body = bodyHtml
        } else {
            postCreateReq.ContentType = nested.CONTENT_TYPE_TEXT_PLAIN
            postCreateReq.Body = bodyPlain
        }

        _Log.Debugf("Content Type: %s", postCreateReq.ContentType)

        // Validate Sender
        _Log.Debug("Sender: %s", senderID)

        // Validate Targets and Separate places and emails
        mapPlaceIDs := make(map[string]bool)
        mapEmails := make(map[string]bool)
        for _, v := range targets {
            if idx := strings.Index(v, "@"); idx != -1 {
                if strings.HasSuffix(strings.ToLower(v), fmt.Sprintf("@%s", _Config.GetString("DOMAIN"))) {
                    // TODO:: Security bug ?!!
                    mapPlaceIDs[v[:idx]] = true
                } else {
                    mapEmails[v] = true
                }
            } else if _Model.Place.Exists(v) {
                mapPlaceIDs[v] = true
            }
        }
        for placeID := range mapPlaceIDs {
            postCreateReq.PlaceIDs = append(postCreateReq.PlaceIDs, placeID)
        }
        for recipient := range mapEmails {
            postCreateReq.Recipients = append(postCreateReq.Recipients, recipient)
        }

        if post := _Model.Post.AddPost(postCreateReq); post == nil {
            _Log.Error("Post add error:")
            return fmt.Errorf("could not create post")
        } else {
            _Log.Info("Post added: %s", post.ID)
            _ClientNtfy.ExternalPushPlaceActivityPostAdded(post)
            for _, pid := range post.PlaceIDs {
                // Internal
                place := _Model.Place.GetByID(pid, nil)
                memberIDs := place.GetMemberIDs()
                _ClientNtfy.InternalPlaceActivitySyncPush(memberIDs, pid, nested.PLACE_ACTIVITY_ACTION_POST_ADD)
            }
        }

        return nil
    }

    // Create one post for CCs
    _Log.Info("Gonna add post to:", nonBlindTargets)
    if err := postCreate(nonBlindTargets); err != nil {
        _Log.Error("Post add error:", err)

        return err
    }

    // Create Individual Posts for BCCs
    for _, recipient := range blindPlaceIDs {
        _Log.Info("Gonna add post to:", recipient)
        if err := postCreate([]string{recipient}); err != nil {
            _Log.Error("Post add error:", err)

            return err
        }
    }

    return nil
}
