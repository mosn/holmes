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
	"log"
	"runtime"
	"testing"
	"time"
)

var h *Holmes

func TestMain(m *testing.M) {
	log.Println("holmes initialing")
	h, _ = New(
		WithCollectInterval("1s"),
		WithTextDump(),
		WithGoroutineDump(10, 25, 80, 90, time.Minute),
	)
	log.Println("holmes initial success")
	h.EnableGoroutineDump().Start()
	time.Sleep(10 * time.Second)
	log.Println("on running")
	m.Run()
}

// -gcflags=all=-l
func TestResetCollectInterval(t *testing.T) {
	before := h.collectCount
	go func() {
		h.Set(WithCollectInterval("2s"))
		defer h.Set(WithCollectInterval("1s"))
		time.Sleep(6 * time.Second)
		// if collect interval not change, collectCount would increase 5 at least
		if h.collectCount-before >= 5 {
			log.Fatalf("fail, before %v, now %v", before, h.collectCount)
		}
	}()
	time.Sleep(8 * time.Second)
}

func TestSetGrOpts(t *testing.T) {
	// decrease min trigger, if our set api is effective,
	// gr profile would be trigger and grCoolDown increase.
	min, diff, abs := 3, 10, 1
	before := h.grCoolDownTime

	err := h.Set(
		WithGoroutineDump(min, diff, abs, 90, time.Minute))
	if err != nil {
		log.Fatalf("fail to set opts on running time.")
	}

	time.Sleep(5 * time.Second)
	if before.Equal(h.grCoolDownTime) {
		log.Fatalf("fail")
	}
}

func TestCpuCore(t *testing.T) {
	h.Set(
		WithCGroup(false),
		WithGoProcAsCPUCore(false),
	)
	cpuCore1, _ := h.getCPUCore()
	goProc1 := runtime.GOMAXPROCS(-1)

	// system cpu core matches go procs
	if cpuCore1 != float64(goProc1) {
		log.Fatalf("cpuCore1 %v not equal goProc1 %v", cpuCore1, goProc1)
	}

	// go proc = system cpu core + 1
	runtime.GOMAXPROCS(goProc1 + 1)

	cpuCore2, _ := h.getCPUCore()
	goProc2 := runtime.GOMAXPROCS(-1)
	if cpuCore2 != float64(goProc2)-1 {
		log.Fatalf("cpuCore2 %v not equal goProc2-1 %v", cpuCore2, goProc2)
	}

	// set cpu core directly
	h.Set(
		WithCPUCore(cpuCore1 + 5),
	)

	cpuCore3, _ := h.getCPUCore()
	if cpuCore3 != cpuCore1+5 {
		log.Fatalf("cpuCore3 %v not equal cpuCore1+5 %v", cpuCore3, cpuCore1+5)
	}
}

func createThread(n int, blockTime time.Duration) {
	for i := 0; i < n; i++ {
		go func() {
			runtime.LockOSThread()
			time.Sleep(blockTime)

			runtime.UnlockOSThread()
		}()
	}
}

func TestWithShrinkThread(t *testing.T) {
	before := h.shrinkThreadTriggerCount

	err := h.Set(
		// delay 5 seconds, after the 50 threads unlocked
		WithThreadDump(10, 10, 10, time.Minute),
		WithShrinkThread(20, time.Second*5),
		WithCollectInterval("1s"),
	)
	h.EnableShrinkThread()
	if err != nil {
		log.Fatalf("fail to set opts on running time.")
	}

	threadNum1 := getThreadNum()
	// 50 threads exists 3 seconds
	createThread(50, time.Second*3)

	time.Sleep(time.Second)
	threadNum2 := getThreadNum()
	if threadNum2-threadNum1 < 40 {
		log.Fatalf("create thread failed, before: %v, now: %v", threadNum1, threadNum2)
	}
	log.Printf("created 50 threads, before: %v, now: %v", threadNum1, threadNum2)

	time.Sleep(10 * time.Second)

	if before+1 != h.shrinkThreadTriggerCount {
		log.Fatalf("shrink thread not triggered, before: %v, now: %v", before, h.shrinkThreadTriggerCount)
	}

	threadNum3 := getThreadNum()
	if threadNum2-threadNum3 < 30 {
		log.Fatalf("shrink thread failed, before: %v, now: %v", threadNum2, threadNum3)
	}

	h.DisableShrinkThread()
}
