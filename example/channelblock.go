package main

import (
	"net/http"
	"time"

	"github.com/mosn/holmes"
)

func init() {
	http.HandleFunc("/chanblock", channelBlock)
	go http.ListenAndServe(":10003", nil)
}

var h = holmes.New("5s", "1m", "/tmp", false).
	EnableGoroutineDump().Config(10, 25, 20000)

func main() {
	h.Start()
	time.Sleep(time.Hour)
}

var nilCh chan int

func channelBlock(wr http.ResponseWriter, req *http.Request) {
	nilCh <- 1
}
