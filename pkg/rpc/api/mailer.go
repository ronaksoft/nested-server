package api

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"path"
	"strings"
	"time"

	"git.ronaksoft.com/nested/server/nested"
	"github.com/dustin/go-humanize"
	"github.com/globalsign/mgo/bson"
	"github.com/jaytaylor/html2text"
	"gopkg.in/mail.v2"
)

type MailRequest struct {
	Host     string
	Port     int
	Username string
	Password string
	PostID   bson.ObjectId
}

type MailTemplate struct {
	Body        template.HTML
	Attachments []AttachmentTemplate
	SenderName  string
}

type AttachmentTemplate struct {
	Url          string
	Group        string
	HasThumbnail bool
	Info         *nested.FileInfo
	Src          template.URL
	Ext          string
	HumanSize    string
}


// Mailer
// "Url":          fmt.Sprintf("%s/file/download/%s", ew.jh.conf.GetString("CYRUS_URL"), attachment.Token),
// "Group":        fileGroup(*attachment.FileInfo),
// "HasThumbnail": attachment.ThumbInfo != nil,
// "Info":         attachment.FileInfo,
// "SrcX":         template.URL(fmt.Sprintf("cid:%s", contentId)),
// "Ext":          strings.ToUpper(path.Ext(attachment.FileInfo.Filename)[1:]),
// "HumanSize":    humanize.Bytes(uint64(attachment.FileInfo.Size)),
type Mailer struct {
	worker          *Worker
	domain          string
	cyrusUrl        string
	defaultSMTPUser string
	defaultSMTPPass string
	defaultSMTPHost string
	defaultSMTPPort int
	template        *template.Template
	chRequests      chan MailRequest
}

func NewMailer(worker *Worker) *Mailer {
	m := new(Mailer)
	m.worker = worker
	m.domain = worker.Config().GetString("SENDER_DOMAIN")
	m.defaultSMTPHost = worker.Config().GetString("SMTP_HOST")
	m.defaultSMTPPort = worker.Config().GetInt("SMTP_PORT")
	m.defaultSMTPUser = worker.Config().GetString("SMTP_USER")
	m.defaultSMTPPass = worker.Config().GetString("SMTP_PASS")
	m.cyrusUrl = worker.Config().GetString("CYRUS_URL")
	m.chRequests = make(chan MailRequest, 1000)
	if tpl, err := template.ParseFiles("/ronak/templates/post_email.html"); err != nil {
		log.Println(err.Error())
	} else {
		m.template = tpl
	}

	// run the mail in background
	// TODO:: maybe a watch dog and graceful shutdown mechanic
	go m.Run()

	return m
}

func (m *Mailer) Run() {
	for req := range m.chRequests {
		if req.Host == "" {
			req.Host = m.defaultSMTPHost
			req.Username = m.defaultSMTPUser
			req.Password = m.defaultSMTPPass
			req.Port = m.defaultSMTPPort
		}

		d := mail.NewDialer(req.Host, req.Port, req.Username, req.Password)
		d.StartTLSPolicy = mail.MandatoryStartTLS
		d.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         req.Host,
		}

		if msg := m.createMessage(req.PostID); msg != nil {
			if err := d.DialAndSend(msg); err != nil {
				log.Println("Mailer::Run", err.Error(), req.Host, req.Port, req.Username, req.Password)
			}
		}
	}
}

func (m *Mailer) SendRequest(req MailRequest) {
	m.chRequests <- req
}

