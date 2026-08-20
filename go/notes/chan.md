# Channel

> 环境：`go version go1.26.3`。底层结构与流程以该版本源码为准：`runtime/chan.go`（hchan、发送/接收/关闭）、`runtime/select.go`（selectgo）、`cmd/compile/internal/walk/select.go`（select 的编译期降级）、`runtime/runtime2.go`（sudog）。不同版本行内逻辑略有调整（如 Go 1.23 起 timer channel 改为由 runtime timer 直接驱动），但核心模型多年未变。

## 一、基础使用

### 1.1 声明与初始化

```go
var ch chan int              // 仅声明，值为 nil，读写都永久阻塞（见 3.3）
ch = make(chan int)          // 无缓冲（同步）channel
ch = make(chan int, 3)       // 有缓冲（异步）channel，容量 3

done := make(chan struct{})  // 只做信号通知，元素类型用空结构体，不占内存
```

- `chan T` 变量本身就是**一个指针**（指向运行时的 `hchan`），`unsafe.Sizeof(ch) == 8`（64 位）。因此 channel 传参、赋值只拷贝这个指针，多个变量共享同一个 channel，天然是"引用语义"。
- **必须 `make` 才能用**：`nil` channel 不是"空 channel"，它是永久阻塞的 channel。

### 1.2 发送、接收、关闭

```go
ch <- 42        // 发送
v := <-ch       // 接收，丢弃 ok
<-ch            // 接收并丢弃值（只取"事件发生了"这个信号）
close(ch)       // 关闭
```

三个操作的阻塞语义：

| 操作      | 无缓冲                       | 有缓冲                     | nil            | 已关闭                     |
| --------- | ---------------------------- | -------------------------- | -------------- | -------------------------- |
| 发送      | 阻塞到有接收者**同时**就绪   | 缓冲区满则阻塞             | **永久阻塞**   | **panic**                  |
| 接收      | 阻塞到有发送者**同时**就绪   | 缓冲区空则阻塞             | **永久阻塞**   | 立即返回（先读完缓冲区）   |
| close     | 正常                         | 正常                       | **panic**      | **panic**                  |

只需记住一句：**"nil 全阻塞，close 后写和再 close 都 panic，读永远安全"**。

### 1.3 comma-ok 接收与关闭语义

```go
v, ok := <-ch
// ok == true ：正常收到一个值
// ok == false：channel 已关闭 **且** 缓冲区已被读空，此时 v 是元素类型的零值
```

关键点：**`close` 不丢数据**。关闭只是打上"不会再有新值"的标记，缓冲区里已有的值仍然可以被完整读出，读完之后才开始返回 `(零值, false)`：

```go
ch := make(chan int, 2)
ch <- 1
close(ch)
fmt.Println(<-ch)      // 1        —— 缓冲区里的数据照常读出
v, ok := <-ch          // 0, false —— 读空之后才是"关闭"语义
```

因此 `ok == false` 的含义精确地是"**关闭且已读空**"，而不是"关闭了"。

### 1.4 range 遍历 channel

```go
for v := range ch {   // 等价于 for { v, ok := <-ch; if !ok { break }; ... }
    fmt.Println(v)
}
```

- `range` 会一直读到 channel **被关闭且读空**才退出，只 `close` 不写入也能正常退出循环。
- **没有人 `close`，`range` 就永远不会结束**（见 3.7），这是 goroutine 泄漏的最常见来源之一。

### 1.5 单向 channel

```go
func producer(out chan<- int) { // 只能发送
    for i := 0; i < 3; i++ {
        out <- i
    }
    close(out)                  // 只发送的一端才有资格 close
}

func consumer(in <-chan int) {  // 只能接收
    for v := range in {
        fmt.Println(v)
    }
    // close(in)                // 编译错误：cannot close receive-only channel
}

ch := make(chan int, 3)
go producer(ch)                 // 双向 channel 可隐式转成单向
consumer(ch)
```

- 双向 → 单向是**隐式允许**的赋值转换；单向 → 双向**不允许**，单向 channel 之间也不能互转。
- 单向类型是**编译期的接口约束**，运行时仍是同一个 `hchan`，零开销。它的价值是把"谁负责发送、谁负责关闭"写进函数签名里（对应 3.5 的原则）。

### 1.6 len 与 cap

```go
ch := make(chan int, 5)
ch <- 1
fmt.Println(len(ch), cap(ch)) // 1 5  —— 缓冲区中现有元素数 / 缓冲区容量

u := make(chan int)
fmt.Println(len(u), cap(u))   // 0 0  —— 无缓冲 channel 两者恒为 0

var n chan int
fmt.Println(len(n), cap(n))   // 0 0  —— nil channel 不 panic，返回 0
```

`len`/`cap` 只反映缓冲区，**不包含阻塞在 `sendq`/`recvq` 上的等待者**，而且读出来的瞬间就可能过期，只适合做监控和调试，不能用来做控制流判断（见 3.10）。

### 1.7 select 多路复用

```go
select {
case v := <-ch1:
    fmt.Println("from ch1:", v)
case ch2 <- 42:
    fmt.Println("sent to ch2")
case <-time.After(time.Second):
    fmt.Println("timeout")
}
```

- **多个 case 同时就绪时，随机选一个**（伪随机，实测均匀：两路都就绪时 10 万次约 49914 : 50086，见 2.8），因此 case 的书写顺序**不构成优先级**。
- 所有 case 都没就绪时：有 `default` 立即走 `default`（select 变成非阻塞），没有 `default` 就阻塞等待第一个就绪的 case。
- `case` 里的 channel 表达式和发送值在进入等待**之前**就会被求值一次——这就是 `time.After` 每次执行 select 都新建一个 timer 的原因（见 3.8）。

三种常用写法：

```go
// ① 非阻塞尝试
select {
case v := <-ch:
    use(v)
default:
    // 没有数据，不等
}

// ② 超时控制
select {
case v := <-ch:
    use(v)
case <-time.After(500 * time.Millisecond):
    return errTimeout
}

// ③ 可取消的循环（推荐用 ctx.Done()）
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case job := <-jobs:
        handle(job)
    }
}
```

### 1.8 常用并发模式

**① 信号通知 / 广播退出**——利用"关闭后所有接收者立即返回"实现一对多广播：

```go
done := make(chan struct{})
for i := 0; i < 3; i++ {
    go func(id int) {
        <-done                 // 三个 goroutine 都阻塞在这里
        fmt.Println(id, "exit")
    }(i)
}
close(done)                    // 一次 close 唤醒全部（见 2.6）
```

**② worker pool（任务分发）**：

```go
jobs := make(chan int, 100)
results := make(chan int, 100)

for w := 0; w < 4; w++ {       // 4 个 worker 抢同一个 jobs channel
    go func() {
        for j := range jobs {
            results <- j * 2
        }
    }()
}

for i := 1; i <= 10; i++ {
    jobs <- i
}
close(jobs)                    // 关闭后所有 worker 的 range 自然退出
```

