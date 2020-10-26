package holmes

import (
	"time"
)

// Config for holmes
type Config struct {
	InText        bool // default dump to binary profile, set to true if you want a text profile
	DumpFullStack bool // only dump top 10, if set to false, dump all, only effective when in_text = true

	// enable the goroutine dumper, should dump if one of the following requirements is matched
	//   1. goroutine_num > GoroutineTriggerMin && goroutine_diff > GoroutineTriggerDiff
	//   2. goroutine_num > GoroutineTriggerAbs
	EnableGoroutineDump  bool
	GoroutineTriggerMin  int
	GoroutineTriggerDiff int
	GoroutineTriggerAbs  int

	// enable the cpu dumper, should dump if one of the following requirements is matched
	//   1. cpu usage > CPUTriggerMin && cpu usage diff > CPUTriggerDiff
	//   2. cpu usage > CPUTriggerAbs
	EnableCPUDump  bool
	CPUTriggerMin  int // cpu trigger min in percent
	CPUTriggerDiff int // cpu trigger diff in percent
	CPUTriggerAbs  int // cpu trigger abs inpercent

	// enable the heap dumper, should dump if one of the following requirements is matched
	//   1. memory usage > MemTriggerMin && cpu usage diff > MemTriggerDiff
	//   2. memory usage > MemTriggerAbs
	EnableMemDump  bool
	MemTriggerMin  int // mem trigger minimum in percent
	MemTriggerAbs  int // mem trigger absolute in percent
	MemTriggerDiff int // mem trigger diff in percent

	// interval for dump loop, default 5s
	CollectInterval time.Duration
	// the cooldown time after every type of dump
	// the cpu/mem/goroutine have different cooldowns of their own
	CoolDown time.Duration // cooldown 的时间间隔，单位为分钟
}
