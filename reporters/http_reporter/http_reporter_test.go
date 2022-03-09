package http_reporter

import (
	"github.com/gin-gonic/gin"
	"log"
	"testing"
	"time"
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
	r := gin.Default()
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