**③ fan-in（扇入合并）**：

```go
func merge(chans ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup
    for _, c := range chans {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for v := range c {
                out <- v
            }
        }(c)
    }
    go func() { wg.Wait(); close(out) }() // 全部上游结束后才 close（见 3.6）
    return out
}
```

**④ pipeline（流水线）**：每个阶段"接收上游、处理、发给下游、关闭下游"。

```go
func gen(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            out <- n
        }
    }()
    return out
}

func sq(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            out <- n * n
        }
    }()
    return out
}

for v := range sq(gen(1, 2, 3)) { // 1 4 9
    fmt.Println(v)
}
```

**⑤ 带缓冲 channel 当信号量做并发限流**：

```go
sem := make(chan struct{}, 10)  // 最多 10 个并发
for _, task := range tasks {
    sem <- struct{}{}           // 获取令牌，满了就阻塞
    go func(t Task) {
        defer func() { <-sem }() // 释放令牌
        t.Do()
    }(task)
}
```

**⑥ 一次性结果返回（future）**：容量 1 的 channel 让生产者不必等消费者，避免生产者泄漏。

```go
func doAsync() <-chan Result {
    ch := make(chan Result, 1) // 关键：缓冲 1，即使调用方放弃接收也不会泄漏
    go func() { ch <- compute() }()
    return ch
}
```

## 二、底层原理

### 2.1 hchan：channel 的运行时结构

`chan.go:34`：

```go
type hchan struct {
    qcount   uint           // 缓冲区中现有元素个数 → len(ch)
    dataqsiz uint           // 环形缓冲区容量        → cap(ch)，创建后不可变
    buf      unsafe.Pointer // 指向 dataqsiz 个元素的数组（环形队列）
    elemsize uint16         // 元素大小（因此元素类型大小上限是 64KB）
    closed   uint32         // 关闭标志，0/1，只能 0→1
    timer    *timer         // 非 nil 表示这是 timer/ticker channel（见 2.11）
    elemtype *_type         // 元素类型，拷贝时需要它做类型化 memmove + 写屏障
    sendx    uint           // 环形缓冲区的写下标
    recvx    uint           // 环形缓冲区的读下标
    recvq    waitq          // 阻塞在接收上的 goroutine 队列（FIFO）
    sendq    waitq          // 阻塞在发送上的 goroutine 队列（FIFO）
    bubble   *synctestBubble// testing/synctest 支持
    lock     mutex          // 保护上面所有字段
}
```

四个要点：

1. **channel 不是无锁结构**。`hchan.lock` 是一把运行时互斥锁，所有可能修改状态的操作都在锁内完成。只有"非阻塞且注定失败"的判断走无锁快路径（见 2.3）。
2. **buf 是环形队列**，`sendx`/`recvx` 各自单调前进并回绕，所以 FIFO 顺序有严格保证：**channel 里的数据顺序 = 发送完成的顺序**。
3. **`dataqsiz == 0` 就是无缓冲 channel**，此时 `buf` 不指向真实数据（只用作 race detector 的同步地址），数据永远在两个 goroutine 的栈之间直接拷贝（见 2.5）。
4. **`elemsize` 是 `uint16`**，`makechan` 里显式检查 `elem.Size_ >= 1<<16` 就 `throw`，所以元素类型大小不能达到 64KB。大对象应该传指针。

创建（`makechan`，`chan.go:75`）按元素是否含指针分三种分配策略：

```go
switch {
case mem == 0:              // 无缓冲，或元素是空结构体 → 只分配 hchan
    c = (*hchan)(mallocgc(hchanSize, nil, true))
    c.buf = c.raceaddr()
case !elem.Pointers():      // 元素不含指针 → hchan 和 buf 一次分配，连续内存
    c = (*hchan)(mallocgc(hchanSize+mem, nil, true))
    c.buf = add(unsafe.Pointer(c), hchanSize)
default:                    // 元素含指针 → buf 单独分配，带类型信息交给 GC 扫描
    c = new(hchan)
    c.buf = mallocgc(mem, elem, true)
}
```

`make(chan T, n)` 一定发生**堆分配**（channel 要跨 goroutine 共享，逃逸分析必然判定逃逸），所以高频创建 channel 是有成本的。

### 2.2 sudog 与 waitq：谁在等，等什么

阻塞的 goroutine 不是直接挂在 channel 上，而是包装成 `sudog`（`runtime2.go:406`）：

```go
type sudog struct {
    g        *g                 // 哪个 goroutine 在等
    next, prev *sudog           // waitq 双向链表
    elem     maybeTraceablePtr  // 数据槽：发送者指向待发送值，接收者指向接收目标
    isSelect bool               // 是否因 select 而入队（唤醒时要 CAS 抢占，见 2.8）
    success  bool               // 唤醒原因：true=真的收发到了数据，false=channel 被关闭
    waitlink *sudog             // 同一个 G 在 select 中挂在多个 channel 上时串成链
    c        maybeTraceableChan // 挂在哪个 channel 上
    ...
}
```

`sudog` 从 P 本地的缓存池 `acquireSudog`/`releaseSudog` 复用，避免每次阻塞都分配。

`waitq` 就是一条 `sudog` 双向链表（`chan.go:56`），`enqueue` 尾插、`dequeue` 头取（`chan.go:872`/`886`），因此**等待者按 FIFO 顺序被唤醒**。`dequeue` 里有一段专门处理 select 的竞争：

```go
if sgp.isSelect {
    if !sgp.g.selectDone.CompareAndSwap(0, 1) {
        continue    // 这个 G 已经被别的 case 唤醒了，跳过，取下一个
    }
}
```

因为一个执行 select 的 G 会同时挂在多个 channel 的队列上，谁先 CAS 成功谁才有权唤醒它，失败的一方必须继续找下一个等待者。

`hchan` 的注释还给出了两条关键不变量：

- `sendq` 和 `recvq` **至少有一个是空的**（唯一例外：无缓冲 channel 上同一个 G 通过 select 同时收发）。
- 有缓冲时：`qcount > 0` ⇒ `recvq` 为空；`qcount < dataqsiz` ⇒ `sendq` 为空。

翻译成人话：**缓冲区里还有数据，就不可能有人在等着收；缓冲区还没满，就不可能有人在等着发。** 这条不变量是后面所有路径判断的基础。

### 2.3 发送：chansend 的四条路径

`c <- v` 由编译器翻译成 `chansend1` → `chansend(c, ep, true, pc)`（`chan.go:160`/`176`）。

