package holmes

import (
	"bytes"
	"io/ioutil"
	"math"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	mem_util "github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

// copied from https://github.com/containerd/cgroups/blob/318312a373405e5e91134d8063d04d59768a1bff/utils.go#L251
func parseUint(s string, base, bitSize int) (uint64, error) {
	v, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil &&
			intErr.(*strconv.NumError).Err == strconv.ErrRange &&
			intValue < 0 {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

// copied from https://github.com/containerd/cgroups/blob/318312a373405e5e91134d8063d04d59768a1bff/utils.go#L243
func readUint(path string) (uint64, error) {
	v, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseUint(strings.TrimSpace(string(v)), 10, 64)
}

// only reserve the top 10.
func trimResult(buffer bytes.Buffer) string {
	arr := strings.Split(buffer.String(), "\n\n")
	if len(arr) > 10 {
		arr = arr[:10]
	}
	return strings.Join(arr, "\n\n")
}

// return cpu percent, mem in MB, goroutine num
// cgroup ver.
func getUsageCGroup() (float64, float64, int, int, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, 0, 0, 0, err
	}

	cpuPercent, err := p.Percent(time.Second)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	cpuPeriod, err := readUint(cgroupCpuPeriodPath)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	cpuQuota, err := readUint(cgroupCpuQuotaPath)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	cpuCore := float64(cpuQuota) / float64(cpuPeriod)

	// the same with physical machine
	// need to divide by core number
	cpuPercent = cpuPercent / cpuCore
	mem, err := p.MemoryInfo()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	memLimit, err := getCGroupMemoryLimit()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	// mem.RSS / cgroup limit in bytes
	memPercent := float64(mem.RSS) * 100 / float64(memLimit)

	gNum := runtime.NumGoroutine()

	tNum := getThreadNum()

	return cpuPercent, memPercent, gNum, tNum, nil
}

func getCGroupMemoryLimit() (uint64, error) {
	usage, err := readUint(cgroupMemLimitPath)
	if err != nil {
		return 0, err
	}
	machineMemory, err := mem_util.VirtualMemory()
	if err != nil {
		return 0, err
	}
	limit := uint64(math.Min(float64(usage), float64(machineMemory.Total)))
	return limit, nil
}

// return cpu percent, mem in MB, goroutine num
// not use cgroup ver.
func getUsageNormal() (float64, float64, int, int, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, 0, 0, 0, err
	}

	cpuPercent, err := p.Percent(time.Second)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// The default percent is from all cores, multiply by runtime.NumCPU()
	// but it's inconvenient to calculate the proper percent
	// here we divide by core number, so we can set a percent bar more intuitively
	cpuPercent = cpuPercent / float64(runtime.NumCPU())

	mem, err := p.MemoryPercent()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	gNum := runtime.NumGoroutine()

	tNum := getThreadNum()

	return cpuPercent, float64(mem), gNum, tNum, nil
}

func getThreadNum() int {
	return pprof.Lookup("threadcreate").Count()
}

var getUsage func() (float64, float64, int, int, error)

// cpu mem goroutine err.
func collect() (int, int, int, int, error) {
	cpu, mem, gNum, tNum, err := getUsage()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	return int(cpu), int(mem), gNum, tNum, nil
}

func matchRule(history ring, curVal, ruleMin, ruleAbs, ruleDiff int) bool {
	// should bigger than rule min
	if curVal < ruleMin {
		return false
	}

	// the current peak load exceed the absolute value
	if curVal > ruleAbs {
		return true
	}

	// the peak load matches the rule
	avg := history.avg()
	return curVal >= avg*(100+ruleDiff)/100
}

func getBinaryFileName(filePath string, dumpType configureType) string {
	var (
		binarySuffix = time.Now().Format("20060102150405") + ".bin"
	)

	return path.Join(filePath, type2name[dumpType]+"."+binarySuffix)
}
