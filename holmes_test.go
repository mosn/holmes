package holmes

import (
	"log"
	"os"
	"testing"
	"time"
)

var h *Holmes

func TestMain(m *testing.M) {
	log.Println("holmes initialing")
	h, _ = New(
		WithCollectInterval("1s"),
		WithCoolDown("1s"),
		WithDumpPath("./"),
		WithTextDump(),
		WithGoroutineDump(10, 25, 80, 90),
	)
	log.Println("holmes initial success")
	h.EnableGoroutineDump().Start()
	time.Sleep(10 * time.Second)
	log.Println("on running")
	m.Run()
}

// -gcflags=all=-l
func TestResetCollectInterval(t *testing.T) {
	before := h.collectCount
	go func() {
		h.Set(WithCollectInterval("2s"))
		defer h.Set(WithCollectInterval("1s"))
		time.Sleep(6 * time.Second)
		// if collect interval not change, collectCount would increase 5 at least
		if h.collectCount-before >= 5 {
			log.Fatalf("fail, before %v, now %v", before, h.collectCount)
		}
	}()
	time.Sleep(8 * time.Second)
}

func TestSetDumpPath(t *testing.T) {
	h.Set(WithDumpPath("./test_case_gen"))
	defer h.Set(WithDumpPath("./"))

	if h.opts.Logger.Load().(*os.File).Name()[:13] != "test_case_gen" {
		log.Fatalf("fail")
	}
}

func TestSetGrOpts(t *testing.T) {
	// decrease min trigger, if our set api is effective,
	// gr profile would be trigger and grCoolDown increase.
	min, diff, abs := 3, 10, 1
	before := h.grCoolDownTime

	err := h.Set(
		WithGoroutineDump(min, diff, abs, 90))
	if err != nil {
		log.Fatalf("fail to set opts on running time.")
	}

	time.Sleep(5 * time.Second)
	if before.Equal(h.grCoolDownTime) {
		log.Fatalf("fail")
	}
}