```
chansend
 ├─ c == nil            → 阻塞版：gopark 永久休眠（"chan send (nil chan)"）
 │                        非阻塞版（select+default）：直接 return false
 ├─ 无锁快路径          → !block && !closed && full(c) → return false
 │
 ├─ lock(&c.lock)
 ├─ closed != 0         → unlock + panic("send on closed channel")
 ├─ ① recvq 有等待者    → send()：数据直接交给接收者，goready 唤醒它，返回
 ├─ ② 缓冲区没满        → typedmemmove 拷进 buf[sendx]，sendx++ 回绕，qcount++
 ├─ ③ !block            → unlock + return false
 └─ ④ 阻塞              → 造 sudog 挂进 sendq，gopark 休眠
                          被唤醒后：success==false 说明是被 close 唤醒的 → panic
```

**无锁快路径**（`chan.go:203`）只服务于 `select` 的 `default` 分支：

```go
if !block && c.closed == 0 && full(c) {
    return false
}
```

它读两个字（`c.closed` 和 `full(c)`）都不加锁。源码里有很长一段注释论证这个无锁读的正确性：因为**已关闭的 channel 不可能从"可发送"变回"不可发送"**，所以即使两次读之间发生了 close，也一定存在一个"既未关闭又不可发送"的时刻，返回 `false` 就是在报告那个时刻的状态——这对非阻塞操作是合法答案。

**路径 ① 是最重要的优化**：有等待的接收者时，值**绕过缓冲区**直接送到接收者（`send`，`chan.go:318`）。结合 2.2 的不变量，这条路对有缓冲 channel 只在缓冲区为空时才可能命中。

**路径 ④ 的唤醒后检查**（`chan.go:295`）解释了一个常见困惑："发送时另一个 goroutine 把 channel 关了会怎样"：

```go
closed := !mysg.success
...
if closed {
    panic(plainError("send on closed channel"))
}
```

阻塞中的发送者被 `close` 唤醒时 `success == false`，于是**在唤醒点 panic**。也就是说 `send on closed channel` 有两个触发点：进入时就已关闭，以及阻塞期间被关闭。

另外注意 `gopark` 之后有一行 `KeepAlive(ep)`：`sudog.elem` 指向发送者的栈对象，但 sudog 不是 GC 的栈扫描根，必须显式保活到接收者拷走数据为止。

### 2.4 接收：chanrecv 的四条路径

`<-c` 编译成 `chanrecv1`（丢弃 ok）或 `chanrecv2`（comma-ok），最终都进 `chanrecv`（`chan.go:524`）。

```
chanrecv
 ├─ c == nil            → 阻塞版：gopark 永久休眠（"chan receive (nil chan)"）
 ├─ 无锁快路径          → !block && empty(c) 且确认未关闭 → return (false,false)
 │                        若已关闭且确实为空 → 清零 *ep，return (true,false)
 ├─ lock(&c.lock)
 ├─ closed != 0 且 qcount == 0
 │                      → 清零 *ep，return (true, false)   ← "零值 + ok=false"
 ├─ ① sendq 有等待者    → recv()：见下，返回 (true,true)
 ├─ ② 缓冲区有数据      → 从 buf[recvx] 拷出，清零原槽位，recvx++ 回绕，qcount--
 ├─ ③ !block            → unlock + return (false,false)
 └─ ④ 阻塞              → 造 sudog 挂进 recvq，gopark 休眠
                          唤醒后返回 (true, mysg.success)
```

两处细节值得注意：

**a) 关闭检查在 sendq 检查之前**，而且只在 `qcount == 0` 时才走"零值 + false"。若 channel 已关闭但缓冲区仍有数据，代码会落到路径 ②，先把缓冲区读干——这就是 1.3 中"close 不丢数据"的实现依据。源码注释写得很直白：`// The channel has been closed, but the channel's buffer have data.`

**b) 无锁快路径的读序不能反**（`chan.go:548`）：必须先判 `empty` 再判 `closed`，且两次都用原子读。反过来会出现这种错误：channel 原本"未关闭且非空"→ 被关闭 → 被读空，乱序的两次读可能得出"未关闭且为空"，从而让一个本该返回 `(true,false)` 的接收错误地返回"未就绪"。

**路径 ① 的 `recv`（`chan.go:702`）分两种情况**，这是最容易被忽略的一段逻辑：

```go
if c.dataqsiz == 0 {
    // 无缓冲：直接从发送者栈拷到接收者栈
    recvDirect(c.elemtype, sg, ep)
} else {
    // 有缓冲且缓冲区必然是满的（否则发送者不会阻塞）
    qp := chanbuf(c, c.recvx)
    typedmemmove(c.elemtype, ep, qp)        // 队头 → 接收者
    typedmemmove(c.elemtype, qp, sg.elem.get()) // 阻塞发送者的值 → 同一个槽位（队尾）
    c.recvx++
    if c.recvx == c.dataqsiz { c.recvx = 0 }
    c.sendx = c.recvx                       // 满队列时队头 == 队尾
}
```

有缓冲 channel 满了、且有发送者在排队时，接收者拿走的是**队头的旧数据**，阻塞发送者的值被放到**队尾**——由于队列是满的，队头和队尾恰好是同一个槽位，一次拷贝就完成了"出队 + 入队"。这保证了即使中间经历了阻塞，**FIFO 顺序也不会被打乱**。

### 2.5 sendDirect / recvDirect：绕过缓冲区的直接拷贝

无缓冲 channel（以及缓冲区为空时命中等待者的有缓冲 channel）的数据传递是**从一个 goroutine 的栈直接拷到另一个 goroutine 的栈**（`chan.go:392`/`405`）：

```go
func sendDirect(t *_type, sg *sudog, src unsafe.Pointer) {
    dst := sg.elem.get()
    typeBitsBulkBarrier(t, uintptr(dst), uintptr(src), t.Size_)
    memmove(dst, src, t.Size_)
}
```

这是 Go 运行时里**唯一一处"一个运行中的 goroutine 直接写另一个运行中 goroutine 的栈"**的地方，因此不能用普通的 `typedmemmove`：GC 假设栈写入只由 goroutine 自己完成，`bulkBarrierPreWrite` 只处理堆目标，对栈目标无效。所以这里手工调用 `typeBitsBulkBarrier` + `memmove`，把写屏障补上。

由此得到一个性能结论：

| channel 类型 | 一次传递的拷贝次数 | 说明                                 |
| ------------ | ------------------ | ------------------------------------ |
| 无缓冲       | **1 次**           | 发送者栈 → 接收者栈                  |
| 有缓冲       | **2 次**           | 发送者栈 → buf → 接收者栈            |
| 有缓冲但正好有等待者 | **1 次**   | 命中路径 ①，绕过 buf                 |

拷贝的是**元素的完整值**，不是引用。传 1KB 的 struct 就要拷 1KB（有缓冲则拷两遍），所以大对象走 channel 应该传指针，代价是要自己保证所有权移交后不再触碰原对象。

### 2.6 close：一次性唤醒所有等待者

`closechan`（`chan.go:414`）：

```go
if c == nil    → panic("close of nil channel")
lock
if c.closed != 0 → unlock + panic("close of closed channel")
c.closed = 1
① 清空 recvq：每个等待接收者的 elem 清零、success = false，收集到 glist
② 清空 sendq：每个等待发送者的 elem 置 nil、success = false，收集到 glist
unlock
③ 逐个 goready(glist)
```