func (m *Mailer) createMessage(postID bson.ObjectId) *mail.Message {
	post := m.worker.Model().Post.GetPostByID(postID)
	if post == nil {
		return nil
	}
	postSender := m.worker.Model().Account.GetByID(post.SenderID, nil)
	if postSender == nil {
		return nil
	}

	msg := mail.NewMessage(
		mail.SetEncoding(mail.Base64),
		mail.SetCharset("UTF-8"),
	)

	mailTemplate := new(MailTemplate)
	mailTemplate.Body = template.HTML(post.Body)
	mailTemplate.SenderName = fmt.Sprintf("%s %s", postSender.FirstName, postSender.LastName)

	// Set MessageID
	msg.SetHeader("Message-ID", fmt.Sprintf("<%s@%s>", post.ID.Hex(), m.domain))

	// Set From
	msg.SetHeader("From",
		msg.FormatAddress(
			fmt.Sprintf("%s@%s", postSender.ID, m.domain),
			fmt.Sprintf("%s %s", postSender.FirstName, postSender.LastName),
		),
	)

	// Set Date
	msg.SetHeader("Date", msg.FormatDate(time.Now()))

	// Set To
	recipients := make([]string, 0, len(post.PlaceIDs)+len(post.Recipients))
	places := m.worker.Model().Place.GetPlacesByIDs(post.PlaceIDs)
	for _, recipient := range post.Recipients {
		recipients = append(recipients, recipient)
	}
	for _, place := range places {
		if place.Privacy.Receptive == nested.PlaceReceptiveExternal {
			if place.ID == postSender.ID || place.GrandParentID == postSender.ID {
				continue
			}
			recipients = append(recipients, msg.FormatAddress(fmt.Sprintf("%s@%s", place.ID, m.domain), place.Name))
		}
	}
	msg.SetHeader("To", recipients...)

	// Set Subject
	msg.SetHeader("Subject", post.Subject)

	// Set InReplyTo
	if post.ReplyTo.Valid() {
		msg.SetHeader("In-Reply-To", fmt.Sprintf("<%s@%s>", post.ReplyTo.Hex(), m.domain))
	}

	for _, universalID := range post.AttachmentIDs {
		fileInfo := m.worker.Model().File.GetByID(universalID, nil)
		downloadToken, err := m.worker.Model().Token.CreateFileToken(universalID, postSender.ID, "")
		if err != nil {
			return nil
		}

		attachmentTemplate := AttachmentTemplate{
			Url:   fmt.Sprintf("%s/file/download/%s", m.cyrusUrl, downloadToken),
			Group: m.fileGroup(fileInfo),
			Info:  fileInfo,

			Ext:       strings.ToUpper(path.Ext(fileInfo.Filename)[1:]),
			HumanSize: humanize.Bytes(uint64(fileInfo.Size)),
		}
		if fileInfo.Type == nested.FileTypeImage {
			attachmentTemplate.Src = template.URL(fmt.Sprintf("%s/file/view/x/%s", m.cyrusUrl, fileInfo.Thumbnails.X64))
			attachmentTemplate.HasThumbnail = true
		} else {
			attachmentTemplate.HasThumbnail = false
		}
		mailTemplate.Attachments = append(mailTemplate.Attachments, attachmentTemplate)
	}

	// Set Body
	body := new(bytes.Buffer)
	if err := m.template.Execute(body, mailTemplate); err != nil {
		log.Println("Template Execute Error:", err.Error())
	}
	bodyHtml := body.String()
	bodyText := ""
	if txt, err := html2text.FromString(bodyHtml); err != nil {
		return nil
	} else {
		bodyText = txt
	}
	msg.AddAlternative("text/plain", bodyText)
	msg.AddAlternative("text/html", bodyHtml)

	return msg
}

func (m *Mailer) fileGroup(info *nested.FileInfo) string {
	switch info.Type {
	case nested.FileTypeAudio, nested.FileTypeVideo:
		return "MULTIMEDIA"
	case nested.FileTypeImage:
		return "IMAGE"
	case nested.FileTypeDocument:
		switch info.MimeType {
		case "application/pdf":
			return "PDF"

		default:
			return "DOCUMENT"
		}
	case nested.FileTypeOther:
		switch info.MimeType {
		case "application/zip", "application/x-rar-compressed":
			return "ZIP"
		}
	}
	return "OTHER"
}
