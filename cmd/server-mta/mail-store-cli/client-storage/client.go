package client_storage

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"git.ronaksoft.com/nested/server/model"
	"git.ronaksoft.com/nested/server/pkg/protocol"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type File interface {
	io.Reader
	Stat() (os.FileInfo, error)
}

type UploadedFile struct {
	Type                string             `json:"type"`
	Name                string             `json:"name"`
	Size                int64              `json:"size"`
	Thumbs              nested.Picture     `json:"thumbs,omitempty"`
	UniversalId         nested.UniversalID `json:"universal_id"`
	ExpirationTimestamp uint64             `json:"expiration_timestamp"`
}

type UploadOutput struct {
	Files []UploadedFile `json:"files"`
}

type UploadResponse struct {
	Payload UploadOutput `json:"data"`
}

type Client struct {
	url         string
	apiKey      string
	insecureTls bool
}

func NewClient(url, apiKey string, insecure bool) (*Client, error) {
	c := &Client{
		url:         url,
		apiKey:      apiKey,
		insecureTls: insecure,
	}
	return c, nil
}

// Uploads files
func (c Client) Upload(uploadType string, files ...File) (*UploadOutput, error) {
	var req *http.Request
	var res *http.Response

	pr, pw := io.Pipe()
	frm := multipart.NewWriter(pw)

	go func() {
		for k, f := range files {
			var fname string
			if info, err := f.Stat(); err != nil {
				fname = fmt.Sprintf("File%d", k)
			} else {
				fname = info.Name()
			}

			if p, err := frm.CreateFormFile("files[]", fname); err != nil {
				fmt.Println(err.Error())
				pw.CloseWithError(err)
			} else if _, err := io.Copy(p, f); err != nil {
				fmt.Println(err.Error())
				pw.CloseWithError(err)
			}
		}

		if err := frm.Close(); err != nil {
			fmt.Println(err.Error())
			pw.CloseWithError(err)

		} else {
			pw.Close()
		}
	}()

	if r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/system/upload/%s/%s", c.url, uploadType, c.apiKey), pr); err != nil {
		fmt.Println(err.Error())
		return nil, err

	} else {
		req = r
		req.Header.Set("Content-Type", fmt.Sprintf("%s; boundary=\"%s\"", "multipart/form-data", frm.Boundary()))
	}

	var client *http.Client
	if c.insecureTls {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	} else {
		client = http.DefaultClient
	}

	if r, err := client.Do(req); err != nil {
		fmt.Println(err.Error())

		return nil, err
	} else if http.StatusOK != r.StatusCode {
		fmt.Println("Upload request response error")

		sErr := protocol.Error{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&sErr); err != nil {
			return nil, protocol.NewUnknownError(protocol.D{"error": err, "status": r.Status})
		} else {
			return nil, sErr
		}
	} else {
		res = r
	}

	defer res.Body.Close()

	response := new(UploadResponse)

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(response); err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return &response.Payload, nil
}
