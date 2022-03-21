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
	http.HandleFunc("/leak", leak)
	go http.ListenAndServe(":10003", nil)
}

func main() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("2s"),
		holmes.WithDumpPath("/tmp"),
		holmes.WithTextDump(),
		holmes.WithGoroutineDump(10, 25, 80, 10000, time.Minute),
	)
	h.EnableGoroutineDump().Start()
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
