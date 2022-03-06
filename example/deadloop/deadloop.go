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

func main() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("2s"),
		holmes.WithCoolDown("1m"),
		holmes.WithLogger(holmes.NewFileLog("./tmp", false, "")),
		holmes.WithCPUDump(10, 25, 80),
	)
	h.EnableCPUDump().Start()
	time.Sleep(time.Hour)
}

func deadloop(wr http.ResponseWriter, req *http.Request) {
	for i := 0; i < 4; i++ {
		for {
			time.Sleep(time.Millisecond)
		}
	}
}
