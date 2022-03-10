package main

import (
	"net/http"
	"time"

	"mosn.io/holmes"
)

func init() {
	http.HandleFunc("/chanblock", channelBlock)
	go http.ListenAndServe(":10003", nil)
}

func main() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("5s"),
		holmes.WithCoolDown("1m"),
		holmes.WithDumpPath("/tmp"),
		holmes.WithLogger(holmes.NewFileLog("/tmp", false, "")),
		holmes.WithTextDump(),
		holmes.WithGoroutineDump(10, 25, 2000, 10000),
	)
	h.EnableGoroutineDump().Start()
	time.Sleep(time.Hour)
}

var nilCh chan int

func channelBlock(wr http.ResponseWriter, req *http.Request) {
	nilCh <- 1
}
