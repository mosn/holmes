package main

import (
	"net/http"
	"time"

	"github.com/mosn/holmes"
)

func init() {
	http.HandleFunc("/cpuex", cpuex)
	go http.ListenAndServe(":10003", nil)
}

var h = holmes.New("2s", "1m", "/tmp", false).
	EnableCPUDump().Config(20, 25, 80)

func main() {
	h.Start()
	time.Sleep(time.Hour)
}

func cpuex(wr http.ResponseWriter, req *http.Request) {
	go func() {
		for {
		}
	}()
}
