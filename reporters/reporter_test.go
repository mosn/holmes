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

package reporters

import (
	"log"
	"testing"
	"time"

	"mosn.io/holmes"
)

var h *holmes.Holmes

func TestMain(m *testing.M) {
	log.Println("holmes initialing")
	h, _ = holmes.New(
		holmes.WithCollectInterval("1s"),
		holmes.WithDumpPath("./"),
		holmes.WithTextDump(),
	)
	log.Println("holmes initial success")
	h.EnableGoroutineDump().EnableCPUDump().Start()
	time.Sleep(11 * time.Second)
	log.Println("on running")
	m.Run()
}

var grReportCount int
var cpuReportCount int

type mockReporter struct {
}

func (m *mockReporter) Report(pType string, buf []byte, reason string, eventID string) error {
	log.Printf("call %s report \n", pType)

	switch pType {
	case "goroutine":
		grReportCount++
	case "cpu":
		cpuReportCount++

	}
	return nil
}

var grReopenReportCount int

type mockReopenReporter struct {
}

func (m *mockReopenReporter) Report(pType string, buf []byte, reason string, eventID string) error {
	log.Printf("call %s report \n", pType)

	switch pType {
	case "goroutine":
		grReopenReportCount++
	}
	return nil
}

func TestReporter(t *testing.T) {
	grReportCount = 0
	cpuReportCount = 0
	r := &mockReporter{}
	err := h.Set(
		holmes.WithProfileReporter(r),
		holmes.WithGoroutineDump(5, 10, 20, 90, time.Second),
		holmes.WithCPUDump(0, 2, 80, time.Second),
		holmes.WithCollectInterval("5s"),
	)
	if err != nil {
		log.Fatalf("fail to set opts on running time.")
	}
	go cpuex()
	time.Sleep(10 * time.Second)

	if grReportCount == 0 {
		log.Fatalf("not grReport")
	}

	if cpuReportCount == 0 {
		log.Fatalf("not cpuReport")
	}

	// test reopen feature
	h.Stop()
	h.Start()
	grReopenReportCount = 0
	h.Set(
		holmes.WithProfileReporter(&mockReopenReporter{}))
	time.Sleep(10 * time.Second)

	time.Sleep(5 * time.Second)

	if grReopenReportCount == 0 {
		log.Fatalf("fail to reopen")
	}
}

func TestReporterReopen(t *testing.T) {
	grReportCount = 0
	cpuReportCount = 0
	r := &mockReporter{}
	err := h.Set(
		holmes.WithProfileReporter(r),
		holmes.WithGoroutineDump(5, 10, 20, 90, time.Second),
		holmes.WithCPUDump(0, 2, 80, time.Second),
		holmes.WithCollectInterval("5s"),
		holmes.WithDumpToLogger(true),
	)
	if err != nil {
		log.Fatalf("fail to set opts on running time.")
	}
	go cpuex()
	time.Sleep(10 * time.Second)

	if grReportCount == 0 {
		log.Fatalf("not grReport")
	}

	if cpuReportCount == 0 {
		log.Fatalf("not cpuReport")
	}

	// test reopen feature
	h.DisableProfileReporter()

	h.EnableProfileReporter()

	grReopenReportCount = 0
	h.Set(
		holmes.WithProfileReporter(&mockReopenReporter{}))
	time.Sleep(10 * time.Second)

	time.Sleep(5 * time.Second)

	if grReopenReportCount == 0 {
		log.Fatalf("fail to reopen")
	}
}

func cpuex() {
	go func() {
		for {
			time.Sleep(time.Millisecond)
		}
	}()
}
