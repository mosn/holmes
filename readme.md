# holmes

[中文版](./readme_cn.md)

**WARNING** : holmes is under heavy development now, so API will make breaking change during dev. If you want to use it in production, please wait for the first release.

Self-aware Golang profile dumper.

Our online system often crashes(mostly killed by OOM) in the midnight, as a lazy developer, we don't want to wake up at night and wait for the online bug to reproduce.

holmes comes to rescue.

## how to use

### dump goroutine when goroutine number spikes

```go
var h = holmes.New("5s", "1m", "/tmp", false).
     EnableGoroutineDump().Config(500, 25, 20000)
     
// start the metrics collect and dump loop
h.Start()

// stop the dumper
h.Stop()
```

* 5s means the system metrics are collected once 5 seconds
* 1m means once a dump happened, the next dump will not happen before cooldown finish-1 minute.
* /tmp means the dump binary file(binary mode) or the dump log file(text mode) will write content to /tmp dir
* false means not in binary mode, so it's text mode profiles
* Config(500, 25, 20000) means dump will happen when current_goroutine_num > 500 && current_goroutine_num > 125% * previous_average_goroutine_num or current_goroutine_num > 20000

### dump cpu profile when cpu load spikes

```go
var h = holmes.New("5s", "1m", "/tmp", false).
     EnableCPUDump().Config(10, 25, 80)

// start the metrics collect and dump loop
h.Start()

// stop the dumper
h.Stop()
```

* 5s means the system metrics are collected once 5 seconds
* 1m means once a dump happened, the next dump will not happen before cooldown finish-1 minute.
* /tmp means the dump binary file(binary mode) or the dump log file(text mode) will write content to /tmp dir
* true or false doesn't affect the CPU profile dump, because the pprof standard library doesn't support text mode dump
* Config(10, 25, 80) means dump will happen when cpu usage > 10% && cpu usage > 125% * previous cpu usage recorded or cpu usage > 80%

### dump heap profile when RSS spikes

```go
var h = holmes.New("5s", "1m", "/tmp", false).
	EnableMemDump().Config(30, 25, 80)

// start the metrics collect and dump loop
h.Start()

// stop the dumper
h.Stop()
```

* 5s means the system metrics are collected once 5 seconds
* 1m means once a dump happened, the next dump will not happen before cooldown finish-1 minute.
* /tmp means the dump binary file(binary mode) or the dump log file(text mode) will write content to /tmp dir
* true or false doesn't affect the CPU profile dump, because the pprof standard library doesn't support text mode dump
* Config(30, 25, 80) means dump will happen when memory usage > 10% && memory usage > 125% * previous memory usage or memory usage > 80%

### enable them all!

It's easy.

```go
var h = holmes.New("5s", "1m", "/tmp", false).
     EnableMemDump().Config(30, 25, 80).
     EnableCPUDump().Config(10, 25, 80).
     EnableGoroutineDump().Config(500, 25, 20000)
```

## known risks

