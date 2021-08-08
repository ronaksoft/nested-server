package lmtp

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/log"
	"github.com/emersion/go-smtp"
	"github.com/jhillyerd/enmime"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"sync"
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
	uploader   *uploadClient
	pusher     *pusherClient
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
		nestedMail = &NestedMail{
			Attachments:       map[string]nested.FileInfo{},
			InlineAttachments: map[string]string{},
		}
	)
	if envelope, err = enmime.ReadEnvelope(r); err != nil {
		log.Warn("got error on read envelope", zap.Error(err))
		return
	}
	if err = s.extractSender(nestedMail, envelope); err != nil {
		log.Warn("got error on extract sender", zap.Error(err))
		return
	}
	if err = s.extractHeader(nestedMail, envelope); err != nil {
		log.Warn("got error on extract header", zap.Error(err))
		return
	}
	if err = s.storeMail(nestedMail, envelope); err != nil {
		log.Warn("got error on store mail", zap.Error(err))
		return
	}
	if err = s.extractGravatar(nestedMail, envelope); err != nil {
		log.Warn("got error on extract gravatar", zap.Error(err))
		return
	}
	if err = s.extractInlineAttachments(nestedMail, envelope); err != nil {
		log.Warn("got error on extract inline attachments", zap.Error(err))
		return
	}
	if err = s.extractInlineAttachments(nestedMail, envelope); err != nil {
		log.Warn("got error on extract inline attachments", zap.Error(err))
		return
	}
	if err = s.extractAttachments(nestedMail, envelope); err != nil {
		log.Warn("got error on extract attachments", zap.Error(err))
		return
	}
	if err = s.store(nestedMail, envelope); err != nil {
		log.Warn("got error on store", zap.Error(err))
		return
	}

	return
}
func (s *Session) extractSender(nm *NestedMail, mailEnvelope *enmime.Envelope) error {
	from := mailEnvelope.GetHeader("From")
	if addr, err := mail.ParseAddress(s.from); err != nil {
		log.Error("got error on parsing sender address", zap.String("Sender", s.from), zap.Error(err))
		nm.SenderID = s.from
	} else {
		nm.SenderID = addr.Address
		nm.SenderName = addr.Name
	}

	if addr, err := mail.ParseAddress(from); err != nil {
		log.Error("got error on parsing FROM header", zap.Error(err), zap.String("FROM", from))
	} else {
		if addr.Address == nm.SenderID && nm.SenderName == "" {
			nm.SenderName = addr.Name
		}
	}
	return nil
}

