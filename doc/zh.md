# holmes

基于规则的自动Golang Profile Dumper.

作为一名"懒惰"的程序员，如何避免在线上Golang系统半夜宕机
（一般是OOM导致的）时起床保存现场呢？又或者如何dump压测时性能尖刺时刻的profile文件呢？

holmes 或许能帮助你解决以上问题。

## 设计

holmes 每隔一段时间收集一次以下应用指标：

* 协程数，通过`runtime.NumGoroutine`。
* 当前应用所占用的RSS，通过[gopsutil](https://github.com/shirou/gopsutil)第三方库。
* CPU使用率，比如8C的机器，如果使用了4C，则使用率为50%，通过[gopsutil](https://github.com/shirou/gopsutil)第三方库。

除此之外，holmes还会根据Gc周期收集RSS指标，如果你开启了`GCheap dump`的话。

在预热阶段（应用启动后，holmes会收集十次指标）结束后，holmes会比较当前指标是否满足用户所设置的阈值/规则，如果满足的话，则dump profile，
以日志或者二进制文件的格式保留现场。

## 如何使用

```shell
  go get https://github.com/mosn/holmes
```
在应用初始化逻辑加上对应的holmes配置。
```go
func main() {
	
  h := initHolmes()
  
  // start the metrics collect and dump loop
  h.Start()
  ......
  
  // quit the application and stop the dumper
  h.Stop()
}
func initHolmes() *Holmes{
    h, _ := holmes.New(
    holmes.WithCollectInterval("5s"),
    holmes.WithDumpPath("/tmp"),
    holmes.WithCPUDump(20, 25, 80),
    holmes.WithCPUMax(90),
    )
    h.EnableCPUDump()
    return h
}

```

holmes 支持对以下几种应用指标进行监控:
	* mem: 内存分配     
	* cpu: cpu使用率      
	* thread: 线程数    
	* goroutine: 协程数
	* gcHeap: 基于GC周期的内存分配


### Dump Goroutine profile

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

* WithCollectInterval("5s") 每5s探测一次当前应用的各项指标，该值建议设置为大于1s。
* WithDumpPath("/tmp") profile文件保存路径。
* WithTextDump() profile的内容将会输出到日志中。
* WithGoroutineDump(10, 25, 2000, 100*1000,time.Minute) 当goroutine指标满足任意以下条件时，将会触发dump操作。
  current_goroutine_num > `10` && current_goroutine_num < `100*1000` && 
  current_goroutine_num > `125`% * previous_average_goroutine_num or current_goroutine_num > `2000`.
  `time.Minute` 是两次dump操作之间最小时间间隔，避免频繁profiling对性能产生的影响。
  > WithGoroutineDump(min int, diff int, abs int, max int, coolDown time.Duration)
  > 当应用所启动的goroutine number大于`Max` 时，holmes会跳过dump操作，因为当goroutine number很大时，
  > dump goroutine profile操作成本很高（STW && dump），有可能拖垮应用。当`Max`=0 时代表没有限制。

### Dump cpu profile

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

* WithCollectInterval("5s") 每5s探测一次当前应用的各项指标，该值建议设置为大于1s。
* WithDumpPath("/tmp") profile文件保存路径。
* cpu profile支持保存文件，不支持输出到日志中，所以WithBinaryDump()和 WithTextDump()在这场景会失效。
* WithCPUDump(10, 25, 80, time.Minute) 会在满足以下条件时dump profile cpu usage > `10%` &&
  cpu usage > `125%` * previous cpu usage recorded or cpu usage > `80%`.
  `time.Minute` 是两次dump操作之间最小时间间隔，避免频繁profiling对性能产生的影响。
* WithCPUMax 当cpu使用率大于`Max`, holmes会跳过dump操作，以防拖垮系统。

### Dump Heap Memory Profile

```go
h, _ := holmes.New(
    holmes.WithCollectInterval("5s"),
    holmes.WithDumpPath("/tmp"),
    holmes.WithTextDump(),
    holmes.WithMemDump(30, 25, 80, time.Mintue),
)

h.EnableMemDump()

// start the metrics collect and dump loop
h.Start()

// stop the dumper
h.Stop()
```
* WithCollectInterval("5s") 每5s探测一次当前应用的各项指标，该值建议设置为大于1s。
* WithDumpPath("/tmp") profile文件保存路径。
* WithTextDump() profile的内容将会输出到日志中。
* WithMemDump(30, 25, 80, time.Minute) 会在满足以下条件时抓取heap profile memory usage > `10%` &&
  memory usage > `125%` * previous memory usage or memory usage > `80%`，
  `time.Minute` 是两次dump操作之间最小时间间隔，避免频繁profiling对性能产生的影响。

### 基于Gc周期的Heap Memory Dump

在一些场景下，我们无法通过定时的heap memory抓取到有效的profile, 比如应用在一个`CollectInterval`周期内分配了大量内存，
又快速回收了它们，holmes在周期前后的探测到堆使用率没有产生变化，与实际情况不符。为了解决这种情况，holmes开发了基于GC周期的
Dump类型，holmes会在堆内存使用率飙高的前后两个GC周期内各dump一次profile，然后开发人员可以使用`pprof --base`命令去对比
两个profile之间的差异。 [具体实现介绍](https://uncledou.site/2022/go-pprof-heap/)。
Note: 如果开启了`GcHeap Dump`的话，没必要再开启常规的`Memory Dump`

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

### 开启全部

holmes当然不是只支持一个类型的dump啦，你可以按需选择你需要的dump类型。

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

### 在docker 或者cgroup环境下运行 holmes

```go
h, _ := holmes.New(
    holmes.WithCollectInterval("5s"),
    holmes.WithDumpPath("/tmp"),
    holmes.WithTextDump(),

    holmes.WithCPUDump(10, 25, 80),
    holmes.WithCGroup(true), // set cgroup to true
)
```

## 已知风险
Gorountine dump 会导致 STW，[从而导致时延](https://github.com/golang/go/issues/33250) 。

## 使用示例
[点击这里](./example/readme.md)
