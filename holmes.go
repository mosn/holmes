package holmes

import (
	"bytes"
	"os"
	"path"
	"runtime/pprof"
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
	h.conf.GoroutineTriggerAbs = defaultGoroutineTriggerAbs
	h.conf.GoroutineTriggerDiff = defaultGoroutineTriggerDiff
	h.conf.GoroutineTriggerMin = defaultGoroutineTriggerMin

	return WithType{
		h:   h,
		typ: goroutine,
	}
}

// EnableCPUDump enables the CPU dump and set the config for cpu profile dump
func (h *Holmes) EnableCPUDump() WithType {
	h.conf.EnableCPUDump = true
	h.conf.CPUTriggerAbs = defaultCPUTriggerAbs
	h.conf.CPUTriggerDiff = defaultCPUTriggerDiff
	h.conf.CPUTriggerMin = defaultCPUTriggerMin

	return WithType{
		h:   h,
		typ: cpu,
	}
}

// EnableMemDump enables the Mem dump and set the config for memory profile dump
func (h *Holmes) EnableMemDump() WithType {
	h.conf.EnableMemDump = true
	h.conf.MemTriggerAbs = defaultMemTriggerAbs
	h.conf.MemTriggerDiff = defaultMemTriggerDiff
	h.conf.MemTriggerMin = defaultMemTriggerMin

	return WithType{
		h:   h,
		typ: mem,
	}
}

// Start starts the dump loop of holmes
func (h *Holmes) Start() {
	h.initEnvironment()
	go h.startDumpLoop()
}

func (h *Holmes) initEnvironment() {
	// is this a docker environment or physical?
	if _, err := readUint(cgroupMemLimitPath); err == nil {
		// docker
		getUsage = getUsageDocker
	} else {
		// physical machine
		getUsage = getUsagePhysical
	}
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
	for range time.Tick(h.conf.CollectInterval) {
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
	if !matchRule(h.gNumStats, gNum, c.GoroutineTriggerMin, c.GoroutineTriggerAbs, c.GoroutineTriggerDiff) {
		h.debugf("NODUMP goroutine, config_min : %v, config_diff : %v, config_abs : %v,  previous : %v, current : %v",
			c.GoroutineTriggerMin, c.GoroutineTriggerDiff, c.GoroutineTriggerAbs,
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
	if !matchRule(h.memStats, rss, c.MemTriggerMin, c.MemTriggerAbs, c.MemTriggerDiff) {
		// let user know why this should not dump
		h.debugf("NODUMP, memory, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current : %v",
			c.MemTriggerMin, c.MemTriggerDiff, c.MemTriggerAbs,
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
	if !matchRule(h.cpuStats, curCPUUsage, c.CPUTriggerMin, c.CPUTriggerAbs, c.CPUTriggerDiff) {
		// let user know why this should not dump
		h.debugf("NODUMP cpu, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current: %v",
			c.CPUTriggerMin, c.CPUTriggerDiff, c.CPUTriggerAbs,
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
		c.CPUTriggerMin, c.CPUTriggerDiff, c.CPUTriggerAbs,
		h.cpuStats.data, curCPUUsage)

	return true
}

func (h *Holmes) writeProfileDataToFile(data bytes.Buffer, dumpType configureType, currentStat int) {
	binFileName := getBinaryFileName(h.dumpPath, dumpType)

	switch dumpType {
	case mem:
		h.logf("pprof memory, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current : %v",
			h.conf.MemTriggerMin, h.conf.MemTriggerDiff, h.conf.MemTriggerAbs,
			h.memStats.data, currentStat)
	case goroutine:
		h.logf("pprof goroutine, config_min : %v, config_diff : %v, config_abs : %v,  previous : %v, current : %v",
			h.conf.GoroutineTriggerMin, h.conf.GoroutineTriggerDiff, h.conf.GoroutineTriggerAbs,
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
