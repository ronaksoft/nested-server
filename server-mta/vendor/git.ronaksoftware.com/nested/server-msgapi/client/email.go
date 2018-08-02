package msgapi

import (
  "time"
  "bytes"
  "strings"
  "encoding/json"

  "git.ronaksoftware.com/common/server-protocol"
)

type cEmail struct {
  c *Client
}

const (
  CMD_CATEGORY_EMAIL = "email"

  CMD_EMAIL_SEND_POST = "email/send_post"
  CMD_EMAIL_SEND_VERIFICATION = "email/send_verification"
)

func newEmailClient(c *Client) (*cEmail, error) {
  ec := new(cEmail)
  ec.c = c

  return ec, nil
}

// --Send Post

type EmailSendPostInput struct {
  PostId string   `json:"post_id"`
}

type EmailSendPostOutput struct {
  FailedRecipients []string `json:"failed_recipients"`
}

func (ec *cEmail) PrepareSendPost(postId string) protocol.Packet {
  return protocol.NewPacket(SUBJECT, protocol.NewRequest(CMD_EMAIL_SEND_POST, EmailSendPostInput{
    PostId: postId,
  }))
}

// Perform a request to msgapi in order to send nested post
// as email to post's recipients
func (ec *cEmail) SendPost(postId string) (*EmailSendPostOutput, error) {
  pkt := ec.PrepareSendPost(postId)
  res := new(protocol.GenericResponse)

  if err := ec.c.conn.Request(pkt.Address(), pkt.Datagram(), res, time.Minute * 1); err != nil {
    return nil, protocol.NewUnknownError(protocol.D{"error": err.Error()})
  } else if protocol.STATUS_FAILURE == res.Status() {
    return nil, res.Data().(protocol.Error)
  }

  b, _ := json.Marshal(res.Data())
  data := EmailSendPostOutput{}
  if err := json.Unmarshal(b, &data); err != nil {
    return nil, protocol.NewUnknownError(protocol.D{"error": err.Error()})
  }

  return &data, nil
}

// --/Send Post

// --Send Post

type EmailSendVerificationInput struct {
  VerificationId string `json:"verification_id"`
}

type EmailSendVerificationOutput struct {
  FailedRecipients []string `json:"failed_recipients"`
}

func (ec *cEmail) PrepareSendVerification(verificationId string) protocol.Packet {
  return protocol.NewPacket(SUBJECT, protocol.NewRequest(CMD_EMAIL_SEND_VERIFICATION, EmailSendVerificationInput{
    VerificationId: verificationId,
  }))
}

// Perform a request to msgapi in order to send nested post
// as email to post's recipients
func (ec *cEmail) SendVerification(verificationId string) (*EmailSendVerificationOutput, error) {
  pkt := ec.PrepareSendVerification(verificationId)
  res := new(protocol.GenericResponse)

  if err := ec.c.conn.Request(pkt.Address(), pkt.Datagram(), res, time.Minute * 1); err != nil {
    return nil, protocol.NewUnknownError(protocol.D{"error": err.Error()})
  } else if protocol.STATUS_FAILURE == res.Status() {
    return nil, res.Data().(protocol.Error)
  }

  b, _ := json.Marshal(res.Data())
  data := EmailSendVerificationOutput{}
  if err := json.Unmarshal(b, &data); err != nil {
    return nil, protocol.NewUnknownError(protocol.D{"error": err.Error()})
  }

  return &data, nil
}

// --/Send Post

// Gets Post ID and Domain and Creates an Email Message-ID
func EmailMessageIdEncode(postId, domain string) string {
  var msgId bytes.Buffer
  msgId.WriteRune('<')
  msgId.WriteString(postId)
  msgId.WriteRune('@')
  msgId.WriteString(domain)
  msgId.WriteRune('>')

  return msgId.String()
}

// Gets an Email Message-ID and returns Post ID and Domain
func EmailMessageIdDecode(msgId string) (postId string, domain string) {
  if msgId[0] != '<' || msgId[len(msgId) - 1] != '>' {
    return "", ""
  }

  parts := strings.Split(msgId[1:len(msgId) - 1], "@")

  if 2 != len(parts) {
    return "", ""
  }

  return parts[0], parts[1]
}
