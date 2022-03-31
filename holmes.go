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
	"io/ioutil"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"
)

// Holmes is a self-aware profile dumper.
type Holmes struct {
	opts *options

	// stats
	collectCount             int
	gcCycleCount             int
	threadTriggerCount       int
	cpuTriggerCount          int
	memTriggerCount          int
	grTriggerCount           int
	gcHeapTriggerCount       int
	shrinkThreadTriggerCount int

	// cooldown
	threadCoolDownTime    time.Time
	cpuCoolDownTime       time.Time
	memCoolDownTime       time.Time
	gcHeapCoolDownTime    time.Time
	grCoolDownTime        time.Time
	shrinkThrCoolDownTime time.Time

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

	// lock Protect the following
	sync.Mutex
	// channel for GC sweep finalizer event
	gcEventsCh chan struct{}
	// profiler reporter channels
	rptEventsCh chan rptEvent
}

type ProfileReporter interface {
	Report(pType string, buf []byte, reason string, eventID string) error
}

// New creates a holmes dumper.
func New(opts ...Option) (*Holmes, error) {
	holmes := &Holmes{

		opts:    newOptions(),
		stopped: 1, // Initialization should be off
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
	h.opts.threadOpts.Enable = true
	return h
}

// DisableThreadDump disables the goroutine dump.
func (h *Holmes) DisableThreadDump() *Holmes {
	h.opts.threadOpts.Enable = false
	return h
}

// EnableGoroutineDump enables the goroutine dump.
func (h *Holmes) EnableGoroutineDump() *Holmes {
	h.opts.grOpts.Enable = true
	return h
}

// DisableGoroutineDump disables the goroutine dump.
func (h *Holmes) DisableGoroutineDump() *Holmes {
	h.opts.grOpts.Enable = false
	return h
}

// EnableCPUDump enables the CPU dump.
func (h *Holmes) EnableCPUDump() *Holmes {
	h.opts.cpuOpts.Enable = true
	return h
}

// DisableCPUDump disables the CPU dump.
func (h *Holmes) DisableCPUDump() *Holmes {
	h.opts.cpuOpts.Enable = false
	return h
}

// EnableMemDump enables the mem dump.
func (h *Holmes) EnableMemDump() *Holmes {
	h.opts.memOpts.Enable = true
	return h
}

// DisableMemDump disables the mem dump.
func (h *Holmes) DisableMemDump() *Holmes {
	h.opts.gCHeapOpts.Enable = false
	return h
}

// EnableGCHeapDump enables the GC heap dump.
func (h *Holmes) EnableGCHeapDump() *Holmes {
	h.opts.gCHeapOpts.Enable = true
	return h
}

// DisableGCHeapDump disables the gc heap dump.
func (h *Holmes) DisableGCHeapDump() *Holmes {
	h.opts.gCHeapOpts.Enable = false
	return h
}

// EnableShrinkThread enables shrink thread
func (h *Holmes) EnableShrinkThread() *Holmes {
	h.opts.ShrinkThrOptions.Enable = true
	return h
}

// DisableShrinkThread disables shrink thread
func (h *Holmes) DisableShrinkThread() *Holmes {
	h.opts.ShrinkThrOptions.Enable = false
	return h
}

func finalizerCallback(gc *gcHeapFinalizer) {
	defer func() {
		if r := recover(); r != nil {
			gc.h.Errorf("Panic in finalizer callback: %v", r)
		}
	}()
	// disable or stop gc clean up normally
	if atomic.LoadInt64(&gc.h.stopped) == 1 {
		return
	}

	// register the finalizer again
	runtime.SetFinalizer(gc, finalizerCallback)

	// read channel should be atomic.
	ch := gc.h.gcEventsCh
	if ch == nil {
		return
	}
	// Notice: here may be a litte race, will panic when ch is closed now.
	// we just leave it since it is very small and there is a recover.
	select {
	case ch <- struct{}{}:
	default:
		gc.h.Errorf("can not send event to finalizer channel immediately, may be analyzer blocked?")
	}
}

// it won't fit into tiny span since this struct contains point.
type gcHeapFinalizer struct {
	h *Holmes
}

func (h *Holmes) startGCCycleLoop(ch chan struct{}) {
	h.gcHeapStats = newRing(minCollectCyclesBeforeDumpStart)

	gc := &gcHeapFinalizer{
		h,
	}

	runtime.SetFinalizer(gc, finalizerCallback)

	go gc.h.gcHeapCheckLoop(ch)
}

// Start starts the dump loop of holmes.
func (h *Holmes) Start() {
	h.Lock()
	defer h.Unlock()

	if !atomic.CompareAndSwapInt64(&h.stopped, 1, 0) {
		//nolint
		h.Errorf("Holmes has started, please don't start it again.")
		return
	}

	gcEventsCh := make(chan struct{}, 1)
	rptCh := make(chan rptEvent, 32)
	h.gcEventsCh = gcEventsCh
	h.rptEventsCh = rptCh

	h.initEnvironment()
	go h.startDumpLoop()
	go h.startReporter(rptCh)

	h.startGCCycleLoop(gcEventsCh)
}

// Stop the dump loop.
func (h *Holmes) Stop() {
	h.Lock()
	defer h.Unlock()

	if !atomic.CompareAndSwapInt64(&h.stopped, 0, 1) {
		//nolint
		fmt.Println("Holmes has stop, please don't stop it again.")
		return
	}

	if gcEventsCh := h.gcEventsCh; gcEventsCh != nil {
		h.gcEventsCh = nil
		close(gcEventsCh)
	}
	if rptEventsCh := h.rptEventsCh; rptEventsCh != nil {
		h.rptEventsCh = nil
		close(rptEventsCh)
	}
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

	for {
		select {
		case <-h.opts.intervalResetting:
			// wait for go version update to 1.15
			// can use Reset API directly here. pkg.go.dev/time#Ticker.Reset
			// we can't use the `for-range` here, because the range loop
			// caches the variable to be lopped and then it can't be overwritten
			itv := h.opts.CollectInterval
			h.Infof("[Holmes] collect interval is resetting to [%v]\n", itv) //nolint:forbidigo
			ticker = time.NewTicker(itv)

		default:
			// bug fix: https://github.com/mosn/holmes/issues/63
			// make sure that the message inside intervalResetting channel
			// would be consumed before ticker.C.
			<-ticker.C
			if atomic.LoadInt64(&h.stopped) == 1 {
				h.Infof("[Holmes] dump loop stopped") //nolint:forbidigo
				return
			}

			cpuCore, err := h.getCPUCore()
			if cpuCore == 0 || err != nil {
				h.Errorf("[Holmes] get CPU core failed, CPU core: %v, error: %v", cpuCore, err)
				return
			}

			memoryLimit, err := h.getMemoryLimit()
			if memoryLimit == 0 || err != nil {
				h.Errorf("[Holmes] get memory limit failed, memory limit: %v, error: %v", memoryLimit, err)
				return
			}

			cpu, mem, gNum, tNum, err := collect(cpuCore, memoryLimit)
			if err != nil {
				h.Errorf("failed to collect resource usage: %v", err.Error())

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
				h.Debugf("[Holmes] warming up cycle : %d", h.collectCount)

				continue
			}

			if err := h.EnableDump(cpu); err != nil {
				h.Infof("[Holmes] unable to dump: %v", err)

				continue
			}

			h.goroutineCheckAndDump(gNum)
			h.memCheckAndDump(mem)
			h.cpuCheckAndDump(cpu)
			h.threadCheckAndDump(tNum)
			h.threadCheckAndShrink(tNum)
		}
	}
}

// goroutine start.
func (h *Holmes) goroutineCheckAndDump(gNum int) {
	// get a copy instead of locking it
	grOpts := h.opts.GetGrOpts()
	if !grOpts.Enable {
		return
	}

	if h.grCoolDownTime.After(time.Now()) {
		h.Debugf("[Holmes] goroutine dump is in cooldown")
		return
	}
	// grOpts is a struct, no escape.
	if triggered := h.goroutineProfile(gNum, grOpts); triggered {
		h.grCoolDownTime = time.Now().Add(grOpts.CoolDown)
		h.grTriggerCount++
	}
}

func (h *Holmes) goroutineProfile(gNum int, c grOptions) bool {
	match, reason := matchRule(h.grNumStats, gNum, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, c.GoroutineTriggerNumMax)
	if !match {
		h.Infof(UniformLogFormat, "NODUMP", check2name[goroutine],
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs,
			c.GoroutineTriggerNumMax, h.grNumStats.data, gNum)
		return false
	}

	h.Alertf("holmes.goroutine", UniformLogFormat, "pprof ", check2name[goroutine],
		c.TriggerMin, c.TriggerDiff, c.TriggerAbs,
		c.GoroutineTriggerNumMax,
		h.grNumStats.data, gNum)

	var buf bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, goroutine, "")

	h.ReportProfile(type2name[goroutine], buf.Bytes(), reason, "")
	return true
}

