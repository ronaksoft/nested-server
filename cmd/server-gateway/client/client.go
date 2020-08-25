package nestedGateway

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"git.ronaksoft.com/nested/server/model"
	"git.ronaksoft.com/nested/server/pkg/protocol"

	"go.uber.org/zap"
)

var (
	_Log      *zap.Logger
	_LogLevel zap.AtomicLevel
)

type File interface {
	io.Reader
	Stat() (os.FileInfo, error)
}

type Client struct {
	url         string
	apiKey      string
	insecureTls bool
}

func NewClient(url, apiKey string, insecure bool) (*Client, error) {
	if nil == _Log {
		_LogLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
		zap.NewProductionConfig()
		logConfig := zap.NewProductionConfig()
		logConfig.Encoding = "console"
		logConfig.Level = _LogLevel
		if v, err := logConfig.Build(); err != nil {
			os.Exit(1)
		} else {
			_Log = v
		}
	}

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
				_Log.Warn(err.Error())
				pw.CloseWithError(err)
			} else if _, err := io.Copy(p, f); err != nil {
				_Log.Warn(err.Error())
				pw.CloseWithError(err)
			}
		}

		if err := frm.Close(); err != nil {
			_Log.Warn(err.Error())
			pw.CloseWithError(err)

		} else {
			pw.Close()
		}
	}()

	if r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/system/upload/%s/%s", c.url, uploadType, c.apiKey), pr); err != nil {
		_Log.Warn(err.Error())
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
		_Log.Warn(err.Error())

		return nil, err
	} else if http.StatusOK != r.StatusCode {
		_Log.Warn("Upload request response error")

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
		_Log.Warn(err.Error())
		return nil, err
	}

	return &response.Payload, nil
}

// Prepare Download Query
func (c Client) PrepareDownload(uid string) (*http.Request, error) {
	if r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/system/download/%s/%s", c.url, c.apiKey, uid), nil); err != nil {
		_Log.Warn(err.Error())
		return nil, err

	} else {
		return r, nil
	}
}

// Download files
func (c Client) Download(uid string) (io.Reader, error) {
	var req *http.Request
	if r, err := c.PrepareDownload(uid); err != nil {
		return nil, err
	} else {
		req = r
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
		_Log.Warn(err.Error())

		return nil, err
	} else if http.StatusOK != r.StatusCode {

		_Log.Warn("download request response error",
			zap.String("STATUS", r.Status),
		)

		sErr := protocol.Error{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&sErr); err != nil {
			return nil, protocol.NewUnknownError(protocol.D{"error": err, "status": r.Status})
		} else {
			return nil, sErr
		}
	} else {
		return r.Body, nil
	}
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

func NewUploadResponse(data UploadOutput) UploadResponse {
	return UploadResponse{
		Payload: data,
	}
}
func (r UploadResponse) Type() protocol.DatagramType {
	return protocol.DATAGRAM_TYPE_RESPONSE
}
func (r UploadResponse) Status() protocol.ResponseStatus {
	return protocol.STATUS_SUCCESS
}
func (r UploadResponse) Data() protocol.Payload {
	return r.Payload
}
func (r UploadResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":   r.Type(),
		"status": r.Status(),
		"data":   r.Data(),
	})
}
