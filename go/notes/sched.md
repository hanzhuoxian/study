# 调度器

> 环境：`go version go1.26.3 darwin/amd64`。源码：`runtime/proc.go`。配套代码：`notes/sched/`。
>
> **本文讲"调度行为"**：查找顺序、work stealing、抢占、系统调用移交、可观测性。**G/M/P 三个结构体本身**见 gmp.md，**进程启动到第一个 goroutine** 见 exec.md。
>
> 版本演进：
> - **1.1**：引入 P，从 GM 模型变成 GMP（work stealing 成为可能）。
> - **1.2**：协作式抢占（函数调用处检查抢占标记）。
> - **1.14**：**异步抢占**（SIGURG + `asyncPreempt`），紧密循环终于能被抢占。
> - **1.21**：`runtime/trace` 重写，调度事件可流式解析。
> - **1.25**：**容器感知的 `GOMAXPROCS`**——读 cgroup CPU limit 自动设置，sysmon 每秒检查变化。
> - **1.26**：`GODEBUG=schedtrace` 的输出新增 `schedticks=[...]`（每个 P 的调度次数）。

## 一、基础

### 1.1 GOMAXPROCS

```text
runtime.NumCPU()      = 8    机器/cgroup 可见的 CPU 数
runtime.GOMAXPROCS(0) = 8    P 的数量
```

三个容易混淆的事实：

- **P 的数量决定并行度**（同时能跑多少 G），**不限制 goroutine 总数**；
- **M（线程）数量上限是 `sched.maxmcount = 10000`**（`proc.go:863`），和 P 无关——阻塞式 syscall 会让 M 远多于 P（见 3.2）；
- `GOMAXPROCS(n)` 运行时可改，但会触发 **STW**（`stopTheWorld` + `procresize`），不要在热路径上调。

**Go 1.25 起 runtime 会读 cgroup 的 CPU limit 自动设置 `GOMAXPROCS`**，并由 sysmon 每秒检查一次变化（`sysmonUpdateGOMAXPROCS`）。1.25 之前 `GOMAXPROCS` = 宿主机核数，容器里会让 GC（固定占 25% 的 GOMAXPROCS）和调度抢走远超配额的 CPU——这就是 `uber-go/automaxprocs` 存在的理由（见 gc.md 3.9）。

### 1.2 goroutine 的状态机

| 状态 | 含义 |
| --- | --- |
| `_Gidle` | 刚分配，还没初始化 |
| `_Grunnable` | 在运行队列里等 P |
| `_Grunning` | 正在某个 M 上跑（占着 P） |
| `_Gsyscall` | 在系统调用里（可能已交出 P） |
| `_Gwaiting` | 被阻塞（chan/锁/timer/netpoll），**不在**运行队列里 |
| `_Gdead` | 刚退出或刚从 `gfree` 取出来 |
| `_Gcopystack` | 栈正在被拷贝（`morestack`/`shrinkstack`，见 mem.md 3.2） |
| `_Gpreempted` | 被抢占停下，等待恢复（1.14 引入） |

两组核心转换：

```text
gopark:  _Grunning -> _Gwaiting    主动让出（chan 阻塞、锁等待、Sleep）
goready: _Gwaiting -> _Grunnable   被唤醒，塞回运行队列
```

**阻塞的是 G 不是 M**：M 会去跑别的 G。这是 Go 并发模型的地基（chan.md 2.7、sync.md 2.4）。

`runtime.Stack(buf, true)` 打出来的 dump 里方括号就是状态（实测子进程输出）：

```text
[running]
[chan receive]
[chan send]
[sync.Mutex.Lock]
[sleep]
[select (no cases)]
```

排查线上卡死时，这些字符串就是第一手线索（profile.md 4.1）。

### 1.3 主动让出

```go
runtime.Gosched()    // 让出 P，把当前 G 放回**全局队列**，重新调度
time.Sleep(0)        // 直接返回，不让出（time.md 4.4）
runtime.Goexit()     // 结束当前 goroutine，但**会执行完所有 defer**
```

`Goexit` 实测顺序：`[Goexit 之前, defer 执行了]`——`testing` 的 `t.Fatal` 就是靠它实现的（所以在子 goroutine 里调 `t.Fatal` 只会结束那个 goroutine，见 test.md 面试题 2）。在 main goroutine 里调用会 `fatal error: no goroutines (main called runtime.Goexit) - deadlock!`。