// memory start.
func (h *Holmes) memCheckAndDump(mem int) {
	// get a copy instead of locking it
	memOpts := h.opts.GetMemOpts()
	if !memOpts.Enable {
		return
	}

	if h.memCoolDownTime.After(time.Now()) {
		h.Debugf("[Holmes] mem dump is in cooldown")
		return
	}
	// memOpts is a struct, no escape.
	if triggered := h.memProfile(mem, memOpts); triggered {
		h.memCoolDownTime = time.Now().Add(memOpts.CoolDown)
		h.memTriggerCount++
	}
}

func (h *Holmes) memProfile(rss int, c typeOption) bool {
	match, reason := matchRule(h.memStats, rss, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig)
	if !match {
		// let user know why this should not dump
		h.Infof(UniformLogFormat, "NODUMP", check2name[mem],
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
			h.memStats.data, rss)

		return false
	}

	h.Alertf("holmes.memory", UniformLogFormat, "pprof", check2name[mem],
		c.TriggerMin, c.TriggerDiff, c.TriggerAbs,
		NotSupportTypeMaxConfig, h.memStats, rss)

	var buf bytes.Buffer
	_ = pprof.Lookup("heap").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck

	h.writeProfileDataToFile(buf, mem, "")

	h.ReportProfile(type2name[mem], buf.Bytes(), reason, "")
	return true
}

