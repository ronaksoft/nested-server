package nestedServiceAuth

import (
	"net/url"
	"strings"
	"net/http"
	"fmt"
	"log"
	"encoding/json"
	"io/ioutil"
	"time"
	"math/rand"
)

type TwilioClient struct {
	accountSID     string
	accountToken   string
	smsNumber      string
	messageUrl     string
	callUrl        string
	lookupPhoneUrl string
	callNumbers    []string
}

func NewTwilioClient(sid, token, smsNumber string, callNumbers []string) *TwilioClient {
	tc := new(TwilioClient)
	tc.accountSID = sid
	tc.accountToken = token
	tc.smsNumber = smsNumber
	tc.callNumbers = callNumbers
	tc.messageUrl = fmt.Sprintf("https://api.twilio.nested.me/2010-04-01/Accounts/%s/Messages", tc.accountSID)
	tc.callUrl = fmt.Sprintf("https://api.twilio.nested.me/2010-04-01/Accounts/%s/Calls", tc.accountSID)
	tc.lookupPhoneUrl = fmt.Sprintf("https://lookups.twilio.nested.me/v1/PhoneNumbers/")
	return tc
}

func (tc *TwilioClient) SendSms(phoneNumber, txt string) (delivered bool, err error) {
	v := url.Values{}
	v.Set("To", phoneNumber)
	v.Set("From", tc.smsNumber)
	v.Set("Body", fmt.Sprintf("Nested verification code is: %s", txt))
	rb := strings.NewReader(v.Encode())
	c := http.DefaultClient
	if req, err := http.NewRequest("POST", tc.messageUrl, rb); err != nil {
		return false, err
	} else {
		req.SetBasicAuth(tc.accountSID, tc.accountToken)
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		if _, err := c.Do(req); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (tc *TwilioClient) SendCall(phoneNumber, callbackUrl string) (err error) {
	v := url.Values{}
	v.Set("To", phoneNumber)
	rand.Seed(time.Now().UnixNano())
	v.Set("From", tc.callNumbers[rand.Intn(len(tc.callNumbers))])
	v.Set("Url", callbackUrl)
	v.Set("Method", "GET")
	rb := strings.NewReader(v.Encode())
	c := http.DefaultClient
	req, _ := http.NewRequest("POST", tc.callUrl, rb)
	req.SetBasicAuth(tc.accountSID, tc.accountToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if _, err := c.Do(req); err != nil {
		return err
	}
	return nil
}

func (tc *TwilioClient) LookupPhone(phone string) (string, string, error) {
	c := http.DefaultClient
	twilioResponse := struct {
		CountryCode    string `json:"country_code"`
		PhoneNumber    string `json:"phone_number"`
		NationalFormat string `json:"national_format"`
	}{}
	getURL := fmt.Sprintf("%s%s", tc.lookupPhoneUrl, phone)
	req, _ := http.NewRequest("GET", getURL, nil)
	req.SetBasicAuth(tc.accountSID, tc.accountToken)
	req.Header.Add("Accept", "application/json")
	if resp, err := c.Do(req); err != nil {
		log.Println("lookupPhone::Error1::", err.Error())
		return "", "", err
	} else {
		if d, err := ioutil.ReadAll(resp.Body); err != nil {
			log.Println("lookupPhone::Error2::", err)
			return "", "", err
		} else {
			if err := json.Unmarshal(d, &twilioResponse); err != nil {
				log.Println("Response:", string(d))
				log.Println("lookupPhone::Error3::", err.Error())
				return "", "", err
			}
		}
	}
	return twilioResponse.CountryCode, twilioResponse.PhoneNumber, nil
}
