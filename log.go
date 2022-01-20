package holmes

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
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
	if h.opts.logOpts.Enable {
		state, _ := h.opts.Logger.Stat()
		if state.Size() > h.opts.logOpts.SplitLoggerSize && atomic.CompareAndSwapInt32(&h.changelog, 0, 1) {
			defer atomic.StoreInt32(&h.changelog, 0)
			suffix := fmt.Sprintf(time.Now().Format("20060102150405"))
			srcPath := filepath.Join(h.opts.DumpPath, defaultLoggerName)
			dstPath := filepath.Join(h.opts.DumpPath, defaultLoggerName+"_"+suffix+".back")
			if err := os.Rename(srcPath, dstPath); err != nil {
				h.opts.logOpts.Enable = false
				return
			}
			if newLogger, err := os.OpenFile(srcPath, defaultLoggerFlags, defaultLoggerPerm); err != nil {
				h.opts.logOpts.Enable = false
				return
			} else {
				h.opts.Logger, newLogger = newLogger, h.opts.Logger
				_ = newLogger.Close()
			}
		}
	}
	if _, err := h.opts.Logger.WriteString(content); err != nil {
		fmt.Println(err) // where to write this log?
	}
}
