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

func main() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("2s"),
		holmes.WithCoolDown("1m"),
		holmes.WithDumpPath("./tmp"),
		holmes.WithTextDump(),
		holmes.WithMemDump(3, 25, 80),
	)
	h.EnableMemDump().Start()
	time.Sleep(time.Hour)
}

func make1gbslice(wr http.ResponseWriter, req *http.Request) {
	var a = make([]byte, 1073741824)
	_ = a

}
