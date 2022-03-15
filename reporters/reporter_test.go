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
		holmes.WithGoroutineDump(5, 10, 20, 90, time.Minute),
		holmes.WithCPUDump(0, 2, 80, time.Minute),
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
		holmes.WithGoroutineDump(5, 10, 20, 90, time.Minute),
		holmes.WithCPUDump(0, 2, 80, time.Minute),
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
