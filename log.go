package holmes

import (
	mlog "mosn.io/pkg/log"
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
