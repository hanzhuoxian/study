# 性能分析：pprof 与 trace

> 环境：`go version go1.26.3 darwin/amd64`。配套代码：`notes/prof/`——一个故意写了 CPU 热点、内存泄漏、goroutine 泄漏、锁竞争的程序，`go run ./prof` 会把五种 profile 写到 `notes/prof/out/`。本文所有输出都是那份代码的真实运行结果。

## 一、六种 profile

| profile | 采集什么 | 默认开启 | 采集方式 |
| --- | --- | --- | --- |
| **cpu** | 每 10ms 一次的栈采样（`SIGPROF`） | 需显式开始/停止 | `pprof.StartCPUProfile` |
| **heap** | 存活对象 + 累计分配（4 个视图） | **是**（`MemProfileRate=512KB`） | `pprof.WriteHeapProfile` |
| **goroutine** | 当前所有 goroutine 的栈 | 是 | `pprof.Lookup("goroutine")` |
| **allocs** | 累计分配（等价于 heap 的 alloc 视图） | 是 | 同上 |
| **mutex** | 锁的**等待**时间 | **否** | `runtime.SetMutexProfileFraction(n)` |
| **block** | goroutine 阻塞时间（chan/锁/select/syscall） | **否** | `runtime.SetBlockProfileRate(n)` |

另外还有 `threadcreate`（很少用）和 `execution trace`（不是 profile，是事件流，见第四节）。

### 1.1 采集方式一：代码里写

```go
// CPU
f, _ := os.Create("cpu.pprof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()

// Heap：官方建议先 GC，去掉待回收对象的噪声
runtime.GC()
pprof.WriteHeapProfile(f)

// 任意命名 profile（goroutine/mutex/block/allocs/threadcreate）
pprof.Lookup("goroutine").WriteTo(f, 0)   // 0 = protobuf，给 go tool pprof
pprof.Lookup("goroutine").WriteTo(f, 1)   // 1 = 人可读文本（栈聚合）
pprof.Lookup("goroutine").WriteTo(f, 2)   // 2 = 每个 goroutine 一条完整栈
```

### 1.2 采集方式二：HTTP 端点（线上标准做法）

```go
import _ "net/http/pprof"    // 注册 /debug/pprof/* 到 DefaultServeMux

func main() {
    runtime.SetMutexProfileFraction(5)     // 生产建议：1/5 采样
    runtime.SetBlockProfileRate(int(1e6))  // 只记 1ms 以上的阻塞
    go http.ListenAndServe("localhost:6060", nil)   // ⚠️ 只监听 localhost
    ...
}
```

```bash
go tool pprof -http=:8080 'http://localhost:6060/debug/pprof/profile?seconds=30'
go tool pprof 'http://localhost:6060/debug/pprof/heap'
curl -o trace.out 'http://localhost:6060/debug/pprof/trace?seconds=5'
curl 'http://localhost:6060/debug/pprof/goroutine?debug=2' | head -50   # 全部 goroutine 栈
curl 'http://localhost:6060/debug/pprof/'                              # 看有哪些端点和计数
```

**三条安全纪律**：

1. **只绑 `localhost`**，或者放在单独的管理端口 + 鉴权后面。`/debug/pprof` 暴露到公网等于把源码结构、内存内容和一个 DoS 开关送出去；
2. 用 `http.NewServeMux` 时 `import _ "net/http/pprof"` **不会**自动注册（它注册到 `DefaultServeMux`），要手工 `mux.HandleFunc("/debug/pprof/", pprof.Index)` 等；
3. `?seconds=N` 期间程序会被采样，CPU profile 有约 **1-5%** 的开销，可以接受；`trace` 的开销大得多（见 4.3）。

## 二、CPU profile

### 2.1 读 `-top` 的两列

