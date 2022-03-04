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
	part.Write(buf)
	writer.WriteField("token", r.token)
	writer.WriteField("profile_type", ptype)
	writer.WriteField("event_id", eventID)
	writer.WriteField("comment", reason)
	writer.Close()
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
	defer response.Body.Close()

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
