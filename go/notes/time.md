# time

> 环境：`go version go1.26.3 darwin/amd64`。源码：`time/{time,sleep,tick,format}.go`、`runtime/time.go`。配套代码：`notes/tm/`（目录名避开 `time`）。
>
> 版本演进（timer 这块 1.23 是分水岭）：
> - **1.9**：`time.Time` 引入**单调时钟**，`time.Since` 从此不受改系统时间影响。
> - **1.14**：runtime 的 timer 改成**每个 P 一个四叉堆**，取消了专门的 `timerproc` goroutine。
> - **1.15**：`time/tzdata` 包，可以把时区数据编进二进制。
> - **1.20**：`time.DateOnly`/`TimeOnly`/`DateTime` 三个常用 layout 常量。
> - **1.23**：**timer 三个行为变化**——未触发的 timer 可被 GC 回收、timer 通道改为同步（无缓冲）、`Stop`/`Reset` 之后不再需要手工 drain。回退开关 `GODEBUG=asynctimerchan=1`（1.27 可能移除）。

## 一、基础

### 1.1 `time.Time` 的结构

```go
// time/time.go:140
type Time struct {
    wall uint64      // 1 bit hasMonotonic | 33 bit 秒 | 30 bit 纳秒
    ext  int64       // hasMonotonic=1 时存单调时钟读数，否则存完整秒数
    loc  *Location   // nil 表示 UTC
}
// Sizeof(time.Time) = 24
```

三个要点：

- **一个 Time 可能同时带两个时钟读数**：墙上时间（wall clock）+ 单调时钟（monotonic）；
- `loc == nil` 就是 UTC（所有 UTC 时间都用 nil，从不用 `&utcLoc`）；
- 33 位秒字段以 1885 年为基准，能表示到 2157 年；超范围就退化成只有 wall。

**含 `*Location` 指针意味着含 `time.Time` 的 struct 一定落在 scan span**，GC 每轮都要扫它（见 4.6、mem.md 1.5）。

### 1.2 单调时钟

```text
time.Now()        2026-08-28 01:01:15.280825 +0900 JST m=+0.000162504
                                                       ~~~~~~~~~~~~~~~ 单调时钟读数
t.Round(0)        2026-08-28 01:01:15.280825 +0900 JST     ← m= 不见了
```

会**剥掉单调时钟**的操作：`Round`/`Truncate`/`UTC`/`Local`/`In`/`AddDate`。

```go
start := time.Now(); ...; time.Since(start)   // ✓ 单调时钟，不受 NTP 校时/手动改表影响
t2.Round(0).Sub(t1.Round(0))                  // ✗ 退化成墙上时间相减
```

铁律：**测耗时一律 `time.Since(start)`**；跨进程/跨机器的时间差只能用墙上时间（且要接受时钟漂移）。

### 1.3 `Duration`

```go
type Duration int64   // 纳秒，所以上限约 292 年（2562047h47m16s）
```

两个高频陷阱：

```go
n := 5
n * time.Second                 // ✗ 编译错误：int 不能和 Duration 相乘
time.Duration(n) * time.Second  // ✓
5 * time.Second                 // ✓ 无类型常量会自动转换

sec := 3
time.Duration(sec)                 // ✗ 3ns（不是 3 秒！）
time.Duration(sec) * time.Second   // ✓ 3s
```

第二个尤其致命：从配置里读到 `timeout: 30`（秒），写成 `time.Duration(cfg.Timeout)` 就变成了 30 纳秒，超时立刻触发。

### 1.4 格式化

Go 不用 `%Y-%m-%d`，而是用**参考时间**：

```text
Mon Jan 2 15:04:05 MST 2006     ← 记：1月2日3点4分5秒2006年，时区 -0700
```

```go
time.RFC3339        // "2006-01-02T15:04:05Z07:00"
time.RFC3339Nano
time.DateOnly       // "2006-01-02"        1.20+
time.TimeOnly       // "15:04:05"          1.20+
time.DateTime       // "2006-01-02 15:04:05" 1.20+
"2006-01-02 15:04:05.000"    // 毫秒用 .000（补零）或 .999（不补零）
```

`Parse` 对补零**严格匹配**：layout 写 `01` 就要求两位，`2026-8-28` 得用 layout `2006-1-2`。

性能上 `AppendFormat` 比 `Format` 省一次分配：

