# 网络轮询器 netpoll

> 环境：`go version go1.26.3 darwin/amd64`。源码：`runtime/netpoll.go`、`runtime/netpoll_{epoll,kqueue}.go`、`internal/poll/fd_unix.go`。配套代码：`notes/netpoll/`。
>
> **本文讲"同步的 API 怎么跑在异步的内核接口上"**：fd 怎么注册、G 怎么挂起、谁来唤醒、deadline 怎么实现。**调度器什么时候调用 `netpoll()`** 见 sched.md 2.1/3.3，**G/M/P 结构体**见 gmp.md。
>
> 一句话：**Go 把 "每连接一线程" 换成了 "每连接一 goroutine + 一个全局 epoll"**，同步的写法，异步的成本。
>
> 版本演进（只列能在源码里对上号的）：
> - **1.2**：网络轮询器集成进 runtime（`netpoll.go` 的版权年份就是 2013）。此前是独立的 poll server goroutine。
> - **1.14**：引入 `netpollBreak()`——在此之前没法唤醒阻塞在 `epoll_wait` 里的 M，timer 精度被 netpoll 拖累。
> - **1.26**：新增 `/sched/goroutines/{running,runnable,waiting,not-in-go}` 指标（1.25 的 `description.go` 里还没有），能直接看有多少 G 挂在 IO 上。`netpoll.go` 本身 1.25 → 1.26 一行未改，只有 `netpoll_epoll.go` 跟着 `internal/runtime/syscall` → `internal/runtime/syscall/linux` 改了包名。
>
> 几个没有版本号但值得知道的实现细节：Linux 的 `netpollBreak` 用 **eventfd**（早期是 pipe，占两个 fd）；`ev.Data` 里存的是带 `fdseq` 的 **tagged pointer**，防止 fd 复用导致唤错 goroutine；deadline 靠 runtime timer 实现，不再需要额外 goroutine。

## 一、fd 是怎么进到 netpoll 里的

### 1.1 从 `conn.Read()` 到 `epoll_wait`

中间隔着六层，每层只干一件事：

| 层 | 文件 | 职责 |
| --- | --- | --- |
| `net.TCPConn` | `net/tcpsock.go` | 用户看到的**同步阻塞** API |
| `net.netFD` | `net/fd_posix.go` | 网络语义（地址、类型、错误包装） |
| `internal/poll.FD` | `internal/poll/fd_unix.go` | **重试循环**：`EAGAIN` → `waitRead` → 重试 |
| `internal/poll.pollDesc` | `internal/poll/fd_poll_runtime.go` | 一个 `runtimeCtx uintptr`，靠 `go:linkname` 打到 runtime |
| `runtime.pollDesc` | `runtime/netpoll.go:75` | `rg`/`wg` 两个信号量 + 两个 deadline timer |
| `netpollopen` / `netpoll` | `runtime/netpoll_{epoll,kqueue}.go` | `epoll_ctl` / `kevent` |

runtime 只要求每个平台实现 5 个函数（`netpoll.go` 顶部注释）：

```text
netpollinit()                      初始化，只调用一次
netpollopen(fd, pd) errno          把 fd 以**边缘触发**注册进来
netpollclose(fd) errno             注销
netpoll(delta) (gList, int32)      delta<0 阻塞 / ==0 非阻塞 / >0 超时；返回就绪的 G 列表
netpollBreak()                     唤醒阻塞在 netpoll 里的那个 M
```

**注册永远是边缘触发、读写一起注册、注册一次管一辈子**：

```go
// linux: netpoll_epoll.go
ev.Events = EPOLLIN | EPOLLOUT | EPOLLRDHUP | EPOLLET
EpollCtl(epfd, EPOLL_CTL_ADD, fd, &ev)

// darwin/BSD: netpoll_kqueue.go
ev[0].filter = EVFILT_READ;  ev[0].flags = EV_ADD | EV_CLEAR
ev[1].filter = EVFILT_WRITE; ev[1].flags = EV_ADD | EV_CLEAR
```