```text
$ go tool pprof -top -nodecount=12 prof/out/cpu.pprof
Duration: 2.11s, Total samples = 1.74s (82.5%)
      flat  flat%   sum%        cum   cum%
     1.46s 83.91% 83.91%      1.46s 83.91%  crypto/internal/fips140/sha256.blockSHANI
     0.08s  4.60% 88.51%      0.08s  4.60%  runtime.madvise
     0.03s  1.72% 92.53%      0.03s  1.72%  runtime.(*mspan).init
     0.01s  0.57% 98.85%      0.09s  5.17%  runtime.concatstrings
```

- **flat**：这个函数**自己**消耗的时间（不含它调用的函数）；
- **cum**（cumulative）：这个函数**及其所有被调用者**消耗的时间；
- **叶子函数** flat ≈ cum；**框架函数** flat≈0 而 cum 很大。

**看 flat 找"真正在烧 CPU 的代码"，看 cum 找"哪条调用链贵"**：

```text
$ go tool pprof -top -cum prof/out/cpu.pprof | grep main
      0     0%      1.62s 92.57%  main.slowWork
      0     0%      1.52s 86.86%  main.hashSlow      ← 86% 的时间在这条链上
      0     0%      0.09s  5.14%  main.concatSlow
```

### 2.2 常用命令

```bash
go tool pprof -http=:8080 cpu.pprof        # ★ 浏览器：火焰图 / 调用图 / 源码级注解
go tool pprof -top -nodecount=20 cpu.pprof
go tool pprof -cum -top cpu.pprof
go tool pprof -list 'hashSlow' cpu.pprof   # 源码级：每一行花了多少
go tool pprof -peek 'sha256' cpu.pprof     # 谁调用了它、它调用了谁
go tool pprof -focus 'main\.' cpu.pprof    # 只看匹配的子图
go tool pprof -ignore 'runtime\.' cpu.pprof
go tool pprof -base old.pprof new.pprof    # ★ 两份 profile 做差（优化前后对比）
```

`-http=:8080` 里最有用的三个视图：**Flame Graph**（看占比结构）、**Source**（看具体哪一行）、**Peek**（看调用关系）。

### 2.3 flat 里全是 `runtime.*` 说明什么

我第一版的示例程序（`s += ...` 拼接 2000 次）跑出来是这样：

```text
     750ms 27.27%  runtime.kevent
     540ms 19.64%  runtime.madvise
     410ms 14.91%  runtime.pthread_cond_wait
     280ms 10.18%  runtime.pthread_cond_signal
```

用户代码一个都没上榜。这不是 profile 坏了，而是**程序被 GC 和调度支配了**——`kevent`/`pthread_cond_*` 是 M 在休眠/唤醒，`madvise` 是 scavenger 在还内存。

这种 profile 的正确读法：

1. `-cum` 看用户代码在哪条链上（`main.concatSlow` cum 5%）；
2. **转而看 heap profile 的 `alloc_space`**——GC 压力的根因是分配（见 3.2）；
3. `GODEBUG=gctrace=1` 确认 GC 占比（见 gc.md 2.4）。

**经验规则：flat 前几名是 `runtime.mallocgc`/`gcDrain`/`madvise`/`kevent` 时，别在 CPU profile 上继续挖，去看分配。**

### 2.4 采样原理与盲区

CPU profile 靠 `setitimer(ITIMER_PROF)` 每 10ms 发一次 `SIGPROF`，信号处理函数记录当前栈。三个推论：

- **采样率固定 100Hz**（`runtime.SetCPUProfileRate` 可改，但 `StartCPUProfile` 会覆盖它），所以**跑不到 1 秒的东西测不准**——至少采 10-30 秒；
- **看不到不消耗 CPU 的等待**：阻塞在 IO、锁、channel 上的时间在 CPU profile 里完全不可见。那是 block/mutex profile 和 trace 的活；
- `Total samples = 1.74s (82.5%)` 里的百分比是"采样总时长 / wall duration"，**超过 100% 说明多核并行**（我第一版跑出 130%）。

## 三、内存 profile

### 3.1 四个视图

heap profile 里有四组数据，`-sample_index` 切换：