## 二、调度循环

### 2.1 `schedule()` → `findRunnable()` 的查找顺序

`runtime/proc.go` 的 `findRunnable()`，按代码顺序：

1. trace reader / GC worker（有的话优先）；
2. **每 61 次调度检查一次全局队列**：`if pp.schedtick%61 == 0 && !sched.runq.empty()`；
3. 唤醒 finalizer G / cleanup G；
4. **本地运行队列** `runqget(pp)`（含 `runnext` 单槽）；
5. **全局运行队列** `globrunqgetbatch`——一次搬 `len(local)/2` 个到本地；
6. `netpoll(0)` 非阻塞轮询（有等待者时的优化）；
7. **work stealing**：随机顺序遍历其他 P，偷一半（见 2.2）；
8. 再检查全局队列 / GC 工作 / 其他 P 的 timer；
9. `netpoll(阻塞)` 或 `stopm()` 把 M 挂起。

第 ⑥/⑨ 步的 `netpoll` 内部怎么工作（fd 注册、G 挂起、批量唤醒）见 netpoll.md 二。

**`runnext` 是个关键设计**：`newproc` 创建的新 G 优先放进这个单槽，让"刚 `go` 出来的 G"尽快执行（缓存局部性好、常见于"生产者立刻唤醒消费者"）。代价是它会插队到本地队列前面。

### 2.2 work stealing

```text
一个 goroutine 创建 20000 个 G，全部完成用了 44ms（8 核跑满）
```

本地队列容量固定 **256**（`runq [256]guintptr`）。放不下时批量搬一半到全局队列，其他 P 从全局队列取或直接来偷。

偷取规则（`stealWork`/`runqsteal`）：

- **随机起点 + 随机步长**遍历所有 P——避免所有 M 都从 P0 开始偷（惊群）；
- 一次偷对方本地队列的**一半**；
- **最多 4 轮**；只有最后一轮才允许偷 `runnext`，而且要**先等 3µs**，给对方一个自己执行的机会；
- 偷不到 G 就顺手检查对方的 **timer 堆**（time.md 3.1）——所以 timer 也是"可偷的工作"。

### 2.3 为什么要"每 61 次看一次全局队列"

设想两个 G 互相唤醒（ping-pong）：

```text
G1 跑完 -> 唤醒 G2 放进 runnext -> G2 跑完 -> 唤醒 G1 放进 runnext -> ...
```

它们会永远待在同一个 P 的本地队列里，全局队列里的 G **永远等不到**。所以有了那句 `schedtick%61 == 0`。

**61 是质数**，避免和其他周期性行为共振。这是 Go 调度器**唯一的强制公平点**——它整体上优先吞吐（局部性）而不是延迟公平，这和 Linux CFS 的取向完全不同。

### 2.4 spinning M：为什么要自旋

`schedtrace` 里的 `spinningthreads` 指的是"正在找活但还没找到"的 M。设计动机是**避免频繁的线程休眠/唤醒**（futex 系统调用很贵）。

规则：

- 最多允许 `GOMAXPROCS/2` 个 M 处于 spinning；
- 有新 G 就绪时，如果已经有 spinning M，就**不唤醒新的 M**（那个自旋的会捡到）；
- spinning M 找不到活就走 `stopm()` 真正休眠。

**`spinningthreads` 持续大于 0 通常说明任务粒度太细**：G 频繁创建和结束，M 一直在"找活—没找到—又有活"之间打转。

## 三、抢占

### 3.1 协作式 → 异步（1.14）

**1.14 之前是协作式抢占**：编译器在函数入口插入栈检查，`sysmon` 想抢占时把 `stackguard0` 设成 `stackPreempt`，下一次函数调用就会进调度器。

问题：**没有函数调用的紧密循环无法被抢占**。

```go
func main() {
    runtime.GOMAXPROCS(1)
    go func() { for {} }()      // 1.14 之前：整个程序卡死，连 GC 都进不来
    time.Sleep(time.Millisecond)
    fmt.Println("到不了这里")
}
```

**1.14 引入异步抢占**：

```text
sysmon 发现 G 连续运行超过 forcePreemptNS = 10ms（proc.go:6628）
  -> 给对应线程发 SIGURG
  -> 信号处理函数把 G 的 PC 改到 runtime.asyncPreempt
  -> asyncPreempt 保存全部寄存器，进入调度器
```

