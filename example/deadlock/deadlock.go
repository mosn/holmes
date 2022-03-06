package main

import (
	"net/http"
	"sync"
	"time"

	"mosn.io/holmes"
)

func init() {
	http.HandleFunc("/lockorder1", lockorder1)
	http.HandleFunc("/lockorder2", lockorder2)
	http.HandleFunc("/req", req)
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

var l1 sync.Mutex
var l2 sync.Mutex

func req(wr http.ResponseWriter, req *http.Request) {
	l1.Lock()
	defer l1.Unlock()
}

func lockorder1(wr http.ResponseWriter, req *http.Request) {
	l1.Lock()
	defer l1.Unlock()

	time.Sleep(time.Minute)

	l2.Lock()
	defer l2.Unlock()
}

func lockorder2(wr http.ResponseWriter, req *http.Request) {
	l2.Lock()
	defer l2.Unlock()

	time.Sleep(time.Minute)

	l1.Lock()
	defer l1.Unlock()
}