| 视图 | 含义 | 用来查 |
| --- | --- | --- |
| `inuse_space`（默认） | **当前存活**的字节数 | **内存泄漏** |
| `inuse_objects` | 当前存活的对象数 | 小对象太多、GC 慢 |
| `alloc_space` | **累计分配**的字节数 | GC 压力大、分配热点 |
| `alloc_objects` | 累计分配的对象数 | **分配次数**热点（见 mem.md 4.1） |

```text
$ go tool pprof -top -nodecount=6 prof/out/heap.pprof              # inuse_space
   74.38MB 98.67%  main.leakMemory        ← 泄漏点一眼可见

$ go tool pprof -top -sample_index=alloc_space -nodecount=6 prof/out/heap.pprof
   76.03MB 46.61%  main.concatSlow        ← 分配热点（但已经被回收了）
   75.86MB 46.51%  main.leakMemory
```

两个视图看到的是**完全不同的问题**：`concatSlow` 分配了 76MB 但都回收了（是 GC 压力问题），`leakMemory` 的 74MB 还活着（是泄漏问题）。

### 3.2 排查内存泄漏的标准流程

```bash
# ① 隔一段时间抓两次
curl -o h1.pprof 'http://host:6060/debug/pprof/heap'
sleep 600
curl -o h2.pprof 'http://host:6060/debug/pprof/heap'

# ② 做差，只看增量
go tool pprof -base h1.pprof -top -nodecount=20 h2.pprof

# ③ 定位到函数后看具体行
go tool pprof -base h1.pprof -list 'leakMemory' h2.pprof
```

`-base` 是内存排查的核心手段：**绝对值受启动期和缓存影响，增量才说明泄漏**。

如果 `inuse_space` 不涨但 RSS 涨，那不是堆泄漏，去看 gc.md 3.5 的排查表（goroutine 栈、`heap_released`、cgo）。

### 3.3 采样率

`MemProfileRate` 默认 512KB——**平均每分配 512KB 记录一次栈**，不是每次都记。所以：

- heap profile 的数字是**统计估计**（runtime 会做缩放补偿），小量分配可能完全不出现；
- 要精确（比如写单元测试断言分配量）：`runtime.MemProfileRate = 1`，但开销显著；
- 只在 `main` 最开头改它才有意义（改之前的分配已经按老速率采样了）。

## 四、goroutine / mutex / block profile

### 4.1 goroutine profile：查泄漏

`debug=1` 的文本格式最直观（按栈聚合 + 计数）：

```text
$ head -22 prof/out/goroutine.txt
goroutine profile: total 1001
500 @ 0x107b89ce 0x10749cbc 0x107498b7 0x1099b5fe 0x107c08c1
#	0x1099b5fd	main.leakGoroutines.func1+0x1d	.../prof/main.go:177

500 @ 0x107b89ce 0x1074ac8e 0x1074a7d2 0x1099b5b9 0x107c08c1
#	0x1099b5b8	main.leakGoroutines.func2+0x18	.../prof/main.go:181
```

**"500 @ 同一个栈" 就是泄漏的指纹**。对应代码：

```go
ch := make(chan int)
go func() { ch <- 1 }()      // 无人接收 -> 永久阻塞（第 177 行）

never := make(chan struct{})
go func() { <-never }()      // 永远不会被关闭（第 181 行）
```

线上排查用 `?debug=2`：输出每个 goroutine 的完整栈**和状态**（`[chan send, 10 minutes]`），带阻塞时长，比 `debug=1` 更适合找"卡了很久"的那批。

也可以直接 `kill -QUIT <pid>`（`SIGQUIT`）让进程打印全部 goroutine 栈然后退出——**没有 pprof 端点时的救命手段**。

### 4.2 mutex profile：查锁竞争

**必须先打开**：`runtime.SetMutexProfileFraction(n)`（n=1 全采，n=5 表示 1/5）。

```text
$ go tool pprof -top prof/out/mutex.pprof
Type: delay
  352.65ms 99.37%  sync.(*Mutex).Unlock (inline)
        0     0%    353.86ms  main.writeContentionProfiles.func2
```

两个反直觉的点：

