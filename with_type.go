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
		h.conf.MemTriggerPercentAbs = abs
		h.conf.MemTriggerPercentDiff = diff
		h.conf.MemTriggerPercentMin = min
	case cpu:
		h.conf.CPUTriggerPercentAbs = abs
		h.conf.CPUTriggerPercentDiff = diff
		h.conf.CPUTriggerPercentMin = min
	case goroutine:
		h.conf.GoroutineTriggerNumAbs = abs
		h.conf.GoroutineTriggerPercentDiff = diff
		h.conf.GoroutineTriggerNumMin = min
	}

	return wt.h
}
