package holmes

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"
)

// Holmes is a self-aware profile dumper.
type Holmes struct {
	opts *options

	// stats
	changelog          int32
	collectCount       int
	gcCycleCount       int
	threadTriggerCount int
	cpuTriggerCount    int
	memTriggerCount    int
	grTriggerCount     int
	gcHeapTriggerCount int

	// channel for GC sweep finalizer event
	finCh chan time.Time

	// cooldown
	threadCoolDownTime time.Time
	cpuCoolDownTime    time.Time
	memCoolDownTime    time.Time
	gcHeapCoolDownTime time.Time
	grCoolDownTime     time.Time

	// GC heap triggered, need to dump next time.
	gcHeapTriggered bool

	// stats ring
	memStats    ring
	cpuStats    ring
	grNumStats  ring
	threadStats ring
	gcHeapStats ring

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
	h.opts.grOpts.SetEnable(true)
	return h
}

// DisableGoroutineDump disables the goroutine dump.
func (h *Holmes) DisableGoroutineDump() *Holmes {
	h.opts.grOpts.SetEnable(false)
	return h
}

// EnableCPUDump enables the CPU dump.
func (h *Holmes) EnableCPUDump() *Holmes {
	h.opts.cpuOpts.SetEnable(true)
	return h
}

// DisableCPUDump disables the CPU dump.
func (h *Holmes) DisableCPUDump() *Holmes {
	h.opts.cpuOpts.SetEnable(false)
	return h
}

// EnableMemDump enables the mem dump.
func (h *Holmes) EnableMemDump() *Holmes {
	h.opts.memOpts.SetEnable(true)
	return h
}

// EnableGCHeapDump enables the GC heap dump.
func (h *Holmes) EnableGCHeapDump() *Holmes {
	h.opts.GCHeapOpts.Enable = true
	return h
}

// DisableMemDump disables the mem dump.
func (h *Holmes) DisableMemDump() *Holmes {
	h.opts.memOpts.SetEnable(false)
	return h
}

// it won't fit into tiny span since this struct contains point.
type foo struct {
	h *Holmes
}

func finalizerCallback(f *foo) {
	// register the finalizer again
	runtime.SetFinalizer(f, finalizerCallback)

	select {
	case f.h.finCh <- time.Time{}:
	default:
		f.h.logf("can not send event to finalizer channel immediately, may be analyzer blocked?")
	}
}

func (h *Holmes) startGCCycleLoop() {
	h.gcHeapStats = newRing(minCollectCyclesBeforeDumpStart)

	f := &foo{
		h: h,
	}
	runtime.SetFinalizer(f, finalizerCallback)

	go f.h.gcHeapCheckLoop()
}

// Start starts the dump loop of holmes.
func (h *Holmes) Start() {
	atomic.StoreInt64(&h.stopped, 0)
	h.initEnvironment()
	go h.startDumpLoop()

	h.startGCCycleLoop()
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

		if err := h.EnableDump(cpu); err != nil {
			h.logf("[Holmes] unable to dump: %v", err)
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
	// get a copy instead of locking it
	coolDown := h.opts.CoolDown

	grOpts, exist := h.opts.GetGrOpts()
	if !exist {
		h.logf("goroutine option has not been initialized")
		return
	}
	if !grOpts.Enable {
		return
	}

	if h.grCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] goroutine dump is in cooldown")
		return
	}

	if triggered := h.goroutineProfile(gNum, &grOpts); triggered {
		h.grCoolDownTime = time.Now().Add(coolDown)
		h.grTriggerCount++
	}
}

func (h *Holmes) goroutineProfile(gNum int, c *grOptions) bool {

	if !matchRule(h.grNumStats, gNum, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, c.GoroutineTriggerNumMax) {
		h.debugf(UniformLogFormat, "NODUMP", type2name[goroutine],
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs,
			c.GoroutineTriggerNumMax, h.grNumStats.data, gNum)
		return false
	}

	var buf bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeGrProfileDataToFile(buf, c, goroutine, gNum)
	return true
}

// memory start.
func (h *Holmes) memCheckAndDump(mem int) {
	// get a copy instead of locking it
	coolDown := h.opts.CoolDown

	memOpts, exist := h.opts.GetMemOpts()
	if !exist {
		h.logf("memory option has not been initialized")
		return
	}

	if !memOpts.Enable {
		return
	}

	if h.memCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] mem dump is in cooldown")
		return
	}

	if triggered := h.memProfile(mem, &memOpts); triggered {
		h.memCoolDownTime = time.Now().Add(coolDown)
		h.memTriggerCount++
	}
}

