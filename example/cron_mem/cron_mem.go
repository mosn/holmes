package main

import (
	"net/http"
	"time"

	"github.com/mosn/holmes"
	"github.com/robfig/cron/v3"
)

// run with -gcflags '-N -l'
func init() {
	http.HandleFunc("/make1gb", make1gbslice)
	go http.ListenAndServe(":10003", nil)
}

func main() {
	everyMinute := "0/1 * * * ?"

	h, _ := holmes.New(
		holmes.WithCollectInterval("2s"),
		holmes.WithCoolDown("1m"),
		holmes.WithDumpPath("/tmp"),
		holmes.WithTextDump(),
		holmes.WithMemDump(3, 25, 80),
		holmes.WithMemCron(everyMinute), // every minute
	)
	// simulate cron and self-warning run at the same time
	go func() {
		c := cron.New()
		_, _ = c.AddFunc(everyMinute, func() {
			var a = make([]byte, 1073741824)
			_ = a
		})
		c.Start()
	}()

	h.EnableMemDump().Start()
	time.Sleep(time.Hour)

}
func make1gbslice(wr http.ResponseWriter, req *http.Request) {
	var a = make([]byte, 1073741824)
	_ = a
}

/*
holmes.log
[2022-01-16 18:03:00.603]cron mem profile is being dumped, so stop warning dump
[2022-01-16 18:03:01.613]mem cron dump profile
[2022-01-16 18:03:01.613][Holmes] pprof mem, config_min : 3, config_diff : 25, config_abs : 80, previous : [0 0 0 0 0 6 0 0 0 0], current: 6

*/
