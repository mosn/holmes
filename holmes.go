package holmes

import (
	"bytes"
	"fmt"
	"os"
	"runtime/pprof"
	"sync/atomic"
	"time"
)

// Holmes is a self-aware profile dumper.
type Holmes struct {
	opts *options

	// stats
	collectCount       int
	threadTriggerCount int
	cpuTriggerCount    int
	memTriggerCount    int
	grTriggerCount     int

	// cooldown
	threadCoolDownTime time.Time
	cpuCoolDownTime    time.Time
	memCoolDownTime    time.Time
	grCoolDownTime     time.Time

	// stats ring
	memStats    ring
	cpuStats    ring
	grNumStats  ring
	threadStats ring

	// switch
	stopped int64
}

// New creates a holmes dumper.
func New(opts ...Option) (*Holmes, error) {
	holmes := &Holmes{
		opts: newOptions(),
	}

	for _, opt := range opts {
		if err := opt.apply(holmes.opts); err != nil {
			return nil, err
		}
	}

	return holmes, nil
}

// EnableThreadDump enables the goroutine dump.
func (h *Holmes) EnableThreadDump() *Holmes {
	h.opts.ThreadOpts.Enable = true
	return h
}

// DisableThreadDump disables the goroutine dump.
func (h *Holmes) DisableThreadDump() *Holmes {
	h.opts.ThreadOpts.Enable = false
	return h
}

// EnableGoroutineDump enables the goroutine dump.
func (h *Holmes) EnableGoroutineDump() *Holmes {
	h.opts.GrOpts.Enable = true
	return h
}

// DisableGoroutineDump disables the goroutine dump.
func (h *Holmes) DisableGoroutineDump() *Holmes {
	h.opts.GrOpts.Enable = false
	return h
}

// EnableCPUDump enables the CPU dump.
func (h *Holmes) EnableCPUDump() *Holmes {
	h.opts.CPUOpts.Enable = true
	return h
}

// DisableCPUDump disables the CPU dump.
func (h *Holmes) DisableCPUDump() *Holmes {
	h.opts.CPUOpts.Enable = false
	return h
}

// EnableMemDump enables the mem dump.
func (h *Holmes) EnableMemDump() *Holmes {
	h.opts.MemOpts.Enable = true
	return h
}

// DisableMemDump disables the mem dump.
func (h *Holmes) DisableMemDump() *Holmes {
	h.opts.MemOpts.Enable = false
	return h
}

// Start starts the dump loop of holmes.
func (h *Holmes) Start() {
	atomic.StoreInt64(&h.stopped, 0)
	h.initEnvironment()
	go h.startDumpLoop()
}

// Stop the dump loop.
func (h *Holmes) Stop() {
	atomic.StoreInt64(&h.stopped, 1)
}

func (h *Holmes) startDumpLoop() {
	// init previous cool down time
	now := time.Now()
	h.cpuCoolDownTime = now
	h.memCoolDownTime = now
	h.grCoolDownTime = now

	// init stats ring
	h.cpuStats = newRing(minCollectCyclesBeforeDumpStart)
	h.memStats = newRing(minCollectCyclesBeforeDumpStart)
	h.grNumStats = newRing(minCollectCyclesBeforeDumpStart)
	h.threadStats = newRing(minCollectCyclesBeforeDumpStart)

	// dump loop
	ticker := time.NewTicker(h.opts.CollectInterval)
	defer ticker.Stop()
	for range ticker.C {
		if atomic.LoadInt64(&h.stopped) == 1 {
			fmt.Println("[Holmes] dump loop stopped")
			return
		}

		cpu, mem, gNum, tNum, err := collect()
		if err != nil {
			h.logf(err.Error())
			continue
		}

		h.cpuStats.push(cpu)
		h.memStats.push(mem)
		h.grNumStats.push(gNum)
		h.threadStats.push(tNum)

		h.collectCount++
		if h.collectCount < minCollectCyclesBeforeDumpStart {
			// at least collect some cycles
			// before start to judge and dump
			h.logf("[Holmes] warming up cycle : %d", h.collectCount)
			continue
		}

		h.goroutineCheckAndDump(gNum)
		h.memCheckAndDump(mem)
		h.cpuCheckAndDump(cpu)
		h.threadCheckAndDump(tNum)
	}
}

