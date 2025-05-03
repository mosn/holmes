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
	"path/filepath"
	"strconv"
	"strings"
)

type mountPointFormatInvalidError struct {
	line string
}

func (err mountPointFormatInvalidError) Error() string {
	return fmt.Sprintf("invalid format for MountPoint: %q", err.line)
}

type pathNotExposedFromMountPointError struct {
	mountPoint string
	root       string
	path       string
}

func (err pathNotExposedFromMountPointError) Error() string {
	return fmt.Sprintf("path %q is not a descendant of mount point root %q and cannot be exposed from %q", err.path, err.root, err.mountPoint)
}

const (
	_mountInfoSep               = " "
	_mountInfoOptsSep           = ","
	_mountInfoOptionalFieldsSep = "-"
)

const (
	_miFieldIDMountID = iota
	_miFieldIDParentID
	_miFieldIDDeviceID
	_miFieldIDRoot
	_miFieldIDMountPoint
	_miFieldIDOptions
	_miFieldIDOptionalFields

	_miFieldCountFirstHalf
)

const (
	_miFieldOffsetFSType = iota
	_miFieldOffsetMountSource
	_miFieldOffsetSuperOptions

	_miFieldCountSecondHalf
)

const _miFieldCountMin = _miFieldCountFirstHalf + _miFieldCountSecondHalf

// MountPoint is the data structure for the mount points in
// `/proc/$PID/mountinfo`. See also proc(5) for more information.
type MountPoint struct {
	MountID        int
	ParentID       int
	DeviceID       string
	Root           string
	MountPoint     string
	Options        []string
	OptionalFields []string
	FSType         string
	MountSource    string
	SuperOptions   []string
}

// NewMountPointFromLine parses a line read from `/proc/$PID/mountinfo` and
// returns a new *MountPoint.
func NewMountPointFromLine(line string) (*MountPoint, error) {
	fields := strings.Split(line, _mountInfoSep)

	if len(fields) < _miFieldCountMin {
		return nil, mountPointFormatInvalidError{line}
	}

	mountID, err := strconv.Atoi(fields[_miFieldIDMountID])
	if err != nil {
		return nil, err
	}

	parentID, err := strconv.Atoi(fields[_miFieldIDParentID])
	if err != nil {
		return nil, err
	}

	for i, field := range fields[_miFieldIDOptionalFields:] {
		if field == _mountInfoOptionalFieldsSep {
			fsTypeStart := _miFieldIDOptionalFields + i + 1

			if len(fields) != fsTypeStart+_miFieldCountSecondHalf {
				return nil, mountPointFormatInvalidError{line}
			}

			miFieldIDFSType := _miFieldOffsetFSType + fsTypeStart
			miFieldIDMountSource := _miFieldOffsetMountSource + fsTypeStart
			miFieldIDSuperOptions := _miFieldOffsetSuperOptions + fsTypeStart

			return &MountPoint{
				MountID:        mountID,
				ParentID:       parentID,
				DeviceID:       fields[_miFieldIDDeviceID],
				Root:           fields[_miFieldIDRoot],
				MountPoint:     fields[_miFieldIDMountPoint],
				Options:        strings.Split(fields[_miFieldIDOptions], _mountInfoOptsSep),
				OptionalFields: fields[_miFieldIDOptionalFields:(fsTypeStart - 1)],
				FSType:         fields[miFieldIDFSType],
				MountSource:    fields[miFieldIDMountSource],
				SuperOptions:   strings.Split(fields[miFieldIDSuperOptions], _mountInfoOptsSep),
			}, nil
		}
	}

	return nil, mountPointFormatInvalidError{line}
}

// Translate converts an absolute path inside the *MountPoint's file system to
// the host file system path in the mount namespace the *MountPoint belongs to.
func (mp *MountPoint) Translate(absPath string) (string, error) {
	relPath, err := filepath.Rel(mp.Root, absPath)

	if err != nil {
		return "", err
	}
	if relPath == ".." || strings.HasPrefix(relPath, "../") {
		return "", pathNotExposedFromMountPointError{
			mountPoint: mp.MountPoint,
			root:       mp.Root,
			path:       absPath,
		}
	}

	return filepath.Join(mp.MountPoint, relPath), nil
}

// parseMountInfo parses procPathMountInfo (usually at `/proc/$PID/mountinfo`)
// and yields parsed *MountPoint into newMountPoint.
func parseMountInfo(procPathMountInfo string) ([]*MountPoint, error) {
	mountInfoFile, err := os.Open(procPathMountInfo)
	if err != nil {
		return nil, err
	}
	defer mountInfoFile.Close() // nolint: errcheck

	scanner := bufio.NewScanner(mountInfoFile)

	mps := make([]*MountPoint, 0, 10)
	for scanner.Scan() {
		mountPoint, err := NewMountPointFromLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		mps = append(mps, mountPoint)
	}

	return mps, nil
}