```text
BenchmarkFormatRFC3339-8    60.28 ns/op   24 B/op   1 allocs/op
BenchmarkAppendFormat-8     39.90 ns/op    0 B/op   0 allocs/op
BenchmarkParseRFC3339-8     36.48 ns/op    0 B/op   0 allocs/op
BenchmarkUnixVsFormat-8      1.12 ns/op    0 B/op   0 allocs/op   ← 能用时间戳就别格式化
```

### 1.5 时区

```go
sh, err := time.LoadLocation("Asia/Shanghai")   // 读 /usr/share/zoneinfo 或 $ZONEINFO
utc.In(sh)                                       // 同一时刻，换个表示
utc.Unix() == utc.In(sh).Unix()                  // true —— Unix 时间戳与时区无关
```

三条实践规则：

1. **存储和传输一律 UTC 或 Unix 时间戳**；
2. 只在**展示层**转成用户时区；
3. **容器镜像（scratch/alpine）里 `LoadLocation` 会失败**——要么装 tzdata 包，要么 `import _ "time/tzdata"`（1.15+，约 450KB 编进二进制）。

`time.Local` 取决于 `TZ` 环境变量和 `/etc/localtime`，这意味着**同一份代码在不同机器上 `time.Now()` 的展示结果不同**——又一个"只在展示层碰时区"的理由。

## 二、Timer / Ticker

### 2.1 Timer 与 `Stop` 的返回值

```go
t := time.NewTimer(10 * time.Millisecond)
<-t.C     // 收到触发时刻
```

`Stop()` 返回值的四种情形（**实测，Go 1.26**）：

| 情形 | 返回值 |
| --- | --- |
| 还没到期就 Stop | `true`（成功阻止） |
| 再 Stop 一次 | `false` |
| 已经收过值再 Stop | `false` |
| **时间已过但没人接收** | **`true`** ← 1.23 之前是 `false` |

最后一行是 1.23 最容易踩的坑。实测对比：

```text
Go 1.26 默认：
  时间已过但没收过 Stop() = true，len(t4.C) = 0
  ⚠️ 此时再写 <-t4.C 会永久阻塞

GODEBUG=asynctimerchan=1（模拟 1.22）：
  时间已过但没收过 Stop() = false，len(t.C) = 1
  → 值已经在缓冲通道里躺着，必须 drain 才能复用
```

**结论：老代码里那段 `if !t.Stop() { <-t.C }` 在 1.23+ 是 bug**，会在"时间已过但没接收"这种情况下永久阻塞。

### 2.2 Go 1.23 的三个变化

`time/sleep.go` 的 `NewTimer` 文档原文把三件事说得很清楚：

**① 未触发的 timer 现在能被 GC 回收**

> Before Go 1.23, the garbage collector did not recover timers that had not yet expired or been stopped, so code often immediately deferred t.Stop after calling NewTimer... As of Go 1.23, the garbage collector can recover unreferenced timers.

所以 `defer t.Stop()` 不再是"帮 GC 一把"，只用来表达"我不想它响了"。

**② 通道从有缓冲（cap=1）变成同步（cap=0）**

```text
len(t.C)=0 cap(t.C)=0
```

（源码里其实还是 `make(chan Time, 1)`，但 `syncTimer(c)` 让 runtime 把它当同步通道处理，`cap` 也报 0。）

**③ 因此 `Stop`/`Reset` 之后不需要 drain**

> as of Go 1.23, any receive from t.C after Stop has returned is guaranteed to block rather than receive a stale time value from before the Stop

新代码直接 `t.Reset(d)` 就行。要兼容 1.22 及更早的库，drain 逻辑还得留着（而且必须写成 `select { case <-t.C: default: }` 才不会阻塞）。

### 2.3 Ticker

```go
tk := time.NewTicker(5 * time.Millisecond)
defer tk.Stop()      // 必须！
for range 3 { <-tk.C }
```

```text
tick 1 at 6ms
tick 2 at 11ms
tick 3 at 16ms
```

三个语义要点：

- 通道容量 1，**来不及消费的 tick 被丢弃**（不会堆积、不会补发）。所以 Ticker 保证的是"不早于周期"，不是"每周期一次"；
- **Ticker 必须 `Stop`**——1.23 的 GC 改进**不覆盖 Ticker**（它自己在 runtime 里注册并不断重新装填）。文档原文：*the underlying Ticker is not recovered by the garbage collector*；
- `time.Tick(d)` 拿不到 Ticker 对象，**永远无法 Stop**，只能用于"活到进程结束"的场景。

