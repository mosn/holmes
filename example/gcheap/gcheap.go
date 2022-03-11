package main

import (
	"fmt"
	mlog "mosn.io/pkg/log"

	//"mosn.io/holmes/reporters/http_reporter"
	"math/rand"
	"net/http"
	"time"

	"mosn.io/holmes"
)

func init() {
	http.HandleFunc("/rand", randAlloc)
	http.HandleFunc("/spike", spikeAlloc)
	go http.ListenAndServe(":10024", nil)
}

func main() {
	// reporter := http_reporter.NewReporter("TOKEN", "URL")
	h, _ := holmes.New(
		holmes.WithCoolDown("10s"),
		holmes.WithDumpPath("/tmp"),
		holmes.WithLogger(holmes.NewFileLog("/tmp/holmes.log", mlog.INFO)),
		holmes.WithBinaryDump(),
		holmes.WithMemoryLimit(100*1024*1024), // 100MB
		holmes.WithGCHeapDump(10, 20, 40),
		// holmes.WithProfileReporter(reporter),
	)
	h.EnableGCHeapDump().Start()
	time.Sleep(time.Hour)
}

var (
	base = make([]byte, 1024*1024*10) // 10 MB long live memory.
)

func randAlloc(wr http.ResponseWriter, req *http.Request) {
	var s = make([][]byte, 0) // short live
	for i := 0; i < 1024; i++ {
		len := rand.Intn(1024)
		bytes := make([]byte, len)

		s = append(s, bytes)

		if len == 0 {
			s = make([][]byte, 0)
		}
	}
	time.Sleep(time.Millisecond * 10)
	fmt.Fprintf(wr, "slice current length: %v\n", len(s))
}

func spikeAlloc(wr http.ResponseWriter, req *http.Request) {
	var s = make([][]byte, 0, 1024) // spike, 10MB
	for i := 0; i < 10; i++ {
		bytes := make([]byte, 1024*1024)
		s = append(s, bytes)
	}
	// live for a while
	time.Sleep(time.Millisecond * 500)
	fmt.Fprintf(wr, "spike slice length: %v\n", len(s))
}
