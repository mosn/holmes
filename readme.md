# holmes
[中文版](./doc/zh.md)

Self-aware Golang profile dumper.

Our online system often crashes at midnight (usually killed by the OS due to OOM). 
As lazy developers, we don't want to be woken up at midnight and waiting for the online error to recur.

holmes comes to rescue.

## Design

Holmes collects the following stats every interval passed:

* Goroutine number by `runtime.NumGoroutine`.
* RSS used by the current process with [gopsutil](https://github.com/shirou/gopsutil)
* CPU percent a total. eg total 8 core, use 4 core = 50% with [gopsutil](https://github.com/shirou/gopsutil)

In addition, holmes will collect `RSS` based on GC cycle, if you enable `GC heap`.

After warming up(10 times collects after starting application) phase finished, 
Holmes will compare the current stats with the average
of previous collected stats(10 cycles). If the dump rule is matched, Holmes will dump
the related profile to log(text mode) or binary file(binary mode).

When you get warning messages sent by your own monitor system, e.g, memory usage exceed 80%,
OOM killed, CPU usage exceed 80%, goroutine num exceed 100k. The profile is already dumped
to your dump path. You could just fetch the profile and see what actually happened without pressure.


## how to use

### dump goroutine when goroutine number spikes

```go
h, _ := holmes.New(
    holmes.WithCollectInterval("5s"),
    holmes.WithDumpPath("/tmp"),
    holmes.WithTextDump(),
    holmes.WithGoroutineDump(10, 25, 2000, 10*1000),
)
h.EnableGoroutineDump()

// start the metrics collect and dump loop
h.Start()

// stop the dumper
h.Stop()
```

* WithCollectInterval("5s") means the system metrics are collected once 5 seconds
* WithDumpPath("/tmp") means the dump binary file(binary mode)  will write content to `/tmp` dir.
* WithTextDump() means not in binary mode, so it's text mode profiles
* WithGoroutineDump(10, 25, 2000, 100*1000,time.Minute) means dump will happen when current_goroutine_num > 10 && 
  current_goroutine_num < `100*1000` && current_goroutine_num > `125%` * previous_average_goroutine_num or current_goroutine_num > `2000`,
  `time.Minute` means once a dump happened, the next dump will not happen before cooldown
  finish-1 minute.
  > WithGoroutineDump(min int, diff int, abs int, max int, coolDown time.Duration)
  > 100*1000 means max goroutine number, when current goroutines number is greater 100k, holmes would not 
  > dump goroutine profile. Cuz if goroutine num is huge, e.g, 100k goroutine dump will also become a 
  > heavy action: stw && stack dump. Max = 0 means no limit.
  
### dump cpu profile when cpu load spikes

```go
h, _ := holmes.New(
    holmes.WithCollectInterval("5s"),
    holmes.WithDumpPath("/tmp"),
    holmes.WithCPUDump(20, 25, 80, time.Minute),
    holmes.WithCPUMax(90),
)
h.EnableCPUDump()

// start the metrics collect and dump loop
h.Start()

// stop the dumper
h.Stop()
```

* WithCollectInterval("5s") means the system metrics are collected once 5 seconds
* WithDumpPath("/tmp") means the dump binary file(binary mode)  will write content to `/tmp` dir.
* WithBinaryDump() or WithTextDump() doesn't affect the CPU profile dump, because the pprof 
  standard library doesn't support text mode dump.
* WithCPUDump(10, 25, 80,time.Minute) means dump will happen when cpu usage > `10%` && 
  cpu usage > `125%` * previous cpu usage recorded or cpu usage > `80%`.
  `time.Minute` means once a dump happened, the next dump will not happen before
  cooldown finish-1 minute.
* WithCPUMax means holmes would not dump all types profile when current cpu 
  usage percent is greater than CPUMaxPercent.  

### dump heap profile when RSS spikes

```go
h, _ := holmes.New(
    holmes.WithCollectInterval("5s"),
    holmes.WithDumpPath("/tmp"),
    holmes.WithTextDump(),
    holmes.WithMemDump(30, 25, 80,time.Mintue),
)

h.EnableMemDump()

// start the metrics collect and dump loop
h.Start()

// stop the dumper
h.Stop()
```

* WithCollectInterval("5s") means the system metrics are collected once 5 seconds
* WithDumpPath("/tmp") means the dump binary file(binary mode)  will write content to `/tmp` dir.
* WithTextDump() means not in binary mode, so it's text mode profiles
* WithMemDump(30, 25, 80, time.Minute) means dump will happen when memory usage > `10%` && 
  memory usage > `125%` * previous memory usage or memory usage > `80%`. 
  `time.Minute` means once a dump happened, the next dump will not happen before
  cooldown finish-1 minute.
  
### Dump heap profile when RSS spikes based GC cycle

In some situations we can not get useful information, such the application allocates heap memory and 
collects it between one `CollectInterval`. So we design a new heap memory monitor rule, which bases on
GC cycle, to control holmes dump. It will dump twice heap profile continuously while RSS spike, then devs
can compare the profiles through `pprof base` command.

Note: we recommend enabling one of dump type between `Memory Dump` and `GcHeap Dump`.

```go
	h, _ := holmes.New(
		holmes.WithDumpPath("/tmp"),
		holmes.WithLogger(holmes.NewFileLog("/tmp/holmes.log", mlog.INFO)),
		holmes.WithBinaryDump(),
		holmes.WithMemoryLimit(100*1024*1024), // 100MB
		holmes.WithGCHeapDump(10, 20, 40, time.Minute),
		// holmes.WithProfileReporter(reporter),
	)
	h.EnableGCHeapDump().Start()
	time.Sleep(time.Hour)
```
  
### enable them all!

It's easy.

```go
h, _ := holmes.New(
    holmes.WithCollectInterval("5s"),
    holmes.WithDumpPath("/tmp"),
    holmes.WithTextDump(),

    holmes.WithCPUDump(10, 25, 80, time.Minute),
    //holmes.WithMemDump(30, 25, 80, time.Minute),
    holmes.WithGCHeapDump(10, 20, 40, time.Minute),
    holmes.WithGoroutineDump(500, 25, 20000, 0),
)

    h.EnableCPUDump().
    EnableGoroutineDump().
	EnableMemDump().
	EnableGCHeapDump().

```

### running in docker or other cgroup limited environment

```go
h, _ := holmes.New(
    holmes.WithCollectInterval("5s"),
    holmes.WithDumpPath("/tmp"),
    holmes.WithTextDump(),

    holmes.WithCPUDump(10, 25, 80),
    holmes.WithCGroup(true), // set cgroup to true
)
```

## known risks

Collect a goroutine itself [may cause latency spike](https://github.com/golang/go/issues/33250) because of the STW.

## Show cases
[Click here](./example/readme.md)