所以 Go 里**没有** "重新 arm fd" 这一步（`netpollarm` 在 epoll/kqueue 上直接 `throw("unused")`）——省掉了每次 IO 一次 `epoll_ctl` 的系统调用，代价是 ET 模式下必须一直读到 `EAGAIN`（`internal/poll` 的循环已经替你做了）。

`ev.Data` 里存的不是裸指针而是 **tagged pointer**（`taggedPointerPack(pd, pd.fdseq)`）：fd 关闭后可能被立刻复用，内核里残留的旧事件带着旧 `fdseq`，`netpoll` 一比对就丢弃，避免唤错 goroutine。

### 1.2 谁进得了 netpoll：看 `O_NONBLOCK`

Go 只给"能被 poll 的 fd"设 `O_NONBLOCK`，所以这个标志就是 pollable 的指示灯。用 `SyscallConn().Control` + `fcntl(F_GETFL)` 实测（darwin）：

```text
TCP conn               O_NONBLOCK=true   -> 注册进 netpoll，阻塞时只挂起 G
UDP conn               O_NONBLOCK=true   -> 注册进 netpoll，阻塞时只挂起 G
os.Pipe（管道）        O_NONBLOCK=true   -> 注册进 netpoll，阻塞时只挂起 G
os.File（普通文件）    O_NONBLOCK=false  -> 没进 netpoll，阻塞时占住整个线程
os.Stdin               O_NONBLOCK=false  -> 没进 netpoll，阻塞时占住整个线程
```

两个平台的**失败方式不一样**：

- **Linux**：`os.OpenFile` 照样调 `epoll_ctl(ADD)`，内核对普通文件返回 `EPERM`，`FD.Init` 里 fallback 成 `isBlocking = 1`；
- **darwin/BSD**：`os` 包**主动跳过**普通文件和目录（`os/file_unix.go` `newFile`），因为 kqueue 对它们行为不正确（FreeBSD 上普通文件永远报可写，Dragonfly/NetBSD/OpenBSD 只报一次）；darwin 上 **FIFO 也被跳过**，因为关掉最后一个写端不产生 kqueue 事件（issue #19093 / #24164 / #66239）。

**结论：网络 fd 一定 pollable，普通文件一定不 pollable。** 这就是"网络 IO 不涨线程、文件 IO 涨线程"的全部根因（3.1/3.2、sched.md 3.2）。

> `io_uring` 能让普通文件也异步，但 Go 至今没用（提案 #31908 挂了多年）：跨平台抽象、安全边界、以及 `netpoll` 这套接口很难兼容。

### 1.3 坑：`f.Fd()` 会把 fd 退回阻塞模式

```go
pr, pw, _ := os.Pipe()
nonblock(pr)   // true
_ = pr.Fd()    // 副作用就在这一行
nonblock(pr)   // false
```

`os/file_unix.go`：

```go
func (f *File) fd() uintptr {
	// 历史上 Fd() 一直返回阻塞 fd，为了兼容只能保持
	if f.nonblock {
		f.pfd.SetBlocking()
	}
	return uintptr(f.pfd.Sysfd)
}
```

**代价**：这个 `File` 从此脱离 netpoll，之后每次读写都占住一个 OS 线程。要拿 fd 做 `setsockopt` 之类的事，用 `SyscallConn().Control(fn)`——它只在回调期间加引用计数，不改阻塞模式。

## 二、读写路径

### 2.1 从 `Read` 到 `gopark`

`internal/poll/fd_unix.go` 的 `(*FD).Read` 是整个机制的核心，只有十几行：

```go
for {
	n, err := ignoringEINTRIO(syscall.Read, fd.Sysfd, p)
	if err != nil {
		n = 0
		if err == syscall.EAGAIN && fd.pd.pollable() {
			if err = fd.pd.waitRead(fd.isFile); err == nil {
				continue           // <- 被唤醒后重新读
			}
		}
	}
	return n, fd.eofError(n, err)
}
```

关键点：

