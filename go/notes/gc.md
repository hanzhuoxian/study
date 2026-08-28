# GC

> 环境：`go version go1.26.3`。源码：`runtime/{mgc,mgcpacer,mgcmark,mgcsweep,mbarrier,mwbbuf,mgcscavenge}.go`。配套代码：`notes/gc/`。
>
> 版本演进（面试爱问，且网上资料版本混乱）：
> - **1.3**：mark & sweep，STW 标记，停顿几百 ms。
> - **1.5**：并发标记清扫（CMS 风格三色标记），插入写屏障，停顿降到十毫秒级。
> - **1.8**：**混合写屏障**（Dijkstra 插入 + Yuasa 删除），消掉 mark termination 里的**栈重扫**，停顿进入亚毫秒级。这是 GC 停顿"与堆大小解耦"的分水岭。
> - **1.12**：清扫改成并发 + 惰性；`mheap` 的 treap 改造。
> - **1.14**：**异步抢占**，解决"紧密循环里没有函数调用导致 GC 无法进入 STW"的老问题。
> - **1.18**：pacer 重写（`mgcpacer.go` 的 PI 控制器形态），把栈和全局的扫描量也纳入计算。
> - **1.19**：**`GOMEMLIMIT`** 软内存上限；GC 指南（`doc/gc-guide.html`）进入官方文档。
> - **1.24**：`runtime.AddCleanup` 取代 `SetFinalizer`。
> - **1.25/1.26**：**Green Tea 标记算法**，1.26 起**默认开启**（`internal/buildcfg/exp.go` 的 baseline 里 `GreenTeaGC: true`）。

## 一、基础使用

### 1.1 GC 什么时候触发

`runtime/mgc.go` 里 `gcTrigger` 有三种：

| 触发源 | 条件 | 说明 |
| --- | --- | --- |
| `gcTriggerHeap` | 堆分配量达到 heap goal | 绝大多数 GC 都是这么来的 |
| `gcTriggerTime` | 距上次 GC 超过 `forcegcperiod = 2 分钟` | sysmon 里发起，保证空闲程序也会回收 |
| `gcTriggerCycle` | `runtime.GC()` / `debug.FreeOSMemory()` | 手动，`gctrace` 里带 `(forced)` |

```text
分配 40MB 垃圾后 GC 次数: 0 -> 11      # 堆触发
runtime.GC() 之后:        11 -> 12     # 手动触发
```

### 1.2 heap goal：GOGC 到底控制什么

```text
heap goal ≈ live × (1 + GOGC/100)
```

`live` 是**上一轮 GC 结束时的存活量**。实测（`notes/gc/`）：

```text
GOGC=50   live=20MB goal=30MB （1.5x）
GOGC=100  live=20MB goal=41MB （2.0x）
GOGC=400  live=20MB goal=102MB（5.0x）
```

两个细节：

- **有下限**：`defaultHeapMinimum = 4MB`（GOGC=100 时，`mgcpacer.go:57`）。小堆程序不会因为 live 只有几十 KB 就疯狂 GC。
- **实际公式比这个复杂**：1.18 之后的 pacer 还把**栈扫描量**和**全局变量扫描量**算进 goal，并用一个反馈控制器预测"什么时候开始标记才刚好在 goal 处结束"（`gcControllerState.revise()`）。`GODEBUG=gcpacertrace=1` 能看到每轮的决策。

### 1.3 MemStats：能看什么，代价是什么

```go
var s runtime.MemStats
runtime.ReadMemStats(&s)   // ⚠️ 会 STW
```

```text
HeapAlloc      20.2 MB      当前存活对象字节数（= 累计分配 - 累计释放）
HeapSys        27.6 MB      从 OS 拿到的堆虚拟内存
HeapIdle       7.1 MB       空闲 span（其中 HeapReleased 已还给 OS）
HeapReleased   6.9 MB       已经还给 OS 的字节数
NextGC         40.6 MB      下一轮的 heap goal
NumGC          16           已完成轮数
PauseTotalNs   826µs        所有 STW 累计
GCCPUFraction  9.5%         启动至今 GC 占的 CPU
```

