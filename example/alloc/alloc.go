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
	"fmt"
	"net/http"
	"time"

	mlog "mosn.io/pkg/log"

	"mosn.io/holmes"
)

// run `curl http://localhost:10003/alloc` after 15s(warn up)
func init() {
	http.HandleFunc("/alloc", alloc)
	go http.ListenAndServe(":10003", nil) //nolint:errcheck
}

func main() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("2s"),
		holmes.WithDumpPath("./tmp"),
		holmes.WithLogger(holmes.NewFileLog("./tmp/holmes.log", mlog.DEBUG)),
		holmes.WithTextDump(),
		holmes.WithMemDump(3, 25, 80, time.Minute),
	)
	h.EnableMemDump().Start()
	time.Sleep(time.Hour)
}

func alloc(wr http.ResponseWriter, req *http.Request) {
	var m = make(map[string]string, 1073741824)
	for i := 0; i < 1000; i++ {
		m[fmt.Sprint(i)] = fmt.Sprint(i)
	}
	_ = m
}
