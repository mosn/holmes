# holmes

Self-aware Golang profile dumper.

Our online system often crashes(mostly killed by OOM or cpu use outage) in the midnight, as a lazy developer, we don't want to wake up at night and wait for the online bug to reproduce.

holmes comes to rescue.

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

### goroutine explosion caused by channel block

see this [example](examples/channelblock.go)

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

The producer forget to close the task channel after produce finishes, so every request to this URI will leak a goroutine, we could curl http://localhost:10003/leak some time and got the following log:

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

### large thread allocation caused by cgo block 

Will be supported in the future
