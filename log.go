package holmes

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

type Logger interface {
	Print(context string)
}

type fileLog struct {
	changelog       int32        // changelog marker bit
	dumpPath        string       // dumpPath
	rotateEnable    bool         // rotateEnable Turn on syncopation
	splitLoggerSize int64        // splitLoggerSize The size of the log split
	logger          atomic.Value // Logger *os.File
}

func (f *fileLog) Print(context string) {
	logger, ok := f.logger.Load().(*os.File)
	if !ok || logger == nil {
		//nolint
		fmt.Println("write fail,logger is null or assert fail ", context) // where to write this log?
		return
	}

	if _, err := logger.WriteString(context); err != nil {
		//nolint
		fmt.Println(err) // where to write this log?
		return
	}

	if !f.rotateEnable {
		return
	}

	state, err := logger.Stat()
	if err != nil {
		f.rotateEnable = false
		//nolint
		fmt.Println("get logger stat:", err, "from now on, it will be disabled split log")

		return
	}

	if state.Size() > f.splitLoggerSize && atomic.CompareAndSwapInt32(&f.changelog, 0, 1) {
		defer atomic.StoreInt32(&f.changelog, 0)

		var (
			newLogger *os.File
			err       error
			dumpPath  = f.dumpPath
			suffix    = time.Now().Format("20060102150405")
			srcPath   = filepath.Clean(filepath.Join(dumpPath, defaultLoggerName))
			dstPath   = srcPath + "_" + suffix + ".back"
		)

		err = os.Rename(srcPath, dstPath)

		if err != nil {
			f.rotateEnable = false
			//nolint
			fmt.Println("rename err:", err, "from now on, it will be disabled split log")

			return
		}

		newLogger, err = os.OpenFile(filepath.Clean(srcPath), defaultLoggerFlags, defaultLoggerPerm)

		if err != nil {
			f.rotateEnable = false

			//nolint
			fmt.Println("open new file err:", err, "from now on, it will be disabled split log")

			return
		}

		old := logger

		f.logger.Store(newLogger)

		_ = old.Close()
	}
}

type stdLog struct {
	*os.File
}

func (s *stdLog) Print(context string) {
	if _, err := s.WriteString(context); err != nil {
		//nolint
		fmt.Println(err) // where to write this log?
		return
	}
}

// log write content to log file.
func (h *Holmes) logf(pattern string, args ...interface{}) {
	if h.opts.LogLevel.Allow(LogLevelInfo) {
		timestamp := "[" + time.Now().Format("2006-01-02 15:04:05.000") + "]"
		h.opts.Logger.Print(fmt.Sprintf(timestamp+pattern+"\n", args...))
	}
}

// log write content to log file.
func (h *Holmes) debugf(pattern string, args ...interface{}) {
	if h.opts.LogLevel.Allow(LogLevelDebug) {
		h.opts.Logger.Print(fmt.Sprintf(pattern+"\n", args...))
	}
}

// log write content to log file.
func (h *Holmes) warnf(pattern string, args ...interface{}) {
	if h.opts.LogLevel.Allow(LogLevelWarn) {
		h.opts.Logger.Print(fmt.Sprintf(pattern+"\n", args...))
	}
}

// log write content to log file.
func (h *Holmes) errorf(pattern string, args ...interface{}) {
	if h.opts.LogLevel.Allow(LogLevelError) {
		h.opts.Logger.Print(fmt.Sprintf(pattern+"\n", args...))
	}
}
