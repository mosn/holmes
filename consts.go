package holmes

const (
	defaultCPUTriggerMin  = 10 // 10%%
	defaultCPUTriggerAbs  = 70 // 70%%
	defaultCPUTriggerDiff = 25 // 25%

	defaultGoroutineTriggerMin  = 3000   // 3000 goroutines
	defaultGoroutineTriggerAbs  = 200000 // 200k goroutines
	defaultGoroutineTriggerDiff = 20     // 20%  diff

	defaultMemTriggerMin  = 10 // 10%
	defaultMemTriggerAbs  = 80 // 80%
	defaultMemTriggerDiff = 25 // 25%
)

type configureType int

const (
	mem configureType = iota
	cpu
	goroutine
)

var type2name = map[configureType]string{
	mem:       "mem",
	cpu:       "cpu",
	goroutine: "goroutine",
}

const cgroupMemLimitPath = "/sys/fs/cgroup/memory/memory.limit_in_bytes"

const minCollectCyclesBeforeDumpStart = 10
