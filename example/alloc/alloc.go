package main

import (
	"fmt"
	mlog "mosn.io/pkg/log"
	"net/http"
	"time"

	"mosn.io/holmes"
)

func init() {
	http.HandleFunc("/alloc", alloc)
	go http.ListenAndServe(":10003", nil)
}

func main() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("2s"),
		holmes.WithDumpPath("/tmp"),
		holmes.WithLogger(holmes.NewFileLog("/tmp/holmes.log", mlog.INFO)),
		holmes.WithTextDump(),
		holmes.WithMemDump(3, 25, 80, time.Minute),
	)
	h.EnableMemDump().Start()
	time.Sleep(time.Hour)
}

func alloc(wr http.ResponseWriter, req *http.Request) {
	var m = make(map[string]string, 102400)
	for i := 0; i < 1000; i++ {
		m[fmt.Sprint(i)] = fmt.Sprint(i)
	}
	_ = m
}
