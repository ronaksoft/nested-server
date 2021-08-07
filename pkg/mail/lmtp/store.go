package lmtp

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/emersion/go-smtp"
	"github.com/jhillyerd/enmime"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/mail"
	"strings"
)

/*
   Creation Time: 2021 - Aug - 06
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type Session struct {
	hostname   string
	remoteAddr string
	from       string
	rcpts      []string
	opts       smtp.MailOptions
	model      *nested.Manager
}

func (s *Session) Reset() {
	s.opts = smtp.MailOptions{}
	s.from = ""
	s.rcpts = s.rcpts[:0]
	log.Info("Session Reset", zap.String("H", s.hostname), zap.String("Remote", s.remoteAddr))
}

func (s *Session) Logout() error {
	log.Info("Session Logout", zap.String("H", s.hostname), zap.String("Remote", s.remoteAddr))
	return nil
}

func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	s.from = from
	s.opts = opts
	log.Info("Session Mail",
		zap.String("H", s.hostname),
		zap.String("Remote", s.remoteAddr),
		zap.String("From", from),
		zap.Any("MO", opts),
	)
	return nil
}

func (s *Session) Rcpt(to string) error {
	log.Info("Session To", zap.String("H", s.hostname), zap.String("Remote", s.remoteAddr), zap.String("TO", to))
	s.rcpts = append(s.rcpts, to)
	return nil
}

func (s *Session) Data(r io.Reader) (err error) {
	var (
		envelope   *enmime.Envelope
		nestedMail = &NestedMail{}
	)
	if envelope, err = enmime.ReadEnvelope(r); err != nil {
		return
	}
	if err = s.extractSender(nestedMail, envelope); err != nil {
		return
	}
	if err = s.storeMail(nestedMail, envelope); err != nil {
		return
	}
	if err = s.extractGravatar(nestedMail, envelope); err != nil {
		return
	}
	// if err = s.extractInlineAttachments(nestedMail, envelope); err != nil {
	// 	return
	// }

	// return s.store(envelope)
	return nil
}
func (s *Session) extractSender(nm *NestedMail, envelope *enmime.Envelope) error {
	nm.SenderID, nm.SenderName = s.parseSender(envelope)
	replyToHeader := strings.TrimSpace(envelope.GetHeader("Reply-To"))
	if len(replyToHeader) >= 0 {
		// No Reply-To has been set
		addr, err := mail.ParseAddress(replyToHeader)
		if err != nil {
			log.Error("ERROR::Reply-to address is invalid:", zap.Any("replyToHeader", replyToHeader))
			return err
		}
		nm.ReplyTo = addr.Address
	}
	return nil
}
func (s *Session) extractRecipients(nm *NestedMail, envelope *enmime.Envelope) error {
	recipientGroup := NewRecipientGroup(envelope)

	for _, rcpt := range s.rcpts {
		rcpt = strings.ToLower(rcpt)
		_, isTo := recipientGroup.ToMap[rcpt]
		_, isCc := recipientGroup.CcMap[rcpt]
		_, isBcc := recipientGroup.BccMap[rcpt]

		req := strings.Split(rcpt, "@")
		// TODO: Check alias
		mailbox := req[0]

		if isTo || isCc {
			nm.NonBlindPlaceIDs = append(nm.NonBlindPlaceIDs, mailbox)
			nm.NonBlindTargets = append(nm.NonBlindTargets, rcpt)
		} else if isBcc {
			nm.BlindPlaceIDs = append(nm.BlindPlaceIDs, mailbox)
		}
	}
	nm.AttachOwners = append(nm.AttachOwners, nm.NonBlindPlaceIDs...)
	nm.AttachOwners = append(nm.AttachOwners, nm.BlindPlaceIDs...)
	return nil
}
func (s *Session) storeMail(nm *NestedMail, envelope *enmime.Envelope) (err error) {
	buff := &bytes.Buffer{}
	err = envelope.Root.Encode(buff)
	if err != nil {
		return
	}

	fileInfo := s.model.Store.Save(buff,
		nested.GenerateFileInfo(
			fmt.Sprintf("%s-%s.eml", envelope.GetHeader("MessageID"), envelope.GetHeader("Subject")),
			nm.SenderID,
			nested.FileTypeOther,
			nil,
		),
	)
	if fileInfo == nil {
		return
	}
	s.model.File.SetStatus(fileInfo.ID, nested.FileStatusAttached)
	nm.RawUniversalID = fileInfo.ID
	return
}
func (s *Session) extractGravatar(nm *NestedMail, envelope *enmime.Envelope) error {
	encoder := md5.New()
	encoder.Write([]byte(nm.SenderID))
	senderIdHash := hex.EncodeToString(encoder.Sum(nil))
	senderPictureUrl := fmt.Sprintf("https://www.gravatar.com/avatar/%s?size=%d&rating=g&default=404", senderIdHash, 512)
	req, err := http.NewRequest(http.MethodGet, senderPictureUrl, nil)
	if err != nil {
		return nil
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	if res.StatusCode != http.StatusOK {
		return nil
	}

	fileInfo := s.model.Store.Save(res.Body,
		nested.GenerateFileInfo(
			fmt.Sprintf("%s.jpg", senderIdHash),
			nm.SenderID,
			nested.FileTypeImage,
			nil,
		),
	)
	if fileInfo == nil {
		return nil
	}
	s.model.File.SetStatus(fileInfo.ID, nested.FileStatusPublic)
	// TODO:: nm.SenderPic

	_ = res.Body.Close()
	return nil
}

// func (s *Session) extractInlineAttachments(nm *NestedMail, envelope *enmime.Envelope) error {
// 	chInlineAttachments := make(chan multipartAttachment, len(mailEnvelope.Inlines))
// 	wg := sync.WaitGroup{}
// 	for k, att := range envelope.Inlines {
// 		wg.Add(1)
// 		go func(att *enmime.Part, index int) {
// 			defer wg.Done()
//
// 			// Upload File
// 			var cid string
// 			if c := att.Header.Get("Content-Id"); len(c) > 0 {
// 				cid = c[1 : len(c)-1]
// 				log.Info("CID: ", zap.String("", c), zap.String("", cid))
// 			}
//
// 			filename := att.FileName
// 			if 0 == len(filename) {
// 				filename = fmt.Sprintf("inline_attachment_%d", index)
// 			}
//
// 			fileInfo := s.model.Store.Save(bytes.NewReader(att.Content), nested.GenerateFileInfo(
// 				filename, nm.SenderID, att.ContentType,
// 			))
// 			if fileInfo != nil {
// 				return
// 			}
// 			s.model.File.SetStatus(fileInfo.ID, nested.FileStatusAttached)
//
// 			if finfo, err := uploadFile(filename, senderID, nested.FileStatusAttached, attachmentOwners, bytes.NewReader(att.Content), m.Storage); err != nil {
// 				log.Error("ERROR::::::Error adding inline attachment:", zap.Error(err))
// 			} else {
// 				log.Info("Gonna create file token for %s", zap.Any("", finfo.UniversalId))
// 				if tk, err := m.CreateFileToken(finfo.UniversalId, "", ""); err != nil {
// 					log.Error("ERROR::::::Error creating file token for inline attachment: ", zap.Error(err))
// 					benc := make([]byte, base64.StdEncoding.EncodedLen(len(att.Content)))
// 					base64.StdEncoding.Encode(benc, att.Content)
//
// 					chInlineAttachments <- multipartAttachment{
// 						contentId: cid,
// 						content:   fmt.Sprintf("data:image/png;base64, %s", string(benc)),
// 						finfo:     *finfo,
// 					}
// 				} else {
// 					chInlineAttachments <- multipartAttachment{
// 						contentId: cid,
// 						content:   fmt.Sprintf("%s/file/view/%s", m.CyrusURL, tk),
// 						finfo:     *finfo,
// 					}
// 				}
// 				log.Info("Uploaded Inline:", zap.Any("", filename))
// 			}
// 		}(att, k)
// 	}
// 	wg.Wait()
// }
// func (s *Session) store(mailEnvelope *enmime.Envelope) error {
// 	var (
// 		bodyHtml  = mailEnvelope.HTML
// 		bodyPlain = mailEnvelope.Text
// 		messageID = mailEnvelope.GetHeader("Message-ID")
// 		inReplyTo = mailEnvelope.GetHeader("In-Reply-To")
// 		subject   = mailEnvelope.GetHeader("Subject")
// 	)
//
// 	// --Save Attachments
// 	wg := sync.WaitGroup{}
//
// 	type multipartAttachment struct {
// 		finfo     client_storage.UploadedFile
// 		content   string
// 		contentId string
// 	}
//
// 	// --Inline Attachments
// 	log.Info("Going to save inline attachments")
//
// 	// --/Inline Attachments
//
// 	// --Attachments
// 	log.Info("Going to save attachments")
// 	chAttachments := make(chan multipartAttachment, len(mailEnvelope.Attachments))
// 	for k, att := range mailEnvelope.Attachments {
// 		wg.Add(1)
// 		go func(att *enmime.Part, index int) {
// 			defer wg.Done()
//
// 			// Upload File
// 			var cid string
// 			if c := att.Header.Get("Content-Id"); len(c) > 0 {
// 				cid = c[1 : len(c)-1]
// 			}
//
// 			filename := att.FileName
// 			if 0 == len(filename) {
// 				filename = fmt.Sprintf("attachment_%d", index)
// 			}
// 			log.Info("Uploading Inline: ",
// 				zap.Any("", att.ContentType),
// 				zap.String("", filename),
// 				zap.String("", cid),
// 				zap.Any("", att.Header),
// 			)
//
// 			if finfo, err := uploadFile(att.FileName, senderID, nested.FileStatusAttached, attachmentOwners, bytes.NewReader(att.Content), m.Storage); err != nil {
// 				log.Error("ERROR::::::Error adding inline attachment:", zap.Error(err))
// 			} else {
// 				log.Info("Gonna create file token for %s", zap.Any("", finfo.UniversalId))
// 				if tk, err := m.CreateFileToken(finfo.UniversalId, "", ""); err != nil {
// 					log.Error("ERROR::::::Error creating file token for inline attachment: ", zap.Error(err))
// 					benc := make([]byte, base64.StdEncoding.EncodedLen(len(att.Content)))
// 					base64.StdEncoding.Encode(benc, att.Content)
//
// 					chAttachments <- multipartAttachment{
// 						contentId: cid,
// 						content:   fmt.Sprintf("data:image/png;base64, %s", string(benc)),
// 						finfo:     *finfo,
// 					}
// 				} else {
// 					chAttachments <- multipartAttachment{
// 						contentId: cid,
// 						content:   fmt.Sprintf("%s/file/view/%s", m.CyrusURL, tk),
// 						finfo:     *finfo,
// 					}
// 				}
// 				log.Info("Uploaded Inline:", zap.String("", filename))
// 			}
// 		}(att, k)
// 	}
//
// 	// Wait for files to be saved
// 	wg.Wait()
// 	log.Info("All attachments jobs have been done")
// 	close(chAttachments)
// 	close(chInlineAttachments)
//
// 	inlineAttachments := make(map[string]string, len(mailEnvelope.Attachments)+len(mailEnvelope.Inlines))
// 	attachments := make(map[string]nested.FileInfo, len(mailEnvelope.Attachments)+len(mailEnvelope.Inlines))
// 	for att := range chInlineAttachments {
// 		if len(att.contentId) > 0 && strings.Count(bodyHtml, att.contentId)+strings.Count(bodyPlain, att.contentId) > 0 {
// 			inlineAttachments[att.contentId] = att.content
// 		} else {
// 			log.Info("Not found %s in body", zap.String("", att.contentId))
// 			if att.finfo.Size > 0 {
// 				attachments[string(att.finfo.UniversalId)] = nested.FileInfo{
// 					Size:     att.finfo.Size,
// 					Filename: att.finfo.Name,
// 				}
// 			}
// 		}
// 	}
// 	for att := range chAttachments {
// 		if len(att.contentId) > 0 && strings.Count(bodyHtml, att.contentId)+strings.Count(bodyPlain, att.contentId) > 0 {
// 			inlineAttachments[att.contentId] = att.content
// 		} else {
// 			log.Info("Not found %s in body", zap.String("", att.contentId))
// 		}
//
// 		if att.finfo.Size > 0 {
// 			attachments[string(att.finfo.UniversalId)] = nested.FileInfo{
// 				Size:     att.finfo.Size,
// 				Filename: att.finfo.Name,
// 			}
// 		}
// 	}
// 	// --/Attachments
// 	// --/Save Attachments
//
// 	// --Prepare Body
// 	for k, v := range inlineAttachments {
// 		bodyHtml = strings.Replace(bodyHtml, fmt.Sprintf("\"cid:%s\"", k), fmt.Sprintf("\"%s\"", v), -1)
// 		bodyPlain = strings.Replace(bodyPlain, fmt.Sprintf("\"cid:%s\"", k), fmt.Sprintf("\"%s\"", v), -1)
// 	}
// 	// --/Prepare Body
//
// 	postCreate := func(targets []string) error {
// 		postCreateReq := nested.PostCreateRequest{
// 			SenderID: senderID,
// 			Subject:  subject,
// 			EmailMetadata: nested.EmailMetadata{
// 				Name:           senderName,
// 				RawMessageFile: rawMsgFileID,
// 				MessageID:      messageID,
// 				InReplyTo:      inReplyTo,
// 				ReplyTo:        replyTo,
// 				Picture:        senderPicture,
// 			},
// 			PlaceIDs:   []string{},
// 			Recipients: []string{},
// 		}
// 		postAttachmentIDs := make([]nested.UniversalID, 0, len(attachments))
// 		postAttachmentSizes := make([]int64, 0, len(attachments))
// 		for attachID, inf := range attachments {
// 			postAttachmentIDs = append(postAttachmentIDs, nested.UniversalID(attachID))
// 			postAttachmentSizes = append(postAttachmentSizes, int64(inf.Size))
// 		}
// 		postCreateReq.AttachmentIDs = postAttachmentIDs
// 		postCreateReq.AttachmentSizes = postAttachmentSizes
// 		if len(bodyHtml) > 0 {
// 			postCreateReq.ContentType = nested.ContentTypeTextHtml
// 			postCreateReq.Body = bodyHtml
// 		} else {
// 			postCreateReq.ContentType = nested.ContentTypeTextPlain
// 			postCreateReq.Body = bodyPlain
// 		}
// 		// Validate Targets and Separate places and emails
// 		mapPlaceIDs := make(map[string]bool)
// 		mapEmails := make(map[string]bool)
// 		log.Debug("targets:", zap.Any("", targets))
// 		log.Debug("domain:", zap.Any("", domain))
// 		for _, v := range targets {
// 			if idx := strings.Index(v, "@"); idx != -1 {
// 				log.Debug("fmt.Sprintf(@%s, domain)", zap.String("", fmt.Sprintf("@%s", domain)))
//
// 				if strings.HasSuffix(strings.ToLower(v), fmt.Sprintf("@%s", domain)) && !m.IsBlocked(v[:idx], senderID) {
// 					log.Debug("", zap.String("", fmt.Sprintf("%s", v[:idx])))
// 					mapPlaceIDs[v[:idx]] = true
// 				} else {
// 					mapEmails[v] = true
// 				}
// 			} else if m.PlaceExist(v) && !m.IsBlocked(v, senderID) {
// 				mapPlaceIDs[v] = true
// 			}
// 		}
// 		for placeID := range mapPlaceIDs {
// 			postCreateReq.PlaceIDs = append(postCreateReq.PlaceIDs, placeID)
// 		}
// 		for recipient := range mapEmails {
// 			postCreateReq.Recipients = append(postCreateReq.Recipients, recipient)
// 		}
// 		log.Debug("postCreateReq", zap.Any("AttachmentIDs", postCreateReq.AttachmentIDs))
// 		if post := m.AddPost(postCreateReq); post == nil {
// 			log.Error("ERROR::::::Post add error:")
// 			return fmt.Errorf("could not create post")
// 		} else {
// 			log.Debug("Post added to nested instance", zap.String("post.SenderID", post.SenderID))
// 			log.Debug("Post added to nested instance", zap.String("post.Subject", post.Subject))
// 			log.Debug("Post added to nested instance", zap.Strings("post.Places", post.PlaceIDs))
//
// 			m.ExternalPushPlaceActivityPostAdded(post)
// 			for _, pid := range post.PlaceIDs {
// 				// Internal
// 				place := s.model.Place.GetByID(pid)
// 				memberIDs := place.GetMemberIDs()
// 				m.InternalPlaceActivitySyncPush(memberIDs, pid, nested.PlaceActivityActionPostAdd)
// 			}
// 		}
// 		return nil
// 	}
//
// 	// Create one post for CCs
// 	log.Debug("Gonna add post to:", zap.Any("nonBlindTargets", nonBlindTargets))
// 	if err := postCreate(nonBlindTargets); err != nil {
// 		log.Error("ERROR::::::Post add error:", zap.Error(err))
// 		return err
// 	}
//
// 	// Create Individual Posts for BCCs
// 	for _, recipient := range blindPlaceIDs {
// 		log.Debug("Gonna add post to:", zap.String("recipient", recipient))
// 		if err := postCreate([]string{recipient}); err != nil {
// 			log.Error("ERROR::::::Post add error:", zap.Error(err))
// 			return err
// 		}
// 	}
// 	return nil
// }

func (s *Session) parseSender(mailEnvelope *enmime.Envelope) (senderID, senderName string) {
	from := mailEnvelope.GetHeader("From")
	if addr, err := mail.ParseAddress(s.from); err != nil {
		log.Error("ERROR::Parse sender address error:", zap.String("Sender", s.from), zap.Error(err))
		senderID = s.from
	} else {
		senderID = addr.Address
		senderName = addr.Name
	}

	if addr, err := mail.ParseAddress(from); err != nil {
		log.Error("ERROR::::::Parse from address error:", zap.Error(err))
	} else {
		if addr.Address == senderID && senderName == "" {
			senderName = addr.Name
		}
	}
	return
}
