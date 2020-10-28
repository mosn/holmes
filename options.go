package holmes

import (
	"os"
	"path"
	"time"
)

type options struct {
	DumpPath        string          // full path to put the profile files, default /tmp
	DumpProfileType dumpProfileType // default dump to binary profile, set to true if you want a text profile
	DumpFullStack   bool            // only dump top 10 if set to false, otherwise dump all, only effective when in_text = true
	Logger          *os.File

	// interval for dump loop, default 5s
	CollectInterval time.Duration
	// the cooldown time after every type of dump
	// the cpu/mem/goroutine have different cooldowns of their own
	CoolDown time.Duration // interval for cooldown，default 1m

	GrOpts  *grOptions
	MemOpts *memOptions
	CPUOpts *cpuOptions
}

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
		Logger:          os.Stdout,
		CollectInterval: defaultInterval,
		CoolDown:        defaultCooldown,
		DumpPath:        defaultDumpPath,
		DumpProfileType: defaultDumpProfileType,
		DumpFullStack:   false,
	}
}

// interval must be valid time duration string,
// eg. "ns", "us" (or "µs"), "ms", "s", "m", "h".
func WithCollectInterval(interval string) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.CollectInterval, err = time.ParseDuration(interval)
		return
	})
}

// coolDown must be valid time duration string,
// eg. "ns", "us" (or "µs"), "ms", "s", "m", "h".
func WithCoolDown(coolDown string) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.CoolDown, err = time.ParseDuration(coolDown)
		return
	})
}

func WithDumpPath(dumpPath string) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.DumpPath = dumpPath
		opts.Logger, err = os.OpenFile(path.Join(dumpPath, defaultLoggerName), defaultLoggerFlags, defaultLoggerPerm)
		return
	})
}

func WithBinaryDump() Option {
	return withDumpProfileType(binaryDump)
}

func WithTextDump() Option {
	return withDumpProfileType(textDump)
}

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

func WithMemDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.MemOpts.MemTriggerPercentMin = min
		opts.MemOpts.MemTriggerPercentDiff = diff
		opts.MemOpts.MemTriggerPercentAbs = abs
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

func WithCPUDump(min int, diff int, abs int) Option {
	return optionFunc(func(opts *options) (err error) {
		opts.CPUOpts.CPUTriggerPercentMin = min
		opts.CPUOpts.CPUTriggerPercentDiff = diff
		opts.CPUOpts.CPUTriggerPercentAbs = abs
		return
	})
}
