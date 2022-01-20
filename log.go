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
	if _, err := h.opts.Logger.WriteString(content); err != nil {
		fmt.Println(err) // where to write this log?
	}

	if !h.opts.logOpts.Enable {
		return
	}

	state, err := h.opts.Logger.Stat()
	if err != nil {
		h.opts.logOpts.Enable = false

		return
	}

	if state.Size() > h.opts.logOpts.SplitLoggerSize && atomic.CompareAndSwapInt32(&h.changelog, 0, 1) {
		defer atomic.StoreInt32(&h.changelog, 0)

		var (
			newLogger *os.File
			err       error
			dumpPath  = h.opts.DumpPath
			suffix    = time.Now().Format("20060102150405")
			srcPath   = filepath.Clean(filepath.Join(dumpPath, defaultLoggerName))
			dstPath   = srcPath + "_" + suffix + ".back"
		)

		err = os.Rename(srcPath, dstPath)

		if err != nil {
			h.opts.logOpts.Enable = false

			return
		}

		newLogger, err = os.OpenFile(filepath.Clean(srcPath), defaultLoggerFlags, defaultLoggerPerm)

		if err != nil {
			h.opts.logOpts.Enable = false

			return
		}

		h.opts.Logger, newLogger = newLogger, h.opts.Logger
		_ = newLogger.Close()
	}
}
