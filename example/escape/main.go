package main

import (
	"time"

	"github.com/mosn/holmes"
)

//  go run -gcflags "-m -l" example/escape/main.go
func main() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("2s"),
		holmes.WithCoolDown("2s"),
		holmes.WithDumpPath("/tmp"),
		holmes.WithTextDump(),

		holmes.WithGoroutineDump(3, 25, 80, 90),
		holmes.WithMemDump(3, 25, 80),
		holmes.WithCPUDump(3, 25, 80),
		holmes.WithThreadDump(3, 25, 80),
	)

	h.EnableGoroutineDump().
		EnableMemDump().EnableThreadDump().EnableCPUDump().Start()
	time.Sleep(time.Hour)
}