1. **先读再说**（乐观）。数据已经在内核缓冲区时根本不进 netpoll，一次 syscall 就返回——这是高吞吐场景的常态。
2. 只有 `EAGAIN` 才挂起，且必须 `pollable()`（否则退化成阻塞读）。
3. 唤醒表示的是"**fd 可读了**"而不是"拿到数据了"，所以必须 `continue` 重新 `syscall.Read`。ET 模式的正确性也靠这个循环保证。

`waitRead` → `runtime_pollWait` → `netpollblock` → 全流程唯一的挂起点：

```go
// runtime/netpoll.go:575
gopark(netpollblockcommit, unsafe.Pointer(gpp), waitReasonIOWait, traceBlockNet, 5)
```

`waitReasonIOWait` 就是 dump 里的 `[IO wait]`（`runtime2.go:1225`），**全 runtime 只有这一处产生它**。实测：

```text
dump: goroutine 33 [IO wait]:
栈帧: internal/poll.runtime_pollWait(0x8243c00, 0x72)
栈帧: internal/poll.(*pollDesc).wait(...)
栈帧: internal/poll.(*pollDesc).waitRead(...)
```

看到这个栈就可以确定：**这个 goroutine 没占线程，它在等对端**。排查"服务卡住"时，满屏 `[IO wait]` 通常不是病因而是症状。

### 2.2 `pollDesc.rg` / `wg` 的四个状态

读和写各一个 `atomic.Uintptr`，是一个手写的二值信号量：

| 值 | 含义 |
| --- | --- |
| `pdNil (0)` | 什么都没有 |
| `pdReady (1)` | **就绪通知已挂账**，等 G 来取（取走时置回 `pdNil`） |
| `pdWait (2)` | 有 G 准备挂起，但还没真正 park（CAS 的中间态） |
| G 指针 | G 已经挂在上面了 |

`pdReady` 这个状态解决了经典竞态：**IO 在 G 挂起之前就绪**。此时 `netpollready` 把状态置成 `pdReady`，`netpollblock` 一看不是 `pdNil` 就直接返回、根本不 park。反过来 `pdWait` 保证 `gopark` 的 `commit` 函数（`netpollblockcommit`）能原子地把 G 指针写进去，写失败说明有人抢先 ready 了，park 会被撤销。

### 2.3 唤醒：`netpoll()` 一次捞一批

```go
func netpoll(delay int64) (gList, int32)
```

平台实现拿到内核返回的事件数组（epoll 一次最多 128 个），对每个事件调 `netpollready`，把对应的 G 串成一个 **`gList`** 一次性返回。调用方（`findRunnable`）再 `injectglist` 把整批塞进运行队列。

**批量是关键**：一万个连接同时来数据，是一次 `epoll_wait` + 一次 `injectglist`，不是一万次唤醒。

调度器在三个地方调它（sched.md 2.1、3.3）：

| 位置 | 参数 | 目的 |
| --- | --- | --- |
| `findRunnable` 第 ⑥ 步 | `netpoll(0)` 非阻塞 | 顺手看一眼有没有就绪的 IO |
| `findRunnable` 最后（`stopm` 之前） | `netpoll(delay)` 阻塞 | **这个 M 没活干了，就替所有人守着 epoll** |
| `sysmon` | `netpoll(0)` 非阻塞 | 兜底：距上次 poll 超过 **10ms** 就强制捞一次（`proc.go:6569`） |

那个阻塞的 M 是通过 `sched.lastpoll.Swap(0)` 抢到的——**同一时刻只有一个 M 阻塞在 netpoll 上**，其他 M 该干嘛干嘛。

### 2.4 `netpollBreak`：怎么叫醒守夜的那个 M

M 阻塞在 `epoll_wait(timeout = 最近的 timer)` 上时，如果这时创建了一个**更早**到期的 timer，它就得被提前叫醒。1.14 之前没有这个能力，timer 精度受 netpoll 拖累。

现在的做法是往 poller 自己注册的一个 fd 上写一个字节：