### 2.4 `AfterFunc`

```go
t := time.AfterFunc(5*time.Millisecond, func() { /* 在新 goroutine 里执行 */ })
t.Stop()
```

- 回调在**新 goroutine** 里跑，不是调用者的 goroutine；
- `AfterFunc` 返回的 Timer 的 `C` 字段是 **nil**，不能接收；
- `Stop()` 返回 `false` **不代表回调已完成**——它可能正在跑，要同步得自己配合（文档明确说 *Stop does not wait for f to complete*）；
- 用途：超时后触发动作（取消 context、关连接），比 `select + timer` 省一个 goroutine。`context.WithTimeout` 内部就是 `AfterFunc`。

## 三、runtime 侧的实现

### 3.1 数据结构

```go
// runtime/time.go:131
type timers struct {
    mu          mutex          // per-P，但别的 P 会来偷，所以还是要锁
    heap        []timerWhen    // 四叉小顶堆，按 when 排序
    len         atomic.Uint32
    zombies     atomic.Int32   // 已标记删除但还在堆里的数量
    minWhenHeap atomic.Int64   // heap[0].when，让 wakeTime 不用加锁就能判断
    ...
}
```

### 3.2 演进

| 版本 | 实现 |
| --- | --- |
| ≤1.9 | 一个**全局** timer 堆 + 一个专门的 `timerproc` goroutine，锁竞争严重 |
| 1.10–1.13 | 分成 **64 个全局桶**，缓解锁竞争 |
| **1.14+** | **每个 P 一个 timer 堆**，由调度器在 `schedule()` 里顺便检查；**没有 timer goroutine 了** |
| 1.23 | timer 可被 GC 回收；通道语义改为同步 |

触发路径：`schedule()` → `checkTimers()` → 执行到期 timer 的 `f`（`sendTime` 往通道发送，或 `goroutineReady` 唤醒 `Sleep` 的 goroutine）。`sysmon` 也会检查最近的 `when`，必要时唤醒空闲 P。netpoll 的超时同样走这套——`SetDeadline` 就是在 `pollDesc` 上挂一个 runtime timer（netpoll.md 4.1）。

所以"**大量 timer 会不会成为瓶颈**"的答案是：分散在各 P 上比全局堆好很多，但单个 P 上百万 timer 的插入/删除仍是 O(log n) 且要抢那把 `mu`。做百万连接的心跳时，通常改成**时间轮**或按秒分桶的方案，而不是给每个连接一个 timer。

### 3.3 取时间的成本

```text
BenchmarkTimeNow-8            84.32 ns/op    ← 两次时钟读取（wall + monotonic）
BenchmarkTimeSince-8          40.09 ns/op    ← 只读单调时钟，便宜一半
BenchmarkTimeNowUnixNano-8    83.37 ns/op
```

`time.Now()` 在 darwin 上要走 `vDSO`/`gettimeofday`，约 84ns——**在每秒百万次的热路径上是能被测出来的开销**。常见对策：

- 只需要"过了多久" → `time.Since`（省一次墙上时间读取）；
- 日志/指标打时间戳 → 用一个后台 goroutine 每毫秒更新一个 `atomic.Int64` 缓存（牺牲精度换性能，很多高性能日志库这么干）。

## 四、常见陷阱

### 4.1 `time.After` 在循环里

```go
for {
    select {
    case v := <-ch:
    case <-time.After(time.Minute):   // ✗ 每次循环新建一个 timer
    }
}
```

- **1.23 之前**：每个 timer 挂在 runtime 里直到到期，循环快的话瞬间几万个 timer，内存和 CPU 都吃紧（经典的"`time.After` 泄漏"）；
- **1.23 之后**：GC 能回收，泄漏问题基本消失，**但每次循环仍要新建 timer 并插堆**。

实测：

```text
BenchmarkSelectAfter-8         347.2 ns/op   248 B/op   3 allocs/op
BenchmarkSelectReuseTimer-8    191.3 ns/op     0 B/op   0 allocs/op
BenchmarkSelectNoTimer-8         5.68 ns/op    0 B/op   0 allocs/op
```

正确写法：

```go
t := time.NewTimer(time.Minute)
defer t.Stop()
for {
    t.Reset(time.Minute)     // 1.23+ 不需要先 drain
    select {
    case v := <-ch:
    case <-t.C:
    }
}
```