func (h *Holmes) threadCheckAndShrink(threadNum int) {
	opts := h.opts.GetShrinkThreadOpts()

	if !opts.Enable {
		return
	}

	if h.shrinkThrCoolDownTime.After(time.Now()) {
		return
	}

	if threadNum > opts.Threshold {
		// 100x Delay time a cooldown time as default
		delay := opts.Delay * 100
		// one hour at least
		if delay < time.Hour {
			delay = time.Hour
		}
		if delay > time.Hour*24 {
			delay = time.Hour * 24
		}
		h.shrinkThrCoolDownTime = time.Now().Add(delay)

		h.Alertf("holmes.thread", "current thread number(%v) larger than threshold(%v), will start to shrink thread after %v", threadNum, opts.Threshold, opts.Delay)

		// do not shrink thread immediately
		time.AfterFunc(opts.Delay, func() {
			h.startShrinkThread()
		})
	}
}

// thread start.
func (h *Holmes) threadCheckAndDump(threadNum int) {
	threadOpts := h.opts.GetThreadOpts()
	if !threadOpts.Enable {
		return
	}

	if h.threadCoolDownTime.After(time.Now()) {
		h.Debugf("[Holmes] thread dump is in cooldown")
		return
	}
	// threadOpts is a struct, no escape.
	if triggered := h.threadProfile(threadNum, threadOpts); triggered {
		h.threadCoolDownTime = time.Now().Add(threadOpts.CoolDown)
		h.threadTriggerCount++
	}
}

