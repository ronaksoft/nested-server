package lmtp

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/global"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

/*
   Creation Time: 2021 - Aug - 07
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type File interface {
	io.Reader
	Stat() (os.FileInfo, error)
}

type UploadedFile struct {
	Type                string             `json:"type"`
	Name                string             `json:"name"`
	Size                int64              `json:"size"`
	Thumbs              nested.Picture     `json:"thumbs,omitempty"`
	UniversalID         nested.UniversalID `json:"universal_id"`
	ExpirationTimestamp uint64             `json:"expiration_timestamp"`
}

type UploadOutput struct {
	Files []UploadedFile `json:"files"`
}

type UploadResponse struct {
	Payload UploadOutput `json:"data"`
}

type MultipartFile struct {
	file      UploadedFile
	content   string
	contentID string
}

type uploadClient struct {
	url    string
	apiKey string
	client *http.Client
}

func newUploadClient(url, apiKey string, skipVerify bool) (*uploadClient, error) {
	c := &uploadClient{
		url:    url,
		apiKey: apiKey,
	}
	if strings.HasPrefix(url, "https://") {
		c.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
			},
		}
	} else {
		c.client = http.DefaultClient
	}
	return c, nil
}

func (c *uploadClient) upload(uploadType string, files ...File) (*UploadOutput, error) {
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
		return nil, err

	} else {
		req = r
		req.Header.Set("Content-Type", fmt.Sprintf("%s; boundary=\"%s\"", "multipart/form-data", frm.Boundary()))
	}

	if r, err := c.client.Do(req); err != nil {
		return nil, err
	} else if http.StatusOK != r.StatusCode {
		sErr := global.Error{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&sErr); err != nil {
			return nil, global.NewUnknownError(global.DataPayload{"error": err, "status": r.Status})
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

func (c *uploadClient) uploadFile(filename, uploaderId, status string, ownerIds []string, r io.Reader) (*UploadedFile, error) {
	f := &MyFile{
		r:        r,
		name:     filename,
		uploader: uploaderId,
		status:   status,
		owners:   ownerIds,
	}

	if res, err := c.upload(nested.UploadTypeFile, f); err != nil {
		return nil, err
	} else if 0 == len(res.Files) {
		return nil, errors.New("file not uploaded")
	} else {
		return &res.Files[0], nil
	}
}

type MyFile struct {
	r        io.Reader
	name     string
	uploader string
	status   string
	owners   []string
}

func (f *MyFile) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func (f MyFile) Name() string {
	return f.name
}

func (f MyFile) Size() int64 {
	return 0
}

func (f MyFile) Mode() os.FileMode {
	return os.ModePerm
}

func (f MyFile) ModTime() time.Time {
	return time.Now()
}

func (f MyFile) IsDir() bool {
	return false
}

func (f MyFile) Sys() interface{} {
	return nil
}

func (f *MyFile) Stat() (os.FileInfo, error) {
	return f, nil
}
