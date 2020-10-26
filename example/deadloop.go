package main

import (
	"net/http"
	"time"

	"github.com/mosn/holmes"
)

func init() {
	http.HandleFunc("/deadloop", deadloop)
	go http.ListenAndServe(":10003", nil)
}

var h = holmes.New("2s", "1m", "/tmp", false).
	EnableCPUDump().Config(10, 25, 80)

func main() {
	h.Start()
	time.Sleep(time.Hour)
}

func deadloop(wr http.ResponseWriter, req *http.Request) {
	for i := 0; i < 4; i++ {
		for {
		}
	}
}