- **Type 是 `delay`（等待时间），不是次数**——它回答"竞争让别人等了多久"；
- **栈顶记在 `Unlock` 上**，因为 runtime 是在解锁、把锁交给下一个等待者时才知道"这个等待者等了多久"。所以要看 cum 那一列找**是哪段业务代码在争这把锁**。

### 4.3 block profile：查阻塞

`runtime.SetBlockProfileRate(n)`：n 是纳秒阈值，1 = 全采，`1e6` = 只记 1ms 以上。

```text
$ go tool pprof -top prof/out/block.pprof
  229.25ms 47.71%  runtime.chansend1        ← channel 发送阻塞
  221.99ms 46.20%  sync.(*Mutex).Lock       ← 锁等待
   29.25ms  6.09%  sync.(*WaitGroup).Wait
```

block 和 mutex 的区别：**block 覆盖所有阻塞**（channel 收发、select、`WaitGroup.Wait`、锁），mutex 只覆盖锁竞争。查"为什么延迟高但 CPU 不高"就看 block。

**生产上两者都不要设成 1**——全采会给每次阻塞记一次栈，开销可观。常用值：`SetMutexProfileFraction(5)` + `SetBlockProfileRate(1e6)`（只关心 1ms 以上）。

## 五、execution trace

### 5.1 trace 和 profile 的区别

profile 是**统计采样**（"大概 80% 时间在这里"），trace 是**事件流**（"第 3.21ms 时 G17 在 P2 上开始跑，3.24ms 被 GC 打断"）。

trace 能回答 profile 回答不了的问题：

- 为什么 P99 延迟高（单次请求的完整时间线）；
- GC 什么时候发生、抢了谁的 CPU、STW 多久；
- goroutine 为什么迟迟不被调度（P 全忙？还是在 syscall？）；
- 并行度够不够（8 核只跑满 2 个？）。

```go
trace.Start(f); defer trace.Stop()

// 加业务语义标注
ctx, task := trace.NewTask(context.Background(), "demo-request")
defer task.End()
trace.WithRegion(ctx, "compute", func() { ... })
trace.Log(ctx, "category", "message")
```

```bash
go tool trace prof/out/trace.out          # 打开浏览器
curl -o t.out 'http://host:6060/debug/pprof/trace?seconds=5'
```

### 5.2 `go tool trace` 的六个视图

| 视图 | 看什么 |
| --- | --- |
| **View trace** | 时间线：每个 P 上跑了哪些 G、GC 何时发生、STW 多长 |
| **Goroutine analysis** | 按创建位置聚合，看每类 goroutine 的执行/阻塞时间分解 |
| **Network/Sync/Syscall blocking profile** | 各类阻塞的火焰图 |
| **Scheduler latency profile** | goroutine 从 runnable 到 running 等了多久（**调度延迟**） |
| **User-defined tasks/regions** | 你用 `NewTask`/`WithRegion` 标注的业务时间线 |
| **Minimum mutator utilization** | mutator（业务代码）能拿到的最小 CPU 比例，衡量 GC 干扰 |

**Goroutine analysis 是最被低估的一个**：它把每个 goroutine 的时间拆成 Execution / Network wait / Sync block / Syscall / Scheduler wait / GC，一眼看出"时间花在等什么"。

### 5.3 trace 的代价

trace 记录**每一个**调度事件、GC 事件、channel 操作，开销远大于 pprof：

- 吞吐下降通常 **10-30%**（分配和 goroutine 切换密集时更多）；
- 文件增长快（本示例 1 秒的 trace 就有几十 KB，真实服务每秒几 MB）；
- Go 1.21 重写了 trace 实现（分区、可流式解析），比以前好很多，但仍不适合常开。

**用法：怀疑调度/GC/延迟问题时抓 3-5 秒**，别长跑。

## 六、实战排查路线

### 6.1 CPU 高

```text
① GODEBUG=gctrace=1 看 GC 占比 → 高（>20%）就转去查分配
② go tool pprof -http 看火焰图，找最宽的那块
③ flat 前几名是 runtime.* → 查 alloc_space（2.3 节）
④ 定位到函数后 -list 看具体行
⑤ 优化后用 -base 做差确认
```