func (h *Holmes) memProfile(rss int, c *memOptions) bool {

	if !matchRule(h.memStats, rss, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig) {
		// let user know why this should not dump
		h.debugf(UniformLogFormat, "NODUMP", type2name[mem],
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
			h.memStats.data, rss)

		return false
	}

	var buf bytes.Buffer
	_ = pprof.Lookup("heap").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck

	h.writeMemProfileDataToFile(buf, c, mem, rss)
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
	if !matchRule(h.threadStats, curThreadNum, c.ThreadTriggerPercentMin, c.ThreadTriggerPercentAbs, c.ThreadTriggerPercentDiff, NotSupportTypeMaxConfig) {
		// let user know why this should not dump
		h.debugf(UniformLogFormat, "NODUMP", type2name[thread],
			c.ThreadTriggerPercentMin, c.ThreadTriggerPercentDiff, c.ThreadTriggerPercentAbs, NotSupportTypeMaxConfig,
			h.threadStats.data, curThreadNum)

		return false
	}

	var buf bytes.Buffer
	_ = pprof.Lookup("threadcreate").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	_ = pprof.Lookup("goroutine").WriteTo(&buf, int(h.opts.DumpProfileType))    // nolint: errcheck

	h.writeProfileDataToFile(buf, thread, curThreadNum)

	return true
}

// thread end.

// cpu start.
func (h *Holmes) cpuCheckAndDump(cpu int) {
	// get a copy instead of locking it
	coolDown := h.opts.CoolDown

	cpuOpts, exist := h.opts.GetCPUOpts()
	if !exist {
		h.logf("cpu option has not been initialized")
		return
	}

	if !cpuOpts.Enable {
		return
	}

	if h.cpuCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] cpu dump is in cooldown")
		return
	}

	if triggered := h.cpuProfile(cpu, &cpuOpts); triggered {
		h.cpuCoolDownTime = time.Now().Add(coolDown)
		h.cpuTriggerCount++
	}
}

