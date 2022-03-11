package holmes

import (
	"fmt"
	mlog "mosn.io/pkg/log"
	"os"
	"time"
)

func (h *Holmes) getLogger() mlog.ErrorLogger {
	h.opts.L.RLock()
	defer h.opts.L.RUnlock()
	return h.opts.logger
}

func (h *Holmes) Debugf(format string, args ...interface{}) {
	logger := h.getLogger()
	if logger == nil {
		return
	}
	logger.Debugf(format, args...)
}

func (h *Holmes) Infof(format string, args ...interface{}) {
	logger := h.getLogger()
	if logger == nil {
		return
	}
	logger.Infof(format, args...)
}

func (h *Holmes) Warnf(format string, args ...interface{}) {
	logger := h.getLogger()
	if logger == nil {
		return
	}
	logger.Warnf(format, args...)
}

func (h *Holmes) Errorf(format string, args ...interface{}) {
	logger := h.getLogger()
	if logger == nil {
		return
	}
	logger.Errorf(format, args...)
}

func (h *Holmes) Alertf(alert string, format string, args ...interface{}) {
	logger := h.getLogger()
	if logger == nil {
		return
	}
	logger.Alertf(alert, format, args...)
}

type stdLog struct {
	level mlog.Level
	file  *os.File
}

func formatter(lv string, alert string, format string) string {
	t := time.Now().Format("2006-01-02 15:04:05")
	if alert == "" {
		return t + " " + lv + " " + format
	}
	return t + " " + lv + " [" + alert + "] " + format
}

func (l *stdLog) Alertf(alert string, format string, args ...interface{}) {
	if l.level < mlog.ERROR {
		return
	}
	ft := formatter(mlog.ErrorPre, alert, format)
	fmt.Printf(ft, args...)
}

func (l *stdLog) Infof(format string, args ...interface{}) {
	if l.level < mlog.INFO {
		return
	}
	ft := formatter(mlog.InfoPre, "", format)
	fmt.Printf(ft, args...)
}

func (l *stdLog) Debugf(format string, args ...interface{}) {
	if l.level < mlog.DEBUG {
		return
	}
	ft := formatter(mlog.DebugPre, "", format)
	fmt.Printf(ft, args...)
}

func (l *stdLog) Warnf(format string, args ...interface{}) {
	if l.level < mlog.WARN {
		return
	}
	ft := formatter(mlog.WarnPre, "", format)
	fmt.Printf(ft, args...)
}

func (l *stdLog) Errorf(format string, args ...interface{}) {
	if l.level < mlog.ERROR {
		return
	}
	ft := formatter(mlog.ErrorPre, "", format)
	fmt.Printf(ft, args...)
}

func (l *stdLog) Tracef(format string, args ...interface{}) {
	if l.level < mlog.TRACE {
		return
	}
	ft := formatter(mlog.TracePre, "", format)
	fmt.Printf(ft, args...)
}

func (l *stdLog) Fatalf(format string, args ...interface{}) {
	if l.level < mlog.FATAL {
		return
	}
	ft := formatter(mlog.FatalPre, "", format)
	fmt.Printf(ft, args...)
}

func (l *stdLog) SetLogLevel(level mlog.Level) {
}

func (l *stdLog) GetLogLevel() mlog.Level {
	return l.level
}

func (l *stdLog) Toggle(disable bool) {
}

func (l *stdLog) Disable() bool {
	return true
}

func NewStdLogger() mlog.ErrorLogger {
	return &stdLog{
		file:  os.Stdout,
		level: mlog.INFO,
	}
}
