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

package http_reporter

import (
	"log"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHttpReporter_Report(t *testing.T) {
	newMockServer()

	reporter := NewReporter("test", "http://127.0.0.1:8080/profile/upload")

	buf := []byte("test-data")
	if err := reporter.Report("goroutine", buf, "test", "test-id"); err != nil {
		log.Fatalf("failed to report: %v", err)
	}
}

func newMockServer() {
	r := gin.New()
	r.POST("/profile/upload", ProfileUploadHandler)
	go r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")

	time.Sleep(time.Millisecond * 100)
}

func ProfileUploadHandler(c *gin.Context) {
	ret := map[string]interface{}{}
	ret["code"] = 1
	ret["message"] = "success"
	c.JSON(200, ret)
	return
}
