package holmes

import (
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/go-units"
)

type options struct {
	// whether use the cgroup to calc memory or not
	UseCGroup bool

	// overwrite the system level memory limitation when > 0.
	memoryLimit uint64

	*DumpOptions

	LogLevel int
	Logger   atomic.Value

	// interval for dump loop, default 5s
	CollectInterval time.Duration

	// the cooldown time after every type of dump
	// interval for cooldown，default 1m
	// the cpu/mem/goroutine have different cooldowns of their own

	// todo should we move CoolDown into Gr/CPU/MEM/GCheap Opts and support
	// set different `CoolDown` for different opts?
	CoolDown time.Duration

	// if current cpu usage percent is greater than CPUMaxPercent,
	// holmes would not dump all types profile, cuz this
	// move may result of the system crash.
	CPUMaxPercent int

	logOpts    *loggerOptions
	grOpts     *grOptions
	memOpts    *memOptions
	GCHeapOpts *gcHeapOptions
	cpuOpts    *cpuOptions
	threadOpts *threadOptions
}

// DumpOptions contains configuration about dump file.
type DumpOptions struct {
	// full path to put the profile files, default /tmp
	DumpPath string
	// default dump to binary profile, set to true if you want a text profile
	DumpProfileType dumpProfileType
	// only dump top 10 if set to false, otherwise dump all, only effective when in_text = true
	DumpFullStack bool
}

// GetMemOpts return a copy of memOpts
// if memOpts not exist return a empty memOptions and false
// memOptions which be returned
func (o *options) GetMemOpts() (memOptions, bool) {
	if o.memOpts == nil {
		return memOptions{}, false
	}
	o.memOpts.L.RLock()
	defer o.memOpts.L.RUnlock()
	return *o.memOpts, true
}

// GetCPUOpts return a copy of cpuOpts
// if cpuOpts not exist return a empty cpuOptions and false
func (o *options) GetCPUOpts() (cpuOptions, bool) {
	if o.cpuOpts == nil {
		return cpuOptions{}, false
	}
	o.cpuOpts.L.RLock()
	defer o.cpuOpts.L.RUnlock()
	return *o.cpuOpts, true
}

// GetGrOpts return a copy of memOpts
// if grOpts not exist return a empty grOptions and false
func (o *options) GetGrOpts() (grOptions, bool) {
	if o.grOpts == nil {
		return grOptions{}, false
	}
	o.grOpts.L.RLock()
	defer o.grOpts.L.RUnlock()
	return *o.grOpts, true
}

// GetThreadOpts return a copy of memOpts
// if threadOpts not exist return a empty threadOptions and false
func (o *options) GetThreadOpts() (threadOptions, bool) {
	if o.threadOpts == nil {
		return threadOptions{}, false
	}
	o.threadOpts.L.RLock()
	defer o.threadOpts.L.RUnlock()
	return *o.threadOpts, true
}

func (o *options) SetCoolDown(new time.Duration) {
	o.CoolDown = new
}

// Option holmes option type.
type Option interface {
	apply(*options) error
}

type optionFunc func(*options) error

func (f optionFunc) apply(opts *options) error {
	return f(opts)
}

func newOptions() *options {
	o := &options{
		logOpts:         newLoggerOptions(),
		grOpts:          newGrOptions(),
		memOpts:         newMemOptions(),
		GCHeapOpts:      newGCHeapOptions(),
		cpuOpts:         newCPUOptions(),
		threadOpts:      newThreadOptions(),
		LogLevel:        LogLevelDebug,
		CollectInterval: defaultInterval,
		CoolDown:        defaultCooldown,
		DumpOptions: &DumpOptions{
			DumpPath:        defaultDumpPath,
			DumpProfileType: defaultDumpProfileType,
			DumpFullStack:   false,
		},
	}
	o.Logger.Store(os.Stdout)
	return o
}

// WithCollectInterval : interval must be valid time duration string,
// eg. "ns", "us" (or "µs"), "ms", "s", "m", "h".
func WithCollectInterval(interval string) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.CollectInterval, err = time.ParseDuration(interval)
		return
	})
}

// WithCoolDown : coolDown must be valid time duration string,
// eg. "ns", "us" (or "µs"), "ms", "s", "m", "h".
func WithCoolDown(coolDown string) Option {
	return optionFunc(func(opts *options) (err error) {
		cd, err := time.ParseDuration(coolDown)
		if err != nil {
			return err
		}
		opts.SetCoolDown(cd)
		return
	})
}

