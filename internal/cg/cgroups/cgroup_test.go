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
)

func TestCGroupParamPath(t *testing.T) {
	cgroup := NewCGroup("/sys/fs/cgroup/cpu")
	assert.Equal(t, "/sys/fs/cgroup/cpu", cgroup.Path())
	assert.Equal(t, "/sys/fs/cgroup/cpu/cpu.cfs_quota_us", cgroup.ParamPath("cpu.cfs_quota_us"))
}

func TestCGroupReadFirstLine(t *testing.T) {
	testTable := []struct {
		name            string
		paramName       string
		expectedContent string
		shouldHaveError bool
	}{
		{
			name:            "set",
			paramName:       "cpu.cfs_period_us",
			expectedContent: "100000",
			shouldHaveError: false,
		},
		{
			name:            "absent",
			paramName:       "cpu.stat",
			expectedContent: "",
			shouldHaveError: true,
		},
		{
			name:            "empty",
			paramName:       "cpu.cfs_quota_us",
			expectedContent: "",
			shouldHaveError: true,
		},
	}

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, "v1", "cpu", tt.name)
		cgroup := NewCGroup(cgroupPath)

		content, err := cgroup.readFirstLine(tt.paramName)
		assert.Equal(t, tt.expectedContent, content, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}

func TestCGroupReadInt(t *testing.T) {
	testTable := []struct {
		name            string
		paramName       string
		expectedValue   int
		shouldHaveError bool
	}{
		{
			name:            "set",
			paramName:       "cpu.cfs_period_us",
			expectedValue:   100000,
			shouldHaveError: false,
		},
		{
			name:            "empty",
			paramName:       "cpu.cfs_quota_us",
			expectedValue:   0,
			shouldHaveError: true,
		},
		{
			name:            "invalid",
			paramName:       "cpu.cfs_quota_us",
			expectedValue:   0,
			shouldHaveError: true,
		},
		{
			name:            "absent",
			paramName:       "cpu.cfs_quota_us",
			expectedValue:   0,
			shouldHaveError: true,
		},
	}

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, "v1", "cpu", tt.name)
		cgroup := NewCGroup(cgroupPath)

		value, err := cgroup.readInt(tt.paramName)
		assert.Equal(t, tt.expectedValue, value, "%s/%s", tt.name, tt.paramName)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