`ReadMemStats` **会 STW**，指标采集里每秒调几次就是自己给自己加停顿。现代做法是 `runtime/metrics`。

### 1.4 runtime/metrics：现在应该用这个

```go
samples := []metrics.Sample{{Name: "/gc/heap/live:bytes"}, {Name: "/gc/pauses:seconds"}}
metrics.Read(samples)   // 不 STW
```

本机 go1.26.3 共导出 **112** 个指标，GC 相关最常用的：

| 指标 | 含义 |
| --- | --- |
| `/gc/heap/live:bytes` | 上轮 GC 后的存活量 |
| `/gc/heap/goal:bytes` | 当前 heap goal |
| `/gc/heap/allocs:bytes` `/gc/heap/frees:bytes` | 累计分配/释放 |
| `/gc/cycles/total:gc-cycles` | GC 总轮数 |
| `/gc/pauses:seconds` | **STW 停顿直方图**（可以直接算 p99） |
| `/cpu/classes/gc/total:cpu-seconds` | GC 消耗的 CPU 秒数 |
| `/memory/classes/heap/objects:bytes` | 堆里对象占用 |
| `/memory/classes/heap/released:bytes` | 已归还 OS |
| `/memory/classes/total:bytes` | runtime 管理的总内存（≈ RSS 的上界） |
| `/gc/gomemlimit:bytes` `/gc/gogc:percent` | 当前生效的两个旋钮（1.23+） |

实测本机某轮：`/gc/pauses:seconds p50=16µs p99=262µs`。

## 二、底层原理

### 2.1 三色抽象与实际实现

概念上对象分三色：**白**（未访问，待回收）、**灰**（已访问、子对象未扫描）、**黑**（自己和子对象都扫过）。

Go 的实现里**没有"灰色位"**：颜色 = mark bit + 是否在工作队列里。

- 白 = mark bit 未置位；
- 灰 = mark bit 已置位 **且** 对象在 `gcWork` 工作队列（workbuf）里；
- 黑 = mark bit 已置位 **且** 已从队列取出扫描完。

mark bit 存在 span 的 `gcmarkBits` 位图里（不在对象头上，所以对象没有额外的 header 开销）。

### 2.2 混合写屏障：为什么 Go 的停顿和堆大小无关

`runtime/mbarrier.go` 开头的伪代码就是全部答案：

```go
writePointer(slot, ptr):
    shade(*slot)               // Yuasa 删除屏障：把被覆盖的旧值染灰
    if current stack is grey:
        shade(ptr)             // Dijkstra 插入屏障：把新写入的值染灰
    *slot = ptr
```

三条注释解释了为什么这两个 shade 缺一不可：

1. `shade(*slot)` 防止 mutator 把"堆里指向某对象的唯一指针"搬到自己栈上来隐藏它；
2. `shade(ptr)` 防止 mutator 把"栈上指向某对象的唯一指针"塞进一个**已经是黑色**的堆对象里来隐藏它；
3. **一旦某个 goroutine 的栈被扫黑，第 2 条就不再必要**——因为扫完之后它栈上只指向已染色的对象，藏不了东西了。

这第 3 点正是 1.8 那个提案（`17503-eliminate-rescan.md`）的核心收益：

- **1.8 之前**（只有插入屏障）：栈上的写没有屏障（太贵），所以并发标记结束后必须在 STW 里**重新扫描所有 goroutine 栈**，停顿 ∝ goroutine 数量和栈大小。
- **1.8 之后**（混合屏障）：栈扫过一次就永久变黑，**mark termination 不需要重扫栈**，停顿变成一个只做收尾工作的固定小开销。

实现细节：屏障不是直接调函数，而是把两个指针塞进 P 本地的 `wbBuf`（`runtime/mwbbuf.go`），满了才批量 flush 到工作队列——避免每次指针写都动全局结构。汇编入口是 `gcWriteBarrier`，编译器在**可能指向堆的指针写**处插入（栈上局部变量的写、写入 `int` 之类无指针类型都不插）。

