package main

import (
  "strings"
  "github.com/jhillyerd/enmime"
)

type RecipientGroup struct {
  ToMap map[string]string
  CcMap map[string]string
  BccMap map[string]string

  To  []string
  Cc  []string
  Bcc []string
}

// Retrieve all (to, cc, bcc) recipients
func (r *RecipientGroup) GetAll() []string {
  return append(append(r.To, r.Cc...), r.Bcc...)
}

// Retrieve all non-blind (to, cc) recipients
func (r *RecipientGroup) GetAllNonBlind() []string {
  return append(r.To, r.Cc...)
}

// Retrieve direct (to) recipients located under specific domain
func (r *RecipientGroup) GetToByDomain(domain string) []string {
  recipients := r.To

  myRecipients := recipients[:0]
  for _, addr := range recipients {
    if belongsToDomain(addr, domain) {
      myRecipients = append(myRecipients, addr)
    }
  }

  return myRecipients
}

// Retrieve carbon-copy (cc) recipients located under specific domain
func (r *RecipientGroup) GetCcByDomain(domain string) []string {
  recipients := r.Cc

  myRecipients := recipients[:0]
  for _, addr := range recipients {
    if belongsToDomain(addr, domain) {
      myRecipients = append(myRecipients, addr)
    }
  }

  return myRecipients
}

// Retrieve blind-carbon-copy (bcc) recipients located under specific domain
func (r *RecipientGroup) GetBccByDomain(domain string) []string {
  recipients := r.Bcc

  myRecipients := recipients[:0]
  for _, addr := range recipients {
    if belongsToDomain(addr, domain) {
      myRecipients = append(myRecipients, addr)
    }
  }

  return myRecipients
}

// Retrieve all (to, cc, bcc) recipients located under specific domain
func (r *RecipientGroup) GetAllByDomain(domain string) []string {
  recipients := r.GetAll()

  myRecipients := recipients[:0]
  for _, addr := range recipients {
    if belongsToDomain(addr, domain) {
      myRecipients = append(myRecipients, addr)
    }
  }

  return myRecipients
}

// Retrieve all non-blind (to, cc) recipients located under specific domain
func (r *RecipientGroup) GetAllNonBlindByDomain(domain string) []string {
  recipients := r.GetAllNonBlind()

  myRecipients := recipients[:0]
  for _, addr := range recipients {
    if belongsToDomain(addr, domain) {
      myRecipients = append(myRecipients, addr)
    }
  }

  return myRecipients
}

func NewRecipientGroup(mime *enmime.Envelope) *RecipientGroup {
  toAddrs, _ := mime.AddressList("To")
  ccAddrs, _ := mime.AddressList("Cc")
  bccAddrs, _ := mime.AddressList("Bcc")

  recipients := new(RecipientGroup)
  recipients.ToMap = make(map[string]string)
  recipients.CcMap = make(map[string]string)
  recipients.BccMap = make(map[string]string)

  for _, addr := range toAddrs {
    recipients.To = append(recipients.To, addr.Address)
    recipients.ToMap[addr.Address] = addr.Address
  }

  for _, addr := range ccAddrs {
    recipients.Cc = append(recipients.Cc, addr.Address)
    recipients.CcMap[addr.Address] = addr.Address
  }

  for _, addr := range bccAddrs {
    recipients.Bcc = append(recipients.Bcc, addr.Address)
    recipients.BccMap[addr.Address] = addr.Address
  }

  return recipients
}

func GetMailboxes(addrs []string) []string {
  mailboxes := addrs[:0]
  for _, addr := range addrs {
    mailbox := strings.Split(addr, "@")[0]
    mailboxes = append(mailboxes, mailbox)
  }

  return mailboxes
}

func belongsToDomain(addr string, domain string) bool {
  actDomain := strings.Split(addr, "@")[1]

  return domain == actDomain
}