### 6.2 内存高 / 疑似泄漏

```text
① /debug/pprof/heap 抓两次，-base 做差
② inuse_space 涨 → 真泄漏，看是谁持有（常见：全局 map、cache 无淘汰、slice 截取、context 未 cancel）
③ inuse 不涨但 RSS 涨 → 看 goroutine 数（每个栈至少 2KB）、heap_released、cgo
④ 顺手看 inuse_objects：对象数暴涨会拖慢每一轮 GC
```

### 6.3 延迟高但 CPU 不高

```text
① goroutine profile：goroutine 数是否在涨（泄漏会拖慢调度和 GC 扫描）
② block profile：在等 channel 还是等锁
③ mutex profile：哪把锁在被争
④ trace 的 Scheduler latency profile：是不是根本没被调度上
⑤ 别忘了外部依赖——很多"Go 慢"最后是数据库/下游慢
```

### 6.4 goroutine 泄漏

```text
① curl '/debug/pprof/goroutine?debug=1' | head -40   看"N @ 同一个栈"
② debug=2 看阻塞时长，找卡最久的
③ 常见根因：无缓冲 channel 没人收 / context 没 cancel / WaitGroup 计数不配平
   / time.Ticker 没 Stop / HTTP response body 没 Close
④ 测试里用 go.uber.org/goleak 在 TestMain 里守住（见 test.md 5.1）
```

## 七、其他工具

| 工具 | 用途 |
| --- | --- |
| `GODEBUG=gctrace=1` | 每轮 GC 一行日志（见 gc.md 2.4） |
| `GODEBUG=inittrace=1` | 每个包 init 的耗时和分配（查启动慢） |
| `GODEBUG=schedtrace=1000` | 每秒打印调度器状态（P/M/G 数量、队列长度） |
| `GODEBUG=scheddetail=1` | 配合 schedtrace，输出每个 P/M/G 的细节 |
| `kill -QUIT <pid>` | 打印全部 goroutine 栈并退出（没有 pprof 端点时的救命手段） |
| `runtime/metrics` | 112 个运行时指标，采集不 STW（见 gc.md 1.4） |
| `benchstat` | benchmark 结果的统计对比（见 test.md 2.3） |
| `go build -gcflags='-m'` | 逃逸分析和内联决策（见 mem.md 2.1） |
| `go tool objdump` / `-gcflags=-S` | 看汇编 |
| `dlv`（delve） | 调试器；`dlv attach <pid>` 可以现场看变量 |
| **PGO**（1.21+） | 用 CPU profile 指导编译优化，见下 |

### 7.1 PGO（profile-guided optimization）

```bash
# ① 从生产环境抓一份有代表性的 CPU profile
curl -o default.pgo 'http://host:6060/debug/pprof/profile?seconds=60'

# ② 放到 main 包目录下，命名为 default.pgo
mv default.pgo ./cmd/myapp/default.pgo

# ③ 正常构建，go build 自动启用
go build ./cmd/myapp
```

编译器用 profile 决定**更激进的内联**和**基本块布局**（热路径靠前，减少分支预测失败）。官方数据是 **2-7% 的 CPU 提升**，对本来就有明显热点的服务效果更好。

注意：profile 要有代表性（用生产流量，不是压测），且随代码演进要定期更新。

## 八、常见面试题

**1. Go 有哪几种 profile？哪些默认不开？**
cpu、heap、goroutine、allocs、mutex、block（外加 threadcreate 和 execution trace）。**mutex 和 block 默认关闭**，要 `SetMutexProfileFraction`/`SetBlockProfileRate` 显式打开（见 1.1）。

**2. `flat` 和 `cum` 的区别？**
`flat` 是函数自身消耗，`cum` 含它调用的所有函数。找"烧 CPU 的代码"看 flat，找"贵的调用链"看 cum。叶子函数两者接近，框架函数 flat≈0 而 cum 很大（见 2.1）。

