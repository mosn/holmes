package main

import (
	"net/http"
	"time"

	"github.com/mosn/holmes"
)

func init() {
	http.HandleFunc("/leak", leak)
	go http.ListenAndServe(":10003", nil)
}

var h = holmes.New("2s", "1m", "/tmp", false).
	EnableGoroutineDump().Config(10, 25, 80)

func main() {
	h.Start()
	time.Sleep(time.Hour)
}

func leak(wr http.ResponseWriter, req *http.Request) {
	taskChan := make(chan int)
	consumer := func() {
		for task := range taskChan {
			_ = task // do some tasks
		}
	}

	producer := func() {
		for i := 0; i < 10; i++ {
			taskChan <- i // generate some tasks
		}
		// forget to close the taskChan here
	}

	go consumer()
	go producer()
}
