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

import (
	"net/http"
	"time"

	"mosn.io/holmes"
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
		holmes.WithDumpPath("/tmp"),
		holmes.WithTextDump(),
		holmes.WithMemDump(3, 25, 80, time.Minute),
		holmes.WithCPUDump(60, 10, 80, time.Minute),
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