- **Linux**：`eventfd`（早期实现是 pipe，占两个 fd）；
- **darwin/BSD**：`EVFILT_USER` 事件，`ident = 0xee1eb9f4`（这个魔数在源码注释里写着"当年 Google 搜不到，所以有人搜到它就能找回这里"）。

`netpollWakeSig` 是个 CAS 保护的去重标志：已经有一次唤醒在路上就不重复写，避免惊群。

## 三、为什么网络 IO 不涨线程

### 3.1 N 个空闲连接 = N 个 goroutine + 0 个新线程

实测（8 核，500 条空闲 TCP 连接）：

```text
开始：           goroutines=1    threads=4   GOMAXPROCS=8
500 个空闲连接后：goroutines=502  threads=10
```

goroutine 数量线性增长，线程数几乎不动。所有 G 都 `_Gwaiting` 在各自 `pollDesc.rg` 上，**没有一个占着 M**。这就是 Go 敢让你"一个连接一个 goroutine"的底气：一个 goroutine 起步 2KB 栈（mem.md 3.2），一万条连接约 20MB，而一万个线程光栈就是 80GB 虚拟内存 + 调度灾难。

### 3.2 反例：同样的 fd，脱离 netpoll 之后线程就涨了

把 1.3 那个坑当实验用——100 个管道，唯一区别是有没有调过 `Fd()`：

```text
100 个管道（netpoll 接管）   goroutines +100   threads +0
100 个管道（Fd() 后阻塞）    goroutines +100   threads +91
```

**同样的 fd、同样的 `io.Copy`，并发数直接翻译成了线程数。** 机制见 sched.md 3.2：阻塞式读走 `entersyscall`，sysmon 的 `retake()` 发现 P 卡在 `_Psyscall` 超过一个周期就 `handoffp()` 给新 M。上限是 `sched.maxmcount = 10000`，超了 fatal。

普通文件 IO 走的就是这条路。所以：

- **高并发读写文件必须自己限流**（`semaphore.Weighted` 或固定 worker 池），别指望 goroutine 便宜就无脑起；
- cgo 调用、DNS 解析（走 cgo resolver 时）同理；
- 判断依据很简单：`pprof.Lookup("threadcreate").Count()` 远大于 `GOMAXPROCS` 就有问题。

## 四、deadline

### 4.1 `SetDeadline` 是唯一能打断阻塞 IO 的正规手段

```text
Read 返回耗时 81ms（deadline 设的 80ms）
err                                      = read tcp ...: i/o timeout
errors.As(err, &net.Error) / ne.Timeout()= true / true
errors.Is(err, os.ErrDeadlineExceeded)   = true
errors.Is(err, context.DeadlineExceeded) = false      ← 两套体系，别混
```

实现（`poll_runtime_pollSetDeadline`，`netpoll.go:372`）：在 `pollDesc` 上挂一个 **runtime timer**（`rt`/`rd`/`rseq` 三件套，time.md 3.1）。到期时 `netpolldeadlineimpl` 把 `rd` 置为 `-1` → `publishInfo()` → `netpollunblock` 唤醒 G；G 醒来后 `netpollcheckerr` 返回 `pollErrTimeout`，转成 `ErrDeadlineExceeded`。

`rseq`/`wseq` 是序号，防止"timer 已触发但 deadline 已被改掉"的 stale 唤醒。

**读写 deadline 相同时（`rd == wd`）只挂一个 timer**（源码里的 `combo` 分支）。实测这个优化值一半的钱：

```text
BenchmarkSetDeadline-8      228.7 ns/op    ← SetDeadline(t)：一次调用，一个 timer
BenchmarkSetRWDeadline-8    411.3 ns/op    ← SetReadDeadline + SetWriteDeadline
```

所以读写超时一样时用 `SetDeadline`，别拆成两句。放到真实往返里这点开销看不见（本机 RTT 就有 ~20µs），但每秒百万次的代理/网关上是实打实的。

### 4.2 deadline 是绝对时间点，不会自动续期