// TODO: better only shrink the threads that are idle.
func (h *Holmes) startShrinkThread() {

	curThreadNum := getThreadNum()
	opts := h.opts.GetShrinkThreadOpts()

	n := curThreadNum - opts.Threshold

	// check again after the timer triggered
	if opts.Enable && n > 0 {
		h.shrinkThreadTriggerCount++
		h.Infof("[holmes] start to shrink %v threads, now: %v", n, curThreadNum)

		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			// avoid close too much thread in batch.
			time.Sleep(time.Millisecond * 100)

			go func() {
				defer wg.Done()
				runtime.LockOSThread()
			}()
		}
		wg.Wait()

		h.Infof("[holmes] finished shrink threads, now: %v", getThreadNum())
	}
}

func (h *Holmes) threadProfile(curThreadNum int, c typeOption) bool {
	match, reason := matchRule(h.threadStats, curThreadNum, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig)
	if !match {
		// let user know why this should not dump
		h.Infof(UniformLogFormat, "NODUMP", check2name[thread],
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
			h.threadStats.data, curThreadNum)

		return false
	}

	h.Alertf("holmes.thread", UniformLogFormat, "pprof", check2name[thread],
		c.TriggerMin, c.TriggerDiff, c.TriggerAbs,
		NotSupportTypeMaxConfig, h.threadStats, curThreadNum)

	eventID := fmt.Sprintf("thr-%d", h.threadTriggerCount)
	var buf bytes.Buffer

	_ = pprof.Lookup("threadcreate").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, thread, eventID)

	h.ReportProfile(type2name[thread], buf.Bytes(), reason, eventID)

	buf.Reset()
	_ = pprof.Lookup("goroutine").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, goroutine, eventID)

	h.ReportProfile(type2name[goroutine], buf.Bytes(), reason, eventID)

	return true
}

// thread end.

// cpu start.
func (h *Holmes) cpuCheckAndDump(cpu int) {
	cpuOpts := h.opts.GetCPUOpts()
	if !cpuOpts.Enable {
		return
	}

	if h.cpuCoolDownTime.After(time.Now()) {
		h.Debugf("[Holmes] cpu dump is in cooldown")
		return
	}
	// cpuOpts is a struct, no escape.
	if triggered := h.cpuProfile(cpu, cpuOpts); triggered {
		h.cpuCoolDownTime = time.Now().Add(cpuOpts.CoolDown)
		h.cpuTriggerCount++
	}
}

func (h *Holmes) cpuProfile(curCPUUsage int, c typeOption) bool {
	match, reason := matchRule(h.cpuStats, curCPUUsage, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig)
	if !match {
		// let user know why this should not dump
		h.Infof(UniformLogFormat, "NODUMP", check2name[cpu],
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
			h.cpuStats.data, curCPUUsage)

		return false
	}

	h.Alertf("holmes.cpu", UniformLogFormat, "pprof dump", check2name[cpu],
		c.TriggerMin, c.TriggerDiff, c.TriggerAbs, NotSupportTypeMaxConfig,
		h.cpuStats.data, curCPUUsage)

	bf, binFileName, err := getBinaryFileNameAndCreate(h.opts.DumpPath, cpu, "")
	if err != nil {
		h.Errorf("[Holmes] failed to create cpu profile file: %v", err.Error())
		return false
	}
	defer bf.Close() // nolint: errcheck

	err = pprof.StartCPUProfile(bf)
	if err != nil {
		h.Errorf("[Holmes] failed to profile cpu: %v", err.Error())
		return false
	}

	time.Sleep(defaultCPUSamplingTime)
	pprof.StopCPUProfile()

	if h.opts.DumpToLogger {
		bfCpy, err := ioutil.ReadFile(binFileName)
		if err != nil {
			h.Errorf("encounter error when dumping profile to logger, failed to read cpu profile file: %v", err)
			return true
		}
		h.Infof("[Holmes] CPU profile:: \n" + string(bfCpy))
	}

	if opts := h.opts.GetReporterOpts(); opts.active == 1 {
		bfCpy, err := ioutil.ReadFile(binFileName)
		if err != nil {
			h.Errorf("[holmes reporter] failed to read cpu profile file: %v", err)
			return true
		}
		h.ReportProfile(type2name[cpu], bfCpy, reason, "")
	}

	return true
}