三个设计点：

1. **接收者拿零值，发送者 panic**——都靠 `sg.success = false` 这一个标志区分：接收方把它作为 comma-ok 的 `ok` 返回，发送方看到它就 panic（2.3 路径 ④）。
2. **先收集到 `glist`，解锁之后才 `goready`**。`hchan.lock` 的注释明确要求"持有该锁时不要改变其他 G 的状态"，否则可能与栈收缩（stack shrinking）互相死锁。
3. **close 是一对多广播的基础**：一次 `close` 唤醒 `recvq` 上**所有**等待者，而一次发送只唤醒一个。这就是 1.8 ① 和 `context.Done()` 的实现原理——`Done()` 返回的 channel 从不写入，只被 close。

`close` 是**只能 0→1 的单向操作**，channel 无法重新打开，这也是 2.4 无锁快路径推理成立的前提。

### 2.7 阻塞与唤醒：挂起的是 G，不是线程

channel 阻塞用的是 `gopark` / `goready`，不是操作系统级阻塞：

- `gopark` 把当前 G 的状态从 `_Grunning` 改成 `_Gwaiting`，与 M 解绑，M 立刻回到调度循环去跑别的 G。**一个阻塞在 channel 上的 goroutine 不占用任何 OS 线程**，这是"开十万个 goroutine 收发 channel"可行的根本原因。
- `goready` 把 G 改回 `_Grunnable` 并放进运行队列，由调度器择机执行。**唤醒 ≠ 立即执行**。
- 唤醒时机上有个重要优化：`send`/`recv` 里的 `goready(gp, skip+1)` 会倾向于把被唤醒的 G 放到当前 P 的 `runnext` 槽，让它尽快运行，减少交接延迟。

阻塞前会做一次 `gp.parkingOnChan.Store(true)`，并在 `chanparkcommit`（`chan.go:748`）里设置 `gp.activeStackChans = true` 之后才释放 channel 锁。这对配合的是栈收缩：`sudog.elem` 指向 G 的栈，栈被搬动时必须先锁住相关 channel，这两个标志用来标记"这段窗口不安全，别动我的栈"。

`gopark` 的 `waitReason` 决定 panic/`SIGQUIT` 时 goroutine 的状态文字，排查阻塞问题时很有用：

| 状态文字                   | 含义                             |
| -------------------------- | -------------------------------- |
| `chan send` / `chan receive` | 正常阻塞在收发上                 |
| `chan send (nil chan)` / `chan receive (nil chan)` | 操作 nil channel，**永远不会被唤醒** |
| `select`                   | 阻塞在 select 上                 |

看到 `(nil chan)` 就可以直接定位到"channel 忘了 make"或"select 里的 channel 变量被置 nil 后没有其他可用 case"。

### 2.8 select 的实现：编译期降级 + selectgo 三趟扫描

**第一步：编译器先做降级**（`cmd/compile/internal/walk/select.go:33`），大部分 select 根本不会进 `selectgo`：

| case 数量           | 降级为                                        |
| ------------------- | --------------------------------------------- |
| `select {}`（0 个） | `block()` —— 直接永久 gopark（`select.go:103`）|
| 1 个（无 default）  | 直接的 `chansend` / `chanrecv`，退化成普通收发 |
| 1 个 + default      | `selectnbsend` / `selectnbrecv`，即 `block=false` 的收发（`chan.go:784`/`804`） |
| ≥2 个               | 走通用的 `selectgo`                            |

所以"`select` + `default` 做非阻塞收发"这种写法的开销和普通收发几乎一样，没有 selectgo 的排序和入队成本。

**第二步：`selectgo`（`select.go:122`）**。它维护两个下标数组：

- `pollorder`：**随机打乱**的检查顺序（`select.go:167`），用 `cheaprandn` 做原地 Fisher-Yates 洗牌。这是"多个 case 就绪时随机选一个"的实现——语言层面故意不给 case 优先级，防止代码依赖书写顺序造成饥饿。
- `lockorder`：按 **channel 地址（`sortkey()`）排序**的加锁顺序（`select.go:206`，堆排序保证 O(n log n) 和常数栈开销）。多个 channel 必须按全局一致的顺序加锁，否则两个 select 交叉加锁会**死锁**。这是操作系统课上"按固定顺序获取多把锁"的经典手法。

注意 `cas.c == nil` 的 case 在建表时就被**跳过**（不进 pollorder 也不进 lockorder），这正是 3.3 中"把 channel 置 nil 来动态关闭 select 分支"的实现基础。case 总数上限 65536（为了控制栈开销）。

**第三步：三趟扫描**，全程持有所有相关 channel 的锁：

```
sellock(按 lockorder 加锁)

pass 1（select.go:264）—— 按 pollorder 找一个已经就绪的 case
    接收 case：sendq 有等待者 → recv；缓冲区有数据 → 读缓冲；已关闭 → 返回零值
    发送 case：已关闭 → panic；recvq 有等待者 → send；缓冲区没满 → 写缓冲
    找到就跳出去执行对应分支
  ↓ 全都没就绪
有 default → 解锁，返回 -1，走 default
  ↓ 无 default
pass 2（select.go:309）—— 给每个 case 造一个 sudog，按 lockorder 挂进各自的
    sendq/recvq（同一个 G 的多个 sudog 用 waitlink 串成链挂在 gp.waiting 上），
    然后 gopark 休眠
  ↓ 被某个 channel 唤醒
sellock(重新加锁)
pass 3（select.go:360）—— 遍历所有 sudog：命中的那个记为结果，其余的从对应
    channel 队列中 dequeueSudoG 摘除并归还 sudog 池
```

**pass 3 是必须的**：如果不把没中的 sudog 摘掉，它们会永久堆积在冷清的 channel 队列上（源码注释：`otherwise they stack up on quiet channels`），造成内存泄漏和错误唤醒。

由此可以量化 select 的成本：`selectgo` 一次要做**一次洗牌 + 一次堆排序 + 加 n 把锁 + 最坏 n 次入队和 n 次出队**。所以热点循环里的 select 分支数不宜太多，能用普通收发就别套 select。

### 2.9 内存模型：channel 提供的 happens-before 保证

channel 不只是数据搬运，它同时是**同步原语**。Go 内存模型给出四条保证：

1. **一次发送 happens-before 对应的接收完成**。这是最常用的一条：发送前对任何变量的写，接收方在接收后都能看到。
2. **`close` happens-before 因 channel 关闭而返回零值的接收完成**。所以"close 广播 + 接收方读取共享数据"是安全的。
3. **无缓冲 channel 上的接收 happens-before 对应发送完成**。注意方向：接收在前。这条保证了 `ch <- x` 返回时，对方**确实已经取走了**这个值。
4. **容量为 C 的 channel 上第 k 次接收 happens-before 第 k+C 次发送完成**。这条是用 channel 做信号量（1.8 ⑤）的理论依据。