实测（子进程 `GOMAXPROCS=1` + 无函数调用的死循环）：

```text
GOMAXPROCS=1，死循环 goroutine 存在的情况下，main 仍然被调度到了 ✓
```

代价：

- 需要编译器为**每条指令**都能生成精确的栈图（这是 1.14 那次改动的主要工作量）；
- SIGURG 会让某些系统调用返回 `EINTR`，需要重试逻辑（`internal/poll` 里到处是重试）；
- `GODEBUG=asyncpreemptoff=1` 可以关掉（排查抢占相关的诡异问题时用）。

### 3.2 系统调用与 P 的移交

**进入系统调用**（`entersyscall`）：

- G → `_Gsyscall`，P → `_Psyscall`；
- **P 不立刻交出**——乐观假设：大部分 syscall 很快返回，交出再抢回反而更贵。

**如果 syscall 很慢**：

- sysmon 的 `retake()` 发现 P 处于 `_Psyscall` 且已经过了一个 sysmon 周期（≥20µs）；
- 调 `handoffp()`：把 P 交给别的 M（从 `midle` 取或新建），业务继续跑；
- syscall 返回后（`exitsyscall`）原 M 尝试抢回一个 P；抢不到就把 G 放进全局队列、自己 `stopm()`。

**两类系统调用的区别至关重要**：

| | 处理方式 | 后果 |
| --- | --- | --- |
| **网络 IO** | 被 netpoll 接管，G 直接 `_Gwaiting`，P 立刻去跑别的 | 线程数不涨（netpoll.md） |
| **文件 IO / DNS / cgo** | 真的阻塞线程，走 handoff | **线程数增长** |

实测：

```text
200 个并发阻塞 IO 前后的线程数（threadcreate profile）: 8 -> 73
```

这就是"大量阻塞式 IO 会让 Go 程序线程数暴涨"的机制，上限 `maxmcount = 10000`（超过直接 `fatal error: program exceeds 10000-thread limit`）。对策：限制并发（`semaphore`/worker pool），而不是无脑起 goroutine。

### 3.3 sysmon：不占 P 的后台监控线程

sysmon 是一个**不绑定 P、不参与调度**的特殊 M（`m0` 之外单独创建），死循环干五件事：

```go
delay = 20µs                      // 起步 20µs
if idle > 50 { delay *= 2 }       // 空闲超过 1ms 开始倍增
if delay > 10ms { delay = 10ms }  // 上限 10ms
usleep(delay)
```

1. **`retake()`**：抢占运行超过 10ms 的 G；把卡在 syscall 里的 P 移交出去；
2. **`netpoll(0)`**：如果距上次轮询超过 10ms，主动收一次网络事件；
3. **强制 GC**：距上次 GC 超过 `forcegcperiod = 2 分钟` 就触发一次（gc.md 1.1）；
4. **`sysmonUpdateGOMAXPROCS()`**（1.25+）：每秒检查 cgroup CPU limit 变化；
5. 唤醒 scavenger（gc.md 2.6）、打印 `schedtrace`。

**所有 P 都空闲时 sysmon 会深度睡眠**（`notetsleep`），避免空转耗电——但只要有活跃的 P 就保持 20µs–10ms 的轮询节奏。

### 3.4 `LockOSThread`

```go
runtime.LockOSThread()
defer runtime.UnlockOSThread()
```

绑定后这个 G 只会在这个 OS 线程上跑，别的 G 也不会用这个线程。

用途：

- 调用要求线程本地状态的 C 库（OpenGL、部分 GUI 框架）；
- 需要固定 OS 线程身份的系统调用（Linux 的 `setns`、`gettid`、`unshare`）；
- `runtime.main` 自己就锁在 `m0` 上（信号处理需要）。

代价：

- 该 M 被独占，P 会被 handoff 出去，等价于多占一个线程；
- **忘记 `Unlock` 且 goroutine 退出 → 那个线程会被销毁**（不能复用）；
- 锁定期间该 G 无法迁移，长时间 CPU 密集会拖累这个线程上的其他工作。

## 四、可观测性

### 4.1 `GODEBUG=schedtrace=N`

每 N 毫秒打印一行调度器状态。实测（8 核，200 个 goroutine 密集创建）：