**写屏障只在 `_GCmark`/`_GCmarktermination` 阶段开启**（`setGCPhase` 里 `writeBarrier.enabled = ...`），`_GCoff` 时是一个全局 bool 判断，代价接近零。

### 2.3 一轮 GC 的五个阶段

`runtime/mgc.go` 顶部注释给的顺序，配上函数名：

| 阶段 | 是否 STW | 干什么 |
| --- | --- | --- |
| ① sweep termination | **STW** | 所有 P 到安全点；把上一轮没扫完的 span 扫完（`gcStart` → `stopTheWorldWithSema(stwGCSweepTerm)`） |
| ② mark 准备 | 仍在 STW 内 | `setGCPhase(_GCmark)`：**开写屏障**、开 assist、入队 root 任务。必须 STW，因为要保证所有 P 都看到写屏障已开 |
| ③ 并发标记 | 否 | Start the world。后台 worker + assist 扫 root（栈/全局/off-heap）、排空灰色队列。分布式终止检测（`gcMarkDone`） |
| ④ mark termination | **STW** | `setGCPhase(_GCmarktermination)`：关 worker/assist、flush 所有 mcache、算下一轮 goal |
| ⑤ 并发清扫 | 否 | `setGCPhase(_GCoff)`：**关写屏障**、Start the world。后台 `bgsweep` + 分配时惰性清扫 |

实测（`notes/gc/`，堆 20~100MB）：

```text
最近 16 段 STW：平均 52µs，最大 269µs
/gc/pauses:seconds p50=16µs p99=262µs
```

注意 `PauseNs` 记录的是**每一段** STW（一轮 GC 贡献两段）。

### 2.4 谁在干标记的活：25% + assist

```go
// runtime/mgcpacer.go:39
gcBackgroundUtilization = 0.25   // 后台标记固定占 25% 的 GOMAXPROCS
gcGoalUtilization       = gcBackgroundUtilization
```

三种 mark worker（`runtime/mgc.go:276`）：

| 模式 | 行为 |
| --- | --- |
| `gcMarkWorkerDedicatedMode` | 独占一个 P 全程标记，不被抢占 |
| `gcMarkWorkerFractionalMode` | `GOMAXPROCS × 0.25` 不是整数时，用一个"分数 worker"补齐（跑一小段就让出） |
| `gcMarkWorkerIdleMode` | P 空闲时顺手帮忙标记，随时让位给用户 goroutine |

例：`GOMAXPROCS=8` → 2 个 dedicated worker；`GOMAXPROCS=6` → 1 个 dedicated + 1 个 fractional（0.5）。

**mark assist**：25% 不够时（mutator 分配太猛），在 `mallocgc` 里给分配者记"欠账"，欠得多就**就地干标记工作**（`gcAssistAlloc`）：

```go
gcCreditSlack    = 2000      // 本地攒够 2000 字节扫描信用才同步到全局
gcAssistTimeSlack= 5000      // 5µs
gcOverAssistWork = 64 << 10  // 每次 assist 多干 64KB，预付未来的分配
```

这是"**分配越快，单次分配越慢**"的根源，也是 GC 的自然背压机制。`gctrace` 里 `0.20+0.071/0.26/0.031+0.037 ms cpu` 这段的含义是：

```text
STW清扫终止 + assist/后台/空闲 + STW标记终止
0.20        + 0.071/0.26/0.031 + 0.037
```

第一个数字大 = mutator 被罚得多 = 该调大 GOGC 或降低分配速率。

### 2.5 Green Tea 标记算法（1.26 默认开启）

`runtime/mgcmark_greenteagc.go` 的注释讲得很直白：

> The core idea behind Green Tea is simple: achieve better locality during mark/scan by delaying scanning so that we can accumulate objects to scan within the same span, then scan the objects that have accumulated on the span all together.

传统标记是**逐对象**的：从队列取一个对象 → 读它的类型元数据 → 扫它的指针字段。小对象场景下每次都是一次随机访存 + 一次元数据访问，cache 命中极差。

