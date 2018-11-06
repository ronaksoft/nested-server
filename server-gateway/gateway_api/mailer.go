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

    "git.ronaksoftware.com/nested/server/model"
    "github.com/dustin/go-humanize"
    "github.com/globalsign/mgo/bson"
    "github.com/jaytaylor/html2text"
    "gopkg.in/mail.v2"
)

// MailRequest
type MailRequest struct {
    Host     string
    Port     int
    Username string
    Password string
    PostID   bson.ObjectId
}

// MailTemplate
type MailTemplate struct {
    Body        template.HTML
    Attachments []AttachmentTemplate
    SenderName  string
}

// AttachmentTemplate
type AttachmentTemplate struct {
    Url          string
    Group        string
    HasThumbnail bool
    Info         *nested.FileInfo
    Src          template.URL
    Ext          string
    HumanSize    string
}

// "Url":          fmt.Sprintf("%s/file/download/%s", ew.jh.conf.GetString("CYRUS_URL"), attachment.Token),
// "Group":        fileGroup(*attachment.FileInfo),
// "HasThumbnail": attachment.ThumbInfo != nil,
// "Info":         attachment.FileInfo,
// "SrcX":         template.URL(fmt.Sprintf("cid:%s", contentId)),
// "Ext":          strings.ToUpper(path.Ext(attachment.FileInfo.Filename)[1:]),
// "HumanSize":    humanize.Bytes(uint64(attachment.FileInfo.Size)),
// Mailer
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

// NewMailer
func NewMailer(worker *Worker) *Mailer {
    m := new(Mailer)
    m.worker = worker
    m.domain = worker.Config().GetString("DOMAIN")
    m.defaultSMTPHost = worker.Config().GetString("SMTP_HOST")
    m.defaultSMTPPort = worker.Config().GetInt("SMTP_PORT")
    m.defaultSMTPUser = worker.Config().GetString("SMTP_USER")
    m.defaultSMTPPass = worker.Config().GetString("SMTP_PASS")
    m.cyrusUrl = worker.Config().GetString("CYRUS_URL")
    m.chRequests = make(chan MailRequest, 1000)
    if tpl, err := template.ParseFiles("/ronak/templates/post_email.html"); err != nil {
        log.Fatal(err.Error())
    } else {
        m.template = tpl
    }

    go m.Run()

    return m
}

// Run
func (m *Mailer) Run() {
    for req := range m.chRequests {
        if req.Host == "" {
            req.Host = m.defaultSMTPHost
            req.Username = m.defaultSMTPUser
            req.Password = m.defaultSMTPPass
            req.Port = m.defaultSMTPPort
        }
        log.Println("+++++++++++++++++++++++++++++++++++++++++++++++", req.Host, req.Port, req.Username, req.Password)
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

// SendRequest
func (m *Mailer) SendRequest(req MailRequest) {
    m.chRequests <- req
}

// createMessage
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
        if place.Privacy.Receptive == nested.PLACE_RECEPTIVE_EXTERNAL {
        	if place.ID == postSender.ID || place.GrandParentID == postSender.ID  {
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
        if fileInfo.Type == nested.FILE_TYPE_IMAGE {
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
    case nested.FILE_TYPE_AUDIO, nested.FILE_TYPE_VIDEO:
        return "MULTIMEDIA"
    case nested.FILE_TYPE_IMAGE:
        return "IMAGE"
    case nested.FILE_TYPE_DOCUMENT:
        switch info.MimeType {
        case "application/pdf":
            return "PDF"

        default:
            return "DOCUMENT"
        }
    case nested.FILE_TYPE_OTHER:
        switch info.MimeType {
        case "application/zip", "application/x-rar-compressed":
            return "ZIP"
        }
    }
    return "OTHER"
}