### 4.2 Ticker 忘记 Stop

```text
goroutine 数没变（1 -> 1）：Ticker 不开 goroutine
但它在 runtime 的 timer 堆里注册着，且不断重新装填 -> 永远不会被回收
```

这是**真泄漏**，且 `pprof` 的 goroutine profile 里看不出来（因为不涉及 goroutine）。表现是内存缓慢增长 + GC 时间变长。用 `runtime/metrics` 也看不到 timer 数量，只能靠 code review 和"`NewTicker` 后面一定跟 `defer Stop`"的纪律。

### 4.3 `time.Time` 的比较

```go
t1 == t2         // ✗ struct 比较，连 loc 指针和单调时钟都比
t1.Equal(t2)     // ✓ 只比时刻
```

```text
t1 == t1.Round(0)  ->  false   ← 只是剥掉了单调时钟
utc == utc.In(cst) ->  false   ← 同一时刻，不同时区表示
utc.Equal(...)     ->  true
```

推论：**不要把 `time.Time` 当 map key**（同一时刻的不同表示是不同的 key）。要当 key 就用 `t.UnixNano()` 或 `t.UTC().Format(time.RFC3339Nano)`。

比较性能：`Equal`/`Before` 约 3.2ns，`UnixNano` 直接比较 0.69ns——热路径上存 int64 更划算。

### 4.4 Sleep 的精度

```text
Sleep(1ns)      实际 63µs    （63000x）
Sleep(1µs)      实际 69µs    （69x）
Sleep(100µs)    实际 174µs   （1.7x）
Sleep(1ms)      实际 1.2ms   （1.2x）
```

**保证"至少睡 d"，绝不保证"恰好睡 d"**：受 OS 定时器精度（通常 1ms 量级）+ 调度延迟影响。亚毫秒级定时在通用 OS 上不可靠。

另外 `time.Sleep(0)` **不等于** `runtime.Gosched()`：前者直接返回，不让出 P。

### 4.5 `time.Time` 在 struct 里

```go
type record struct {
    ID        int          // 8
    CreatedAt time.Time    // 24
}                          // 32 字节
```

- 含 `*Location` 指针 → **落在 scan span，GC 每轮都要扫**。海量记录时用 `int64` 存 Unix 纳秒更省（8 字节 + noscan，见 mem.md 1.5）；
- JSON 序列化默认输出 RFC3339Nano，反序列化也**只认这个格式**；自定义格式要实现 `MarshalJSON`/`UnmarshalJSON`（见 json.md 2.1）；
- 零值判断用 `t.IsZero()`，不要写 `t == time.Time{}`。

### 4.6 `Round` / `Truncate` 的方向

```go
d := 90 * time.Minute
d.Round(time.Hour)      // 2h0m0s   ← 四舍五入
d.Truncate(time.Hour)   // 1h0m0s   ← 向下取整
```

`Time.Truncate` 是**相对于零时刻**取整（不是相对于当天零点），跨时区时容易出意外。想要"当天零点"用 `time.Date(y, m, d, 0, 0, 0, 0, loc)`。

### 4.7 定时任务不要用 Ticker 做"整点触发"

`NewTicker(time.Hour)` 从**创建时刻**开始每小时一次，不是"每个整点"。而且它会因为 tick 丢弃而漂移。整点触发的正确做法：

```go
for {
    next := time.Now().Truncate(time.Hour).Add(time.Hour)
    time.Sleep(time.Until(next))
    doWork()
}
```

或者用 `robfig/cron` 这类库（支持 crontab 表达式和时区）。

## 五、常见面试题

**1. `time.Time` 里为什么有两个时钟？**
`wall` 存墙上时间（可能被 NTP 校正、被人手动改），`ext` 在 `hasMonotonic=1` 时存单调时钟读数（进程启动以来的纳秒）。`Sub`/`Since` 优先用单调时钟，所以测耗时不受改系统时间影响（1.9 引入）（见 1.1、1.2）。

**2. 哪些操作会丢掉单调时钟？后果是什么？**
`Round`/`Truncate`/`UTC`/`Local`/`In`/`AddDate`。丢掉之后 `Sub` 退化成墙上时间相减，可能得到负数或跳变。所以"存下来的时间"和"用来测耗时的时间"要分开处理（见 1.2）。

