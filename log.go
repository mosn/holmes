package holmes

import (
	"fmt"
	"time"
)

// log write content to log file.
func (h *Holmes) logf(pattern string, args ...interface{}) {
	if h.opts.LogLevel >= LogLevelInfo {
		timestamp := "[" + time.Now().Format("2006-01-02 15:04:05.000") + "]"
		h.writeString(fmt.Sprintf(timestamp+pattern+"\n", args...))
	}
}

// log write content to log file.
func (h *Holmes) debugf(pattern string, args ...interface{}) {
	if h.opts.LogLevel >= LogLevelDebug {
		h.writeString(fmt.Sprintf(pattern+"\n", args...))
	}
}

func (h *Holmes) writeString(content string) {
	if _, err := h.opts.Logger.WriteString(content); err != nil {
		fmt.Println(err) // where to write this log?
	}
}