Green Tea 改成**逐 span 批量**：

- 每个 span 内联两套位图（`spanInlineMarkBits`）：`marks`（第一次看到指针就置位）和 `scans`（已扫描）。
- 第一次看到指向某 span 的指针时，置 mark 位并**把整个 span 入队**（FIFO，而不是 workbuf 的 LIFO——注释说 FIFO 更利于攒批）。
- 出队时算 `marks` 和 `scans` 的并集/交集：并集写回 `scans`，交集决定这轮要扫哪些对象，从而**保持精确性**。
- 好处：同一 span 内的对象被一起扫，摊薄元数据访问、给硬件预取创造条件，后续还能按 size class 特化甚至上 SIMD（注释里明确说"not yet completed"）。

验证当前是否启用：

```bash
go env GOEXPERIMENT              # 空 = 全是 baseline 默认值
# baseline 里 GreenTeaGC: true（internal/buildcfg/exp.go:83）
GOEXPERIMENT=nogreenteagc go build ...   # 显式关掉做 A/B
```

### 2.6 清扫（sweep）与内存归还

**清扫**只是把 span 里"没被标记的对象"标记为可分配，**不写内存、不 memset**。它是并发的：`bgsweep` goroutine + 分配时按需清扫（`mcentral.cacheSpan` 里）。所以"GC 结束"不等于"内存可用"。

**归还 OS** 是另一套机制（scavenger，`runtime/mgcscavenge.go`）：

```text
持有 100MB:      HeapAlloc=100MB HeapIdle=7MB   HeapReleased=4MB   HeapSys=108MB
丢弃后立即:      HeapAlloc=0MB   HeapIdle=107MB HeapReleased=4MB
FreeOSMemory 后: HeapAlloc=0MB   HeapIdle=107MB HeapReleased=107MB
```

- 空 span 先进 `HeapIdle` 由 runtime 自己留着复用（下次分配不用再进内核）；
- 后台 scavenger 按 pacing 慢慢 `madvise` 归还（`GODEBUG=scavtrace=1` 可观测）；
- `debug.FreeOSMemory()` 立即强制归还（同时强制一轮 GC），代价是下次分配要重新触发缺页；
- Linux/darwin 默认 `MADV_DONTNEED`，`GODEBUG=madvdontneed=0` 改成 `MADV_FREE`（RSS 降得更慢，但重用更快）。

**结论：`RSS` 不降 ≠ 内存泄漏**（见 3.5 的排查顺序）。

### 2.7 关键源码索引

| 位置 | 内容 |
| --- | --- |
| `runtime/mgc.go:25-82` | 整轮 GC 的官方流程注释 |
| `runtime/mgc.go:251,257` | `_GCoff/_GCmark/_GCmarktermination` 与 `setGCPhase`（写屏障开关） |
| `runtime/mgc.go:276-295` | 三种 mark worker 模式 |
| `runtime/mgc.go:733,1015,1344` | `gcStart`、`gcMarkDone`、`gcMarkTermination` |
| `runtime/mgc.go:1750` | `gcBgMarkWorker` |
| `runtime/mbarrier.go:21-60` | 混合写屏障伪代码 + 正确性论证 |
| `runtime/mwbbuf.go` | 写屏障的 P 本地缓冲 |
| `runtime/mgcpacer.go:16-76` | `gcBackgroundUtilization=0.25`、`gcCreditSlack`、`gcOverAssistWork`、`defaultHeapMinimum` |
| `runtime/mgcmark.go` | `gcAssistAlloc`、root 扫描、`scanobject` |
| `runtime/mgcmark_greenteagc.go` | Green Tea 算法与 `spanInlineMarkBits` |
| `runtime/mgcscavenge.go` | 归还 OS 的 pacing |
| `runtime/extern.go:104-135` | `gctrace`/`gcpacertrace`/`gcstoptheworld`/`madvdontneed` 的官方说明 |

## 三、调优与陷阱

### 3.1 GOGC

