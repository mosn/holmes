package main

import (
	"net/http"
	"time"

	"github.com/mosn/holmes"
)

func init() {
	http.HandleFunc("/make1gb", make1gbslice)
	go http.ListenAndServe(":10003", nil)
}

var h = holmes.New("2s", "1m", "/tmp", false).
	EnableMemDump().Config(3, 25, 80)

func main() {
	h.Start()
	time.Sleep(time.Hour)
}

func make1gbslice(wr http.ResponseWriter, req *http.Request) {
	var a = make([]byte, 1073741824)
	_ = a
}