实践含义：

```go
var data []int
done := make(chan struct{})

go func() {
    data = []int{1, 2, 3}   // ① 写共享变量
    close(done)             // ② close
}()

<-done                      // ③ 因关闭而返回
fmt.Println(data)           // ④ 保证能看到 ①，无需额外加锁
```

反过来，第 3 条也解释了 3.11 的陷阱：无缓冲 channel 的发送返回，只保证"对方取走了值"，**不保证对方处理完了**。要等处理完必须再收一个回执。

### 2.10 无缓冲 vs 有缓冲：本质区别

| 维度              | 无缓冲（`dataqsiz == 0`）              | 有缓冲（`dataqsiz > 0`）                 |
| ----------------- | -------------------------------------- | ---------------------------------------- |
| 语义              | **同步交接（rendezvous）**：双方必须同时就绪 | **异步**：缓冲区没满/没空就不阻塞         |
| 数据路径          | 栈 → 栈，1 次拷贝                      | 栈 → buf → 栈，2 次拷贝                  |
| `full()` 判定     | `recvq.first == nil`（没人在等就是"满"） | `qcount == dataqsiz`                     |
| 发送返回时的含义  | 对方**已取走**（内存模型第 3 条）      | 只表示**已放进缓冲区**，对方可能还没看到 |
| 是否天然限流      | 是，生产速度被消费速度完全约束         | 否，缓冲区是一段可积压的空间             |
| 典型用途          | 同步握手、确认、严格的步调对齐         | 解耦生产消费速率、削峰、信号量           |

选型原则：**默认用无缓冲**，因为它的同步语义更强、更容易推理，出问题时立刻阻塞暴露而不是悄悄积压。只有明确需要吸收突发、或需要打破"生产者必须等消费者"的耦合时，才加缓冲，而且容量要有具体依据（例如"就是 worker 数"、"就是并发上限"），不要随手写 100。

### 2.11 timer channel：Go 1.23+ 的特殊 hchan

`time.Timer`/`time.Ticker`/`time.After` 返回的 channel 在 `hchan` 里带一个非 nil 的 `timer` 字段，运行时对它做了特殊处理：

- 它内部**有缓冲**（`dataqsiz > 0`），但 `chanlen`/`chancap`（`chan.go:818`/`835`）对 timer channel **一律返回 0**，对用户伪装成无缓冲。这样运行时可以"撤销"一次已投递但尚未被读取的发送，`Reset`/`Stop` 的语义才干净。实测：

  ```go
  t := time.NewTimer(time.Hour)
  fmt.Println(len(t.C), cap(t.C)) // 0 0，尽管内部实现是有缓冲的
  ```

- `chanrecv`/`empty`/`selectgo` 在遇到 `c.timer != nil` 时会调用 `c.timer.maybeRunChan(c)`，也就是**在读之前顺便推进一下定时器**，让"到点了但 timer goroutine 还没被调度"的窗口尽可能小。
- 因此不要对 timer channel 用 `len`/`cap` 判断"是否已到期"，唯一正确的方式是收/select。

### 2.12 关键源码索引

| 关注点              | 位置                                  |
| ------------------- | ------------------------------------- |
| `hchan` / `waitq`   | `runtime/chan.go:34` / `:56`          |
| `makechan`          | `runtime/chan.go:75`                  |
| `full` / `empty`    | `runtime/chan.go:146` / `:492`        |
| `chansend`          | `runtime/chan.go:176`                 |
| `send`              | `runtime/chan.go:318`                 |
| `sendDirect` / `recvDirect` | `runtime/chan.go:392` / `:405` |
| `closechan`         | `runtime/chan.go:414`                 |
| `chanrecv`          | `runtime/chan.go:524`                 |
| `recv`              | `runtime/chan.go:702`                 |
| `selectnbsend` / `selectnbrecv` | `runtime/chan.go:784` / `:804` |
| `chanlen` / `chancap` | `runtime/chan.go:818` / `:835`      |
| `waitq.enqueue` / `dequeue` | `runtime/chan.go:872` / `:886` |
| `sellock`（按地址加锁） | `runtime/select.go:34`             |
| `selectgo`          | `runtime/select.go:122`               |
| pollorder 洗牌      | `runtime/select.go:167`               |
| lockorder 堆排序    | `runtime/select.go:206`               |
| pass 1 / 2 / 3      | `runtime/select.go:264` / `:309` / `:360` |
| select 编译期降级   | `cmd/compile/internal/walk/select.go:33` |
| `sudog`             | `runtime/runtime2.go:406`             |

## 三、常见陷阱

### 3.1 向已关闭的 channel 发送直接 panic

```go
ch := make(chan int, 1)
close(ch)
ch <- 1   // panic: send on closed channel
```

这个 panic **无法用"发送前检查"来避免**，因为检查和发送之间必然存在竞态：

```go
// 错误：典型的 check-then-act 竞态
if !isClosed(ch) {  // 这里判断"没关"
    ch <- 1         // 执行到这里时可能已经被别人关了 → panic
}
```

Go 也**没有提供**"判断 channel 是否已关闭"的 API，这是有意的设计——正确做法是从架构上保证不会发生（见 3.5、3.6），而不是靠检查。

### 3.2 重复 close 与 close(nil) 都 panic

```go
close(ch); close(ch)  // panic: close of closed channel
var n chan int
close(n)              // panic: close of nil channel
```

多个 goroutine 都可能触发关闭时，用 `sync.Once` 收口：

```go
type Stopper struct {
    done chan struct{}
    once sync.Once
}

func (s *Stopper) Stop() { s.once.Do(func() { close(s.done) }) } // 调多少次都安全
```

### 3.3 nil channel 永久阻塞——既是坑也是特性

```go
var ch chan int
ch <- 1   // 永久阻塞，goroutine 状态 "chan send (nil chan)"
<-ch      // 永久阻塞，goroutine 状态 "chan receive (nil chan)"
```

最常见的成因是**只声明没 `make`**，或者 struct 字段里的 channel 忘了初始化：

```go
type Server struct{ done chan struct{} }
s := &Server{}     // done 是 nil！
<-s.done           // 永久阻塞，而且只在运行时才暴露
```

但 `select` 里的 nil channel 有正当用途：由于 `selectgo` 会**跳过 nil channel 的 case**（2.8），把变量置 nil 就等于**动态关掉这条分支**。这是"两路输入，谁先耗尽就不再监听谁"的标准写法：

```go
for ch1 != nil || ch2 != nil {
    select {
    case v, ok := <-ch1:
        if !ok {
            ch1 = nil   // 关掉这条分支，而不是继续在已关闭的 channel 上空转
            continue
        }
        use(v)
    case v, ok := <-ch2:
        if !ok {
            ch2 = nil
            continue
        }
        use(v)
    }
}
```

如果不置 nil，已关闭的 channel 会**一直立即就绪**并返回零值，select 变成 100% CPU 的忙循环。