```bash
GOGC=100    # 默认：新分配量达到存活量的 100% 就 GC
GOGC=400    # 内存换 CPU：GC 次数降到 1/4 左右
GOGC=off    # 完全关闭（只有 GOMEMLIMIT 还能触发）
```

实测（分配 30MB 存活 + 30MB 垃圾）：

```text
GODEBUG=gctrace=1            -> NumGC=6  HeapAlloc=38MB HeapSys=44MB
GOGC=off GODEBUG=gctrace=1   -> NumGC=0  HeapAlloc=60MB HeapSys=64MB
```

调优原则：**先看 `/cpu/classes/gc/total:cpu-seconds` 占总 CPU 的比例**。低于 5% 基本没必要动；超过 20% 且内存有余量，调大 GOGC 是最简单有效的一步。

### 3.2 GOMEMLIMIT（1.19+）

```go
debug.SetMemoryLimit(4 << 30)   // 4GiB
debug.SetMemoryLimit(-1)        // 只读当前值
```

实测：`live=20MB` 时把 limit 设成 40MB，`goal` 从 41MB 被压到 32MB（pacer 主动提高 GC 频率）。

要点：

- **软上限**：runtime 会更频繁 GC 去逼近它，但**不会为它自杀**。真超了就是继续超（或者被 OS OOM-kill）。
- **统计范围**是 Go runtime 管理的所有内存（堆 + 栈 + 元数据 + 一部分 off-heap），**不含 cgo `malloc`、手动 `mmap`、可执行文件本身**。容器里设置时要留出这部分余量（常见做法是容器 limit 的 80~90%）。
- **死亡螺旋防护**：GC CPU 占比被限制在 **50%**，宁可超过内存上限也不把 CPU 全烧在 GC 上。
- 最实用的组合：**`GOGC=off` + `GOMEMLIMIT=<容器上限的 85%>`**——只在快撞上限时才 GC，把内存吃满换取最少的 GC 次数。前提是你的程序**峰值内存可控**，否则会退化成持续 GC。

### 3.3 ballast（历史技巧，现在别用）

```go
var ballast = make([]byte, 10<<30)   // 10GB 虚拟内存，从不触碰
runtime.KeepAlive(ballast)
```

原理：把 `live` 撑大，`goal = live × 2` 也跟着变大，GC 次数骤降；又因为从没写过，OS 不会真的给物理页，RSS 不涨。

1.19 之后请用 `GOGC` + `GOMEMLIMIT` 代替：语义明确、可观测、不依赖 OS 的懒分配行为，也不会在容器里被 `MADV_DONTNEED`/统计口径差异坑到。

### 3.4 减少 GC 压力的真正手段

调旋钮之外，能从根上减压的（按性价比排序）：

1. **减少分配次数**（不是减少字节数）：`pprof -alloc_objects` 找热点。
2. **预分配容量**：`make([]T, 0, n)`、`strings.Builder.Grow(n)`，避免多次扩容拷贝。
3. **对象复用**：`sync.Pool`（见 pool.md），注意只对"构造成本高 + 本来会逃逸"的对象划算。
4. **避免不必要的逃逸**：`-gcflags=-m` 看逃逸原因；接口装箱、闭包捕获、返回局部指针是三大来源（见 func.md 2.4、mem.md）。
5. **减少指针**：`[]int` 比 `[]*int` 扫得快得多；无指针对象所在的 span 根本不需要扫描（`spanClass.noscan`）。把 `map[string]*T` 换成 `map[string]int`（索引到一个大 slice）能显著缩短标记时间。
6. **控制 goroutine 数量**：每个 goroutine 的栈都是 root，栈越多、越深，root 扫描越久。

### 3.5 陷阱：RSS 不降就以为泄漏

排查顺序（都用 `runtime/metrics`）：

| 现象 | 结论 |
| --- | --- |
| `/memory/classes/heap/objects:bytes` 持续涨 | **真泄漏**：有对象一直可达 |
| objects 平稳、`heap/released` 很低 | 只是没还给 OS，看 scavenger 节奏，正常 |
| `/memory/classes/heap/free:bytes` 很大 | 峰值过后的空 span / 碎片 |
| `total:bytes` 减去 heap 部分很大 | 栈（goroutine 太多）、元数据，或者 cgo |

