package holmes

import (
	"bytes"
	"fmt"
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

// only reserve the top n.
func trimResult(buffer bytes.Buffer) string {
	index := TrimResultTopN
	arr := strings.SplitN(buffer.String(), "\n\n", TrimResultTopN+1)

	if len(arr) <= TrimResultTopN {
		index = len(arr) - 1
	}

	return strings.Join(arr[:index], "\n\n")
}

// return cpu percent, mem in MB, goroutine num, thread num
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

func getNormalMemoryLimit() (uint64, error) {
	machineMemory, err := mem_util.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return machineMemory.Total, nil
}

func getMemoryLimit(h *Holmes) (uint64, error) {
	if h.opts.memoryLimit > 0 {
		return h.opts.memoryLimit, nil
	}

	if h.opts.UseCGroup {
		return getCGroupMemoryLimit()
	}
	return getNormalMemoryLimit()
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

// cpu mem goroutine thread err.
func collect() (int, int, int, int, error) {
	cpu, mem, gNum, tNum, err := getUsage()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	return int(cpu), int(mem), gNum, tNum, nil
}

func matchRule(history ring, curVal, ruleMin, ruleAbs, ruleDiff, ruleMax int) bool {
	// should bigger than rule min
	if curVal < ruleMin {
		return false
	}

	// if ruleMax is enable and current value bigger max, skip dumping
	if ruleMax != NotSupportTypeMaxConfig && curVal >= ruleMax {
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
		binarySuffix = time.Now().Format("20060102150405.000") + ".bin"
	)

	return path.Join(filePath, type2name[dumpType]+"."+binarySuffix)
}

func writeFile(data bytes.Buffer, dumpType configureType, dumpOpts *DumpOptions) error {
	if dumpOpts.DumpProfileType == textDump {
		// write to log
		if dumpOpts.DumpFullStack {
			return fmt.Errorf(trimResult(data))
		} else {
			return fmt.Errorf(data.String())
		}
	}

	binFileName := getBinaryFileName(dumpOpts.DumpPath, dumpType)
	bf, err := os.OpenFile(binFileName, defaultLoggerFlags, defaultLoggerPerm)
	if err != nil {
		return fmt.Errorf("[Holmes] pprof %v write to file failed : %v", type2name[dumpType], err.Error())
	}
	defer bf.Close()

	if _, err = bf.Write(data.Bytes()); err != nil {
		return fmt.Errorf("[Holmes] pprof %v write to file failed : %v", type2name[dumpType], err.Error())
	}
	return nil
}