```text
第 1 次读：n=4 cost=0s     err=<nil>
第 2 次读：n=4 cost=0s     err=<nil>
第 3 次读：n=0 cost=1ms    err=i/o timeout      ← 撞上 120ms 前设的那个点
```

这是最常见的误用。`SetReadDeadline(t)` 设的是**绝对时刻**，不是"每次 Read 的超时"。正确写法是每次读之前重设：

```go
for {
	conn.SetReadDeadline(time.Now().Add(idleTimeout))
	n, err := conn.Read(buf)
	...
}
```

`bufio.Reader` 包一层时尤其容易忘——一次 `bufio.Read` 可能对应多次底层 syscall，deadline 覆盖的是整段。

其他细节：

- `SetReadDeadline(time.Time{})`（零值，`d == 0`）**清除** deadline，永不超时；
- deadline 过期后连接**没有**被关闭，后续读写会立刻返回 timeout；但协议状态多半已经半截了，实践中直接 `Close` 更安全；
- `http.Server` 的 `ReadTimeout`/`WriteTimeout` 就是在每个连接上调 `SetReadDeadline`——它们同样是绝对时间点，覆盖"从连接开始"而不是"从这次请求开始"，长连接场景要用 `ReadHeaderTimeout` + 处理器内部的 `http.ResponseController` 逐次续期。

### 4.3 用 deadline 实现取消

`SetDeadline` 是**并发安全**的，可以在任意 goroutine 调用。把 deadline 设到过去，卡在 `Read` 里的 G 立刻返回：

```go
stop := context.AfterFunc(ctx, func() {
	conn.SetDeadline(time.Now().Add(-time.Second))
})
defer stop()
```

这是把 `context` 接到网络 IO 上的标准写法（`context.AfterFunc` 是 1.21 加的，context.md 1.8）。`Close()` 也能唤醒（`poll_runtime_pollUnblock` → `pollErrClosing`），但那是一次性的：连接不能再用了。

## 五、常见坑

### 5.1 CLOSE_WAIT 堆积 = 你的代码没 Close

```text
客户端全部 Close 之后，服务端 goroutine 仍然 +21
```

| 状态堆积 | 谁的锅 |
| --- | --- |
| **CLOSE_WAIT**（收到 FIN 但没回 FIN） | **你的代码**没调 `Close()` |
| **TIME_WAIT**（主动关闭方等 2MSL） | 正常现象；量大就复用连接或调内核参数 |

`pollDesc` 从 `pollCache` 里分配、标了 `sys.NotInHeap`，**GC 回收不了**——fd 泄漏就是真泄漏，只能靠 `Close`。

### 5.2 `http.Response.Body` 必须读完再 Close

```go
resp, err := client.Do(req)
if err != nil { return err }
defer resp.Body.Close()
// ...用完之后：
io.Copy(io.Discard, resp.Body)   // 少了这句，连接不回池
```

`Transport` 判定连接"脏"就不放回池子，等于每次请求重新三次握手 + TLS。表现是 QPS 上不去 + TIME_WAIT 暴涨，profile 上看不出来（时间花在内核里）。

### 5.3 小包读一定要包 `bufio`

```text
BenchmarkReadRaw-8      1108 ns/op    14 MB/s     ← 每 16 字节一次 syscall
BenchmarkReadBufio-8      32 ns/op   492 MB/s     ← 4KB 缓冲，syscall 降到 1/256
```

**35 倍**。netpoll 省的是线程，省不了 syscall；协议解析类代码（逐字段读长度、读 header）不包 `bufio` 就是纯亏。

### 5.4 多段写用 `net.Buffers`（writev）

```text
BenchmarkWriteTwice-8    8849 ns/op    ← 两次 Write = 两次 syscall
BenchmarkWriteConcat-8   4825 ns/op    ← 用户态拼接，一次 syscall，多一次拷贝
BenchmarkWritev-8        5309 ns/op    ← writev，一次 syscall，零拷贝
```

`head + body` 这种小数据拼接更快（拷贝几十字节比多一次 syscall 便宜）；**body 大的时候 `net.Buffers` 明显占优**——它避免了一次全量拷贝，而且不用为拼接结果分配内存。注意 `WriteTo` 会**修改** `net.Buffers` 切片本身（消费已写出的部分），别复用同一个变量。