真泄漏的常见来源：goroutine 泄漏（连带它的栈和闭包）、全局 map/slice 只增不删、大 slice 截小后仍持有整个底层数组（slice.md 3.3）、`time.Ticker` 忘 `Stop`、`context` 没 cancel（context.md 3.x）。

### 3.6 陷阱：SetFinalizer

```go
r := &resource{}
runtime.SetFinalizer(r, func(r *resource) { r.Close() })
```

问题一箩筐：

- **至少两轮 GC 才能真正释放**：第一轮发现不可达 → 排队 finalizer → 对象**复活**（finalizer 持有它）；第二轮才回收。
- **不保证执行**：进程退出前没轮到就没了。
- **不保证顺序**；循环引用中的对象 finalizer **永不执行**（1.24 之前）。
- finalizer 里让对象重新可达 → 对象**永生**。
- 有 finalizer 的对象**不能是零大小**，也不能是"分配块内部的指针"。

### 3.7 用 `runtime.AddCleanup` 代替（1.24+）

```go
h := &fileHandle{fd: fd}
runtime.AddCleanup(h, func(fd int) { syscall.Close(fd) }, h.fd)  // 注意 arg 不能是 h 本身
```

相比 `SetFinalizer`：

| | SetFinalizer | AddCleanup |
| --- | --- | --- |
| 对象复活 | 会（两轮 GC） | **不会**（一轮即可回收） |
| 同一对象多个钩子 | 不行 | **可以** |
| 循环引用 | 可能永不执行 | **正常执行** |
| 可取消 | `SetFinalizer(obj, nil)` | 返回 `Cleanup`，`Stop()` |
| 参数 | 对象本身 | 只传"底层资源"，**传对象本身会 panic** |

注意 `AddCleanup` 依然**不保证时机**。它的正确定位是"资源忘关时的兜底 + 告警"，正常路径永远是 `defer Close()`。

### 3.8 陷阱：用 runtime.GC() "优化"

`runtime.GC()` 会**阻塞到本轮 GC 完成**（含两次 STW），在请求路径上调用等于自造停顿。合理场景只有三个：基准测试前清场、测内存占用前稳定 live、进程即将长时间空闲（配合 `debug.FreeOSMemory()`）。

### 3.9 陷阱：`GOMAXPROCS` 与 GC 的联动

后台标记固定吃 25% 的 `GOMAXPROCS`。容器里如果没设 `GOMAXPROCS`（1.25 之前不感知 cgroup quota），`GOMAXPROCS` 会等于**宿主机核数**，于是：

- dedicated worker 数量按宿主核算，远超容器实际能用的 CPU；
- 于是 GC 抢走大部分配额，业务 goroutine 严重饥饿。

**Go 1.25 起 runtime 会读 cgroup CPU limit 自动设置 `GOMAXPROCS`**（并在 limit 变化时动态更新）。1.25 之前的版本用 `uber-go/automaxprocs`，或者显式设 `GOMAXPROCS`。

### 3.10 陷阱：大量小对象 vs 少量大对象

同样 100MB：

- `1000 万个 10 字节对象`：标记要遍历 1000 万次，`HeapObjects` 巨大，GC 时间主要花在这里；
- `100 个 1MB 的 []byte`：`noscan` span，标记几乎不花时间。

所以"减少分配"要看的是 **`-alloc_objects`（次数）而不是 `-alloc_space`（字节）**。这也解释了为什么 `map[string][]byte` 换成"一个大 `[]byte` + 偏移索引"能让 GC 时间断崖式下降（很多 cache 库就是这么做的）。

## 四、常见面试题

**1. Go 用的是什么 GC 算法？**
并发的**三色标记清扫**（mark & sweep），**非分代、非压缩（不移动对象）**、**非整理**。写屏障是 **1.8 引入的混合屏障**（Dijkstra 插入 + Yuasa 删除）。1.26 起标记阶段默认使用 Green Tea 算法（按 span 批量扫描以改善局部性）。

