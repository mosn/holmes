package holmes

type ring struct {
	data   []int
	idx    int
	maxLen int
}

func newRing(maxLen int) ring {
	return ring{
		data:   make([]int, 0, maxLen),
		idx:    0,
		maxLen: maxLen,
	}
}

func (r *ring) push(i int) {
	if r.maxLen == 0 {
		return
	}

	// no position to write
	// jump to head
	if r.idx >= cap(r.data) {
		r.idx = 0
	}

	// the first round
	if len(r.data) < cap(r.data) {
		r.data = append(r.data, i)
		return
	}

	// the ring is expanded, just write to the position
	r.data[r.idx] = i
	r.idx++
}

func (r *ring) avg() int {
	if r.maxLen == 0 {
		return 0
	}

	sum := 0
	for i := 0; i < len(r.data); i++ {
		sum += r.data[i]
	}

	return sum / len(r.data)
}
