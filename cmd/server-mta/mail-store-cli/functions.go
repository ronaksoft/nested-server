package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"git.ronaksoft.com/nested/server/cmd/server-mta/mail-store-cli/client-storage"
	"git.ronaksoft.com/nested/server/nested"
	"github.com/jhillyerd/enmime"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"sync"
)

func dispatch(domain, sender string, recipients []string, buf []byte, m *Model) error {
	var mailEnvelope *enmime.Envelope
	// Parse message
	if env, err := enmime.ReadEnvelope(bytes.NewReader(buf)); err != nil {
		_LOG.Error("Read message error: ", zap.Error(err))
		return err
	} else {
		mailEnvelope = env
	}
	subject := mailEnvelope.GetHeader("Subject")
	if v := mailEnvelope.GetHeader("From"); len(v) != 0 {
		sender = mailEnvelope.GetHeader("From")
	}

	switch sender {
	case _Config.GetString("MAILER_DAEMON"):
		pmSubject := mailEnvelope.GetHeader("Postmaster-Subject")

		switch pmSubject {
		case "Postmaster Copy: Undelivered Mail":
			fmt.Println("Undelivered Mail")
			// FIXME: Notify Sender About Delivery Report

		case "Postmaster Warning: Delayed Mail":
			fmt.Println("Undelivered Mail")

		}

		switch subject {
		case "Successful Mail Delivery Report":
			fmt.Println("Successful Mail Delivery Report")

		case "Mail Delivery Status Report":
			fmt.Println("Mail Delivery Status Report")

		}

		if from := mailEnvelope.GetHeader("From"); len(from) > 0 {
			sender = from
		} else {
			sender = fmt.Sprintf("%s@%s", sender, domain)
		}
	}

	return store(domain, sender, recipients, mailEnvelope, bytes.NewReader(buf), m)
}