**2. 为什么 Go 的 GC 不分代？**
分代的前提是"大部分对象很快死"，收益来自只扫年轻代。但 Go 有两个特殊点：① **逃逸分析**已经把大量短命对象放在栈上（栈上对象根本不进 GC），堆里剩下的对象存活率反而偏高；② 分代需要 **card table / remembered set** 这类额外的写屏障状态，而 Go 的写屏障已经为并发标记服务了，再叠一层代价大。官方多次讨论过分代（有原型），结论是收益不足以抵消复杂度。参考同一个原因：Go 也不做**压缩/整理**，因为没有 moving GC 就不需要处理 unsafe.Pointer/cgo 指针失效的问题。

**3. 三色标记是怎么实现的？灰色存在哪？**
没有独立的"灰色位"。颜色 = mark bit + 是否在工作队列里：白 = mark 未置位；灰 = mark 已置位且在 `gcWork` 队列中；黑 = mark 已置位且已扫完出队。mark bit 在 span 的位图里，对象本身没有 header（见 2.1）。

**4. 什么是强/弱三色不变性？写屏障保证的是哪个？**
强三色不变性：黑对象**不允许**指向白对象。弱三色不变性：黑对象可以指向白对象，但**该白对象必须仍有一条从灰对象出发的可达路径**。插入屏障（Dijkstra）保证强不变性，删除屏障（Yuasa）保证弱不变性。Go 的混合屏障同时做两件事：`shade(*slot)` 是删除屏障，`shade(ptr)` 是插入屏障（见 2.2）。

**5. 混合写屏障解决了什么问题？为什么 1.8 之后停顿骤降？**
1.8 之前只有插入屏障，而**栈上的写不能加屏障**（太贵），所以标记结束时必须 STW 重扫所有 goroutine 栈，停顿 ∝ goroutine 数 × 栈深度。混合屏障让"栈扫过一次就永久变黑"，mark termination 不再需要重扫栈，停顿变成固定的小开销，与堆大小基本无关（见 2.2）。

**6. 写屏障的伪代码是什么？两个 shade 各防什么？**
```go
shade(*slot); if 当前栈是灰的 { shade(ptr) }; *slot = ptr
```
`shade(*slot)` 防止把"堆里的唯一指针"搬到栈上藏起来；`shade(ptr)` 防止把"栈上的唯一指针"塞进黑对象里藏起来。栈变黑之后第二个就不必要了（`mbarrier.go` 的三条注释）。

**7. 一轮 GC 有几次 STW？分别在干什么？**
两次：**sweep termination**（让所有 P 到安全点、扫完上轮遗留 span、开写屏障并入队 root 任务）和 **mark termination**（关 worker/assist、flush mcache、计算下轮 goal）。两段都在百微秒级，本机实测 p99 = 262µs（见 2.3）。

**8. GC 的 25% CPU 是怎么来的？assist 是什么？**
`gcBackgroundUtilization = 0.25`：后台标记固定占 `GOMAXPROCS` 的 25%，由 dedicated / fractional / idle 三种 mark worker 实现。不够时启用 **mark assist**：在 `mallocgc` 里给分配者记欠账，欠得多就就地干标记活，所以"分配越猛，单次分配越慢"——这是 GC 的背压机制（见 2.4）。

**9. GOGC 到底控制什么？heap goal 怎么算？**
`goal ≈ live × (1 + GOGC/100)`，`live` 是上轮 GC 结束时的存活量，另有 4MB 下限。1.18 之后 pacer 还把栈和全局的扫描量纳入计算，并用反馈控制器决定何时**开始**标记，使标记刚好在 goal 处结束（见 1.2）。

**10. GOMEMLIMIT 和 GOGC 有什么区别？怎么配？**
GOGC 控制"相对增长比例"，GOMEMLIMIT 控制"绝对内存天花板"。前者在 live 抖动时内存也跟着抖，后者能挡住峰值。它是**软限制**，且只统计 Go runtime 管理的内存（不含 cgo/mmap）。GC CPU 上限 50% 用来防死亡螺旋。生产常用组合：`GOGC=off` + `GOMEMLIMIT = 容器上限 × 85%`（见 3.2）。

