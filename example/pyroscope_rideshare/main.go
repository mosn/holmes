package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"mosn.io/holmes"
	"mosn.io/holmes/reporters/pyroscope_reporter"
	"rideshare/bike"
	"rideshare/car"
	"rideshare/scooter"
)

func bikeRoute(w http.ResponseWriter, r *http.Request) {
	bike.OrderBike(1)
	w.Write([]byte("<h1>Bike ordered</h1>"))
}

func scooterRoute(w http.ResponseWriter, r *http.Request) {
	scooter.OrderScooter(2)
	w.Write([]byte("<h1>Scooter ordered</h1>"))
}

func carRoute(w http.ResponseWriter, r *http.Request) {
	car.OrderCar(3)

	w.Write([]byte("<h1>Car ordered</h1>"))
}

func index(w http.ResponseWriter, r *http.Request) {
	result := "<h1>environment vars:</h1>"
	for _, env := range os.Environ() {
		result += env + "<br>"
	}
	w.Write([]byte(result))
}

var h *holmes.Holmes

func InitHolmes() {
	fmt.Println("holmes initialing")
	h, _ = holmes.New(
		holmes.WithCollectInterval("1s"),
		holmes.WithDumpPath("./log/"),
		// can not set text in pyroscope client
	)
	fmt.Println("holmes initial success")
	h.
		EnableCPUDump().
		EnableGoroutineDump().
		EnableMemDump().
		Start()
	time.Sleep(11 * time.Second)
	fmt.Println("on running")
}

func main() {
	InitHolmes()
	region := os.Getenv("region")
	port := os.Getenv("port")
	fmt.Printf("region is %v port is %v \n", region, port)
	cfg := pyroscope_reporter.RemoteConfig{
		//AuthToken:              "",
		//UpstreamThreads:        4,
		UpstreamAddress:        "http://localhost:4040",
		UpstreamRequestTimeout: 3 * time.Second,
	}

	tags := map[string]string{
		"region": region,
	}

	pReporter, err := pyroscope_reporter.NewPyroscopeReporter("holmes-client", tags, cfg, holmes.NewStdLogger())
	if err != nil {
		fmt.Printf("NewPyroscopeReporter error %v\n", err)
		return
	}

	err = h.Set(
		holmes.WithProfileReporter(pReporter),
		holmes.WithGoroutineDump(2, 2, 20, 90, 20*time.Second),
		holmes.WithCPUDump(2, 2, 80, 20*time.Second),
		holmes.WithMemDump(1, 2, 80, 20*time.Second),
		holmes.WithCollectInterval("5s"),
	)
	if err != nil {
		fmt.Printf("fail to set opts on running time.\n")
		return
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/bike", bikeRoute)
	http.HandleFunc("/scooter", scooterRoute)
	http.HandleFunc("/car", carRoute)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}

	time.Sleep(1 * time.Minute)

}
