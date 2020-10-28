# Holmes 福尔摩斯

自省的 Golang Profile dumper。

现代的 Go 程序大多运行在环境有限制的 docker 里，在比较大的项目里，我们没办法直接很快掌握所有代码细节，但历史项目在线上运行总是难免因为各种遗留 bug 而 crash。这里最讨厌的就是那些总是在半夜 crash 的，很多 crash 在一瞬间发生，半分钟内 OOM kill 重启，从发现有问题到上线去看现场已经来不及了，哪怕是白天都不一定抓得到，晚上的就更尴尬了。

holmes 是为了解决抖动问题定位难而生的。

## case show

### make 1GB slice 导致 RSS 占用抖动, OOM

参考这个 [例子](example/1gbslice.go)

预热结束后, curl http://localhost:10003/make1gb 几次，就可以看到：

```
heap profile: 0: 0 [1: 1073741824] @ heap/1048576
0: 0 [1: 1073741824] @ 0x42ba3ef 0x4252254 0x4254095 0x4254fd3 0x425128c 0x40650a1
#	0x42ba3ee	main.make1gbslice+0x3e			/Users/xargin/go/src/github.com/mosn/holmes/example/1gbslice.go:24
#	0x4252253	net/http.HandlerFunc.ServeHTTP+0x43	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2012
#	0x4254094	net/http.(*ServeMux).ServeHTTP+0x1a4	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2387
#	0x4254fd2	net/http.serverHandler.ServeHTTP+0xa2	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2807
#	0x425128b	net/http.(*conn).serve+0x86b		/Users/xargin/sdk/go1.14.2/src/net/http/server.go:1895
```

1: 1073741824 表示 1 object 和 1GB 的内存消耗，出现这种问题时，监控系统的 RSS 监控应该可以看到相应的抖动。

### 死锁导致 goroutine 堆积，OOM

参考这个 [例子](./example/deadlock.go)

curl localhost:10003/lockorder1

curl localhost:10003/lockorder2

预热结束后，wrk -c 100 http://localhost:10003/req, 然后就可以看到死锁导致的 goroutine 堆积了：

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

req 接口被死锁阻塞了。

定位死锁 bug 需要开启 DumpFullStack，否则栈可能会被截断，会看不到具体锁在哪里。

### channel 阻塞导致 goroutine 堆积，OOM

参考这个 [例子](examples/channelblock.go)

预热结束后, wrk -c100 http://localhost:10003/chanblock

```
goroutine profile: total 203
100 @ 0x4037750 0x4007011 0x4006a15 0x42ba3c9 0x4252234 0x4254075 0x4254fb3 0x425126c 0x4065081
#	0x42ba3c8	main.channelBlock+0x38			/Users/xargin/go/src/github.com/mosn/holmes/example/channelblock.go:26
#	0x4252233	net/http.HandlerFunc.ServeHTTP+0x43	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2012
#	0x4254074	net/http.(*ServeMux).ServeHTTP+0x1a4	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2387
#	0x4254fb2	net/http.serverHandler.ServeHTTP+0xa2	/Users/xargin/sdk/go1.14.2/src/net/http/server.go:2807
#	0x425126b	net/http.(*conn).serve+0x86b		/Users/xargin/sdk/go1.14.2/src/net/http/server.go:1895
```

定位起来比较简单。

### 缓慢泄露 goroutine，直至 OOM

参考这个[例子](example/slowlyleak.go)

producer 在生产完成后忘记关闭 task channel 了，所以每次请求这个接口都会泄露一个 goroutine。我们 curl 这个 http://localhost:10003/leak 接口几次，就可以在 dump 的日志中看到相应的泄露位置了：

```
goroutine profile: total 10
7 @ 0x4038380 0x4008497 0x400819b 0x42bb129 0x4065cb1
#	0x42bb128	main.leak.func1+0x48	/Users/xargin/go/src/github.com/mosn/holmes/example/slowlyleak.go:26
```

有了现场找泄露位置很简单。

### 业务逻辑中的大量内存分配导致的 OOM

参考这个[例子](example/alloc.go), 这个和 make 一个大 slice 那个其实差不多。

预热结束后, wrk -c100 http://localhost:10003/alloc:

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

### CPU 使用尖峰，以死循环为例

参考这个 [例子](./example/cpu_explode).

warming up 结束后，curl localhost:10003/cpuex 几次，然后就可以看到 cpu profile 被 dump 到启动时设置的 dump 目录了。

注意，当前 cpu 的 profile 暂时还不支持 text 模式。需要用 go tool pprof 来浏览。

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

这个 for 循环就是犯罪现场了。

### cgo 阻塞导致线程数暴涨

未来会支持这种场景。

