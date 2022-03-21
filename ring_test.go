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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyRing(t *testing.T) {
	var r = newRing(0)
	assert.Equal(t, r.avg(), 0)

	r = newRing(1)
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