```text
SCHED 0ms:   gomaxprocs=8 idleprocs=6 threads=3 spinningthreads=1 needspinning=0 idlethreads=0 runqueue=0   [ 0 0 0 0 0 0 0 0 ]
SCHED 57ms:  gomaxprocs=8 idleprocs=0 threads=9 spinningthreads=0 needspinning=1 idlethreads=0 runqueue=839 [ 106 115 161 83 3 125 59 204 ]
SCHED 109ms: gomaxprocs=8 idleprocs=0 threads=9 spinningthreads=1 needspinning=1 idlethreads=0 runqueue=966 [ 0 106 44 51 87 0 165 128 ]
SCHED 169ms: gomaxprocs=8 idleprocs=0 threads=9 spinningthreads=0 needspinning=1 idlethreads=0 runqueue=970 [ 6 118 209 87 32 27 117 0 ]
   schedticks=[ 53362 52296 56600 54344 53691 52887 51036 57089 ]     ← 1.26 新增
```

| 字段 | 含义 |
| --- | --- |
| `gomaxprocs` | P 的数量 |
| `idleprocs` | 空闲 P 数（**持续等于 gomaxprocs 说明没活干**） |
| `threads` | 线程总数（含 sysmon、GC worker） |
| `spinningthreads` | 正在自旋找活的 M |
| `needspinning` | 需要有 M 去自旋 |
| `runqueue` | **全局**运行队列长度 |
| `[n n n ...]` | 每个 P 的**本地**队列长度 |
| `schedticks=[...]` | 每个 P 的调度次数（1.26 新增，可以看 P 之间是否均衡） |

**怎么读**：

- `runqueue` 持续很大 + 本地队列也满 → CPU 不够，或者有 G 长时间不让出；
- 本地队列长度**极不均匀** → 任务粒度差异大（work stealing 也救不了长任务）；
- `threads` 远大于 `gomaxprocs` → 大量阻塞式 syscall/cgo（见 3.2）；
- `idleprocs` 长期 > 0 而 `runqueue` 又不空 → 有 P 拿不到 M（线程创建受限）或者刚好在切换。

`GODEBUG=scheddetail=1` 会额外打印每个 P/M/G 的明细（非常长，只在深挖时用）。

### 4.2 其他手段

| 手段 | 看什么 |
| --- | --- |
| `runtime/trace` 的 **Scheduler latency profile** | G 从 runnable 到 running 等了多久（profile.md 5.2） |
| `runtime/trace` 的 **Goroutine analysis** | 每个 G 的时间拆解（执行/网络等待/同步阻塞/syscall/调度等待） |
| `/debug/pprof/goroutine?debug=2` | 每个 G 的状态和阻塞时长（profile.md 4.1） |
| `pprof.Lookup("threadcreate").Count()` | 线程总数 |
| `runtime/metrics` 的 `/sched/goroutines:goroutines`、`/sched/latencies:seconds` | 无 STW 的指标采集 |

## 五、常见面试题

**1. `findRunnable` 的查找顺序是什么？**
① trace/GC worker；② 每 61 次调度看一次全局队列；③ finalizer/cleanup；④ 本地队列（含 `runnext`）；⑤ 全局队列（搬一半过来）；⑥ 非阻塞 netpoll；⑦ work stealing（随机遍历，偷一半，最多 4 轮）；⑧ 再查全局/GC/别人的 timer；⑨ 阻塞 netpoll 或 `stopm()`（见 2.1）。

**2. 为什么每 61 次调度要检查一次全局队列？**
防止两个互相唤醒的 G 借 `runnext` 永远霸占本地队列，让全局队列饿死。61 是质数以避免共振。这是调度器唯一的强制公平点（见 2.3）。

**3. `runnext` 是什么？为什么要它？**
本地队列前面的一个单槽。`go` 出来的新 G 优先放这里，让它尽快执行——利用局部性（生产者刚准备好的数据还在 cache 里）。代价是插队，而且偷取时最后一轮才允许偷它，还要先等 3µs（见 2.1、2.2）。

**4. work stealing 的具体规则？**
随机起点 + 随机步长遍历所有 P（避免惊群），一次偷对方本地队列的**一半**，最多 4 轮，只有最后一轮才偷 `runnext` 且要先等 3µs，偷不到 G 就顺手检查对方的 timer 堆（见 2.2）。

