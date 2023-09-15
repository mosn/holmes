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
	"errors"
	"time"

	"mosn.io/holmes/reporters/pyroscope_reporter/flameql"
)

/*
	Copied from pyroscope-io/client
*/
var (
	ErrCloudTokenRequired = errors.New("Please provide an authentication token. You can find it here: https://pyroscope.io/cloud")
	ErrUpload             = errors.New("Failed to upload a profile")
	ErrUpgradeServer      = errors.New("Newer version of pyroscope server required (>= v0.3.1). Visit https://pyroscope.io/docs/golang/ for more information")
)

const (
	Pprof             UploadFormat = "pprof"
	Trie                           = "trie"
	DefaultUploadRate              = 10 * time.Second
)

type UploadFormat string
type Payload interface {
	Bytes() []byte
}

type ParserState int

const (
	ReservedTagKeyName = "__name__"
)

var (
	heapSampleTypes = map[string]*SampleType{
		"alloc_objects": {
			Units:      "objects",
			Cumulative: false,
		},
		"alloc_space": {
			Units:      "bytes",
			Cumulative: false,
		},
		"inuse_space": {
			Units:       "bytes",
			Aggregation: "average",
			Cumulative:  false,
		},
		"inuse_objects": {
			Units:       "objects",
			Aggregation: "average",
			Cumulative:  false,
		},
	}
	goroutineSampleTypes = map[string]*SampleType{
		"goroutine": {
			DisplayName: "goroutines",
			Units:       "goroutines",
			Aggregation: "average",
		},
	}
)

type SampleType struct {
	Units       string `json:"units,omitempty"`
	Aggregation string `json:"aggregation,omitempty"`
	DisplayName string `json:"display-name,omitempty"`
	Sampled     bool   `json:"sampled,omitempty"`
	Cumulative  bool   `json:"cumulative,omitempty"`
}

type UploadJob struct {
	Name             string
	StartTime        time.Time
	EndTime          time.Time
	SpyName          string
	SampleRate       uint32
	Units            string
	AggregationType  string
	Format           UploadFormat
	Profile          []byte
	PrevProfile      []byte
	SampleTypeConfig map[string]*SampleType
}

type RemoteConfig struct {
	AuthToken              string // holmes not used
	UpstreamThreads        int    // holmes not used
	UpstreamAddress        string
	UpstreamRequestTimeout time.Duration

	ManualStart bool // holmes not used
}

// mergeTagsWithAppName validates user input and merges explicitly specified
// tags with tags from app name.
//
// App name may be in the full form including tags (app.name{foo=bar,baz=qux}).
// Returned application name is always short, any tags that were included are
// moved to tags map. When merged with explicitly provided tags (config/CLI),
// last take precedence.
//
// App name may be an empty string. Tags must not contain reserved keys,
// the map is modified in place.
func mergeTagsWithAppName(appName string, tags map[string]string) (string, error) {
	k, err := flameql.ParseKey(appName)
	if err != nil {
		return "", err
	}
	for tagKey, tagValue := range tags {
		if flameql.IsTagKeyReserved(tagKey) {
			continue
		}
		if err = flameql.ValidateTagKey(tagKey); err != nil {
			return "", err
		}
		k.Add(tagKey, tagValue)
	}
	return k.Normalized(), nil
}
