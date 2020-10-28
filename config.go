package holmes

import (
	"time"
)

// Config for holmes
type Config struct {
	InText        bool // default dump to binary profile, set to true if you want a text profile
	DumpFullStack bool // only dump top 10, if set to false, dump all, only effective when in_text = true

	// enable the goroutine dumper, should dump if one of the following requirements is matched
	//   1. goroutine_num > GoroutineTriggerNumMin && goroutine_diff percent > GoroutineTriggerPercentDiff
	//   2. goroutine_num > GoroutineTriggerNumAbsNum
	EnableGoroutineDump         bool
	GoroutineTriggerNumMin      int // goroutine trigger min in number
	GoroutineTriggerPercentDiff int // goroutine trigger diff in percent
	GoroutineTriggerNumAbs      int // goroutine trigger abs in number

	// enable the cpu dumper, should dump if one of the following requirements is matched
	//   1. cpu usage > CPUTriggerMin && cpu usage diff > CPUTriggerDiff
	//   2. cpu usage > CPUTriggerAbs
	EnableCPUDump         bool
	CPUTriggerPercentMin  int // cpu trigger min in percent
	CPUTriggerPercentDiff int // cpu trigger diff in percent
	CPUTriggerPercentAbs  int // cpu trigger abs inpercent

	// enable the heap dumper, should dump if one of the following requirements is matched
	//   1. memory usage > MemTriggerPercentMin && cpu usage diff > MemTriggerPercentDiff
	//   2. memory usage > MemTriggerPercentAbs
	EnableMemDump         bool
	MemTriggerPercentMin  int // mem trigger minimum in percent
	MemTriggerPercentDiff int // mem trigger diff in percent
	MemTriggerPercentAbs  int // mem trigger absolute in percent

	// interval for dump loop, default 5s
	CollectInterval time.Duration
	// the cooldown time after every type of dump
	// the cpu/mem/goroutine have different cooldowns of their own
	CoolDown time.Duration // cooldown 的时间间隔，单位为分钟
}
