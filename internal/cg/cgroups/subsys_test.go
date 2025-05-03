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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCGroupSubsysFromLine(t *testing.T) {
	testTable := []struct {
		name           string
		line           string
		expectedSubsys *CGroupSubsys
	}{
		{
			name: "single-subsys",
			line: "1:cpu:/",
			expectedSubsys: &CGroupSubsys{
				ID:         1,
				Subsystems: []string{"cpu"},
				Name:       "/",
			},
		},
		{
			name: "multi-subsys",
			line: "8:cpu,cpuacct,cpuset:/docker/1234567890abcdef",
			expectedSubsys: &CGroupSubsys{
				ID:         8,
				Subsystems: []string{"cpu", "cpuacct", "cpuset"},
				Name:       "/docker/1234567890abcdef",
			},
		},
		{
			name: "multi-subsys",
			line: "12:cpu,cpuacct:/system.slice/containerd.service/kubepods-besteffort-podb41662f7_b03a_4c65_8ef9_6e4e55c3cf27.slice:cri-containerd:1753b7cbbf62734d812936961224d5bc0cf8f45214e0d5cdd1a781a053e7c48f",
			expectedSubsys: &CGroupSubsys{
				ID:         12,
				Subsystems: []string{"cpu", "cpuacct"},
				Name:       "/system.slice/containerd.service/kubepods-besteffort-podb41662f7_b03a_4c65_8ef9_6e4e55c3cf27.slice:cri-containerd:1753b7cbbf62734d812936961224d5bc0cf8f45214e0d5cdd1a781a053e7c48f",
			},
		},
	}

	for _, tt := range testTable {
		subsys, err := NewCGroupSubsysFromLine(tt.line)
		assert.Equal(t, tt.expectedSubsys, subsys, tt.name)
		assert.NoError(t, err, tt.name)
	}
}

func TestNewCGroupSubsysFromLineErr(t *testing.T) {
	lines := []string{
		"1:cpu",
		"not-a-number:cpu:/",
	}
	_, parseError := strconv.Atoi("not-a-number")

	testTable := []struct {
		name          string
		line          string
		expectedError error
	}{
		{
			name:          "fewer-fields",
			line:          lines[0],
			expectedError: cgroupSubsysFormatInvalidError{lines[0]},
		},
		{
			name:          "illegal-id",
			line:          lines[1],
			expectedError: parseError,
		},
	}

	for _, tt := range testTable {
		subsys, err := NewCGroupSubsysFromLine(tt.line)
		assert.Nil(t, subsys, tt.name)
		assert.Equal(t, tt.expectedError, err, tt.name)
	}
}