func (h *Holmes) gcHeapCheckLoop(ch chan struct{}) {
	for range ch {
		h.gcHeapCheckAndDump()
	}
}

func (h *Holmes) gcHeapCheckAndDump() {
	gcHeapOpts := h.opts.GetGcHeapOpts()

	if !gcHeapOpts.Enable || atomic.LoadInt64(&h.stopped) == 1 {
		return
	}

	memStats := new(runtime.MemStats)
	runtime.ReadMemStats(memStats)

	// TODO: we can only use NextGC for now since runtime haven't expose heapmarked yet
	// and we hard code the gcPercent is 100 here.
	// may introduce a new API debug.GCHeapMarked? it can also has better performance(no STW).
	nextGC := memStats.NextGC
	prevGC := nextGC / 2 //nolint:gomnd

	memoryLimit, err := h.getMemoryLimit()
	if memoryLimit == 0 || err != nil {
		h.Errorf("[Holmes] get memory limit failed, memory limit: %v, error: %v", memoryLimit, err)
		return
	}

	ratio := int(100 * float64(prevGC) / float64(memoryLimit))
	h.gcHeapStats.push(ratio)

	h.gcCycleCount++
	if h.gcCycleCount < minCollectCyclesBeforeDumpStart {
		// at least collect some cycles
		// before start to judge and dump
		h.Debugf("[Holmes] GC cycle warming up : %d", h.gcCycleCount)
		return
	}

	if h.gcHeapCoolDownTime.After(time.Now()) {
		h.Debugf("[Holmes] GC heap dump is in cooldown")
		return
	}

	if triggered := h.gcHeapProfile(ratio, h.gcHeapTriggered, gcHeapOpts); triggered {
		if h.gcHeapTriggered {
			// already dump twice, mark it false
			h.gcHeapTriggered = false
			h.gcHeapCoolDownTime = time.Now().Add(gcHeapOpts.CoolDown)
			h.gcHeapTriggerCount++
		} else {
			// force dump next time
			h.gcHeapTriggered = true
		}
	}
}

func (h *Holmes) getCPUCore() (float64, error) {
	if h.opts.cpuCore > 0 {
		return h.opts.cpuCore, nil
	}

	if h.opts.UseGoProcAsCPUCore {
		return float64(runtime.GOMAXPROCS(-1)), nil
	}

	if h.opts.UseCGroup {
		return getCGroupCPUCore()
	}

	return float64(runtime.NumCPU()), nil
}

func (h *Holmes) getMemoryLimit() (uint64, error) {
	if h.opts.memoryLimit > 0 {
		return h.opts.memoryLimit, nil
	}

	if h.opts.UseCGroup {
		return getCGroupMemoryLimit()
	}

	return getNormalMemoryLimit()
}

// gcHeapProfile will dump profile twice when triggered once.
// since the current memory profile will be merged after next GC cycle.
// And we assume the finalizer will be called before next GC cycle(it will be usually).
func (h *Holmes) gcHeapProfile(gc int, force bool, c typeOption) bool {
	match, reason := matchRule(h.gcHeapStats, gc, c.TriggerMin, c.TriggerAbs, c.TriggerDiff, NotSupportTypeMaxConfig)
	if !force && !match {
		// let user know why this should not dump
		h.Infof(UniformLogFormat, "NODUMP", check2name[gcHeap],
			c.TriggerMin, c.TriggerDiff, c.TriggerAbs,
			NotSupportTypeMaxConfig,
			h.gcHeapStats.data, gc)

		return false
	}

	h.Alertf("holmes.gcheap", UniformLogFormat, "pprof", check2name[gcHeap],
		c.TriggerMin, c.TriggerDiff, c.TriggerAbs,
		NotSupportTypeMaxConfig, h.gcHeapStats, gc)

	// gcTriggerCount only increased after got both two profiles
	eventID := fmt.Sprintf("heap-%d", h.grTriggerCount)

	var buf bytes.Buffer
	_ = pprof.Lookup("heap").WriteTo(&buf, int(h.opts.DumpProfileType)) // nolint: errcheck
	h.writeProfileDataToFile(buf, gcHeap, eventID)

	h.ReportProfile(type2name[gcHeap], buf.Bytes(), reason, eventID)
	return true
}

