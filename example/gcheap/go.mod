module example.com/m

go 1.17

require (
	mosn.io/holmes v0.0.0-20220125114618-8cb365eb42ac
	mosn.io/pkg v1.6.0
)

require (
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/hashicorp/go-syslog v1.0.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	golang.org/x/sys v0.19.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	mosn.io/api v1.5.0 // indirect
)

replace mosn.io/holmes => ../../
