/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

	"mosn.io/holmes"
)

func init() {
	go func() {
		h, _ := holmes.New(
			holmes.WithCollectInterval("2s"),
			holmes.WithDumpPath("/tmp"),
			holmes.WithTextDump(),
			holmes.WithThreadDump(10, 25, 100, time.Minute),
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