### 5.5 别在 `Accept` 循环里做重活

`Accept` 本身也走 netpoll（listener fd 也注册了）。循环体里做任何阻塞的事（比如同步查数据库判断黑名单）都会拖慢握手队列，表现为客户端连接超时但 CPU 很闲。正确做法永远是 `go handle(conn)`，限流靠信号量而不是靠慢。

## 六、可观测性

```text
/sched/goroutines:goroutines               = 15
/sched/goroutines/running:goroutines       = 1
/sched/goroutines/runnable:goroutines      = 0
/sched/goroutines/waiting:goroutines       = 14
/sched/goroutines/not-in-go:goroutines     = 0
```

Go 1.26 新增的这组 `runtime/metrics` 无 STW，适合常驻采集：

- `waiting` 高、`runnable` 低 → 大量 G 挂在 IO/锁上（连接池的常态，**不是**问题）；
- `runnable` 持续 > 0 → CPU 不够或有 G 不让出（sched.md 4.1）；
- `not-in-go` 高 → 阻塞式 syscall/cgo 多，线程要涨了（3.2）。

其他手段：

| 手段 | 看什么 |
| --- | --- |
| `/debug/pprof/goroutine?debug=2` | `[IO wait]` + `internal/poll.(*pollDesc).wait` 栈帧 |
| `runtime/trace` 的 Network blocking profile | G 在网络上总共等了多久（profile.md 5.2） |
| trace 里的 `GoUnblock(reason=network)` | netpoll 唤醒事件，能看到"poll 到"和"真跑起来"之间的调度延迟 |
| `pprof.Lookup("threadcreate").Count()` | 线程数；远超 `GOMAXPROCS` 说明有非 pollable 的阻塞 |
| `GODEBUG=schedtrace=1000` | `idleprocs` 常年满 + `runqueue` 空 = 在等 IO，不是 CPU 瓶颈 |
| `lsof -p <pid>` / `ss -s` | fd 总数、CLOSE_WAIT 数量 |

## 七、常见面试题

**1. Go 的网络 IO 是同步还是异步？**
API 是同步的，底层是异步的。`conn.Read` 先乐观地 `syscall.Read`，拿到 `EAGAIN` 才把**当前 goroutine**挂起（`gopark`，`_Gwaiting`），线程去跑别的 G；epoll/kqueue 报告可读时再把 G 放回运行队列重新读。用户拿到的是同步的写法和异步的成本（见 2.1）。

**2. netpoll 用的是水平触发还是边缘触发？为什么？**
边缘触发（`EPOLLET` / `EV_CLEAR`），而且读写在注册时一次性 arm，之后不再 `epoll_ctl`。好处是每次 IO 省一次系统调用；代价是必须循环读到 `EAGAIN`——`internal/poll` 的 `for` 循环替你保证了这一点（见 1.1、2.1）。

**3. `[IO wait]` 状态的 goroutine 占线程吗？**
不占。它 `_Gwaiting` 挂在 `pollDesc.rg`/`wg` 上，M 已经去跑别的 G 了。500 条空闲连接实测 goroutine +501、线程 +6（见 3.1）。

**4. 为什么大量文件 IO 会让线程数暴涨，网络 IO 不会？**
普通文件不能被 epoll/kqueue 正确 poll（Linux 上 `epoll_ctl` 返回 `EPERM`，darwin/BSD 上 Go 主动跳过），于是退化成阻塞式 syscall；sysmon 的 `retake()` 把卡住的 P 移交给新 M，并发数直接变成线程数。实测同样 100 个管道，pollable 的线程 +0、非 pollable 的线程 +91（见 1.2、3.2）。

