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

package cg

import (
	"sync"

	"mosn.io/holmes/internal/cg/cgroups"
)

var (
	globalCG cgroups.ICGroups
	once     sync.Once
)

func GetCPUCore() (float64, error) {
	if globalCG == nil {
		if err := initCG(); err != nil {
			return 0, err
		}
	}
	quota, _, err := globalCG.CPUQuota()
	return quota, err
}

func GetMemoryLimit() (int, error) {
	if globalCG == nil {
		if err := initCG(); err != nil {
			return 0, err
		}
	}
	limit, _, err := globalCG.MemLimit()
	return limit, err
}

func initCG() (err error) {
	once.Do(func() {
		globalCG, err = cgroups.LoadCGroupsForCurrentProcess()
	})
	return
}