func (h *Holmes) writeProfileDataToFile(data bytes.Buffer, dumpType configureType, eventID string) {
	fileName, err := writeFile(data, dumpType, h.opts.DumpOptions, eventID)
	if err != nil {
		h.Errorf("failed to write profile to file(%v), err: %s", fileName, err.Error())
		return
	}

	if h.opts.DumpOptions.DumpToLogger {
		h.Infof(fmt.Sprintf("[Holmes] %v profile: \n", check2name[dumpType]) + data.String())
	}

	h.Infof("[Holmes] pprof %v profile write to file %v successfully", check2name[dumpType], fileName)
}

func (h *Holmes) initEnvironment() {
	// whether the max memory is limited by cgroup
	if h.opts.UseCGroup {
		h.Infof("[Holmes] use cgroup to limit memory")
	} else {
		h.Infof("[Holmes] use the default memory percent calculated by gopsutil")
	}
}

func (h *Holmes) EnableDump(curCPU int) (err error) {
	if h.opts.CPUMaxPercent != 0 && curCPU >= h.opts.CPUMaxPercent {
		return fmt.Errorf("current cpu percent [%v] is greater than the CPUMaxPercent [%v]", cpu, h.opts.CPUMaxPercent)
	}
	return nil
}

// Set sets holmes's optional after initialing.
func (h *Holmes) Set(opts ...Option) error {
	h.opts.L.Lock()
	defer h.opts.L.Unlock()

	for _, opt := range opts {
		if err := opt.apply(h.opts); err != nil {
			return err
		}
	}
	return nil
}

func (h *Holmes) DisableProfileReporter() {
	atomic.StoreInt32(&h.opts.rptOpts.active, 0)
}

func (h *Holmes) EnableProfileReporter() {
	opt := h.opts.GetReporterOpts()
	if opt.reporter == nil {
		h.Infof("failed to enable profile reporter since reporter is empty")
		return
	}
	atomic.StoreInt32(&h.opts.rptOpts.active, 1)
}

func (h *Holmes) ReportProfile(pType string, buf []byte, reason string, eventID string) {
	defer func() {
		if r := recover(); r != nil {
			h.Errorf("Panic during report profile: %v", r)
		}
	}()

	if atomic.LoadInt64(&h.stopped) == 1 {
		return
	}

	opts := h.opts.GetReporterOpts()
	if opts.active == 0 {
		return
	}

	msg := rptEvent{
		PType:   pType,
		Buf:     buf,
		Reason:  reason,
		EventID: eventID,
	}

	// read channel should be atomic.
	ch := h.rptEventsCh
	if ch == nil {
		return
	}
	// Notice: here may be a litte race, will panic when ch is closed now.
	// we just leave it since it is very small and there is a recover.
	select {
	case ch <- msg:
	default:
		h.Warnf("reporter channel is full, will ignore it")
	}
}

// startReporter starts a background goroutine to consume event channel,
// and finish it at after receive from cancel channel.
func (h *Holmes) startReporter(ch chan rptEvent) {
	go func() {
		for evt := range ch {
			opts := h.opts.GetReporterOpts()
			if opts.reporter == nil {
				h.Infof("reporter is nil, please initial it before startReporter")
				// drop the event
				continue
			}
			// It's supposed to be sending judgment, isn't it?
			err := opts.reporter.Report(evt.PType, evt.Buf, evt.Reason, evt.EventID) // nolint: errcheck
			if err != nil {
				h.Infof("reporter err:", err)

			}
		}
	}()
}