func (h *Holmes) cpuProfile(curCPUUsage int, c *cpuOptions) bool {

	if !matchRule(h.cpuStats, curCPUUsage, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig) {
		// let user know why this should not dump
		h.debugf(UniformLogFormat, "NODUMP", type2name[cpu],
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
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

	h.logf(UniformLogFormat, "pprof dump to log dir", type2name[cpu],
		c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
		h.cpuStats.data, curCPUUsage)

	return true
}

func (h *Holmes) gcHeapCheckLoop() {
	for {
		// wait for the finalizer event
		<-h.finCh

		if !h.opts.GCHeapOpts.Enable {
			return
		}

		h.gcHeapCheckAndDump()
	}
}

func (h *Holmes) gcHeapCheckAndDump() {
	memStats := new(runtime.MemStats)
	runtime.ReadMemStats(memStats)

	// TODO: we can only use NextGC for now since runtime haven't expose heapmarked yet
	// and we hard code the gcPercent is 100 here.
	// may introduce a new API debug.GCHeapMarked? it can also has better performance(no STW).
	nextGC := memStats.NextGC
	prevGC := nextGC / 2 //nolint:gomnd

	memoryLimit, err := getMemoryLimit(h)
	if memoryLimit == 0 || err != nil {
		h.logf("[Holmes] get memory limit failed, memory limit: %v, error: %v", memoryLimit, err)
		return
	}

	ratio := int(100 * float64(prevGC) / float64(memoryLimit))
	h.gcHeapStats.push(ratio)

	h.gcCycleCount++
	if h.gcCycleCount < minCollectCyclesBeforeDumpStart {
		// at least collect some cycles
		// before start to judge and dump
		h.logf("[Holmes] GC cycle warming up : %d", h.gcCycleCount)
		return
	}

	if h.gcHeapCoolDownTime.After(time.Now()) {
		h.logf("[Holmes] GC heap dump is in cooldown")
		return
	}

	if triggered := h.gcHeapProfile(ratio, h.gcHeapTriggered); triggered {
		if h.gcHeapTriggered {
			// already dump twice, mark it false
			h.gcHeapTriggered = false
			h.gcHeapCoolDownTime = time.Now().Add(h.opts.CoolDown)
			h.gcHeapTriggerCount++
		} else {
			// force dump next time
			h.gcHeapTriggered = true
		}
	}
}

// gcHeapProfile will dump profile twice when triggered once.
// since the current memory profile will be merged after next GC cycle.
// And we assume the finalizer will be called before next GC cycle(it will be usually).
func (h *Holmes) gcHeapProfile(gc int, force bool) bool {
	c := h.opts.GCHeapOpts
	if !force && !matchRule(h.gcHeapStats, gc, c.GCHeapTriggerPercentMin, c.GCHeapTriggerPercentAbs, c.GCHeapTriggerPercentDiff, NotSupportTypeMaxConfig) {
		// let user know why this should not dump
		h.debugf(UniformLogFormat, "NODUMP", type2name[gcHeap],
			c.GCHeapTriggerPercentMin, c.GCHeapTriggerPercentDiff, c.GCHeapTriggerPercentAbs, NotSupportTypeMaxConfig,
			h.gcHeapStats.data, gc)

		return false
	}

	var buf bytes.Buffer
	_ = pprof.Lookup("heap").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, gcHeap, gc)

	return true
}

func (h *Holmes) writeMemProfileDataToFile(data bytes.Buffer, opts *memOptions, dumpType configureType, currentStat int) {
	h.logf(UniformLogFormat, "pprof", type2name[dumpType],
		opts.TriggerMin, opts.TriggerDiff, opts.TriggerAbs,
		NotSupportTypeMaxConfig,
		h.memStats.data, currentStat)

	writeProfileDataToFile(data, dumpType, h.opts.DumpOptions, h.logf)
}

func (h *Holmes) writeGrProfileDataToFile(data bytes.Buffer, opts *grOptions, dumpType configureType, currentStat int) {
	h.logf(UniformLogFormat, "pprof", type2name[dumpType],
		opts.TriggerMin, opts.TriggerDiff, opts.TriggerAbs,
		opts.GoroutineTriggerNumMax,
		h.grNumStats.data, currentStat)

	writeProfileDataToFile(data, dumpType, h.opts.DumpOptions, h.logf)
}

func (h *Holmes) writeProfileDataToFile(data bytes.Buffer, dumpType configureType, currentStat int) {

	switch dumpType {
	//case mem:
	//	opts := h.opts.memOpts
	//	h.logf(UniformLogFormat, "pprof", type2name[dumpType],
	//		opts.TriggerMin, opts.TriggerDiff, opts.TriggerAbs, NotSupportTypeMaxConfig,
	//		h.memStats.data, currentStat)
	case gcHeap:
		opts := h.opts.GCHeapOpts
		h.logf(UniformLogFormat, "pprof", type2name[dumpType],
			opts.GCHeapTriggerPercentMin, opts.GCHeapTriggerPercentDiff, opts.GCHeapTriggerPercentAbs, NotSupportTypeMaxConfig,
			h.gcHeapStats.data, currentStat)
	//case goroutine:
	//	opts := h.opts.grOpts
	//	h.logf(UniformLogFormat, "pprof", type2name[dumpType],
	//		opts.TriggerMin, opts.TriggerDiff, opts.TriggerAbs, opts.GoroutineTriggerNumMax,
	//		h.grNumStats.data, currentStat)
	case thread:
		opts := h.opts.ThreadOpts
		h.logf(UniformLogFormat, "pprof", type2name[dumpType],
			opts.ThreadTriggerPercentMin, opts.ThreadTriggerPercentDiff, opts.ThreadTriggerPercentAbs, NotSupportTypeMaxConfig,
			h.threadStats.data, currentStat)
	}
	writeProfileDataToFile(data, dumpType, h.opts.DumpOptions, h.logf)
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

	logger := h.opts.Logger.Load()

	if (logger == nil || logger == os.Stdout) && h.opts.logOpts.RotateEnable {
		h.opts.logOpts.RotateEnable = false
	}
}

func (h *Holmes) EnableDump(curCPU int) (err error) {
	if h.opts.CPUMaxPercent != 0 && curCPU >= h.opts.CPUMaxPercent {
		return fmt.Errorf("current cpu percent [%v] is greater than the CPUMaxPercent [%v]", cpu, h.opts.CPUMaxPercent)
	}
	return nil
}
