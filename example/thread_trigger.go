package main

/*
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
void output(char *str) {
    sleep(10000);
    printf("%s\n", str);
}
*/
import "C"
import (
	"fmt"
	"net/http"
	"time"
	"unsafe"

	_ "net/http/pprof"

	"github.com/mosn/holmes"
)

func init() {
	go func() {
		h, _ := holmes.New(
			holmes.WithCollectInterval("2s"),
			holmes.WithCoolDown("5s"),
			holmes.WithDumpPath("/tmp"),
			holmes.WithTextDump(),
			holmes.WithThreadDump(10, 25, 100),
		)
		h.EnableThreadDump().Start()
		time.Sleep(time.Hour)
	}()
}

func leak(wr http.ResponseWriter, req *http.Request) {
	go func() {
		for i := 0; i < 1000; i++ {
			go func() {
				str := "hello cgo"
				//change to char*
				cstr := C.CString(str)
				C.output(cstr)
				C.free(unsafe.Pointer(cstr))

			}()
		}
	}()
}

func main() {
	http.HandleFunc("/leak", leak)
	err := http.ListenAndServe(":10003", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	select {}
}
