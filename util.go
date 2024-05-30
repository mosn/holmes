/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package holmes

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"mosn.io/holmes/internal/cg"

	mem_util "github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

// only reserve the top n.
func trimResultTop(buffer bytes.Buffer) []byte {
	index := TrimResultTopN
	arr := strings.SplitN(buffer.String(), "\n\n", TrimResultTopN+1)

	if len(arr) <= TrimResultTopN {
		index = len(arr) - 1
	}

	return []byte(strings.Join(arr[:index], "\n\n"))
}

// only reserve the front n bytes
func trimResultFront(buffer bytes.Buffer) []byte {
	if buffer.Len() <= TrimResultMaxBytes {
		return buffer.Bytes()
	}
	return buffer.Bytes()[:TrimResultMaxBytes-1]
}

// return values:
// 1. cpu percent, not division cpu cores yet,
// 2. RSS mem in bytes,
// 3. goroutine num,
// 4. thread num
func getUsage() (float64, uint64, int, int, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, 0, 0, 0, err
	}
	cpuPercent, err := p.Percent(time.Second)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	mem, err := p.MemoryInfo()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	rss := mem.RSS
	gNum := runtime.NumGoroutine()
	tNum := getThreadNum()

	return cpuPercent, rss, gNum, tNum, nil
}

// get cpu core number limited by CGroup.
func getCGroupCPUCore() (float64, error) {
	quota, err := cg.GetCPUCore()
	if err != nil {
		return 0, err
	}
	if quota == -1 {
		quota = float64(runtime.NumCPU())
	}
	return quota, nil
}

func getCGroupMemoryLimit() (uint64, error) {
	usage, err := cg.GetMemoryLimit()
	if err != nil {
		return 0, err
	}
	machineMemory, err := getNormalMemoryLimit()
	if err != nil {
		return 0, err
	}
	if usage == -1 {
		return machineMemory, nil
	}
	limit := uint64(math.Min(float64(usage), float64(machineMemory)))
	return limit, nil
}

func getNormalMemoryLimit() (uint64, error) {
	machineMemory, err := mem_util.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return machineMemory.Total, nil
}

func getThreadNum() int {
	return pprof.Lookup("threadcreate").Count()
}

// cpu mem goroutine thread err.
func collect(cpuCore float64, memoryLimit uint64) (int, int, int, int, error) {
	cpu, mem, gNum, tNum, err := getUsage()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// The default percent is from all cores, multiply by cpu core
	// but it's inconvenient to calculate the proper percent
	// here we divide by core number, so we can set a percent bar more intuitively
	cpuPercent := cpu / cpuCore

	memPercent := float64(mem) / float64(memoryLimit) * 100

	return int(cpuPercent), int(memPercent), gNum, tNum, nil
}

func matchRule(history ring, curVal, ruleMin, ruleAbs, ruleDiff, ruleMax int) (bool, ReasonType) {
	// should bigger than rule min
	if curVal < ruleMin {
		return false, ReasonCurlLessMin
		//fmt.Sprintf("curVal [%d]< ruleMin [%d]", curVal, ruleMin)
	}

	// if ruleMax is enable and current value bigger max, skip dumping
	if ruleMax != NotSupportTypeMaxConfig && curVal >= ruleMax {
		return false, ReasonCurGreaterMax
	}

	// the current peak load exceed the absolute value
	if curVal > ruleAbs {
		return true, ReasonCurGreaterAbs
		// fmt.Sprintf("curVal [%d] > ruleAbs [%d]", curVal, ruleAbs)
	}

	// the peak load matches the rule
	avg := history.avg()
	if curVal >= avg*(100+ruleDiff)/100 {
		return true, ReasonDiff
		// fmt.Sprintf("curVal[%d] >= avg[%d]*(100+ruleDiff)/100", curVal, avg)
	}
	return false, ReasonCurlGreaterMin
}

func getBinaryFileName(filePath string, dumpType configureType, eventID string) string {
	suffix := time.Now().Format("20060102150405.000") + ".log"
	if len(eventID) == 0 {
		return filepath.Join(filePath, check2name[dumpType]+"."+suffix)
	}

	return filepath.Join(filePath, check2name[dumpType]+"."+eventID+"."+suffix)
}

// fix #89
func getBinaryFileNameAndCreate(dump string, dumpType configureType, eventID string) (*os.File, string, error) {
	filePath := getBinaryFileName(dump, dumpType, eventID)
	f, err := os.OpenFile(filePath, defaultLoggerFlags, defaultLoggerPerm)
	if err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(dump, 0o755); err != nil {
			return nil, filePath, err
		}
		f, err = os.OpenFile(filePath, defaultLoggerFlags, defaultLoggerPerm)
		if err != nil {
			return nil, filePath, err
		}
	}
	return f, filePath, err
}

func writeFile(data bytes.Buffer, dumpType configureType, dumpOpts *DumpOptions, eventID string) (string, error) {
	var buf []byte
	if dumpOpts.DumpProfileType == textDump && !dumpOpts.DumpFullStack {
		switch dumpType {
		case mem, gcHeap, goroutine:
			buf = trimResultTop(data)
		case thread:
			buf = trimResultFront(data)
		default:
			buf = data.Bytes()
		}
	} else {
		buf = data.Bytes()
	}

	file, fileName, err := getBinaryFileNameAndCreate(dumpOpts.DumpPath, dumpType, eventID)
	if err != nil {
		return fileName, fmt.Errorf("pprof %v open file failed : %w", type2name[dumpType], err)
	}
	defer file.Close() //nolint:errcheck,gosec

	if _, err = file.Write(buf); err != nil {
		return fileName, fmt.Errorf("pprof %v write to file failed : %w", type2name[dumpType], err)
	}
	return fileName, nil
}
