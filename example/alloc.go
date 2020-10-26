package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mosn/holmes"
)

func init() {
	http.HandleFunc("/alloc", alloc)
	go http.ListenAndServe(":10003", nil)
}

var h = holmes.New("2s", "1m", "/tmp", false).
	EnableMemDump().Config(3, 25, 80)

func main() {
	h.Start()
	time.Sleep(time.Hour)
}

func alloc(wr http.ResponseWriter, req *http.Request) {
	var m = make(map[string]string, 102400)
	for i := 0; i < 1000; i++ {
		m[fmt.Sprint(i)] = fmt.Sprint(i)
	}
	_ = m
}