**11. 为什么 GC 跑完 RSS 没降？**
GC 只回收对象、清扫只把 span 标为可分配；空 span 先留在 `HeapIdle` 供复用；归还 OS 是 scavenger 的活，按 pacing 慢慢 `madvise`。想立刻还可以 `debug.FreeOSMemory()`。所以要判断泄漏必须看 `/memory/classes/heap/objects:bytes` 而不是 RSS（见 2.6、3.5）。

**12. `runtime.ReadMemStats` 和 `runtime/metrics` 有什么区别？**
`ReadMemStats` **会 STW**，字段固定且部分语义含糊；`metrics.Read` 不 STW，本机导出 112 个指标，还带**直方图**（如 `/gc/pauses:seconds`，可直接算 p99）。监控采集一律用后者（见 1.3、1.4）。

**13. `SetFinalizer` 有什么坑？`AddCleanup` 好在哪？**
finalizer 会让对象复活，至少两轮 GC 才释放；不保证执行、不保证顺序；循环引用可能永不执行；一个对象只能挂一个。`AddCleanup`（1.24+）不复活对象、可挂多个、循环引用也执行、可 `Stop()`，但 arg 不能是对象本身（会 panic）。两者都不保证时机，关资源仍应 `defer Close()`（见 3.6、3.7）。

**14. Green Tea GC 是什么？（1.25/1.26 新内容）**
标记阶段的重写：不再逐对象扫描，而是**攒到同一个 span 再批量扫**。span 内联 `marks`/`scans` 两套位图，第一次看到指向该 span 的指针就置 mark 并把 span 入队（FIFO），出队时用两套位图的交并集算出该扫哪些对象，从而在保持精确的前提下大幅改善 cache 局部性。1.26 起默认开启，可用 `GOEXPERIMENT=nogreenteagc` 关掉做 A/B（见 2.5）。

**15. 如何定位 GC 引起的性能问题？**
① `/cpu/classes/gc/total:cpu-seconds` 看 GC 占了多少 CPU；② `GODEBUG=gctrace=1` 看每轮的 assist 时间占比（第一个 cpu 数字大 = mutator 被罚）和 `#->#->#MB` 的增长；③ `pprof -alloc_objects` 找分配次数热点；④ `runtime/trace` 看 GC 与业务 goroutine 的时间轴重叠。调优优先级：先减分配次数和指针数量，再动 GOGC/GOMEMLIMIT（见 3.4）。

**16. 容器里为什么 GC 会把业务 CPU 挤爆？**
1.25 之前 runtime 不感知 cgroup CPU quota，`GOMAXPROCS` 取宿主核数，于是"25% 的 GOMAXPROCS"按宿主核算，实际远超容器配额，业务 goroutine 严重饥饿。1.25 起 runtime 自动按 cgroup limit 设置并动态更新 `GOMAXPROCS`；旧版本用 `automaxprocs` 或显式设置（见 3.9）。

**17. 为什么"减少分配"要看次数而不是字节数？**
标记的成本 ∝ **对象数量和指针数量**，不是字节数。1000 万个 10 字节对象比 100 个 1MB 的 `[]byte` 慢几个数量级，因为后者是 `noscan` span，压根不用扫。所以优化方向是"合并小对象、减少指针、用索引替代指针"（见 3.10）。

**18. GC 会移动对象吗？为什么？**
不会。Go 是非移动（non-moving）GC。原因：`unsafe.Pointer`、cgo 传出去的指针、以及大量假设"指针稳定"的代码都会因移动而失效；不移动也意味着不需要 read barrier。代价是**内存碎片**——Go 靠 size class 分配器把碎片控制在可接受范围（见 mem.md）。栈是唯一例外：栈增长时会**拷贝整个栈并调整指针**（见 mem.md 的栈增长部分）。
