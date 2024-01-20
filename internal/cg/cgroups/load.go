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
	"errors"
	"fmt"
)

const (
	_procPathCGroup    = "/proc/self/cgroup"
	_procPathMountInfo = "/proc/self/mountinfo"
)

// ErrCGroupFSNotFound indicates that the system is not using cgroups.
var ErrCGroupFSNotFound = errors.New("cgroupfs not found")

type ICGroups interface {
	// CPUQuota returns the CPU quota applied with the CPU cgroup controller.
	// `cpu.cfs_quota_us` was not set, the method returns `(-1, nil)`.
	CPUQuota() (float64, bool, error)
	// MemLimit returns the memory limit with memory cgroup controller.
	// `memory.max` wat not set, the method returns `(-1, nil)`
	MemLimit() (int, bool, error)
	// Version returns CGroup version.
	Version() string
}

func LoadCGroupsForCurrentProcess() (ICGroups, error) {
	return loadCGroups(_procPathMountInfo, _procPathCGroup)
}

func LoadCGroups(pid int) (ICGroups, error) {
	return loadCGroups(fmt.Sprintf("/proc/%d/mountinfo", pid), fmt.Sprintf("/proc/%d/cgroup", pid))
}

func loadCGroups(mountInfoPath, procCGroupPath string) (ICGroups, error) {
	mps, err := parseMountInfo(mountInfoPath)
	if err != nil {
		return nil, err
	}
	subsystems, err := parseCGroupSubsystems(procCGroupPath)
	if err != nil {
		return nil, err
	}

	for _, mp := range mps {
		if mp.FSType == _cgroupv2FSType {
			return newCGroups2(mp, subsystems)
		}
		if mp.FSType == _cgroupFSType {
			return newCGroups(mps, subsystems)
		}
	}
	return nil, ErrCGroupFSNotFound
}