**3. Go 1.23 对 timer 做了哪三个改动？**
① 未触发的 timer 可被 GC 回收（`defer t.Stop()` 不再是为了防泄漏）；② timer 通道从有缓冲改为同步；③ `Stop`/`Reset` 之后不需要手工 drain。回退开关 `GODEBUG=asynctimerchan=1`（见 2.2）。

**4. 1.23 之后 `if !t.Stop() { <-t.C }` 为什么是 bug？**
实测：时间已过但没人接收时，1.23+ 的 `Stop()` 返回 **true**（1.22 返回 false），且通道里没有值。老代码在这种情况下会执行 `<-t.C` 并**永久阻塞**（见 2.1）。

**5. `time.After` 会泄漏吗？**
1.23 之前会：timer 挂在 runtime 里直到到期，循环中大量创建就是泄漏。1.23 之后 GC 能回收，但每次循环仍要新建 timer 并插堆（实测 347ns/248B/3allocs vs 复用 Timer 的 191ns/0B）。正确做法是复用一个 Timer + `Reset`（见 4.1）。

**6. Timer 和 Ticker 谁必须 Stop？**
**Ticker 必须**（1.23 的 GC 改进不覆盖它，它会不断重新装填）。Timer 现在不 Stop 也能被回收。`time.Tick` 拿不到对象，永远无法 Stop（见 2.3、4.2）。

**7. Ticker 的 tick 会堆积吗？**
不会。通道容量 1，消费不及时的 tick 直接丢弃、不补发。所以它保证"不早于周期"，不保证"每周期一次"。要精确计数得自己记时间（见 2.3）。

**8. runtime 是怎么管理 timer 的？有专门的 goroutine 吗？**
1.14 起**每个 P 一个四叉小顶堆**，调度器在 `schedule()` 里顺便 `checkTimers()`，**没有专门的 timer goroutine**（1.9 之前有 `timerproc`，1.10-1.13 是 64 个全局桶）。`sysmon` 也会检查最近的 when 并唤醒空闲 P（见 3.1、3.2）。

**9. 百万连接的心跳超时怎么做？**
不要给每个连接一个 timer——单 P 的 timer 堆插删是 O(log n) 且抢同一把锁。用**时间轮**或"按秒分桶 + 一个 Ticker 扫桶"，把 n 个 timer 压成 1 个（见 3.2）。

**10. 为什么不能用 `==` 比较 `time.Time`？**
`==` 是 struct 比较，会比 `loc` 指针和单调时钟位。同一时刻的 UTC 和 CST 表示、`t` 和 `t.Round(0)` 都不相等。用 `Equal`/`Before`/`After`。也别把 `time.Time` 当 map key（见 4.3）。

**11. `time.Duration(n)` 的坑是什么？**
`time.Duration` 的单位是纳秒。从配置读到 `30`（秒）写成 `time.Duration(30)` 就是 30ns。必须写 `time.Duration(n) * time.Second`。另外 `int * time.Second` 编译不过，只有无类型常量可以（见 1.3）。

**12. `time.Sleep(1ms)` 真的睡 1ms 吗？**
不是。实测 `Sleep(1ns)` 实际 63µs、`Sleep(1ms)` 实际 1.2ms。只保证"至少 d"，受 OS 定时器精度和调度延迟影响。亚毫秒定时在通用 OS 上不可靠（见 4.4）。

**13. `time.Now()` 有多贵？高频取时间怎么优化？**
实测 84ns（两次时钟读取）。`time.Since` 只读单调时钟，40ns。极端场景用后台 goroutine 每毫秒刷新一个 `atomic.Int64` 缓存，用精度换性能（见 3.3）。

**14. 容器里 `time.LoadLocation("Asia/Shanghai")` 报错怎么办？**
scratch/alpine 镜像没有 `/usr/share/zoneinfo`。两条路：装 tzdata 包，或 `import _ "time/tzdata"`（1.15+，把时区库编进二进制，约 450KB）（见 1.5）。

**15. Go 的时间格式化为什么用 "2006-01-02"？**
用一个具体的**参考时间** `Mon Jan 2 15:04:05 MST 2006` 来表达布局，各字段对应 1/2/3/4/5/6/-7，比 `%Y-%m-%d` 更直观（不用记字母含义），代价是要记住这个魔法时间。注意 `Parse` 对补零严格匹配（见 1.4）。
