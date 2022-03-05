package reporters

import (
	"log"
	"testing"
	"time"

	"github.com/mosn/holmes"
)

var h *holmes.Holmes

func TestMain(m *testing.M) {
	log.Println("holmes initialing")
	h, _ = holmes.New(
		holmes.WithCollectInterval("1s"),
		holmes.WithCoolDown("1s"),
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

func TestReporter(t *testing.T) {
	grReportCount = 0
	cpuReportCount = 0
	r := &mockReporter{}
	err := h.Set(
		holmes.WithProfileReporter(r),
		holmes.WithGoroutineDump(5, 10, 20, 90),
		holmes.WithCPUDump(0, 2, 80),
		holmes.WithCollectInterval("5s"),
	)
	if err != nil {
		log.Fatalf("fail to set opts on running time.")
	}
	go cpuex()
	time.Sleep(100 * time.Second)

	if grReportCount == 0 {
		log.Fatalf("not grReport")
	}

	if cpuReportCount == 0 {
		log.Fatalf("not cpuReport")
	}

}

func cpuex() {
	go func() {
		for {
			time.Sleep(time.Millisecond)
		}
	}()
}