// WithCPUMax : set the CPUMaxPercent parameter as max
func WithCPUMax(max int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.CPUMaxPercent = max
		return
	})
}

// WithDumpPath set the dump path for holmes.
func WithDumpPath(dumpPath string, loginfo ...string) Option {
	return optionFunc(func(opts *options) (err error) {
		var logger *os.File
		f := path.Join(dumpPath, defaultLoggerName)
		if len(loginfo) > 0 {
			f = dumpPath + "/" + path.Join(loginfo...)
		}
		opts.DumpPath = filepath.Dir(f)
		logger, err = os.OpenFile(filepath.Clean(f), defaultLoggerFlags, defaultLoggerPerm)
		if err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(opts.DumpPath, 0755); err != nil {
				return
			}
			logger, err = os.OpenFile(filepath.Clean(f), defaultLoggerFlags, defaultLoggerPerm)
			if err != nil {
				return
			}
		}
		opts.Logger.Store(logger)
		return
	})
}

// WithBinaryDump set dump mode to binary.
func WithBinaryDump() Option {
	return withDumpProfileType(binaryDump)
}

// WithTextDump set dump mode to text.
func WithTextDump() Option {
	return withDumpProfileType(textDump)
}

// WithFullStack set to dump full stack or top 10 stack, when dump in text mode.
func WithFullStack(isFull bool) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.DumpFullStack = isFull
		return
	})
}

func withDumpProfileType(profileType dumpProfileType) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.DumpProfileType = profileType
		return
	})
}

type grOptions struct {
	// enable the goroutine dumper, should dump if one of the following requirements is matched
	//   1. goroutine_num > TriggerMin && goroutine_num < GoroutineTriggerNumMax && goroutine diff percent > TriggerDiff
	//   2. goroutine_num > GoroutineTriggerNumAbsNum && goroutine_num < GoroutineTriggerNumMax
	*baseOptions
	GoroutineTriggerNumMax int // goroutine trigger max in number
}

func (g *grOptions) SetTriggerNumMax(new int) {
	g.GoroutineTriggerNumMax = new
}

func newGrOptions() *grOptions {
	base := &baseOptions{
		L:           sync.RWMutex{},
		Enable:      false,
		TriggerAbs:  defaultGoroutineTriggerAbs,
		TriggerDiff: defaultGoroutineTriggerDiff,
		TriggerMin:  defaultGoroutineTriggerMin,
	}
	return &grOptions{baseOptions: base}
}

// WithGoroutineDump set the goroutine dump options.
func WithGoroutineDump(min int, diff int, abs int, max int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.grOpts.SetTriggerMin(min)
		opts.grOpts.SetTriggerDiff(diff)
		opts.grOpts.SetTriggerAbs(abs)
		opts.grOpts.SetTriggerNumMax(max)
		return
	})
}

type baseOptions struct {
	L      sync.RWMutex
	Enable bool
	// mem/cpu/gcheap trigger minimum in percent, goroutine/thread trigger minimum in number
	TriggerMin int

	// mem/cpu/gcheap trigger abs in percent, goroutine/thread trigger abs in number
	TriggerAbs int

	// mem/cpu/gcheap/goroutine/thread trigger diff in percent
	TriggerDiff int
}

func (base *baseOptions) SetEnable(new bool) {
	base.L.Lock()
	defer base.L.Unlock()
	base.Enable = new
}

func (base *baseOptions) SetTriggerMin(new int) {
	base.L.Lock()
	defer base.L.Unlock()
	base.TriggerMin = new
}

func (base *baseOptions) SetTriggerDiff(new int) {
	base.L.Lock()
	defer base.L.Unlock()
	base.TriggerDiff = new
}

func (base *baseOptions) SetTriggerAbs(new int) {
	base.L.Lock()
	defer base.L.Unlock()
	base.TriggerAbs = new
}

type memOptions struct {
	// enable the heap dumper, should dump if one of the following requirements is matched
	// in percent
	//   1. memory usage > TriggerMin && memory usage diff > TriggerDiff
	//   2. memory usage > TriggerAbs
	*baseOptions
}

func newMemOptions() *memOptions {
	base := &baseOptions{
		L:           sync.RWMutex{},
		Enable:      false,
		TriggerMin:  defaultMemTriggerMin,
		TriggerAbs:  defaultMemTriggerAbs,
		TriggerDiff: defaultMemTriggerDiff,
	}
	return &memOptions{base}
}