Collect a goroutine itself [may cause latency spike](https://github.com/golang/go/issues/33250) because of the STW.

## design

Holmes collects the following stats every interval passed:

* Goroutine number by runtime.NumGoroutine.
* RSS used by the current process with [gopsutil](https://github.com/shirou/gopsutil)
* CPU percent a total. eg total 8 core, use 4 core = 50% with [gopsutil](https://github.com/shirou/gopsutil) 

After warming up phase finished, Holmes will compare the current stats with the average of previous collected stats(10 cycles). If the dump rule is matched, Holmes will dump the related profile to log(text mode) or binary file(binary mode).

When you get warning messages sent by your own monitor system, eg. memory usage exceed 80%, OOM killed, CPU usage exceed 80%, goroutine nun exceed 100k. The profile is already dumped to your dump path. You could just fetch the profile and see what actually happend without pressure.

## case show

### RSS peak caused by make a 1GB slice

see this [example](example/1gbslice.go)

after warming up, just curl http://localhost:10003/make1gb for some times, then you'll probably see:

```
heap profile: 0: 0 [1: 1073741824] @ heap/1048576
0: 0 [1: 1073741824] @ 0x42ba3ef 0x4252254 0x4254095 0x4254fd3 0x425128c 0x40650a1
#	0x42ba3ee	main.make1gbslice+0x3e			/Users/xargin/go/src/github.com/mosn/holmes/example/1gbslice.go:24
#	0x4252253	net/http.HandlerFunc.ServeHTTP+0x43	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2012
#	0x4254094	net/http.(*ServeMux).ServeHTTP+0x1a4	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2387
#	0x4254fd2	net/http.serverHandler.ServeHTTP+0xa2	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2807
#	0x425128b	net/http.(*conn).serve+0x86b		/Users/xargin/sdk/go1.14.2/src/net/http/server.go:1895
```

1: 1073741824 means 1 object and 1GB memory consumption.

### goroutine explosion caused by deadlock

See this [example](./example/deadlock.go)

curl localhost:10003/lockorder1

curl localhost:10003/lockorder2

After warming up, wrk -c 100 http://localhost:10003/req, then you'll see the deadlock caused goroutine num peak:

```
100 @ 0x40380b0 0x4048c80 0x4048c6b 0x40489e7 0x406f72c 0x42badfc 0x42badfd 0x4252b94 0x42549d5 0x4255913 0x4251bcc 0x40659e1
#	0x40489e6	sync.runtime_SemacquireMutex+0x46	/Users/xargin/sdk/go1.14.2/src/runtime/sema.go:71
#	0x406f72b	sync.(*Mutex).lockSlow+0xfb		/Users/xargin/sdk/go1.14.2/src/sync/mutex.go:138
#	0x42badfb	sync.(*Mutex).Lock+0x8b			/Users/xargin/sdk/go1.14.2/src/sync/mutex.go:81
#	0x42badfc	main.req+0x8c				/Users/xargin/go/src/github.com/mosn/holmes/example/deadlock.go:30
#	0x4252b93	net/http.HandlerFunc.ServeHTTP+0x43	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2012
#	0x42549d4	net/http.(*ServeMux).ServeHTTP+0x1a4	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2387
#	0x4255912	net/http.serverHandler.ServeHTTP+0xa2	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2807
#	0x4251bcb	net/http.(*conn).serve+0x86b		/Users/xargin/sdk/go1.14.2/src/net/http/server.go:1895
1 @ 0x40380b0 0x4048c80 0x4048c6b 0x40489e7 0x406f72c 0x42bb041 0x42bb042 0x4252b94 0x42549d5 0x4255913 0x4251bcc 0x40659e1

#	0x40489e6	sync.runtime_SemacquireMutex+0x46	/Users/xargin/sdk/go1.14.2/src/runtime/sema.go:71
#	0x406f72b	sync.(*Mutex).lockSlow+0xfb		/Users/xargin/sdk/go1.14.2/src/sync/mutex.go:138
#	0x42bb040	sync.(*Mutex).Lock+0xf0			/Users/xargin/sdk/go1.14.2/src/sync/mutex.go:81
#	0x42bb041	main.lockorder2+0xf1			/Users/xargin/go/src/github.com/mosn/holmes/example/deadlock.go:50
#	0x4252b93	net/http.HandlerFunc.ServeHTTP+0x43	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2012
#	0x42549d4	net/http.(*ServeMux).ServeHTTP+0x1a4	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2387
#	0x4255912	net/http.serverHandler.ServeHTTP+0xa2	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2807
#	0x4251bcb	net/http.(*conn).serve+0x86b		/Users/xargin/sdk/go1.14.2/src/net/http/server.go:1895

1 @ 0x40380b0 0x4048c80 0x4048c6b 0x40489e7 0x406f72c 0x42baf11 0x42baf12 0x4252b94 0x42549d5 0x4255913 0x4251bcc 0x40659e1
#	0x40489e6	sync.runtime_SemacquireMutex+0x46	/Users/xargin/sdk/go1.14.2/src/runtime/sema.go:71
#	0x406f72b	sync.(*Mutex).lockSlow+0xfb		/Users/xargin/sdk/go1.14.2/src/sync/mutex.go:138
#	0x42baf10	sync.(*Mutex).Lock+0xf0			/Users/xargin/sdk/go1.14.2/src/sync/mutex.go:81
#	0x42baf11	main.lockorder1+0xf1			/Users/xargin/go/src/github.com/mosn/holmes/example/deadlock.go:40
#	0x4252b93	net/http.HandlerFunc.ServeHTTP+0x43	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2012
#	0x42549d4	net/http.(*ServeMux).ServeHTTP+0x1a4	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2387
#	0x4255912	net/http.serverHandler.ServeHTTP+0xa2	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2807
#	0x4251bcb	net/http.(*conn).serve+0x86b		/Users/xargin/sdk/go1.14.2/src/net/http/server.go:1895
```

The req API was blocked by deadlock.

Your should set DumpFullStack to true to locate deadlock bug.

### goroutine explosion caused by channel block

see this [example](example/channelblock.go)

after warming up, just  wrk -c100 http://localhost:10003/chanblock

```
goroutine profile: total 203
100 @ 0x4037750 0x4007011 0x4006a15 0x42ba3c9 0x4252234 0x4254075 0x4254fb3 0x425126c 0x4065081
#	0x42ba3c8	main.channelBlock+0x38			/Users/xargin/go/src/github.com/mosn/holmes/example/channelblock.go:26
#	0x4252233	net/http.HandlerFunc.ServeHTTP+0x43	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2012
#	0x4254074	net/http.(*ServeMux).ServeHTTP+0x1a4	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2387
#	0x4254fb2	net/http.serverHandler.ServeHTTP+0xa2	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2807
#	0x425126b	net/http.(*conn).serve+0x86b		/Users/xargin/sdk/go1.14.2/src/net/http/server.go:1895
```

It's easy to locate.

### process slowly leaks goroutines

See this [example](example/slowlyleak.go)

The producer forget to close the task channel after produce finishes, so every request to this URI will leak a goroutine, we could curl http://localhost:10003/leak several time and got the following log:

```
goroutine profile: total 10
7 @ 0x4038380 0x4008497 0x400819b 0x42bb129 0x4065cb1
#	0x42bb128	main.leak.func1+0x48	/Users/xargin/go/src/github.com/mosn/holmes/example/slowlyleak.go:26
```

It's easy to find the leakage reason

### large memory allocation caused by business logic

See this [example](example/alloc.go), this is a similar example as the large slice make.

After warming up finished,  wrk -c100 http://localhost:10003/alloc:

```
pprof memory, config_min : 3, config_diff : 25, config_abs : 80, previous : [0 0 0 4 0 0 0 0 0 0], current : 4
heap profile: 83: 374069984 [3300: 14768402720] @ heap/1048576
79: 374063104 [3119: 14768390144] @ 0x40104b3 0x401024f 0x42bb1ba 0x4252ff4 0x4254e35 0x4255d73 0x425202c 0x4065e41
#	0x42bb1b9	main.alloc+0x69				/Users/xargin/go/src/github.com/mosn/holmes/example/alloc.go:25
#	0x4252ff3	net/http.HandlerFunc.ServeHTTP+0x43	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2012
#	0x4254e34	net/http.(*ServeMux).ServeHTTP+0x1a4	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2387
#	0x4255d72	net/http.serverHandler.ServeHTTP+0xa2	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2807
#	0x425202b	net/http.(*conn).serve+0x86b		/Users/xargin/sdk/go1.14.2/src/net/http/server.go:1895
```

### deadloop caused cpu outage

See this [example](example/cpu_explode.go).

After warming up finished, curl localhost:10003/cpuex several times, then you'll see the cpu profile dump to your dump path.

Notice the cpu profile currently doesn't support text mode.

```
go tool pprof cpu.20201028100641.bin

(pprof) top
Showing nodes accounting for 19.45s, 99.95% of 19.46s total
Dropped 6 nodes (cum <= 0.10s)
      flat  flat%   sum%        cum   cum%
    17.81s 91.52% 91.52%     19.45s 99.95%  main.cpuex.func1
     1.64s  8.43% 99.95%      1.64s  8.43%  runtime.asyncPreempt

(pprof) list func1
Total: 19.46s
ROUTINE ======================== main.cpuex.func1 in /Users/xargin/go/src/github.com/mosn/holmes/example/cpu_explode.go
    17.81s     19.45s (flat, cum) 99.95% of Total
      80ms       80ms      1:package main
         .          .      2:
         .          .      3:import (
         .          .      4:	"net/http"
         .          .      5:	"time"
         .          .      6:
         .          .      7:	"github.com/mosn/holmes"
         .          .      8:)
         .          .      9:
         .          .     10:func init() {
         .          .     11:	http.HandleFunc("/cpuex", cpuex)
         .          .     12:	go http.ListenAndServe(":10003", nil)
         .          .     13:}
         .          .     14:
         .          .     15:var h = holmes.New("2s", "1m", "/tmp", false).
         .          .     16:	EnableCPUDump().Config(20, 25, 80)
         .          .     17:
         .          .     18:func main() {
         .          .     19:	h.Start()
         .          .     20:	time.Sleep(time.Hour)
         .          .     21:}
         .          .     22:
         .          .     23:func cpuex(wr http.ResponseWriter, req *http.Request) {
         .          .     24:	go func() {
    17.73s     19.37s     25:		for {
         .          .     26:		}
         .          .     27:	}()
         .          .     28:}

```

So we find out the criminal.

### large thread allocation caused by cgo block 

Will be supported in the future
