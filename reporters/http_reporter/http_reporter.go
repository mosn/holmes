package http_reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mosn/holmes"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

type HttpReporter struct {
	token string
	url   string
}

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewReporter(token string, url string) holmes.ProfileReporter {
	return &HttpReporter{
		token: token,
		url:   url,
	}
}

func (r *HttpReporter) Report(ptype string, buf []byte, reason string, eventID string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("profile", "go-pprof-profile")
	if err != nil {
		return fmt.Errorf("create form File err: %v", err)
	}
	if _, err := part.Write(buf); err != nil {
		return fmt.Errorf("write buf to file part err: %v", err)
	}
	writer.WriteField("token", r.token)      // nolint: errcheck
	writer.WriteField("profile_type", ptype) // nolint: errcheck
	writer.WriteField("event_id", eventID)   // nolint: errcheck
	writer.WriteField("comment", reason)     // nolint: errcheck
	writer.Close()                           // nolint: errcheck
	request, err := http.NewRequest("POST", r.url, body)
	if err != nil {
		return fmt.Errorf("NewRequest err: %v", err)
	}

	request.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("do Request err: %v", err)
	}
	defer response.Body.Close() // nolint: errcheck

	respContent, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response err: %v", err)
	}

	rsp := &Response{}
	if err := json.Unmarshal(respContent, rsp); err != nil {
		return fmt.Errorf("failed to decode resp json: %v", err)
	}

	if rsp.Code != 1 {
		return fmt.Errorf("code: %d, msg: %s", rsp.Code, rsp.Message)
	}
	return nil
}
