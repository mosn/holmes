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

import "testing"

// TestDisableMemDumpOpts is a regression test for the copy-paste bug where
// DisableMemDump flipped gCHeapOpts.Enable instead of memOpts.Enable, so
// switch=false could never actually turn the mem dump off.
func TestDisableMemDumpOpts(t *testing.T) {
	hg, err := New()
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// mem dump is off by default (newMemOptions leaves Enable at its zero value).
	if hg.opts.GetMemOpts().Enable {
		t.Fatalf("memOpts.Enable should be false by default")
	}

	hg.EnableMemDump()
	if !hg.opts.GetMemOpts().Enable {
		t.Fatalf("EnableMemDump should set memOpts.Enable=true")
	}

	// DisableMemDump must turn memOpts.Enable back off; with the bug it only
	// toggled gCHeapOpts.Enable and left mem dump running.
	hg.DisableMemDump()
	if hg.opts.GetMemOpts().Enable {
		t.Fatalf("DisableMemDump should set memOpts.Enable=false (regression of the copy-paste bug)")
	}
}
