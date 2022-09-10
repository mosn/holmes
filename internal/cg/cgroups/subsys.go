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
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	_cgroupSep       = ":"
	_cgroupSubsysSep = ","
)

const (
	_csFieldIDID = iota
	_csFieldIDSubsystems
	_csFieldIDName
	_csFieldCount
)

type cgroupSubsysFormatInvalidError struct {
	line string
}

func (err cgroupSubsysFormatInvalidError) Error() string {
	return fmt.Sprintf("invalid format for CGroupSubsys: %q", err.line)
}

// CGroupSubsys represents the data structure for entities in
// `/proc/$PID/cgroup`. See also proc(5) for more information.
type CGroupSubsys struct {
	ID         int
	Subsystems []string
	Name       string
}

// NewCGroupSubsysFromLine returns a new *CGroupSubsys by parsing a string in
// the format of `/proc/$PID/cgroup`
func NewCGroupSubsysFromLine(line string) (*CGroupSubsys, error) {
	fields := strings.SplitN(line, _cgroupSep, _csFieldCount)

	if len(fields) != _csFieldCount {
		return nil, cgroupSubsysFormatInvalidError{line}
	}

	id, err := strconv.Atoi(fields[_csFieldIDID])
	if err != nil {
		return nil, err
	}

	cgroup := &CGroupSubsys{
		ID:         id,
		Subsystems: strings.Split(fields[_csFieldIDSubsystems], _cgroupSubsysSep),
		Name:       fields[_csFieldIDName],
	}

	return cgroup, nil
}

// parseCGroupSubsystems parses procPathCGroup (usually at `/proc/$PID/cgroup`)
// and returns a new map[string]*CGroupSubsys.
func parseCGroupSubsystems(procPathCGroup string) (map[string]*CGroupSubsys, error) {
	cgroupFile, err := os.Open(procPathCGroup)
	if err != nil {
		return nil, err
	}
	defer cgroupFile.Close() // nolint: errcheck

	scanner := bufio.NewScanner(cgroupFile)
	subsystems := make(map[string]*CGroupSubsys)

	for scanner.Scan() {
		cgroup, err := NewCGroupSubsysFromLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, subsys := range cgroup.Subsystems {
			subsystems[subsys] = cgroup
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return subsystems, nil
}
