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

func TestLoadCGroups(t *testing.T) {
	cgroupsProcCGroupPath := filepath.Join(testDataProcPath, "cgroup", "cgroup")
	cgroupsProcMountInfoPath := filepath.Join(testDataProcPath, "mountinfo", "mountinfo")

	testTable := []struct {
		subsys string
		path   string
	}{
		{_cgroupSubsysCPU, "/sys/fs/cgroup/cpu,cpuacct"},
		{_cgroupSubsysCPUAcct, "/sys/fs/cgroup/cpu,cpuacct"},
		{_cgroupSubsysCPUSet, "/sys/fs/cgroup/cpuset"},
		{_cgroupSubsysMemory, "/sys/fs/cgroup/memory/large"},
	}

	icg, err := loadCGroups(cgroupsProcMountInfoPath, cgroupsProcCGroupPath)
	assert.NoError(t, err)
	assert.Equal(t, _cgroupFSType, icg.Version())

	cgroups, ok := icg.(CGroups)
	assert.True(t, ok)
	assert.Equal(t, len(testTable), len(cgroups))
	assert.NoError(t, err)

	for _, tt := range testTable {
		cgroup, exists := cgroups[tt.subsys]
		assert.Equal(t, true, exists, "%q expected to present in `cgroups`", tt.subsys)
		assert.Equal(t, tt.path, cgroup.path, "%q expected for `cgroups[%q].path`, got %q", tt.path, tt.subsys, cgroup.path)
	}
}

func TestLoadCGroups2(t *testing.T) {
	tests := []struct {
		procCgroup string
		wantPath   string
		wantError  error
	}{
		{
			procCgroup: "cgroup-no-match",
			wantError:  ErrNotV2,
		},
		{
			procCgroup: "cgroup-root",
			wantPath:   "/",
		},
		{
			procCgroup: "cgroup-subdir",
			wantPath:   "/Example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.procCgroup, func(t *testing.T) {
			mountInfoPath := filepath.Join(testDataProcPath, "mountinfo", "mountinfo-v2")
			procCgroupPath := filepath.Join(testDataProcPath, "cgroup", tt.procCgroup)
			cg, err := loadCGroups(mountInfoPath, procCgroupPath)
			if tt.wantError == nil {
				require.NoError(t, err)
				assert.Equal(t, _cgroupv2FSType, cg.Version())
				cgroups, ok := cg.(*CGroups2)
				assert.True(t, ok)
				assert.Equal(t, tt.wantPath, cgroups.groupPath)
			} else {
				assert.ErrorIs(t, err, tt.wantError)
			}

		})
	}
}

func TestLoadCGroupsWrongFile(t *testing.T) {
	tests := []struct {
		name           string
		mountInfoPath  string
		procCGroupPath string
	}{
		{
			name:           "mountInfoNotFound",
			mountInfoPath:  "non-existing-file",
			procCGroupPath: "/dev/null",
		},
		{
			name:           "procCGroupNotFound",
			mountInfoPath:  "/dev/null",
			procCGroupPath: "non-existing-file",
		},
		{
			name:           "invalid-cgroup",
			mountInfoPath:  "/dev/null",
			procCGroupPath: filepath.Join(testDataProcPath, "cgroup", "invalid"),
		},
		{
			name:           "invalid-mountinfo",
			mountInfoPath:  filepath.Join(testDataProcPath, "mountinfo", "invalid"),
			procCGroupPath: "/dev/null",
		},
		{
			name:           "untranslatable",
			mountInfoPath:  filepath.Join(testDataProcPath, "mountinfo", "untranslatable"),
			procCGroupPath: filepath.Join(testDataProcPath, "cgroup", "untranslatable"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg, err := loadCGroups(tt.mountInfoPath, tt.procCGroupPath)
			assert.Nil(t, cg)
			assert.Error(t, err)
		})
	}
}