func store(domain, sender string, recipients []string, mailEnvelope *enmime.Envelope, rawBody io.Reader, m *Model) error {
	bodyHtml := mailEnvelope.HTML
	bodyPlain := mailEnvelope.Text

	messageID := mailEnvelope.GetHeader("Message-ID")
	inReplyTo := mailEnvelope.GetHeader("In-Reply-To")
	subject := mailEnvelope.GetHeader("Subject")
	from := mailEnvelope.GetHeader("From")

	var senderID, senderName string
	if addr, err := mail.ParseAddress(sender); err != nil {
		_LOG.Error("ERROR::Parse sender address error:", zap.Error(err))
		senderID = sender
	} else {
		senderID = addr.Address
		senderName = addr.Name
	}

	if addr, err := mail.ParseAddress(from); err != nil {
		_LOG.Error("ERROR::::::Parse from address error:", zap.Error(err))
	} else {
		if addr.Address == senderID && senderName == "" {
			senderName = addr.Name
		}
	}

	var replyTo string
	if replyToHeader := strings.TrimSpace(mailEnvelope.GetHeader("Reply-To")); 0 == len(replyToHeader) {
		// No Reply-To has been set
	} else if addr, err := mail.ParseAddress(replyToHeader); err != nil {
		_LOG.Error("ERROR::Reply-to address is invalid:", zap.Any("replyToHeader", replyToHeader))
	} else {
		replyTo = addr.Address
	}

	// --Process Recipients
	recipientGroup := NewRecipientGroup(mailEnvelope)
	_LOG.Info("recipientGroup", zap.Any("recipientGroup", recipientGroup))
	var nonBlindPlaceIDs []string
	var nonBlindTargets []string
	var blindPlaceIDs []string
	for _, rcpt := range recipients {
		rcpt = strings.ToLower(rcpt)
		_, isTo := recipientGroup.ToMap[rcpt]
		_, isCc := recipientGroup.CcMap[rcpt]
		_, isBcc := recipientGroup.BccMap[rcpt]

		req := strings.Split(rcpt, "@")
		// TODO: Check alias
		mailbox := req[0]

		if isTo || isCc {
			nonBlindPlaceIDs = append(nonBlindPlaceIDs, mailbox)
			nonBlindTargets = append(nonBlindTargets, rcpt)
		} else if isBcc {
			blindPlaceIDs = append(blindPlaceIDs, mailbox)
		}
	}

	_LOG.Info("Got email to: nonBlindPlaceIDs, blindPlaceIDs ", zap.Any("nonBlindPlaceIDs", nonBlindPlaceIDs), zap.Any("blindPlaceIDs", blindPlaceIDs))
	// --/Process Recipients

	attachmentOwners := append(nonBlindPlaceIDs, blindPlaceIDs...)

	var rawMsgFileID nested.UniversalID
	if finfo, err := uploadFile(fmt.Sprintf("%s-%s.eml", messageID, subject), senderID, nested.FileStatusAttached, attachmentOwners, rawBody, m.Storage); err != nil {
		// TODO: Retry
		_LOG.Error("ERROR::::::Unable to upload raw message file", zap.Error(err))
	} else {
		_LOG.Debug("File uploaded", zap.String("", fmt.Sprintf("%s-%s.eml", messageID, subject)))
		rawMsgFileID = finfo.UniversalId
	}

	// --Get Sender Picture From Gravatar
	var senderPicture nested.Picture
	encoder := md5.New()
	encoder.Write([]byte(senderID))
	senderIdHash := hex.EncodeToString(encoder.Sum(nil))
	senderPictureUrl := fmt.Sprintf("http://www.gravatar.com/avatar/%s?size=%d&rating=g&default=404", senderIdHash, 512)
	if req, err := http.NewRequest(http.MethodGet, senderPictureUrl, nil); err != nil {
		_LOG.Error("ERROR::::::Unable to create gravatar http request:", zap.Error(err))
	} else if res, err := http.DefaultClient.Do(req); err != nil {
		_LOG.Error("ERROR::::::Unable to do gravatar http request:", zap.Error(err))
	} else if res.StatusCode != http.StatusOK {
		_LOG.Debug("ERROR::::::Gravatar not found")
	} else {
		if finfo, err := uploadFile(fmt.Sprintf("%s.jpg", senderIdHash), senderID, nested.FileStatusPublic, []string{}, res.Body, m.Storage); err != nil {
			_LOG.Error("ERROR::::::Unable set sender profile picture:", zap.Error(err))
		} else {
			senderPicture = finfo.Thumbs
		}
		res.Body.Close()
	}

	// --Save Attachments
	wg := sync.WaitGroup{}

	type multipartAttachment struct {
		finfo     client_storage.UploadedFile
		content   string
		contentId string
	}

	// --Inline Attachments
	_LOG.Info("Going to save inline attachments")
	chInlineAttachments := make(chan multipartAttachment, len(mailEnvelope.Inlines))
	for k, att := range mailEnvelope.Inlines {
		wg.Add(1)
		go func(att *enmime.Part, index int) {
			defer wg.Done()

			// Upload File
			var cid string
			if c := att.Header.Get("Content-Id"); len(c) > 0 {
				cid = c[1 : len(c)-1]
				_LOG.Info("CID: ", zap.String("", c), zap.String("", cid))
			}

			filename := att.FileName
			if 0 == len(filename) {
				filename = fmt.Sprintf("inline_attachment_%d", index)
			}

			_LOG.Info("Uploading Inline: ",
				zap.Any("", att.ContentType),
				zap.String("", filename),
				zap.String("", cid),
				zap.Any("", att.Header),
			)

			if finfo, err := uploadFile(filename, senderID, nested.FileStatusAttached, attachmentOwners, bytes.NewReader(att.Content), m.Storage); err != nil {
				_LOG.Error("ERROR::::::Error adding inline attachment:", zap.Error(err))
			} else {
				_LOG.Info("Gonna create file token for %s", zap.Any("", finfo.UniversalId))
				if tk, err := m.CreateFileToken(finfo.UniversalId, "", ""); err != nil {
					_LOG.Error("ERROR::::::Error creating file token for inline attachment: ", zap.Error(err))
					benc := make([]byte, base64.StdEncoding.EncodedLen(len(att.Content)))
					base64.StdEncoding.Encode(benc, att.Content)

					chInlineAttachments <- multipartAttachment{
						contentId: cid,
						content:   fmt.Sprintf("data:image/png;base64, %s", string(benc)),
						finfo:     *finfo,
					}
				} else {
					chInlineAttachments <- multipartAttachment{
						contentId: cid,
						content:   fmt.Sprintf("%s/file/view/%s", m.CyrusURL, tk),
						finfo:     *finfo,
					}
				}
				_LOG.Info("Uploaded Inline:", zap.Any("", filename))
			}
		}(att, k)
	}
	// --/Inline Attachments

	// --Attachments
	_LOG.Info("Going to save attachments")
	chAttachments := make(chan multipartAttachment, len(mailEnvelope.Attachments))
	for k, att := range mailEnvelope.Attachments {
		wg.Add(1)
		go func(att *enmime.Part, index int) {
			defer wg.Done()

			// Upload File
			var cid string
			if c := att.Header.Get("Content-Id"); len(c) > 0 {
				cid = c[1 : len(c)-1]
			}

			filename := att.FileName
			if 0 == len(filename) {
				filename = fmt.Sprintf("attachment_%d", index)
			}
			_LOG.Info("Uploading Inline: ",
				zap.Any("", att.ContentType),
				zap.String("", filename),
				zap.String("", cid),
				zap.Any("", att.Header),
			)

			if finfo, err := uploadFile(att.FileName, senderID, nested.FileStatusAttached, attachmentOwners, bytes.NewReader(att.Content), m.Storage); err != nil {
				_LOG.Error("ERROR::::::Error adding inline attachment:", zap.Error(err))
			} else {
				_LOG.Info("Gonna create file token for %s", zap.Any("", finfo.UniversalId))
				if tk, err := m.CreateFileToken(finfo.UniversalId, "", ""); err != nil {
					_LOG.Error("ERROR::::::Error creating file token for inline attachment: ", zap.Error(err))
					benc := make([]byte, base64.StdEncoding.EncodedLen(len(att.Content)))
					base64.StdEncoding.Encode(benc, att.Content)

					chAttachments <- multipartAttachment{
						contentId: cid,
						content:   fmt.Sprintf("data:image/png;base64, %s", string(benc)),
						finfo:     *finfo,
					}
				} else {
					chAttachments <- multipartAttachment{
						contentId: cid,
						content:   fmt.Sprintf("%s/file/view/%s", m.CyrusURL, tk),
						finfo:     *finfo,
					}
				}
				_LOG.Info("Uploaded Inline:", zap.String("", filename))
			}
		}(att, k)
	}

	// Wait for files to be saved
	wg.Wait()
	_LOG.Info("All attachments jobs have been done")
	close(chAttachments)
	close(chInlineAttachments)

	inlineAttachments := make(map[string]string, len(mailEnvelope.Attachments)+len(mailEnvelope.Inlines))
	attachments := make(map[string]nested.FileInfo, len(mailEnvelope.Attachments)+len(mailEnvelope.Inlines))
	for att := range chInlineAttachments {
		if len(att.contentId) > 0 && strings.Count(bodyHtml, att.contentId)+strings.Count(bodyPlain, att.contentId) > 0 {
			inlineAttachments[att.contentId] = att.content
		} else {
			_LOG.Info("Not found %s in body", zap.String("", att.contentId))
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
			inlineAttachments[att.contentId] = att.content
		} else {
			_LOG.Info("Not found %s in body", zap.String("", att.contentId))
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
		postAttachmentIDs := make([]nested.UniversalID, 0, len(attachments))
		postAttachmentSizes := make([]int64, 0, len(attachments))
		for attachID, inf := range attachments {
			postAttachmentIDs = append(postAttachmentIDs, nested.UniversalID(attachID))
			postAttachmentSizes = append(postAttachmentSizes, int64(inf.Size))
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
		// Validate Targets and Separate places and emails
		mapPlaceIDs := make(map[string]bool)
		mapEmails := make(map[string]bool)
		_LOG.Debug("targets:", zap.Any("", targets))
		_LOG.Debug("domain:", zap.Any("", domain))
		for _, v := range targets {
			if idx := strings.Index(v, "@"); idx != -1 {
				_LOG.Debug("fmt.Sprintf(@%s, domain)", zap.String("", fmt.Sprintf("@%s", domain)))

				if strings.HasSuffix(strings.ToLower(v), fmt.Sprintf("@%s", domain)) && !m.IsBlocked(v[:idx], senderID) {
					_LOG.Debug("", zap.String("", fmt.Sprintf("%s", v[:idx])))
					mapPlaceIDs[v[:idx]] = true
				} else {
					mapEmails[v] = true
				}
			} else if m.PlaceExist(v) && !m.IsBlocked(v, senderID) {
				mapPlaceIDs[v] = true
			}
		}
		for placeID := range mapPlaceIDs {
			postCreateReq.PlaceIDs = append(postCreateReq.PlaceIDs, placeID)
		}
		for recipient := range mapEmails {
			postCreateReq.Recipients = append(postCreateReq.Recipients, recipient)
		}
		_LOG.Debug("postCreateReq", zap.Any("AttachmentIDs", postCreateReq.AttachmentIDs))
		if post := m.AddPost(postCreateReq); post == nil {
			_LOG.Error("ERROR::::::Post add error:")
			return fmt.Errorf("could not create post")
		} else {
			_LOG.Debug("Post added to nested instance", zap.String("post.SenderID", post.SenderID))
			_LOG.Debug("Post added to nested instance", zap.String("post.Subject", post.Subject))
			_LOG.Debug("Post added to nested instance", zap.Strings("post.Places", post.PlaceIDs))
			m.ExternalPushPlaceActivityPostAdded(post)
			for _, pid := range post.PlaceIDs {
				// Internal
				place := m.GetPlaceByID(pid)
				memberIDs := place.GetMemberIDs()
				m.InternalPlaceActivitySyncPush(memberIDs, pid, nested.PlaceActivityActionPostAdd)
			}
		}
		return nil
	}

	// Create one post for CCs
	_LOG.Debug("Gonna add post to:", zap.Any("nonBlindTargets", nonBlindTargets))
	if err := postCreate(nonBlindTargets); err != nil {
		_LOG.Error("ERROR::::::Post add error:", zap.Error(err))
		return err
	}

	// Create Individual Posts for BCCs
	for _, recipient := range blindPlaceIDs {
		_LOG.Debug("Gonna add post to:", zap.String("recipient", recipient))
		if err := postCreate([]string{recipient}); err != nil {
			_LOG.Error("ERROR::::::Post add error:", zap.Error(err))
			return err
		}
	}
	return nil
}