func (s *Session) extractHeader(nm *NestedMail, envelope *enmime.Envelope) error {
	replyToHeader := strings.TrimSpace(envelope.GetHeader("Reply-To"))
	if len(replyToHeader) > 0 {
		// Reply-To has been set
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

	uploadedFile, err := s.uploader.uploadFile(
		fmt.Sprintf("%s-%s.eml", envelope.GetHeader("MessageID"), envelope.GetHeader("Subject")),
		nm.SenderID,
		nested.FileStatusAttached, nm.AttachOwners, buff,
	)
	if err != nil {
		return err
	}
	nm.RawUniversalID = uploadedFile.UniversalID
	return
}
func (s *Session) extractGravatar(nm *NestedMail, _ *enmime.Envelope) error {
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

	uploadedFile, err := s.uploader.uploadFile(fmt.Sprintf("%s.jpg", senderIdHash), nm.SenderID, nested.FileStatusPublic, []string{}, res.Body)
	if err != nil {
		return err
	}
	nm.SenderPic = uploadedFile.Thumbs
	_ = res.Body.Close()
	return nil
}
func (s *Session) extractInlineAttachments(nm *NestedMail, envelope *enmime.Envelope) error {
	chInlineAttachments := make(chan MultipartFile, len(envelope.Inlines))
	wg := sync.WaitGroup{}
	for k, att := range envelope.Inlines {
		wg.Add(1)
		go func(att *enmime.Part, index int) {
			defer wg.Done()

			// Upload File
			var cid string
			if c := att.Header.Get("Content-Id"); len(c) > 0 {
				cid = c[1 : len(c)-1]
				log.Info("CID: ", zap.String("", c), zap.String("", cid))
			}

			filename := att.FileName
			if 0 == len(filename) {
				filename = fmt.Sprintf("inline_attachment_%d", index)
			}

			uploadedFile, err := s.uploader.uploadFile(filename, nm.SenderID, nested.FileStatusAttached, nm.AttachOwners, bytes.NewReader(att.Content))
			if err != nil {
				log.Error("got error on uploading inline attachments (upload)", zap.Error(err), zap.String("Sender", nm.SenderName))
				return
			}
			tk, err := s.model.Token.CreateFileToken(uploadedFile.UniversalID, "", "")
			if err != nil {
				benc := make([]byte, base64.StdEncoding.EncodedLen(len(att.Content)))
				base64.StdEncoding.Encode(benc, att.Content)
				chInlineAttachments <- MultipartFile{
					contentID: cid,
					content:   fmt.Sprintf("data:image/png;base64, %s", string(benc)),
					file:      *uploadedFile,
				}
			} else {
				chInlineAttachments <- MultipartFile{
					contentID: cid,
					content:   fmt.Sprintf("%s/file/view/%s", config.GetString(config.CyrusURL), tk),
					file:      *uploadedFile,
				}
			}
		}(att, k)
	}
	wg.Wait()
	close(chInlineAttachments)

	for att := range chInlineAttachments {
		if len(att.contentID) > 0 && strings.Count(envelope.HTML, att.contentID)+strings.Count(envelope.Text, att.contentID) > 0 {
			nm.InlineAttachments[att.contentID] = att.content
		} else {
			if att.file.Size > 0 {
				nm.Attachments[string(att.file.UniversalID)] = nested.FileInfo{
					Size:     att.file.Size,
					Filename: att.file.Name,
				}
			}
		}
	}
	return nil
}
func (s *Session) extractAttachments(nm *NestedMail, envelope *enmime.Envelope) error {
	chAttachments := make(chan MultipartFile, len(envelope.Attachments))
	wg := sync.WaitGroup{}
	for k, att := range envelope.Attachments {
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
			uploadedFile, err := s.uploader.uploadFile(att.FileName, nm.SenderID, nested.FileStatusAttached, nm.AttachOwners, bytes.NewReader(att.Content))
			if err != nil {
				log.Error("got error on uploading attachments (upload)", zap.Error(err), zap.String("Sender", nm.SenderName))
				return
			}
			tk, err := s.model.Token.CreateFileToken(uploadedFile.UniversalID, "", "")
			if err != nil {
				benc := make([]byte, base64.StdEncoding.EncodedLen(len(att.Content)))
				base64.StdEncoding.Encode(benc, att.Content)

				chAttachments <- MultipartFile{
					contentID: cid,
					content:   fmt.Sprintf("data:image/png;base64, %s", string(benc)),
					file:      *uploadedFile,
				}
			} else {
				chAttachments <- MultipartFile{
					contentID: cid,
					content:   fmt.Sprintf("%s/file/view/%s", config.GetString(config.CyrusURL), tk),
					file:      *uploadedFile,
				}
			}
		}(att, k)
	}
	wg.Wait()
	close(chAttachments)
	for att := range chAttachments {
		if len(att.contentID) > 0 && strings.Count(envelope.HTML, att.contentID)+strings.Count(envelope.Text, att.contentID) > 0 {
			nm.InlineAttachments[att.contentID] = att.content
		}

		if att.file.Size > 0 {
			nm.Attachments[string(att.file.UniversalID)] = nested.FileInfo{
				Size:     att.file.Size,
				Filename: att.file.Name,
			}
		}
	}
	return nil
}
func (s *Session) store(nm *NestedMail, mailEnvelope *enmime.Envelope) error {
	var (
		bodyHtml  = mailEnvelope.HTML
		bodyPlain = mailEnvelope.Text
		messageID = mailEnvelope.GetHeader("Message-ID")
		inReplyTo = mailEnvelope.GetHeader("In-Reply-To")
		subject   = mailEnvelope.GetHeader("Subject")
	)

	for k, v := range nm.InlineAttachments {
		bodyHtml = strings.Replace(bodyHtml, fmt.Sprintf("\"cid:%s\"", k), fmt.Sprintf("\"%s\"", v), -1)
		bodyPlain = strings.Replace(bodyPlain, fmt.Sprintf("\"cid:%s\"", k), fmt.Sprintf("\"%s\"", v), -1)
	}

	postCreate := func(targets []string) error {
		postCreateReq := nested.PostCreateRequest{
			SenderID: nm.SenderID,
			Subject:  subject,
			EmailMetadata: nested.EmailMetadata{
				Name:           nm.SenderName,
				RawMessageFile: nm.RawUniversalID,
				MessageID:      messageID,
				InReplyTo:      inReplyTo,
				ReplyTo:        nm.ReplyTo,
				Picture:        nm.SenderPic,
			},
			PlaceIDs:   []string{},
			Recipients: []string{},
		}
		postAttachmentIDs := make([]nested.UniversalID, 0, len(nm.Attachments))
		postAttachmentSizes := make([]int64, 0, len(nm.Attachments))
		for attachID, inf := range nm.Attachments {
			postAttachmentIDs = append(postAttachmentIDs, nested.UniversalID(attachID))
			postAttachmentSizes = append(postAttachmentSizes, inf.Size)
		}
		postCreateReq.AttachmentIDs = postAttachmentIDs
		postCreateReq.AttachmentSizes = postAttachmentSizes
		if len(bodyHtml) > 0 {
			postCreateReq.ContentType = nested.ContentTypeTextHtml
			postCreateReq.Body = bodyHtml
		} else {
			postCreateReq.ContentType = nested.ContentTypeTextPlain
			postCreateReq.Body = bodyPlain
		}
		// Validate Targets and Separate places and emails
		mapPlaceIDs := make(map[string]bool)
		mapEmails := make(map[string]bool)
		for _, targetAddr := range targets {
			if idx := strings.Index(targetAddr, "@"); idx != -1 {
				if strings.HasSuffix(strings.ToLower(targetAddr), fmt.Sprintf("@%s", strings.ToLower(config.GetString(config.SenderDomain)))) {
					mapPlaceIDs[targetAddr[:idx]] = true
				} else {
					mapEmails[targetAddr] = true
				}
			} else if s.model.Place.Exists(targetAddr) && !s.model.Place.IsBlocked(targetAddr, nm.SenderID) {
				mapPlaceIDs[targetAddr] = true
			}
		}
		for placeID := range mapPlaceIDs {
			postCreateReq.PlaceIDs = append(postCreateReq.PlaceIDs, placeID)
		}
		for recipient := range mapEmails {
			postCreateReq.Recipients = append(postCreateReq.Recipients, recipient)
		}
		log.Debug("postCreateReq", zap.Any("AttachmentIDs", postCreateReq.AttachmentIDs))
		post := s.model.Post.AddPost(postCreateReq)
		if post == nil {
			return fmt.Errorf("could not create post")
		}

		for _, pid := range post.PlaceIDs {
			s.pusher.PlaceActivity(pid, nested.PlaceActivityActionPostAdd)
		}

		return nil
	}

	// Create one post for CCs
	if err := postCreate(nm.NonBlindTargets); err != nil {
		return err
	}

	// Create Individual Posts for BCCs
	for _, recipient := range nm.BlindPlaceIDs {
		if err := postCreate([]string{recipient}); err != nil {
			return err
		}
	}
	return nil
}
