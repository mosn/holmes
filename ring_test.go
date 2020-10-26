package holmes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyRing(t *testing.T) {
	var r = newRing(0)
	assert.Equal(t, r.avg(), 0)
}

func TestRing(t *testing.T) {
	var cases = []struct {
		slice  []int
		maxLen int
		avg    int
	}{
		{
			slice:  []int{1, 2, 3},
			maxLen: 10,
			avg:    2,
		},
		{
			slice:  []int{1, 2, 3},
			maxLen: 1,
			avg:    3,
		},
	}

	for _, cas := range cases {
		var r = newRing(cas.maxLen)
		for _, elem := range cas.slice {
			r.push(elem)
		}
		assert.Equal(t, r.avg(), cas.avg)
	}
}
