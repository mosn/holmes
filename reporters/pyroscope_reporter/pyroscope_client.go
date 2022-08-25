/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pyroscope_reporter

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"mosn.io/holmes"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	mlog "mosn.io/pkg/log"
)

/*
	Enable holmes to report pprof event to pyroscope as it's client.
*/

type PyroscopeReporter struct {
	AppName string
	Tags    map[string]string

	cfg    RemoteConfig
	client *http.Client
	Logger mlog.ErrorLogger
}

func NewPyroscopeReporter(AppName string, tags map[string]string, cfg RemoteConfig, logger mlog.ErrorLogger) (*PyroscopeReporter, error) {
	appName, err := mergeTagsWithAppName(AppName, tags)
	if err != nil {
		return nil, err
	}

	reporter := &PyroscopeReporter{
		cfg: cfg,
		client: &http.Client{
			Transport: &http.Transport{
				MaxConnsPerHost: cfg.UpstreamThreads,
			},
			Timeout: cfg.UpstreamRequestTimeout,
		},
		Logger:  logger,
		AppName: appName,
	}

	// todo: holmes doesn't support auth token temporary

	return reporter, nil
}

// uploadProfile copied from pyroscope client
func (r *PyroscopeReporter) uploadProfile(j *UploadJob) error {
	u, err := url.Parse(r.cfg.UpstreamAddress)
	if err != nil {
		return fmt.Errorf("url parse: %v", err)
	}

	body := &bytes.Buffer{}

	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile("profile", "profile.pprof")
	fw.Write(j.Profile) // nolint: errcheck
	if err != nil {
		return err
	}
	if j.PrevProfile != nil {
		fw, err = writer.CreateFormFile("prev_profile", "profile.pprof")
		fw.Write(j.PrevProfile) // nolint: errcheck
		if err != nil {
			return err
		}
	}
	writer.Close() // nolint: errcheck

	q := u.Query()
	q.Set("name", j.Name)
	// TODO: I think these should be renamed to startTime / endTime
	q.Set("from", strconv.Itoa(int(j.StartTime.Unix())))
	q.Set("until", strconv.Itoa(int(j.EndTime.Unix())))
	q.Set("spyName", j.SpyName)
	q.Set("sampleRate", strconv.Itoa(int(j.SampleRate)))
	q.Set("units", j.Units)
	q.Set("aggregationType", j.AggregationType)

	u.Path = path.Join(u.Path, "/ingest")
	u.RawQuery = q.Encode()

	r.Logger.Debugf("uploading at %s", u.String())

	// new a request for the job
	request, err := http.NewRequest("POST", u.String(), body)
	//r.Logger.Debugf("body is %s", body.String())
	if err != nil {
		return fmt.Errorf("new http request: %v", err)
	}
	contentType := writer.FormDataContentType()
	r.Logger.Debugf("content type: %s", contentType)
	request.Header.Set("Content-Type", contentType)
	// request.Header.Set("Content-Type", "binary/octet-stream+"+string(j.Format))

	if r.cfg.AuthToken != "" {
		request.Header.Set("Authorization", "Bearer "+r.cfg.AuthToken)
	}

	// do the request and get the response
	response, err := r.client.Do(request)
	if err != nil {
		return fmt.Errorf("do http request: %v", err)
	}
	defer response.Body.Close() // nolint: errcheck

	// read all the response body
	_, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response body: %v", err)
	}

	if response.StatusCode == 422 {
		return ErrUpgradeServer
	}
	if response.StatusCode != 200 {
		return ErrUpload
	}

	return nil
}

func (r *PyroscopeReporter) Report(ptype string, filename string, reason holmes.ReasonType, eventID string, sampleTime time.Time, pprofBytes []byte, scene holmes.Scene) error {
	endTime := sampleTime.Truncate(DefaultUploadRate)
	startTime := endTime.Add(-DefaultUploadRate)
	_, _, _, _, _ = ptype, filename, reason, eventID, scene
	j := &UploadJob{
		Name:            r.AppName,
		StartTime:       startTime,
		EndTime:         endTime,
		SpyName:         "gospy",
		SampleRate:      100,
		Units:           "samples",
		AggregationType: "sum",
		Format:          Pprof,
		Profile:         pprofBytes,
	}

	if err := r.uploadProfile(j); err != nil {
		return err
	}
	return nil
}
