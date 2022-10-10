/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package holmes

type ring struct {
	data   []int
	idx    int
	sum    int
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

	// the first round
	if len(r.data) < r.maxLen {
		r.sum += i
		r.data = append(r.data, i)
		return
	}

	r.sum += i - r.data[r.idx]

	// the ring is expanded, just write to the position
	r.data[r.idx] = i
	r.idx = (r.idx + 1) % r.maxLen
}

func (r *ring) avg() int {
	// Check if the len(r.data) is zero before dividing
	if r.maxLen == 0 || len(r.data) == 0 {
		return 0
	}
	return r.sum / len(r.data)
}

func (r *ring) sequentialData() []int {
	index := r.idx
	slice := make([]int, r.maxLen)
	if index == 0 || len(r.data) < r.maxLen {
		copy(slice, r.data)
		return slice
	}
	copy(slice, r.data[index:])
	copy((slice)[r.maxLen-index:], r.data[:index])
	return slice
}
