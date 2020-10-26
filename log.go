package holmes

import "fmt"

// log write content to log file
func (h *Holmes) logf(pattern string, args ...interface{}) {
	content := fmt.Sprintf(pattern+"\n", args...)
	_, err := h.textFile.WriteString(content)
	if err != nil {
		fmt.Println(err) // where to write this log?
	}
}

// log write content to log file
func (h *Holmes) debugf(pattern string, args ...interface{}) {
	content := fmt.Sprintf(pattern+"\n", args...)
	_, err := h.textFile.WriteString(content)
	if err != nil {
		fmt.Println(err) // where to write this log?
	}
}