**5. 谁去调用 `epoll_wait`？会不会有专门的 poller 线程？**
没有专线程。调度器在 `findRunnable` 里顺手 `netpoll(0)`；某个 M 实在找不到活干、要 `stopm()` 睡觉之前，就用 `sched.lastpoll.Swap(0)` 抢下"守夜权"，阻塞在 `netpoll(delay)` 上——同一时刻只有一个。sysmon 再兜一层底：距上次 poll 超 10ms 就强制捞一次（见 2.3）。

**6. `netpollBreak` 是干什么的？**
唤醒那个阻塞在 `epoll_wait` 上的 M。典型场景：它按"最近的 timer"设了超时，此时又来了一个更早到期的 timer。Linux 上往 `eventfd` 写 1，darwin 上触发 `EVFILT_USER`。`netpollWakeSig` 做 CAS 去重，避免重复唤醒（见 2.4）。

**7. `pollDesc.rg` 的 `pdReady`/`pdWait` 是解决什么问题的？**
IO 就绪和 goroutine 挂起之间的竞态。`pdReady` 表示"通知已挂账"，G 来了直接拿走不 park；`pdWait` 是 CAS 中间态，让 `gopark` 的 commit 函数能原子地判断"我 park 的这一刻有没有人已经 ready 了"，冲突就撤销 park（见 2.2）。

**8. `SetReadDeadline` 是怎么打断一个阻塞的 `Read` 的？**
在 `pollDesc` 上挂 runtime timer；到期时把 `rd` 置 `-1`、`publishInfo`、`netpollunblock` 唤醒 G，G 醒来 `netpollcheckerr` 返回 `pollErrTimeout` → `os.ErrDeadlineExceeded`。它是并发安全的，所以把 deadline 设到过去就等于"取消一次 IO"，这也是 `context` 接网络 IO 的标准做法（见 4.1、4.3）。

**9. `SetReadDeadline` 设一次就一直有效吗？**
不是"每次读的超时"，是**绝对时刻**。循环读必须每次重设 `time.Now().Add(d)`，否则第 N 次读会撞上很久以前设的那个点。传零值 `time.Time{}` 清除（见 4.2）。

**10. `conn.SetDeadline(t)` 和分别调 `SetReadDeadline`/`SetWriteDeadline` 有区别吗？**
有。`rd == wd` 时源码走 `combo` 分支只挂一个 timer，实测 228ns vs 411ns。读写超时相同就用 `SetDeadline`（见 4.1）。

**11. `f.Fd()` 有什么副作用？**
把文件退回阻塞模式（`(*File).fd()` 里的 `f.pfd.SetBlocking()`），从此脱离 netpoll，每次 IO 占一个线程。要拿 fd 做 `setsockopt` 用 `SyscallConn().Control`（见 1.3）。

**12. CLOSE_WAIT 和 TIME_WAIT 堆积分别说明什么？**
CLOSE_WAIT 是**你没 Close**（收到 FIN 却不回）；TIME_WAIT 是主动关闭方的正常 2MSL 等待，量大说明短连接太多，该复用连接。`pollDesc` 是 `NotInHeap` 的，GC 救不了泄漏的 fd（见 5.1）。

**13. `http.Response.Body` 为什么必须读完再 Close？**
没读完的连接被 `Transport` 判为脏连接，不放回池子，每次请求都要重新握手 + TLS。表现是 QPS 上不去、TIME_WAIT 暴涨，而 CPU profile 上什么都看不到（见 5.2）。

**14. netpoll 能省掉 syscall 吗？**
不能。它省的是**线程**。每次实际读写仍然是一次 `read`/`write` 系统调用——小包场景不包 `bufio` 实测慢 35 倍（1108ns vs 32ns），多段写用 `net.Buffers`（writev）能把两次 syscall 并成一次（见 5.3、5.4）。

**15. Go 为什么不用 io_uring？**
提案 #31908 挂了多年。难点：`netpoll` 这套接口是"fd 就绪通知"语义，io_uring 是"操作完成"语义，两者抽象不兼容；还要处理跨平台（只有 Linux 有）、内核版本差异、以及提交队列的内存安全边界。目前 Go 的普通文件 IO 仍然是阻塞 + 多线程（见 1.2）。
