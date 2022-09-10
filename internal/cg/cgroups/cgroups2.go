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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	// _cgroupv2CPUMax is the file name for the CGroup-V2 CPU max and period
	// parameter.
	_cgroupv2CPUMax = "cpu.max"
	_cgroupv2MEMMax = "memory.max"
	// _cgroupFSType is the Linux CGroup-V2 file system type used in
	// `/proc/$PID/mountinfo`.
	_cgroupv2FSType = "cgroup2"

	_cgroupV2CPUMaxDefaultPeriod = 100000
	_cgroupV2CPUMaxQuotaMax      = "max"
	_cgroupV2MEMMaxDefault       = "max"
)

const (
	_cgroupv2CPUMaxQuotaIndex = iota
	_cgroupv2CPUMaxPeriodIndex
)

// ErrNotV2 indicates that the system is not using cgroups2.
var ErrNotV2 = errors.New("not using cgroups2")

// CGroups2 provides access to cgroups data for systems using cgroups2.
type CGroups2 struct {
	mountPoint string
	groupPath  string
}

func newCGroups2(mountInfo *MountPoint, subsystems map[string]*CGroupSubsys) (*CGroups2, error) {
	// Find v2 subsystem by looking for the `0` id
	var v2subsys *CGroupSubsys
	for _, subsys := range subsystems {
		if subsys.ID == 0 {
			v2subsys = subsys
			break
		}
	}

	if v2subsys == nil {
		return nil, ErrNotV2
	}

	return &CGroups2{
		mountPoint: mountInfo.MountPoint,
		groupPath:  v2subsys.Name,
	}, nil
}

// MemLimit returns the memory limit with memory cgroup controller.
// `memory.max` wat not set, the method returns `(-1, nil)`
func (cg *CGroups2) MemLimit() (int, bool, error) {
	memMaxParams, err := os.Open(path.Join(cg.mountPoint, cg.groupPath, _cgroupv2MEMMax))
	if err != nil {
		if os.IsNotExist(err) {
			return -1, false, nil
		}
		return -1, false, err
	}
	defer memMaxParams.Close() // nolint: errcheck

	scanner := bufio.NewScanner(memMaxParams)
	if scanner.Scan() {
		text := scanner.Text()
		if text == _cgroupV2MEMMaxDefault {
			return -1, false, nil
		}
		max, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return -1, false, fmt.Errorf("parse max memory failed, invalid format. %w", err)
		}
		return max, true, nil
	}
	if err = scanner.Err(); err != nil {
		return -1, false, err
	}

	return 0, false, io.ErrUnexpectedEOF
}

// CPUQuota returns the CPU quota applied with the CPU cgroup2 controller.
// It is a result of reading cpu quota and period from cpu.max file.
// It will return `cpu.max / cpu.period`. If cpu.max is set to max, it returns
// (-1, false, nil)
func (cg *CGroups2) CPUQuota() (float64, bool, error) {
	cpuMaxParams, err := os.Open(path.Join(cg.mountPoint, cg.groupPath, _cgroupv2CPUMax))
	if err != nil {
		if os.IsNotExist(err) {
			return -1, false, nil
		}
		return -1, false, err
	}
	defer cpuMaxParams.Close() // nolint: errcheck

	scanner := bufio.NewScanner(cpuMaxParams)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || len(fields) > 2 {
			return -1, false, fmt.Errorf("invalid format")
		}

		if fields[_cgroupv2CPUMaxQuotaIndex] == _cgroupV2CPUMaxQuotaMax {
			return -1, false, nil
		}

		max, err := strconv.Atoi(fields[_cgroupv2CPUMaxQuotaIndex])
		if err != nil {
			return -1, false, err
		}

		var period int
		if len(fields) == 1 {
			period = _cgroupV2CPUMaxDefaultPeriod
		} else {
			period, err = strconv.Atoi(fields[_cgroupv2CPUMaxPeriodIndex])
			if err != nil {
				return -1, false, err
			}
		}

		return float64(max) / float64(period), true, nil
	}

	if err = scanner.Err(); err != nil {
		return -1, false, err
	}

	return 0, false, io.ErrUnexpectedEOF
}

// Version return version of cgroupfs.
func (cg *CGroups2) Version() string {
	return _cgroupv2FSType
}
