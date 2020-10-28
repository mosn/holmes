package holmes

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"runtime/pprof"
	"sync/atomic"
	"time"
)

// Holmes is a self-aware profile dumper
type Holmes struct {
	// normal config
	conf     Config
	dumpPath string // full path to put the profile files

	// stats
	collectCount          int
	cpuTriggerCount       int
	memTriggerCount       int
	goroutineTriggerCount int

	// cooldown
	cpuCoolDownTime       time.Time
	memCoolDownTime       time.Time
	goroutineCoolDownTime time.Time

	// stats ring
	memStats  ring
	cpuStats  ring
	gNumStats ring

	// textLogger
	textFile *os.File

	// switch
	stopped int64
}

// New creates a holmes dumper
// interval and cooldown must be valid time duration string,
// eg. "ns", "us" (or "µs"), "ms", "s", "m", "h".
func New(interval string, cooldown string, dumpPath string, binaryProfile bool) *Holmes {
	intervalTime, err := time.ParseDuration(interval)
	if err != nil {
		panic(err)
	}

	cdTime, err := time.ParseDuration(cooldown)
	if err != nil {
		panic(err)
	}

	logFile, err := os.OpenFile(path.Join(dumpPath, "holmes.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}

	return &Holmes{
		dumpPath: dumpPath,
		textFile: logFile,

		conf: Config{
			InText:          !binaryProfile,
			CollectInterval: intervalTime,
			CoolDown:        cdTime,

			EnableCPUDump:       false,
			EnableGoroutineDump: false,
			EnableMemDump:       false,
		},
	}
}

// EnableGoroutineDump enables the goroutine dump and set the config for goroutine dump
func (h *Holmes) EnableGoroutineDump() WithType {
	h.conf.EnableGoroutineDump = true
	h.conf.GoroutineTriggerNumAbs = defaultGoroutineTriggerAbs
	h.conf.GoroutineTriggerPercentDiff = defaultGoroutineTriggerDiff
	h.conf.GoroutineTriggerNumMin = defaultGoroutineTriggerMin

	return WithType{
		h:   h,
		typ: goroutine,
	}
}

// EnableCPUDump enables the CPU dump and set the config for cpu profile dump
func (h *Holmes) EnableCPUDump() WithType {
	h.conf.EnableCPUDump = true
	h.conf.CPUTriggerPercentAbs = defaultCPUTriggerAbs
	h.conf.CPUTriggerPercentDiff = defaultCPUTriggerDiff
	h.conf.CPUTriggerPercentMin = defaultCPUTriggerMin

	return WithType{
		h:   h,
		typ: cpu,
	}
}

// EnableMemDump enables the Mem dump and set the config for memory profile dump
func (h *Holmes) EnableMemDump() WithType {
	h.conf.EnableMemDump = true
	h.conf.MemTriggerPercentAbs = defaultMemTriggerAbs
	h.conf.MemTriggerPercentDiff = defaultMemTriggerDiff
	h.conf.MemTriggerPercentMin = defaultMemTriggerMin

	return WithType{
		h:   h,
		typ: mem,
	}
}

// Start starts the dump loop of holmes
func (h *Holmes) Start() {
	atomic.StoreInt64(&h.stopped, 0)
	go h.startDumpLoop()
}

// Stop the dump loop
func (h *Holmes) Stop() {
	atomic.StoreInt64(&h.stopped, 1)
}

func (h *Holmes) startDumpLoop() {
	// init previous cool down time
	h.cpuCoolDownTime = time.Now()
	h.memCoolDownTime = time.Now()
	h.goroutineCoolDownTime = time.Now()

	// init stats ring
	h.cpuStats = newRing(minCollectCyclesBeforeDumpStart)
	h.memStats = newRing(minCollectCyclesBeforeDumpStart)
	h.gNumStats = newRing(minCollectCyclesBeforeDumpStart)

	// dump loop
	ticker := time.NewTicker(h.conf.CollectInterval)
	for range ticker.C {
		if atomic.LoadInt64(&h.stopped) == 1 {
			ticker.Stop()
			fmt.Println("dump loop stopped")
			return
		}

		cpu, mem, gNum, err := collect()
		if err != nil {
			h.logf(err.Error())
			continue
		}

		h.cpuStats.push(cpu)
		h.memStats.push(mem)
		h.gNumStats.push(gNum)

		h.collectCount++
		if h.collectCount < minCollectCyclesBeforeDumpStart {
			// at least collect some cycles
			// before start to judge and dump
			h.logf("warming up cycle : %d", h.collectCount)
			continue
		}

		h.goroutineCheckAndDump(gNum)
		h.memCheckAndDump(mem)
		h.cpuCheckAndDump(cpu)
	}
}

// goroutine start
func (h *Holmes) goroutineCheckAndDump(gNum int) {
	if !h.conf.EnableGoroutineDump {
		return
	}

	if h.goroutineCoolDownTime.After(time.Now()) {
		h.logf("goroutine dump is in cooldown")
		return
	}

	if triggered := h.goroutineProfile(gNum, h.conf); triggered {
		h.goroutineCoolDownTime = time.Now().Add(h.conf.CoolDown)
		h.goroutineTriggerCount++
	}
}

func (h *Holmes) goroutineProfile(gNum int, c Config) bool {
	if !matchRule(h.gNumStats, gNum, c.GoroutineTriggerNumMin, c.GoroutineTriggerNumAbs, c.GoroutineTriggerPercentDiff) {
		h.debugf("NODUMP goroutine, config_min : %v, config_diff : %v, config_abs : %v,  previous : %v, current : %v",
			c.GoroutineTriggerNumMin, c.GoroutineTriggerPercentDiff, c.GoroutineTriggerNumAbs,
			h.gNumStats.data, gNum)

		return false
	}

	var buf bytes.Buffer
	if h.conf.InText {
		pprof.Lookup("goroutine").WriteTo(&buf, 1) // nolint: errcheck
	} else {
		pprof.Lookup("goroutine").WriteTo(&buf, 0) // nolint: errcheck
	}
	h.writeProfileDataToFile(buf, goroutine, gNum)

	return true
}

// memory start
func (h *Holmes) memCheckAndDump(mem int) {
	if !h.conf.EnableMemDump {
		return
	}

	if h.memCoolDownTime.After(time.Now()) {
		h.logf("mem dump is in cooldown")
		return
	}

	if triggered := h.memProfile(mem, h.conf); triggered {
		h.memCoolDownTime = time.Now().Add(h.conf.CoolDown)
		h.memTriggerCount++
	}
}

func (h *Holmes) memProfile(rss int, c Config) bool {
	if !matchRule(h.memStats, rss, c.MemTriggerPercentMin, c.MemTriggerPercentAbs, c.MemTriggerPercentDiff) {
		// let user know why this should not dump
		h.debugf("NODUMP, memory, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current : %v",
			c.MemTriggerPercentMin, c.MemTriggerPercentDiff, c.MemTriggerPercentAbs,
			h.memStats.data, rss)

		return false
	}

	var buf bytes.Buffer
	if h.conf.InText {
		pprof.Lookup("heap").WriteTo(&buf, 1) // nolint: errcheck
	} else {
		pprof.Lookup("heap").WriteTo(&buf, 0) // nolint: errcheck
	}

	h.writeProfileDataToFile(buf, mem, rss)
	return true
}

// cpu start
func (h *Holmes) cpuCheckAndDump(cpu int) {
	if !h.conf.EnableCPUDump {
		return
	}

	if h.cpuCoolDownTime.After(time.Now()) {
		h.logf("cpu dump is in cooldown")
		return
	}

	if triggered := h.cpuProfile(cpu, h.conf); triggered {
		h.cpuCoolDownTime = time.Now().Add(h.conf.CoolDown)
		h.cpuTriggerCount++
	}
}

func (h *Holmes) cpuProfile(curCPUUsage int, c Config) bool {
	if !matchRule(h.cpuStats, curCPUUsage, c.CPUTriggerPercentMin, c.CPUTriggerPercentAbs, c.CPUTriggerPercentDiff) {
		// let user know why this should not dump
		h.debugf("NODUMP cpu, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current: %v",
			c.CPUTriggerPercentMin, c.CPUTriggerPercentDiff, c.CPUTriggerPercentAbs,
			h.cpuStats.data, curCPUUsage)

		return false
	}

	binFileName := getBinaryFileName(h.dumpPath, cpu)

	bf, err := os.OpenFile(binFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		h.logf("failed to create cpu profile file: %v", err.Error())
		return false
	}
	defer bf.Close()

	err = pprof.StartCPUProfile(bf)
	if err != nil {
		h.logf("failed to profile cpu: %v", err.Error())
		return false
	}

	// collect 5s cpu profile
	time.Sleep(time.Second * 5)
	pprof.StopCPUProfile()
	h.logf("pprof cpu dump to log dir, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current: %v",
		c.CPUTriggerPercentMin, c.CPUTriggerPercentDiff, c.CPUTriggerPercentAbs,
		h.cpuStats.data, curCPUUsage)

	return true
}

func (h *Holmes) writeProfileDataToFile(data bytes.Buffer, dumpType configureType, currentStat int) {
	binFileName := getBinaryFileName(h.dumpPath, dumpType)

	switch dumpType {
	case mem:
		h.logf("pprof memory, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current : %v",
			h.conf.MemTriggerPercentMin, h.conf.MemTriggerPercentDiff, h.conf.MemTriggerPercentAbs,
			h.memStats.data, currentStat)
	case goroutine:
		h.logf("pprof goroutine, config_min : %v, config_diff : %v, config_abs : %v,  previous : %v, current : %v",
			h.conf.GoroutineTriggerNumMin, h.conf.GoroutineTriggerPercentDiff, h.conf.GoroutineTriggerNumAbs,
			h.gNumStats.data, currentStat)
	}

	if h.conf.InText {
		// write to log
		var res = data.String()
		if !h.conf.DumpFullStack {
			res = trimResult(data)
		}

		h.logf(res)
	} else {
		bf, err := os.OpenFile(binFileName, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			h.logf("pprof memory write to file failed : %v", err.Error())
			return
		}
		defer bf.Close()

		_, err = bf.Write(data.Bytes())
		if err != nil {
			h.logf("pprof memory write to file failed : %v", err.Error())
		}
	}
}
