package holmes

// WithType with a configure type
type WithType struct {
	h   *Holmes
	typ configureType
}

// Config configs values for dumper
func (wt WithType) Config(min int, diff int, abs int) *Holmes {
	h := wt.h

	switch wt.typ {
	case mem:
		h.conf.MemTriggerAbs = abs
		h.conf.MemTriggerDiff = diff
		h.conf.MemTriggerMin = min
	case cpu:
		h.conf.CPUTriggerAbs = abs
		h.conf.CPUTriggerDiff = diff
		h.conf.CPUTriggerMin = min
	case goroutine:
		h.conf.GoroutineTriggerAbs = abs
		h.conf.GoroutineTriggerDiff = diff
		h.conf.GoroutineTriggerMin = min
	}

	return wt.h
}