// WithMemDump set the memory dump options.
func WithMemDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.memOpts.SetTriggerMin(min)
		opts.memOpts.SetTriggerDiff(diff)
		opts.memOpts.SetTriggerAbs(abs)
		return
	})
}

type gcHeapOptions struct {
	// enable the heap dumper, should dump if one of the following requirements is matched
	//   1. GC heap usage > GCHeapTriggerPercentMin && GC heap usage diff > GCHeapTriggerPercentDiff
	//   2. GC heap usage > GCHeapTriggerPercentAbs
	Enable                   bool
	GCHeapTriggerPercentMin  int // GC heap trigger minimum in percent
	GCHeapTriggerPercentDiff int // GC heap trigger diff in percent
	GCHeapTriggerPercentAbs  int // GC heap trigger absolute in percent
}

func newGCHeapOptions() *gcHeapOptions {
	return &gcHeapOptions{
		Enable:                   false,
		GCHeapTriggerPercentAbs:  defaultGCHeapTriggerAbs,
		GCHeapTriggerPercentDiff: defaultGCHeapTriggerDiff,
		GCHeapTriggerPercentMin:  defaultGCHeapTriggerMin,
	}
}

// WithGCHeapDump set the GC heap dump options.
func WithGCHeapDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.GCHeapOpts.GCHeapTriggerPercentMin = min
		opts.GCHeapOpts.GCHeapTriggerPercentDiff = diff
		opts.GCHeapOpts.GCHeapTriggerPercentAbs = abs
		return
	})
}

// WithMemoryLimit overwrite the system level memory limit when it > 0.
func WithMemoryLimit(limit uint64) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.memoryLimit = limit
		return
	})
}

type threadOptions struct {
	*baseOptions
}

func newThreadOptions() *threadOptions {
	base := &baseOptions{
		L:           sync.RWMutex{},
		Enable:      false,
		TriggerMin:  defaultMemTriggerMin,
		TriggerAbs:  defaultMemTriggerAbs,
		TriggerDiff: defaultMemTriggerDiff,
	}
	return &threadOptions{base}
}

// WithThreadDump set the thread dump options.
func WithThreadDump(min, diff, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.cpuOpts.SetTriggerMin(min)
		opts.cpuOpts.SetTriggerDiff(diff)
		opts.cpuOpts.SetTriggerAbs(abs)
		return
	})
}

type cpuOptions struct {
	// enable the cpu dumper, should dump if one of the following requirements is matched
	// in percent
	//   1. cpu usage > CPUTriggerMin && cpu usage diff > CPUTriggerDiff
	//   2. cpu usage > CPUTriggerAbs
	*baseOptions
}

func newCPUOptions() *cpuOptions {
	base := &baseOptions{
		L:           sync.RWMutex{},
		Enable:      false,
		TriggerMin:  defaultCPUTriggerMin,
		TriggerAbs:  defaultCPUTriggerAbs,
		TriggerDiff: defaultCPUTriggerDiff,
	}
	return &cpuOptions{base}
}

// WithCPUDump set the cpu dump options.
func WithCPUDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.cpuOpts.SetTriggerMin(min)
		opts.cpuOpts.SetTriggerDiff(diff)
		opts.cpuOpts.SetTriggerAbs(abs)
		return
	})
}

// WithCGroup set holmes use cgroup or not.
func WithCGroup(useCGroup bool) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.UseCGroup = useCGroup
		return
	})
}

func WithLoggerLevel(level int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.LogLevel = level
		return
	})
}

type loggerOptions struct {
	RotateEnable    bool
	SplitLoggerSize int64 // SplitLoggerSize The size of the log split
}

func newLoggerOptions() *loggerOptions {
	return &loggerOptions{
		RotateEnable:    true,
		SplitLoggerSize: defaultShardLoggerSize,
	}
}

// WithLoggerSplit set the split log options.
// eg. "b/B", "k/K" "kb/Kb" "mb/Mb", "gb/Gb" "tb/Tb" "pb/Pb".
func WithLoggerSplit(enable bool, shardLoggerSize string) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.logOpts.RotateEnable = enable
		if !enable {
			return nil
		}

		parseShardLoggerSize, err := units.FromHumanSize(shardLoggerSize)
		if err != nil || (err == nil && parseShardLoggerSize <= 0) {
			opts.logOpts.SplitLoggerSize = defaultShardLoggerSize
			return
		}

		opts.logOpts.SplitLoggerSize = parseShardLoggerSize
		return
	})
}
