package pyroscope_reporter

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"mosn.io/holmes"
)

var h *holmes.Holmes

func TestMain(m *testing.M) {
	log.Println("holmes initialing")
	h, _ = holmes.New(
		holmes.WithCollectInterval("1s"),
		holmes.WithDumpPath("./"),
		holmes.WithTextDump(),
	)
	log.Println("holmes initial success")
	h.EnableCPUDump().Start()
	time.Sleep(11 * time.Second)
	log.Println("on running")
	newMockServer()
	os.Exit(m.Run())
}

var received = false

func TestPyroscopeClient(t *testing.T) {

	cfg := RemoteConfig{
		//AuthToken:              "",
		//UpstreamThreads:        4,
		UpstreamAddress:        "http://localhost:8080",
		UpstreamRequestTimeout: 3 * time.Second,
	}
	tags := map[string]string{
		"region": "zh",
	}
	pReporter, err := NewPyroscopeReporter("holmes-client-01", tags, cfg, holmes.NewStdLogger())
	if err != nil {
		log.Fatalf("NewPyroscopeReporter error %v", err)
	}

	err = h.Set(
		holmes.WithProfileReporter(pReporter),
		holmes.WithGoroutineDump(5, 10, 20, 90, time.Second),
		holmes.WithCPUDump(0, 2, 80, time.Second),
		holmes.WithCollectInterval("5s"),
	)
	if err != nil {
		log.Fatalf("fail to set opts on running time.")
	}
	go cpuex()
	time.Sleep(20 * time.Second)
	if !received {
		t.Errorf("mock pyroscope server didn't received request")
	}
}

func cpuex() {
	go func() {
		for {
			time.Sleep(time.Millisecond)
		}
	}()
}

func newMockServer() {
	r := gin.New()
	r.POST("/ingest", ProfileUploadHandler)
	go r.Run() //nolint:errcheck // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")

	time.Sleep(time.Millisecond * 100)
}

func ProfileUploadHandler(c *gin.Context) {
	ret := map[string]interface{}{}
	ret["code"] = 1
	ret["message"] = "success"
	c.JSON(200, ret)
	received = true
}
