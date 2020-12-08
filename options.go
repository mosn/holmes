package holmes

import (
	"os"
	"path"
	"path/filepath"
	"time"
)

type options struct {
	// whether use the cgroup to calc memory or not
	UseCGroup bool

	// full path to put the profile files, default /tmp
	DumpPath string
	// default dump to binary profile, set to true if you want a text profile
	DumpProfileType dumpProfileType
	// only dump top 10 if set to false, otherwise dump all, only effective when in_text = true
	DumpFullStack bool

	LogLevel int
	Logger   *os.File

	// interval for dump loop, default 5s
	CollectInterval time.Duration

	// the cooldown time after every type of dump
	// interval for cooldown，default 1m
	// the cpu/mem/goroutine have different cooldowns of their own
	CoolDown time.Duration

	GrOpts     *grOptions
	MemOpts    *memOptions
	CPUOpts    *cpuOptions
	ThreadOpts *threadOptions
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
	return &options{
		GrOpts:          newGrOptions(),
		MemOpts:         newMemOptions(),
		CPUOpts:         newCPUOptions(),
		ThreadOpts:      newThreadOptions(),
		LogLevel:        LogLevelDebug,
		Logger:          os.Stdout,
		CollectInterval: defaultInterval,
		CoolDown:        defaultCooldown,
		DumpPath:        defaultDumpPath,
		DumpProfileType: defaultDumpProfileType,
		DumpFullStack:   false,
	}
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
		opts.CoolDown, err = time.ParseDuration(coolDown)
		return
	})
}

// WithDumpPath set the dump path for holmes.
func WithDumpPath(dumpPath string, loginfo ...string) Option {
	return optionFunc(func(opts *options) (err error) {
		f := path.Join(dumpPath, defaultLoggerName)
		if len(loginfo) > 0 {
			f = dumpPath + "/" + path.Join(loginfo...)
		}
		opts.DumpPath = filepath.Dir(f)
		opts.Logger, err = os.OpenFile(f, defaultLoggerFlags, defaultLoggerPerm)
		if err != nil && os.IsNotExist(err) {
			if err = os.MkdirAll(opts.DumpPath, 0755); err != nil {
				return
			}
			opts.Logger, err = os.OpenFile(f, defaultLoggerFlags, defaultLoggerPerm)
			if err != nil {
				return
			}
		}
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
	//   1. goroutine_num > GoroutineTriggerNumMin && goroutine diff percent > GoroutineTriggerPercentDiff
	//   2. goroutine_num > GoroutineTriggerNumAbsNum
	Enable                      bool
	GoroutineTriggerNumMin      int // goroutine trigger min in number
	GoroutineTriggerPercentDiff int // goroutine trigger diff in percent
	GoroutineTriggerNumAbs      int // goroutine trigger abs in number
}

func newGrOptions() *grOptions {
	return &grOptions{
		Enable:                      false,
		GoroutineTriggerNumAbs:      defaultGoroutineTriggerAbs,
		GoroutineTriggerPercentDiff: defaultGoroutineTriggerDiff,
		GoroutineTriggerNumMin:      defaultGoroutineTriggerMin,
	}
}

// WithGoroutineDump set the goroutine dump options.
func WithGoroutineDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.GrOpts.GoroutineTriggerNumMin = min
		opts.GrOpts.GoroutineTriggerPercentDiff = diff
		opts.GrOpts.GoroutineTriggerNumAbs = abs
		return
	})
}

type memOptions struct {
	// enable the heap dumper, should dump if one of the following requirements is matched
	//   1. memory usage > MemTriggerPercentMin && memory usage diff > MemTriggerPercentDiff
	//   2. memory usage > MemTriggerPercentAbs
	Enable                bool
	MemTriggerPercentMin  int // mem trigger minimum in percent
	MemTriggerPercentDiff int // mem trigger diff in percent
	MemTriggerPercentAbs  int // mem trigger absolute in percent
}

func newMemOptions() *memOptions {
	return &memOptions{
		Enable:                false,
		MemTriggerPercentAbs:  defaultMemTriggerAbs,
		MemTriggerPercentDiff: defaultMemTriggerDiff,
		MemTriggerPercentMin:  defaultMemTriggerMin,
	}
}

// WithMemDump set the memory dump options.
func WithMemDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.MemOpts.MemTriggerPercentMin = min
		opts.MemOpts.MemTriggerPercentDiff = diff
		opts.MemOpts.MemTriggerPercentAbs = abs
		return
	})
}

type threadOptions struct {
	Enable                   bool
	ThreadTriggerPercentMin  int // thread trigger min in number
	ThreadTriggerPercentDiff int // thread trigger diff in percent
	ThreadTriggerPercentAbs  int // thread trigger abs in number
}

func newThreadOptions() *threadOptions {
	return &threadOptions{
		Enable:                   false,
		ThreadTriggerPercentAbs:  defaultCPUTriggerAbs,
		ThreadTriggerPercentDiff: defaultCPUTriggerDiff,
		ThreadTriggerPercentMin:  defaultCPUTriggerMin,
	}
}

// WithThreadDump set the thread dump options.
func WithThreadDump(min, diff, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.ThreadOpts.ThreadTriggerPercentMin = min
		opts.ThreadOpts.ThreadTriggerPercentDiff = diff
		opts.ThreadOpts.ThreadTriggerPercentAbs = abs
		return
	})
}

type cpuOptions struct {
	// enable the cpu dumper, should dump if one of the following requirements is matched
	//   1. cpu usage > CPUTriggerMin && cpu usage diff > CPUTriggerDiff
	//   2. cpu usage > CPUTriggerAbs
	Enable                bool
	CPUTriggerPercentMin  int // cpu trigger min in percent
	CPUTriggerPercentDiff int // cpu trigger diff in percent
	CPUTriggerPercentAbs  int // cpu trigger abs inpercent
}

func newCPUOptions() *cpuOptions {
	return &cpuOptions{
		Enable:                false,
		CPUTriggerPercentAbs:  defaultCPUTriggerAbs,
		CPUTriggerPercentDiff: defaultCPUTriggerDiff,
		CPUTriggerPercentMin:  defaultCPUTriggerMin,
	}
}

// WithCPUDump set the cpu dump options.
func WithCPUDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.CPUOpts.CPUTriggerPercentMin = min
		opts.CPUOpts.CPUTriggerPercentDiff = diff
		opts.CPUOpts.CPUTriggerPercentAbs = abs
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
