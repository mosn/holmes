package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/mosn/holmes"
)

func init() {
	http.HandleFunc("/lockorder1", lockorder1)
	http.HandleFunc("/lockorder2", lockorder2)
	http.HandleFunc("/req", req)
	go http.ListenAndServe(":10003", nil)
}

var h = holmes.New("5s", "1m", "/tmp", false).
	EnableGoroutineDump().Config(10, 25, 20000)

func main() {
	h.Start()
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