**5. 1.14 的异步抢占解决了什么问题？怎么实现的？**
解决"没有函数调用的紧密循环无法被抢占"——1.14 之前 `GOMAXPROCS=1` 加一个 `for {}` 能让整个程序卡死（连 GC 都进不去）。实现：sysmon 发现 G 跑超过 `forcePreemptNS=10ms`，给线程发 **SIGURG**，信号处理函数把 PC 改到 `asyncPreempt`，保存寄存器后进调度器。前提是编译器能为每条指令生成精确栈图（见 3.1）。

**6. goroutine 阻塞在 channel 上时，OS 线程会阻塞吗？**
不会。`gopark` 把 G 置为 `_Gwaiting` 并让出 P，M 去跑别的 G。只有**真正的阻塞式系统调用**（文件 IO、cgo、DNS）才会占着线程（见 1.2、3.2；网络 IO 的完整路径见 netpoll.md 2.1）。

**7. 系统调用时 P 会被立刻交出去吗？**
不会。`entersyscall` 只把 P 标成 `_Psyscall`，乐观地假设很快返回。sysmon 的 `retake()` 发现超过一个 sysmon 周期（≥20µs）才 `handoffp()` 把 P 交给别的 M。`exitsyscall` 时原 M 抢回 P 失败就把 G 放全局队列、自己休眠（见 3.2）。

**8. 为什么大量文件 IO 会让线程数暴涨？上限是多少？**
文件 IO 是真阻塞的系统调用（netpoll 只管网络），每个阻塞的 M 都会导致 P 被 handoff 给新 M。实测 200 个并发阻塞 IO 让线程从 8 涨到 73。上限 `maxmcount = 10000`，超了 fatal error。对策是限制并发而不是无脑起 goroutine（见 3.2；普通文件为什么进不了 netpoll 见 netpoll.md 1.2）。

**9. sysmon 是什么？它干哪些事？**
一个不绑 P、不参与调度的独立 M。轮询间隔 20µs–10ms（空闲时倍增，全部 P 空闲时深度睡眠）。职责：抢占长跑的 G、移交卡在 syscall 的 P、兜底 netpoll、2 分钟强制 GC、（1.25+）每秒更新 GOMAXPROCS、唤醒 scavenger、打印 schedtrace（见 3.3）。

**10. `GOMAXPROCS` 和线程数是什么关系？**
`GOMAXPROCS` 只决定 P 的数量（并行度）。M 的数量由阻塞情况决定，上限 10000。所以"`GOMAXPROCS=8` 的程序有 73 个线程"是完全正常的（见 1.1、3.2）。

**11. 容器里为什么要设 `GOMAXPROCS`？1.25 之后还需要吗？**
1.25 之前 runtime 不感知 cgroup CPU quota，`GOMAXPROCS` 取宿主核数，导致 GC（固定占 25% GOMAXPROCS）和调度按宿主规模运行，业务被挤爆。**1.25 起 runtime 自动读 cgroup limit 并由 sysmon 每秒跟踪变化**，不再需要 `automaxprocs`（见 1.1）。

**12. `runtime.Gosched()`、`time.Sleep(0)`、`runtime.Goexit()` 的区别？**
`Gosched` 让出 P 并把 G 放回**全局队列**；`Sleep(0)` 直接返回、不让出；`Goexit` 结束当前 goroutine 但**会执行 defer**（`t.Fatal` 的实现基础），在 main goroutine 里调用会 fatal（见 1.3）。

**13. spinning M 是什么？为什么需要它？**
"正在找活但没找到"的 M。目的是避免频繁的线程休眠/唤醒（futex 很贵）：有新 G 就绪时若已有 spinning M 就不唤醒新 M。上限约 `GOMAXPROCS/2`。`spinningthreads` 持续偏高通常意味着任务粒度太细（见 2.4）。

**14. 怎么判断调度是否成了瓶颈？**
`GODEBUG=schedtrace` 看 `runqueue` 和各 P 本地队列（持续积压 = CPU 不足或有 G 不让出）、`threads` 远大于 `gomaxprocs`（阻塞式 syscall 多）、本地队列不均匀（任务粒度差异大）；再配合 trace 的 Scheduler latency profile 看 G 从 runnable 到 running 的等待时间（见 4.1、4.2）。

**15. `LockOSThread` 有什么代价？**
该 M 被独占、P 被 handoff（等价多占一个线程）；忘记 `Unlock` 且 goroutine 退出会导致那个线程被销毁而不是复用；锁定期间 G 无法迁移。只在需要线程本地状态时用（cgo、`setns`、GUI）（见 3.4）。