### 3.4 goroutine 泄漏：没有接收者的发送

最经典的一例：函数提前返回，导致后台 goroutine 永久卡在发送上。

```go
// 有泄漏
func query(urls []string) string {
    ch := make(chan string)          // 无缓冲
    for _, u := range urls {
        go func(u string) { ch <- fetch(u) }(u) // 只有第一个能发出去
    }
    return <-ch                      // 只收一个就返回 → 其余 goroutine 永久阻塞
}
```

三种修法，按场景选：

```go
// ① 缓冲区容量 = 发送者数量，让所有发送者都能发完就退出
ch := make(chan string, len(urls))

// ② 用 select + ctx.Done() 让发送方也可以放弃
go func(u string) {
    select {
    case ch <- fetch(u):
    case <-ctx.Done():   // 调用方走了，发送方也退出
    }
}(u)

// ③ 只需要第一个结果时，用 context 通知其他人取消工作
ctx, cancel := context.WithCancel(ctx)
defer cancel()
```

排查手段：`go test -race` **查不出**泄漏（泄漏不是数据竞争），要用 `runtime.NumGoroutine()` 前后对比、`go.uber.org/goleak`，或线上 `pprof` 的 goroutine profile（看 `chan send`/`chan receive` 堆栈聚集在哪一行）。

Go 1.26 还新增了一个专门的 **goroutine 泄漏 profile**，目前仍在 GOEXPERIMENT 阶段（默认关闭）：

```bash
GOEXPERIMENT=goroutineleakprofile go run main.go
```

```go
pprof.Lookup("goroutineleak").WriteTo(os.Stdout, 1)
// goroutineleak profile: total 2
// 2 @ ...
// #  0x...  main.query.func1+0x84  /path/main.go:14   ← 直接指到泄漏那一行
```

它的判定原理很聪明：借 GC 的可达性分析，如果一个 goroutine 阻塞在 channel 上（`chan send`/`chan receive`/`select`），而它等待的那个 channel 已经**不可达**（没有任何存活对象引用它，因此永远不会有人来收发），那这个 goroutine 就**永远不可能被唤醒**，判定为泄漏；`select (no cases)` 和 nil channel 上的阻塞则是定义上必然泄漏（`runtime/mgc.go` 的 `isMaybeRunnable`）。注意 `Count()` 要在 `WriteTo` 触发过一次 leak GC 之后才有值。未开启该实验时 `pprof.Lookup("goroutineleak")` 返回 `nil`，直接调用会 panic。

### 3.5 只有发送方能 close，接收方永远不要 close

```go
// 错误：接收方 close
func consume(ch chan int) {
    for v := range ch { use(v) }
    close(ch)   // 发送方可能还在写 → send on closed channel
}
```

原则：**channel 的关闭权归唯一的发送方**。用单向类型把这条原则写进签名里（1.5）——接收方拿到的是 `<-chan T`，`close` 直接编译不过，从"运行时 panic"变成"编译期错误"。

### 3.6 多个发送者时如何安全关闭

多发送者场景下没有哪个发送者知道"自己是最后一个"，所以**不能由发送者关闭数据 channel**。两种正确模式：

```go
// ① WaitGroup 收口：等所有发送者结束，再由一个协调者关闭
var wg sync.WaitGroup
out := make(chan int)
for i := 0; i < N; i++ {
    wg.Add(1)
    go func() { defer wg.Done(); out <- work() }()
}
go func() { wg.Wait(); close(out) }()   // 唯一的关闭点
for v := range out { use(v) }

// ② 用独立的 done channel 传播"停止"，数据 channel 干脆不关
done := make(chan struct{})
go func() { /* 某个条件满足 */ close(done) }()   // 关的是 done，不是 data
for i := 0; i < N; i++ {
    go func() {
        for {
            select {
            case <-done:
                return
            case data <- produce():
            }
        }
    }()
}
```

要点：**关闭信号和数据分开走两个 channel**。`done`/`ctx.Done()` 只被关闭、从不写入，所以"多方触发关闭"用 `sync.Once` 就能收口（3.2）；而数据 channel 有多个写者时，最简单的正确做法是**根本不关它**——channel 没有引用后会被 GC 回收，不关不会泄漏资源。

### 3.7 for range 不会自己结束

```go
ch := make(chan int, 3)
ch <- 1; ch <- 2; ch <- 3
for v := range ch {   // 打印 1 2 3 之后**永久阻塞**，不是正常结束
    fmt.Println(v)
}
```

`range` 的退出条件是"关闭且读空"，不是"读空"。要么 `close(ch)`，要么别用 `range`（改成读固定次数，或 select + 退出条件）。

### 3.8 select + time.After 的定时器开销

```go
// 每次循环都新建一个 Timer，1 小时内谁都不会被回收
for {
    select {
    case v := <-ch:
        use(v)
    case <-time.After(time.Hour):   // ← 每轮循环 new 一个 timer
        return
    }
}
```

`time.After` 创建的 timer 在到期前**不会被 GC**（它被 runtime 的 timer 堆引用着），高频循环里会堆积大量 timer，吃内存也吃 timer 堆的调整开销。正确做法是复用一个 `Timer`：

```go
t := time.NewTimer(time.Hour)
defer t.Stop()
for {
    if !t.Stop() {        // 复用前先停掉并排空
        select {
        case <-t.C:
        default:
        }
    }
    t.Reset(time.Hour)
    select {
    case v := <-ch:
        use(v)
    case <-t.C:
        return
    }
}
```

Go 1.23+ 修正了 `Timer`/`Ticker` 的 GC 和 `Reset` 语义（未被引用的 timer 可以被回收，`Reset`/`Stop` 后不会再收到旧值），所以在新版本上 `time.After` 的泄漏危害小了很多，但**在循环里复用 Timer 依然是更省的写法**。

### 3.9 default 会让 select 变成忙轮询

```go
// 错误：100% CPU
for {
    select {
    case v := <-ch:
        use(v)
    default:      // 没数据就立刻转下一圈
    }
}
```

`default` 的语义是"现在没就绪就别等"，放在 `for` 里就是自旋。需要"等到有数据"就**去掉 default**；确实需要轮询就加退出条件或退避（`time.Sleep`）；需要同时兼顾多个事件源就把它们都写成 case。

### 3.10 用 len/cap 做控制流是竞态的

```go
// 错误
if len(ch) > 0 {
    v := <-ch        // 执行到这里时可能已被别的 goroutine 取走 → 阻塞
}
if len(ch) < cap(ch) {
    ch <- v          // 同理，可能已经满了 → 阻塞
}
```

`len(ch)` 只是一次无锁读，返回的瞬间就可能失效（源码在 `full`/`empty` 的注释里明确说了这点）。要非阻塞地收发就用 `select` + `default`，那是运行时在锁内做的原子判断：

```go
select {
case v := <-ch:
    use(v)
default:
    // 确实没有
}
```