// goroutine start.
func (h *Holmes) goroutineCheckAndDump(gNum int) {
	if !h.opts.GrOpts.Enable {
		return
	}

	if h.grCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] goroutine dump is in cooldown")
		return
	}

	if triggered := h.goroutineProfile(gNum); triggered {
		h.grCoolDownTime = time.Now().Add(h.opts.CoolDown)
		h.grTriggerCount++
	}
}

func (h *Holmes) goroutineProfile(gNum int) bool {
	c := h.opts.GrOpts
	if !matchRule(h.grNumStats, gNum, c.GoroutineTriggerNumMin, c.GoroutineTriggerNumAbs, c.GoroutineTriggerPercentDiff) {
		h.debugf("[Holmes] NODUMP goroutine, config_min : %v, config_diff : %v, config_abs : %v,  previous : %v, current : %v",
			c.GoroutineTriggerNumMin, c.GoroutineTriggerPercentDiff, c.GoroutineTriggerNumAbs,
			h.grNumStats.data, gNum)

		return false
	}

	var buf bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, goroutine, gNum)

	return true
}

// memory start.
func (h *Holmes) memCheckAndDump(mem int) {
	if !h.opts.MemOpts.Enable {
		return
	}

	if h.memCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] mem dump is in cooldown")
		return
	}

	if triggered := h.memProfile(mem); triggered {
		h.memCoolDownTime = time.Now().Add(h.opts.CoolDown)
		h.memTriggerCount++
	}
}

func (h *Holmes) memProfile(rss int) bool {
	c := h.opts.MemOpts
	if !matchRule(h.memStats, rss, c.MemTriggerPercentMin, c.MemTriggerPercentAbs, c.MemTriggerPercentDiff) {
		// let user know why this should not dump
		h.debugf("[Holmes] NODUMP, memory, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current : %v",
			c.MemTriggerPercentMin, c.MemTriggerPercentDiff, c.MemTriggerPercentAbs,
			h.memStats.data, rss)

		return false
	}

	var buf bytes.Buffer
	_ = pprof.Lookup("heap").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, mem, rss)
	return true
}

// thread start.
func (h *Holmes) threadCheckAndDump(threadNum int) {
	if !h.opts.ThreadOpts.Enable {
		return
	}

	if h.threadCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] thread dump is in cooldown")
		return
	}

	if triggered := h.threadProfile(threadNum); triggered {
		h.threadCoolDownTime = time.Now().Add(h.opts.CoolDown)
		h.threadTriggerCount++
	}
}

func (h *Holmes) threadProfile(curThreadNum int) bool {
	c := h.opts.ThreadOpts
	if !matchRule(h.threadStats, curThreadNum, c.ThreadTriggerPercentMin, c.ThreadTriggerPercentAbs, c.ThreadTriggerPercentDiff) {
		// let user know why this should not dump
		h.debugf("[Holmes] NODUMP thread, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current: %v",
			c.ThreadTriggerPercentMin, c.ThreadTriggerPercentDiff, c.ThreadTriggerPercentAbs,
			h.threadStats.data, curThreadNum)

		return false
	}

	var buf bytes.Buffer
	pprof.Lookup("threadcreate").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	pprof.Lookup("goroutine").WriteTo(&buf, int(h.opts.DumpProfileType))    // nolint: errcheck

	h.writeProfileDataToFile(buf, thread, curThreadNum)

	return true
}

// thread end.

// cpu start.
func (h *Holmes) cpuCheckAndDump(cpu int) {
	if !h.opts.CPUOpts.Enable {
		return
	}

	if h.cpuCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] cpu dump is in cooldown")
		return
	}

	if triggered := h.cpuProfile(cpu); triggered {
		h.cpuCoolDownTime = time.Now().Add(h.opts.CoolDown)
		h.cpuTriggerCount++
	}
}

