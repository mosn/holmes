package holmes

import "fmt"

// log write content to log file
func (h *Holmes) logf(pattern string, args ...interface{}) {
	h.writeString(fmt.Sprintf(pattern+"\n", args...))
}

// log write content to log file
func (h *Holmes) debugf(pattern string, args ...interface{}) {
	h.writeString(fmt.Sprintf(pattern+"\n", args...))
}

func (h *Holmes) writeString(content string) {
	if _, err := h.opts.Logger.WriteString(content); err != nil {
		fmt.Println(err) // where to write this log?
	}
}