### 3.11 无缓冲 channel 的"发送成功"不代表"处理完成"

```go
ch := make(chan Task)
ch <- task            // 返回时，对方只是"取走了 task"
// 此处 task 可能一行代码都还没被执行
```

由内存模型第 3 条（2.9），无缓冲发送返回只保证接收方**完成了接收动作**。要等对方处理完，必须有回执：

```go
type Task struct {
    payload string
    reply   chan error   // 回执 channel
}

t := Task{payload: "x", reply: make(chan error, 1)}
ch <- t
if err := <-t.reply; err != nil {   // 这才是"处理完成"
    return err
}
```

### 3.12 channel 传的是值的拷贝

```go
type Big struct{ buf [4096]byte }
ch := make(chan Big, 100)   // 缓冲区就占 400KB，每次收发还各拷 4KB
```

`typedmemmove` 拷的是完整元素（2.5）。大对象应该传指针 `chan *Big`，但随之而来的是**所有权约定**：值一旦发出去，发送方就不能再改它，否则就是没有锁保护的共享写，`-race` 会报数据竞争。

另一个相关坑是**元素是含指针的 struct 时**，拷贝是浅拷贝：

```go
type Msg struct{ Data []byte }
m := Msg{Data: buf}
ch <- m
buf[0] = 'x'    // 发送方改的是同一块底层数组，接收方看到的数据被篡改了
```

### 3.13 死锁：主 goroutine 上的同步操作

```go
func main() {
    ch := make(chan int)
    ch <- 1        // fatal error: all goroutines are asleep - deadlock!
    fmt.Println(<-ch)
}
```

无缓冲 channel 要求收发双方**同时**就绪，同一个 goroutine 里先发后收必然自锁。运行时的死锁检测器能发现"所有 goroutine 都休眠"这一种情况并 `fatal error`（**注意这是 fatal error，`recover` 抓不住**），但它检测不了"部分 goroutine 泄漏而主流程仍在跑"——那种情况只能靠 pprof。

`select {}`、以及"所有 case 都是 nil channel 且无 default"的 select 也会被检测为死锁：

```go
var a, b chan int
select {           // 所有 case 都被跳过，等价于 select{}
case <-a:
case b <- 1:
}                  // fatal error: all goroutines are asleep - deadlock!
```

### 3.14 select 中同一 channel 出现多个 case

```go
select {
case v := <-ch:   // 分支 A
    handleA(v)
case v := <-ch:   // 分支 B —— 合法，但走哪个是随机的
    handleB(v)
}
```

编译能过，`sellock` 也做了同 channel 去重（相邻同 channel 只加一次锁），但走 A 还是 B 完全取决于 pollorder 的随机洗牌结果。这几乎总是逻辑错误，应该合并成一个 case 后再分派。

### 3.15 关闭顺序：先关数据，还是先等 worker

```go
// 错误：worker 还在往 results 里写，results 就被关了
close(jobs)
close(results)          // worker 可能仍在 results <- ... → panic
for r := range results { ... }
```

顺序必须是：**关闭输入 → 等所有 worker 退出 → 关闭输出**。

```go
close(jobs)                              // ① 让 worker 的 range 退出
go func() { wg.Wait(); close(results) }() // ② worker 全退了才关 results
for r := range results { use(r) }        // ③ 正常读到结束
```

注意 ② 必须放在另一个 goroutine 里：如果在当前 goroutine 直接 `wg.Wait()`，而 worker 正阻塞在 `results <-` 上等人来读，就是互相等待的死锁。

## 四、常见面试题

**1. channel 的底层数据结构是什么？**
`runtime.hchan`：环形缓冲区 `buf` + 读写下标 `recvx`/`sendx` + 元素计数 `qcount` + 容量 `dataqsiz` + 两条等待队列 `recvq`/`sendq`（`sudog` 双向链表）+ 一把 `mutex`。`chan T` 变量本身是指向 `hchan` 的指针，大小 8 字节。详见 2.1。

**2. channel 是无锁的吗？**
不是。`hchan` 自带一把运行时 mutex，所有会修改状态的操作都在锁内完成。只有"非阻塞且注定失败"的判断（select 的 default 分支）走无锁快路径，靠"已关闭的 channel 不可能变回可用"这一单调性保证正确。详见 2.3。

**3. 无缓冲和有缓冲 channel 的本质区别？**
无缓冲是**同步交接**：`dataqsiz == 0`，收发双方必须同时就绪，数据从发送者栈**直接拷到**接收者栈（1 次拷贝），发送返回即代表对方已取走。有缓冲是异步：数据经过 `buf` 中转（2 次拷贝），发送返回只代表已放入缓冲区。详见 2.10。

**4. 数据在 channel 里被拷贝了几次？**
无缓冲 1 次（`sendDirect`：发送者栈 → 接收者栈）；有缓冲一般 2 次（栈 → buf → 栈）；有缓冲但恰好有等待的接收者时也是 1 次（走 `chansend` 的路径 ①，绕过 buf）。详见 2.5。

**5. 为什么无缓冲 channel 的直接拷贝要手工加写屏障？**
这是运行时里唯一"一个运行中的 goroutine 直接写另一个运行中 goroutine 的栈"的场景。GC 假设栈写入只由该 goroutine 自己完成，`typedmemmove` 的 `bulkBarrierPreWrite` 只覆盖堆目标，对栈目标无效，所以 `sendDirect` 手工调 `typeBitsBulkBarrier` + `memmove` 补上写屏障。详见 2.5。

**6. 向已关闭的 channel 发送/接收分别是什么行为？**
发送：**panic** `send on closed channel`——包括"进入时已关闭"和"阻塞期间被关闭"两种，后者在被 `close` 唤醒时通过 `sg.success == false` 判断出来并 panic。接收：**永远安全**，先把缓冲区剩余数据读完，读空后返回 `(零值, false)`。详见 2.3、2.4。

**7. `close` 是怎么唤醒所有等待者的？为什么要先收集再唤醒？**
`closechan` 置 `closed = 1`，然后把 `recvq` 和 `sendq` 里所有 `sudog` 取出、统一标记 `success = false`，收集进 `glist`，**解锁之后**才逐个 `goready`。之所以要等解锁：`hchan.lock` 的约束是"持锁时不得改变其他 G 的状态"，否则可能与栈收缩死锁。接收者把 `success=false` 当作 `ok=false`，发送者看到它就 panic——一个标志区分两种行为。详见 2.6。

**8. 为什么 `close` 能做广播，而发送不能？**
一次发送只从 `recvq` 里 `dequeue` **一个**等待者；`close` 会把 `recvq` **清空**，唤醒全部。这就是 `context.Done()` 的原理：那个 channel 从不写入，只被 close，所以任意多个监听者都能同时收到。详见 2.6、1.8 ①。

