package main

import (
	"net/http"
	"time"

	"github.com/mosn/holmes"
)

func init() {
	http.HandleFunc("/docker", dockermake1gb)
	http.HandleFunc("/docker/cpu", cpuex)
	http.HandleFunc("/docker/cpu_multi_core", cpuMulticore)
	go http.ListenAndServe(":10003", nil)
}

func main() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("2s"),
		holmes.WithCoolDown("1m"),
		holmes.WithLogger(holmes.NewFileLog("./tmp", false, "")),
		holmes.WithTextDump(),
		holmes.WithLoggerLevel(holmes.LogLevelDebug),
		holmes.WithMemDump(3, 25, 80),
		holmes.WithCPUDump(60, 10, 80),
		holmes.WithCGroup(true),
	)
	h.EnableCPUDump()
	h.EnableMemDump()
	h.Start()
	time.Sleep(time.Hour)
}

func cpuex(wr http.ResponseWriter, req *http.Request) {
	go func() {
		for {
		}
	}()
}

func cpuMulticore(wr http.ResponseWriter, req *http.Request) {
	for i := 1; i <= 100; i++ {
		go func() {
			for {
			}
		}()
	}
}

func dockermake1gb(wr http.ResponseWriter, req *http.Request) {
	var a = make([]byte, 1073741824)
	_ = a
}
