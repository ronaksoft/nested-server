package nestedServiceAdmin

import (
	"net/url"
	"strings"
	"net/http"
	"log"
)

// ADP is a SMS Provider
type ADP struct {
	username 	string
	password 	string
	url 			string
}
func NewADP(un, pass, url string) *ADP {
	adp := new(ADP)
	adp.username = un
	adp.password = pass
	adp.url = url
	return adp
}
func (adp *ADP) SendSms(phoneNumber string, txt string) (delivered bool, err error) {
	v := url.Values{}
	v.Set("username", adp.username)
	v.Set("password", adp.password)
	v.Set("dstaddress", phoneNumber)
	v.Set("srcaddress", "98200049112")
	v.Set("body", txt)
	rb := strings.NewReader(v.Encode())
	c := http.DefaultClient
	if req, err := http.NewRequest("POST", adp.url, rb); err != nil {
		log.Println("ADP::Send::Error 1::", err.Error())
		return false, err
	} else {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		if _, err := c.Do(req); err != nil {
			log.Println("ADP::Send::Error 2::", err.Error())
			return false, err
		}
	}
	return true, nil
}