**9. `v, ok := <-ch` 里 `ok == false` 到底意味着什么？**
意味着"channel 已关闭**并且**缓冲区已读空"，不是"channel 已关闭"。`chanrecv` 只在 `closed != 0 && qcount == 0` 时才走零值分支，否则继续读缓冲区。详见 1.3、2.4。

**10. select 是怎么实现的？多个 case 同时就绪时选哪个？**
先看编译期：0 个 case → `block()` 永久 park；1 个 case → 退化成普通收发；1 个 case + default → `selectnbsend`/`selectnbrecv`；≥2 个才进 `selectgo`。`selectgo` 建两个序：`pollorder` 用 `cheaprandn` 随机洗牌（决定就绪时选谁，实测均匀 50/50），`lockorder` 按 channel 地址排序（决定加锁顺序）。然后三趟：找就绪 → 全部入队并 park → 唤醒后摘除未命中的 sudog。详见 2.8。

**11. select 为什么要按 channel 地址排序加锁？**
select 需要同时持有多个 channel 的锁。若两个 select 语句以不同顺序加锁（A→B 和 B→A），就会经典地互相死锁。按 channel 地址（`sortkey()`）排序，保证全进程统一的加锁顺序，从根本上消除环路。详见 2.8。

**12. select 的第三趟扫描（pass 3）为什么不能省？**
执行 select 的 G 会在**每个** case 的 channel 队列上挂一个 sudog。被某一个唤醒后，其余 sudog 还留在别的队列里。不摘掉它们，冷清的 channel 上会不断堆积无效 sudog（源码注释：`otherwise they stack up on quiet channels`），既泄漏内存又会引发错误唤醒。详见 2.8。

**13. 一个 G 挂在多个 channel 上，两个 channel 同时来数据会怎样？**
`sudog.isSelect == true` 时，`waitq.dequeue` 必须先 `sgp.g.selectDone.CompareAndSwap(0, 1)` 才有权唤醒这个 G。CAS 失败说明别的 case 已经抢到了唤醒权，本次 `dequeue` 跳过它继续找下一个等待者。详见 2.2。

**14. `for range ch` 什么时候退出？只 close 不写数据能退出吗？**
退出条件是"channel 已关闭且缓冲区读空"。只 `close` 不写数据可以正常退出（第一次读就是 `ok == false`）；反之如果永远不 `close`，`range` 就永久阻塞——这是 goroutine 泄漏的常见来源。详见 1.4、3.7。

**15. 多个发送者时，谁负责 close？**
没有发送者能安全地关闭。两种正确模式：① `WaitGroup` 等所有发送者退出后，由一个独立的协调 goroutine 关闭；② 关闭信号走独立的 `done`/`ctx.Done()` channel（只 close 不写，多方触发用 `sync.Once` 收口），数据 channel 干脆不关——channel 不被引用后会被 GC 回收，不关不泄漏。详见 3.6。

**16. 怎么判断一个 channel 是否已关闭？**
语言**不提供**这个能力，而且这个需求本身就是错的：判断和后续操作之间必然有竞态窗口。只能通过接收的 comma-ok 得知"关闭且已读空"，或者从架构上约定关闭权归属。详见 3.1。

**17. channel 提供哪些内存可见性保证？**
四条：① 发送 happens-before 对应的接收完成；② `close` happens-before 因关闭而返回零值的接收完成；③ **无缓冲** channel 的接收 happens-before 对应的发送完成；④ 容量 C 的 channel 上第 k 次接收 happens-before 第 k+C 次发送完成。第 ③ 条决定了"无缓冲发送返回 = 对方已取走"；第 ④ 条是拿 channel 当信号量的依据。详见 2.9。

**18. goroutine 阻塞在 channel 上会占用一个 OS 线程吗？**
不会。`gopark` 把 G 置为 `_Gwaiting` 并与 M 解绑，M 立刻回调度循环执行别的 G，被阻塞的 G 只是一个挂在 `hchan` 队列上的 `sudog`。这也是能开十万级 goroutine 收发 channel 的原因。唤醒走 `goready`，只是重新入队，**不保证立即执行**。详见 2.7。

**19. 为什么 channel 元素类型有大小限制？限制是多少？**
`hchan.elemsize` 是 `uint16`，`makechan` 显式检查 `elem.Size_ >= 1<<16` 就 `throw`，因此元素类型大小必须小于 64KB。实践中远早于这个上限就该改用指针了——拷贝成本是逐字节的。详见 2.1、3.12。

**20. `len(ch)`/`cap(ch)` 能用来做流控判断吗？**
不能。它们只是一次无锁读，返回瞬间就可能失效，而且**不包含阻塞在 `sendq`/`recvq` 上的等待者**。非阻塞收发要用 `select` + `default`（运行时在锁内原子判断）。另外 timer channel 的 `len`/`cap` 被特意伪装成 0，更不能用来判断是否到期。详见 1.6、2.11、3.10。

**21. 为什么 `time.After` 在循环里有问题？**
每次 select 求值 case 表达式时都会新建一个 Timer，到期前一直被 runtime 的 timer 堆引用。高频循环会堆积大量 timer。应该复用一个 `time.Timer`（`Stop` + 排空 + `Reset`）。Go 1.23+ 改善了 timer 的 GC 和 `Reset` 语义，危害变小，但复用仍是更省的写法。详见 3.8。

**22. 有缓冲 channel 满了、有发送者在排队时，接收者拿到的是哪个值？**
拿到的是**缓冲区队头的旧值**，阻塞发送者的值被放到队尾。由于队列是满的，队头和队尾是同一个槽位，`recv` 一次拷贝就完成了"出队 + 入队"，之后 `recvx++` 且 `sendx = recvx`。这保证了即使中途经历阻塞，FIFO 顺序也不会乱。详见 2.4。

**23. `select {}` 和 `for {}` 有什么区别？**
`select {}` 被编译成 `block()`，即无条件 `gopark` 永久休眠，**不消耗 CPU**，而且 G 处于 `_Gwaiting`，能被死锁检测器识别（`fatal error: all goroutines are asleep - deadlock!`，goroutine 状态显示为 `select (no cases)`）。`for {}` 是纯自旋，**吃满一个核**，且 G 一直是 `_Grunning`，永远不会被判定为死锁（Go 1.14 起有异步抢占，空循环不会再卡住 GC 和调度，但 CPU 照样白烧）。要永久阻塞当前 goroutine 应该用 `select {}` 或 `<-make(chan struct{})`。详见 2.8。

**24. channel 和 mutex 该怎么选？**
channel 传递**数据所有权和事件**（生产消费、任务分发、取消通知、结果汇聚、流水线），它同时提供同步和通信；mutex 保护**共享状态的临界区**（计数器、缓存、map）。判断依据：如果在用 channel 模拟"加锁改一个变量再解锁"，就应该用 mutex；如果在用 mutex + 条件变量模拟"等待某个事件/传递数据"，就应该用 channel。Go 官方的说法是 "Don't communicate by sharing memory; share memory by communicating"，但也明确说了 mutex 在保护状态这件事上更合适。
