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

func TestCGroupsCPUQuota(t *testing.T) {
	testTable := []struct {
		name            string
		expectedQuota   float64
		expectedDefined bool
		shouldHaveError bool
	}{
		{
			name:            "set",
			expectedQuota:   6.0,
			expectedDefined: true,
			shouldHaveError: false,
		},
		{
			name:            "undefined",
			expectedQuota:   -1.0,
			expectedDefined: false,
			shouldHaveError: false,
		},
		{
			name:            "undefined-period",
			expectedQuota:   -1.0,
			expectedDefined: false,
			shouldHaveError: true,
		},
	}

	cgroups := make(CGroups)

	quota, defined, err := cgroups.CPUQuota()
	assert.Equal(t, -1.0, quota, "nonexistent")
	assert.Equal(t, false, defined, "nonexistent")
	assert.NoError(t, err, "nonexistent")

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, "v1", "cpu", tt.name)
		cgroups[_cgroupSubsysCPU] = NewCGroup(cgroupPath)

		quota, defined, err := cgroups.CPUQuota()
		assert.Equal(t, tt.expectedQuota, quota, tt.name)
		assert.Equal(t, tt.expectedDefined, defined, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}

func TestCGroupsMemLimit(t *testing.T) {
	testTable := []struct {
		name            string
		expectedLimit   int
		expectedDefined bool
		shouldHaveError bool
	}{
		{
			name:            "set",
			expectedLimit:   2147483648,
			expectedDefined: true,
			shouldHaveError: false,
		},
		{
			name:            "undefined",
			expectedLimit:   -1,
			expectedDefined: false,
			shouldHaveError: false,
		},
	}

	cgroups := make(CGroups)

	quota, defined, err := cgroups.MemLimit()
	assert.Equal(t, -1, quota, "nonexistent")
	assert.Equal(t, false, defined, "nonexistent")
	assert.NoError(t, err, "nonexistent")

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, "v1", "memory", tt.name)
		cgroups[_cgroupSubsysMemory] = NewCGroup(cgroupPath)

		quota, defined, err := cgroups.MemLimit()
		assert.Equal(t, tt.expectedLimit, quota, tt.name)
		assert.Equal(t, tt.expectedDefined, defined, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
