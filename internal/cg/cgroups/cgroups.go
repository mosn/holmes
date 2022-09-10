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

const (
	// _cgroupFSType is the Linux CGroup file system type used in
	// `/proc/$PID/mountinfo`.
	_cgroupFSType = "cgroup"
	// _cgroupSubsysCPU is the CPU CGroup subsystem.
	_cgroupSubsysCPU = "cpu"
	// _cgroupSubsysMemory is the Memory CGroup subsystem.
	_cgroupSubsysMemory = "memory"

	// _cgroupCPUCFSQuotaUsParam is the file name for the CGroup CFS quota
	// parameter.
	_cgroupCPUCFSQuotaUsParam = "cpu.cfs_quota_us"
	// _cgroupCPUCFSPeriodUsParam is the file name for the CGroup CFS period
	// parameter.
	_cgroupCPUCFSPeriodUsParam = "cpu.cfs_period_us"
	// _cgroupMemLimitParam is the file name for the CGroup CFS memory
	// parameter.
	_cgroupMemLimitParam = "memory.limit_in_bytes"
)

// CGroups is a map that associates each CGroup with its subsystem name.
type CGroups map[string]CGroup

func newCGroups(mountInfo []*MountPoint, subsystems map[string]*CGroupSubsys) (CGroups, error) {
	cgroups := make(CGroups)
	for _, mp := range mountInfo {
		if mp.FSType != _cgroupFSType {
			continue
		}
		for _, opt := range mp.SuperOptions {
			subsys, exists := subsystems[opt]
			if !exists {
				continue
			}

			cgroupPath, err := mp.Translate(subsys.Name)
			if err != nil {
				return nil, err
			}
			cgroups[opt] = NewCGroup(cgroupPath)
		}

	}
	return cgroups, nil
}

// MemLimit returns the memory limit with memory cgroup controller.
// `memory.max` wat not set, the method returns `(-1, nil)`
func (cg CGroups) MemLimit() (int, bool, error) {
	memCGroup, ok := cg[_cgroupSubsysMemory]
	if !ok {
		return -1, false, nil
	}
	memLimit, err := memCGroup.readInt(_cgroupMemLimitParam)
	if defined := memLimit > 0; err != nil || !defined {
		return -1, defined, err
	}
	return memLimit, true, nil
}

// CPUQuota returns the CPU quota applied with the CPU cgroup controller.
// It is a result of `cpu.cfs_quota_us / cpu.cfs_period_us`. If the value of
// `cpu.cfs_quota_us` was not set (-1), the method returns `(-1, nil)`.
func (cg CGroups) CPUQuota() (float64, bool, error) {
	cpuCGroup, exists := cg[_cgroupSubsysCPU]
	if !exists {
		return -1, false, nil
	}

	cfsQuotaUs, err := cpuCGroup.readInt(_cgroupCPUCFSQuotaUsParam)
	if defined := cfsQuotaUs > 0; err != nil || !defined {
		return -1, defined, err
	}

	cfsPeriodUs, err := cpuCGroup.readInt(_cgroupCPUCFSPeriodUsParam)
	if err != nil {
		return -1, false, err
	}

	return float64(cfsQuotaUs) / float64(cfsPeriodUs), true, nil
}

func (cg CGroups) Version() string {
	return _cgroupFSType
}
