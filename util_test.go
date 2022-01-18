package holmes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTryLock_TryLock(t *testing.T) {
	tl := NewTryLock()
	tl.Lock()
	assert.Equal(t, tl.TryLock(), false)
	tl.Unlock()

	assert.Equal(t, tl.TryLock(), true)
	tl.Unlock()

	tl.TryLock()
	assert.Equal(t, tl.TryLock(), false)
	tl.Unlock()

}
