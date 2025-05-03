// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package cgroups

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCGroupsCPUQuotaV2(t *testing.T) {
	tests := []struct {
		name    string
		want    float64
		wantOK  bool
		wantErr string
	}{
		{
			name:   "set",
			want:   2.5,
			wantOK: true,
		},
		{
			name:   "unset",
			want:   -1.0,
			wantOK: false,
		},
		{
			name:   "only-max",
			want:   5.0,
			wantOK: true,
		},
		{
			name:    "invalid-max",
			wantErr: `parsing "asdf": invalid syntax`,
		},
		{
			name:    "invalid-period",
			wantErr: `parsing "njn": invalid syntax`,
		},
		{
			name:   "nonexistent",
			want:   -1.0,
			wantOK: false,
		},
		{
			name:    "empty",
			wantErr: "unexpected EOF",
		},
		{
			name:    "too-few-fields",
			wantErr: "invalid format",
		},
		{
			name:    "too-many-fields",
			wantErr: "invalid format",
		},
	}

	mountPoint := filepath.Join(testDataCGroupsPath, "v2")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quota, defined, err := (&CGroups2{
				mountPoint: mountPoint,
				groupPath:  tt.name,
			}).CPUQuota()

			if len(tt.wantErr) > 0 {
				require.Error(t, err, tt.name)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err, tt.name)
				assert.Equal(t, tt.want, quota, tt.name)
				assert.Equal(t, tt.wantOK, defined, tt.name)
			}
		})
	}
}

func TestCGroupsMemLimitV2(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantOK  bool
		wantErr string
	}{
		{
			name:   "set",
			want:   2147483648,
			wantOK: true,
		},
		{
			name:   "unset",
			want:   -1,
			wantOK: false,
		},
		{
			name:    "invalid-max",
			wantErr: `parsing "asdf": invalid syntax`,
		},
		{
			name:   "nonexistent",
			want:   -1,
			wantOK: false,
		},
		{
			name:    "empty",
			wantErr: "unexpected EOF",
		},
		{
			name:    "too-many-fields",
			wantErr: `parsing "250000 100000 100": invalid syntax`,
		},
	}

	mountPoint := filepath.Join(testDataCGroupsPath, "v2")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memLimit, defined, err := (&CGroups2{
				mountPoint: mountPoint,
				groupPath:  tt.name,
			}).MemLimit()

			if len(tt.wantErr) > 0 {
				require.Error(t, err, tt.name)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err, tt.name)
				assert.Equal(t, tt.want, memLimit, tt.name)
				assert.Equal(t, tt.wantOK, defined, tt.name)
			}
		})
	}
}