func (h *Holmes) cpuProfile(curCPUUsage int) bool {
	c := h.opts.CPUOpts
	if !matchRule(h.cpuStats, curCPUUsage, c.CPUTriggerPercentMin, c.CPUTriggerPercentAbs, c.CPUTriggerPercentDiff) {
		// let user know why this should not dump
		h.debugf("[Holmes] NODUMP cpu, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current: %v",
			c.CPUTriggerPercentMin, c.CPUTriggerPercentDiff, c.CPUTriggerPercentAbs,
			h.cpuStats.data, curCPUUsage)

		return false
	}

	binFileName := getBinaryFileName(h.opts.DumpPath, cpu)

	bf, err := os.OpenFile(binFileName, defaultLoggerFlags, defaultLoggerPerm)
	if err != nil {
		h.logf("[Holmes] failed to create cpu profile file: %v", err.Error())
		return false
	}
	defer bf.Close()

	err = pprof.StartCPUProfile(bf)
	if err != nil {
		h.logf("[Holmes] failed to profile cpu: %v", err.Error())
		return false
	}

	time.Sleep(defaultCPUSamplingTime)
	pprof.StopCPUProfile()

	h.logf("[Holmes] pprof cpu dump to log dir, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current: %v",
		c.CPUTriggerPercentMin, c.CPUTriggerPercentDiff, c.CPUTriggerPercentAbs,
		h.cpuStats.data, curCPUUsage)

	return true
}

func (h *Holmes) writeProfileDataToFile(data bytes.Buffer, dumpType configureType, currentStat int) {
	binFileName := getBinaryFileName(h.opts.DumpPath, dumpType)

	switch dumpType {
	case mem:
		opts := h.opts.MemOpts
		h.logf("[Holmes] pprof %v, config_min : %v, config_diff : %v, config_abs : %v, previous : %v, current : %v",
			type2name[dumpType], opts.MemTriggerPercentMin,
			opts.MemTriggerPercentDiff, opts.MemTriggerPercentAbs,
			h.memStats.data, currentStat)
	case goroutine:
		opts := h.opts.GrOpts
		h.logf("[Holmes] pprof %v, config_min : %v, config_diff : %v, config_abs : %v,  previous : %v, current : %v",
			type2name[dumpType], opts.GoroutineTriggerNumMin,
			opts.GoroutineTriggerPercentDiff, opts.GoroutineTriggerNumAbs,
			h.grNumStats.data, currentStat)
	case thread:
		opts := h.opts.ThreadOpts
		h.logf("[Holmes] pprof %v, config_min : %v, config_diff : %v, config_abs : %v,  previous : %v, current : %v",
			type2name[dumpType], opts.ThreadTriggerPercentMin,
			opts.ThreadTriggerPercentDiff, opts.ThreadTriggerPercentAbs,
			h.threadStats.data, currentStat)
	}

	if h.opts.DumpProfileType == textDump {
		// write to log
		var res = data.String()
		if !h.opts.DumpFullStack {
			res = trimResult(data)
		}
		h.logf(res)
	} else {
		bf, err := os.OpenFile(binFileName, defaultLoggerFlags, defaultLoggerPerm)
		if err != nil {
			h.logf("[Holmes] pprof %v write to file failed : %v", type2name[dumpType], err.Error())
			return
		}
		defer bf.Close()

		if _, err = bf.Write(data.Bytes()); err != nil {
			h.logf("[Holmes] pprof %v write to file failed : %v", type2name[dumpType], err.Error())
		}
	}
}

func (h *Holmes) initEnvironment() {
	// choose whether the max memory is limited by cgroup
	if h.opts.UseCGroup {
		// use cgroup
		getUsage = getUsageCGroup
		h.logf("[Holmes] use cgroup to limit memory")
	} else {
		// not use cgroup
		getUsage = getUsageNormal
		h.logf("[Holmes] use the default memory percent calculated by gopsutil")
	}
}
