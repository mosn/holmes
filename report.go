package holmes

import "time"

type ProfileReporter interface {
	Report(pType string, filename string, reason ReasonType, eventID string, sampleTime time.Time, pprofBytes []byte, scene Scene) error
}

// rptEvent stands of the args of report event
type rptEvent struct {
	PType      string
	FileName   string
	Reason     ReasonType
	EventID    string
	SampleTime time.Time
	PprofBytes []byte
	Scene      Scene
}

// Scene contains the scene information when profile triggers,
// including current value, average value and configurations.
type Scene struct {
	typeOption

	// current value while dump event occurs
	CurVal int
	// Avg is the average of the past values
	Avg int
}

type ReasonType uint8

const (
	// ReasonCurlLessMin means current value is less than min value.
	ReasonCurlLessMin ReasonType = iota
	// ReasonCurlGreaterMin means current value is greater than min value,
	// but don't meet any trigger conditions.
	ReasonCurlGreaterMin
	// ReasonCurGreaterMax means current value is greater than max value.
	ReasonCurGreaterMax
	// ReasonCurGreaterAbs means current value meets the trigger condition where
	// it is greater than abs value.
	ReasonCurGreaterAbs
	// ReasonDiff means current value is greater than the value: (diff + 1) * agv.
	ReasonDiff
)

func (rt ReasonType) String() string {
	var reason string
	switch rt {
	case ReasonCurlLessMin:
		reason = "curVal < ruleMin"
	case ReasonCurlGreaterMin:
		reason = "curVal >= ruleMin, but don't meet diff trigger condition"
	case ReasonCurGreaterMax:
		reason = "curVal >= ruleMax"
	case ReasonCurGreaterAbs:
		reason = "curVal > ruleAbs"
	case ReasonDiff:
		reason = "curVal >= ruleMin, and meet diff trigger condition"

	}

	return reason
}
