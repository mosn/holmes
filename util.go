package holmes

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

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

// only reserve the top 10
func trimResult(buffer bytes.Buffer) string {
	arr := strings.Split(buffer.String(), "\n\n")
	if len(arr) > 10 {
		arr = arr[:10]
	}
	return strings.Join(arr, "\n\n")
}

// return cpu percent, mem in MB, goroutine num
// docker ver
func getUsageDocker() (float64, float64, int, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, 0, 0, err
	}

	cpuPercent, err := p.CPUPercent()
	if err != nil {
		return 0, 0, 0, err
	}

	// the same with physical machine
	// need to divide by core
	cpuPercent = cpuPercent / float64(runtime.GOMAXPROCS(-1))

	mem, err := p.MemoryInfo()
	if err != nil {
		return 0, 0, 0, err
	}

	memLimit, err := readUint(cgroupMemLimitPath)
	if err != nil {
		return 0, 0, 0, err
	}

	// mem.RSS / cgroup limit in bytes
	memPercent := float64(mem.RSS) * 100 / float64(memLimit)

	gNum := runtime.NumGoroutine()
	return cpuPercent, memPercent, gNum, nil
}

// return cpu percent, mem in MB, goroutine num
// phys ver
func getUsagePhysical() (float64, float64, int, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, 0, 0, err
	}

	cpuPercent, err := p.CPUPercent()
	if err != nil {
		return 0, 0, 0, err
	}

	// The default percent is if you use one core, then 100%, two core, 200%
	// but it's inconvenient to calculate the proper percent
	// here we multiply by core number, so we can set a percent bar more intuitively
	cpuPercent = cpuPercent / float64(runtime.GOMAXPROCS(-1))

	mem, err := p.MemoryPercent()
	if err != nil {
		return 0, 0, 0, err
	}

	gNum := runtime.NumGoroutine()

	return cpuPercent, float64(mem), gNum, nil
}

var getUsage func() (float64, float64, int, error)

// cpu mem goroutine err
func collect() (int, int, int, error) {
	cpu, mem, gNum, err := getUsage()
	if err != nil {
		return 0, 0, 0, err
	}

	return int(cpu), int(mem), int(gNum), nil
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

func getDebugLogFileName(filePath string) string {
	return path.Join(filePath, "holmes_debug.log")
}