**3. CPU profile 的原理？有什么盲区？**
`setitimer(ITIMER_PROF)` 每 10ms 发 `SIGPROF`，信号处理函数记录当前栈，采样率 100Hz。盲区：**不消耗 CPU 的等待完全看不见**（IO、锁、channel 阻塞），短于 1 秒的行为测不准（见 2.4）。

**4. CPU profile 里 flat 全是 `runtime.madvise`/`kevent` 说明什么？**
程序被 GC 和调度支配。应该转去看 heap 的 `alloc_space`（分配是 GC 压力的根因）和 `GODEBUG=gctrace=1`，而不是继续在 CPU profile 上挖（见 2.3）。

**5. heap profile 的四个视图分别查什么？**
`inuse_space` 查泄漏、`inuse_objects` 查小对象过多、`alloc_space` 查 GC 压力、`alloc_objects` 查分配次数热点。前两个是"现在还活着"，后两个是"历史累计"——**它们经常指向完全不同的问题**（见 3.1）。

**6. 怎么用 pprof 定位内存泄漏？**
隔十分钟抓两次 heap，用 **`-base h1.pprof h2.pprof` 做差**只看增量，然后 `-list` 到具体行。绝对值受启动期和缓存干扰，增量才说明问题（见 3.2）。

**7. `MemProfileRate` 是什么？为什么 heap profile 的数字不精确？**
默认 512KB——平均每分配 512KB 才记一次栈，数字是统计估计（runtime 做了缩放补偿）。要精确得设成 1，但开销大，且必须在 `main` 最开头设（见 3.3）。

**8. 怎么发现 goroutine 泄漏？**
`/debug/pprof/goroutine?debug=1` 看有没有"N @ 同一个栈"；`debug=2` 看阻塞时长。常见根因：无缓冲 channel 无人接收、context 未 cancel、WaitGroup 不配平、Ticker 未 Stop、response body 未 Close。测试里用 `goleak`（见 4.1、6.4）。

**9. mutex profile 为什么把栈记在 `Unlock` 上？**
runtime 只在解锁、把锁移交给下一个等待者时才知道"这个等待者等了多久"，所以延迟归属记在 `Unlock` 处。要找争锁的业务代码得看 cum 列。另外它的单位是**等待时间（delay）而不是次数**（见 4.2）。

**10. block profile 和 mutex profile 的区别？**
block 覆盖所有阻塞（channel 收发、select、`WaitGroup.Wait`、锁），mutex 只覆盖锁竞争。"延迟高但 CPU 不高"优先看 block（见 4.3）。

**11. trace 和 pprof 的区别？什么时候用 trace？**
pprof 是统计采样（谁占比高），trace 是完整事件流（什么时刻发生了什么）。查 P99 延迟、GC 干扰、调度延迟、并行度不足用 trace。代价是 10-30% 吞吐下降，只抓 3-5 秒（见 5.1、5.3）。

**12. `go tool trace` 里最有用的视图是哪个？**
**Goroutine analysis**——把每个 goroutine 的时间拆成 Execution / Network wait / Sync block / Syscall / Scheduler wait / GC，一眼看出时间花在等什么。其次是 Scheduler latency profile 和 View trace（见 5.2）。

**13. 线上暴露 `/debug/pprof` 有什么风险？**
泄漏源码结构和内存内容（heap profile 里可能有明文数据）、提供 DoS 入口（`?seconds=3600`）。必须绑 localhost 或放在鉴权后的管理端口。另外注意它注册在 `DefaultServeMux`，用自定义 mux 时要手工注册（见 1.2）。

**14. 没有 pprof 端点、进程卡住了怎么办？**
`kill -QUIT <pid>`（SIGQUIT）让 runtime 打印**全部 goroutine 栈**然后退出。这是 Go 内置的最后手段。想不退出可以 `dlv attach <pid>`（见 4.1、7）。

**15. PGO 是什么？怎么用？**
1.21 起支持 profile-guided optimization：把有代表性的 CPU profile 命名为 `default.pgo` 放在 main 包目录下，`go build` 自动启用，用它指导内联和代码布局，官方数据 2-7% CPU 提升。profile 要来自生产流量并定期更新（见 7.1）。
