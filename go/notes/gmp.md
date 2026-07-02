# GMP 模型

- **G（Goroutine）**：Go 的用户态协程，代表一个待执行的任务单元。创建成本低（初始栈仅 2KB，可动态伸缩），由 Go 运行时调度，数量可达百万级。
- **M（Machine）**：操作系统线程（OS thread），是真正执行代码的实体。M 必须绑定一个 P 才能运行 G，其数量受 `GOMAXPROCS` 及运行时限制（默认上限 10000）。
- **P（Processor）**：逻辑处理器，代表执行 G 所需的调度上下文与资源（如本地可运行 G 队列、内存分配缓存）。P 的数量默认等于 CPU 核数（由 `GOMAXPROCS` 决定），决定了同时并行执行 Go 代码的最大并发度。


*go/src/runtime/runtime2.go*

## G — goroutine 结构体

源码：`runtime/runtime2.go` 的 `type g struct`。

```go
type g struct {
    // ---- 栈相关 ----
    stack       stack   // 栈的内存范围 [stack.lo, stack.hi)
    stackguard0 uintptr // 栈增长检查的边界值；被设为 stackPreempt 时用于触发抢占
    stackguard1 uintptr // 供 //go:systemstack 代码使用的栈边界

    // ---- 执行现场 ----
    _panic    *_panic  // 当前最内层的 panic 链表
    _defer    *_defer  // 当前最内层的 defer 链表
    m         *m       // 当前正在执行本 G 的 M（未运行时为 nil）
    sched     gobuf    // 调度现场：切换 G 时保存/恢复 sp、pc、bp 等寄存器

    // ---- 系统调用 ----
    syscallsp uintptr  // status==Gsyscall 时保存的栈指针，供 GC 使用
    syscallpc uintptr  // status==Gsyscall 时保存的程序计数器

    // ---- 状态与调度 ----
    param        unsafe.Pointer // 唤醒时传递的通用参数（如 channel 唤醒时指向 sudog）
    atomicstatus atomic.Uint32  // G 的当前状态（_Grunnable/_Grunning/_Gwaiting…），原子访问
    goid         uint64         // goroutine 的唯一 id
    schedlink    guintptr       // 指向下一个 G，用于把 G 串成链表（如全局队列）
    waitsince    int64          // 进入阻塞的大致时间
    waitreason   waitReason     // status==Gwaiting 时的阻塞原因（chan 收发、GC、锁等）

    // ---- 抢占 ----
    preempt      bool           // 抢占信号，与 stackguard0=stackPreempt 冗余
    preemptStop  bool           // 被抢占时转入 _Gpreempted 而非只是让出

    // ---- 血缘 / 调试信息 ----
    lockedm      muintptr       // 若 G 被锁定到某个 M（LockOSThread），指向该 M
    parentGoid   uint64         // 创建本 G 的父 goroutine 的 goid
    gopc         uintptr        // 触发创建的 `go` 语句所在的 pc（panic 栈里能看到）
    startpc      uintptr        // goroutine 入口函数的 pc

    waiting      *sudog         // 本 G 正在等待的 sudog 链（用于 channel/select）
    timer        *timer         // time.Sleep 复用的定时器
}
```

**学习要点**

- **G 保存的是"任务的执行现场"**：真正切换 goroutine 时，`sched (gobuf)` 里的 sp/pc/bp 被保存和恢复，这就是"用户态协程切换比线程切换快"的原因——不涉及内核态。
- **栈是可伸缩的**：`stack` 描述当前栈内存，`stackguard0` 在函数序言里做栈溢出检查，触发 `morestack` 进行栈扩容/收缩，所以初始 2KB 栈能按需增长。
- **抢占机制**：把 `stackguard0` 设为 `stackPreempt`，下次函数调用检查栈时就会"顺便"让出 CPU，这是 Go 协作式抢占的核心技巧（1.14 后还有基于信号的异步抢占）。
- **`atomicstatus` 是调度状态机**：G 在 `_Grunnable`（可运行）、`_Grunning`（运行中）、`_Gwaiting`（阻塞）、`_Gsyscall`（系统调用中）等状态间流转。
- **`m` 字段体现 G↔M 关系**：G 运行时指向承载它的 M，让出后置空——G 并不固定绑定某个线程。

## M — machine（OS 线程）结构体

源码：`runtime/runtime2.go` 的 `type m struct`。M 是对操作系统线程的封装。

```go
type m struct {
    // ---- 调度栈 ----
    g0      *g     // 调度专用 goroutine，运行调度器代码时用它的栈（系统栈）
    morebuf gobuf  // morestack 使用的现场

    gsignal *g                // 处理信号用的 g
    tls     [tlsSlots]uintptr // 线程本地存储（TLS）

    // ---- 当前执行状态 ----
    curg    *g       // 当前正在运行的用户 goroutine
    p       puintptr // 当前绑定的 P（执行 Go 代码时非空；不执行用户代码时为 nil）
    nextp   puintptr // 即将绑定的 P（唤醒 M 前先塞好）
    oldp    puintptr // 进入系统调用前绑定的 P

    id       int64
    spinning bool // M 处于自旋态：没活干、正在到处找可运行的 G
    blocked  bool // M 阻塞在 note 上

    // ---- cgo ----
    incgo    bool   // M 正在执行 cgo 调用
    ncgocall uint64 // 累计 cgo 调用次数

    // ---- 链接与锁定 ----
    alllink   *m       // 挂在全局 allm 链表上
    schedlink muintptr // 空闲 M 链表的链接指针
    lockedg   guintptr // 若发生 LockOSThread，指向被锁定到本 M 的 G

    park        note        // M 空闲时在此“睡眠”，被唤醒后继续找活
    createstack [32]uintptr // 创建该线程时的调用栈
    mOS                     // 平台相关的线程字段（如线程句柄）
}
```

**学习要点**

- **`g0` 是 M 的"系统栈"**：调度、栈扩容、GC 等运行时逻辑跑在 `g0` 上，而不是用户 G 的栈。`curg` 指向当前用户 G，`g0` 负责调度——两者来回切换是理解调度的关键。
- **`p` 决定 M 能否跑 Go 代码**：M 必须持有一个 P 才能执行用户 G。`p == nil` 说明当前没在跑用户代码。`oldp` / `nextp` 服务于系统调用前后的 P 交接（handoff）。
- **`spinning` 自旋态**：M 找不到活时不会立刻睡，而是先自旋一小会儿主动找 G，避免频繁地睡眠/唤醒线程。找不到才 `park` 睡眠。
- **`lockedg` 与 LockOSThread**：`runtime.LockOSThread` 会把 G 固定到某个 M 上（常用于必须在同一线程执行的场景，如 OpenGL、某些 C 库），此时 G↔M 变成强绑定。
- **M 是懒创建、可复用的**：新 M 通过 `alllink` 挂到全局 `allm`，空闲后放到 `schedlink` 空闲链表复用；这也解释了阻塞 syscall 多时 M 数量会涨（见前文"M 可以大于 GOMAXPROCS"）。

## P — processor（逻辑处理器）结构体

源码：`runtime/runtime2.go` 的 `type p struct`。P 是"执行 Go 代码所需的资源与上下文"，数量由 `GOMAXPROCS` 决定。

```go
type p struct {
    id     int32
    status uint32   // P 的状态：_Pidle/_Prunning/_Psyscall/_Pgcstop/_Pdead
    m      muintptr // 反向指向绑定的 M（空闲时为 nil）
    mcache *mcache  // 本 P 的内存分配缓存，无锁分配小对象的关键

    // ---- 本地可运行 G 队列（核心）----
    runqhead uint32
    runqtail uint32
    runq     [256]guintptr // 环形队列，容量 256，无锁访问
    runnext  guintptr      // 优先运行的下一个 G（比 runq 更优先）

    gFree gList // 本 P 缓存的已结束(_Gdead) G，复用以减少分配

    // ---- 各类对象缓存（减少锁竞争）----
    deferpool    []*_defer // defer 结构缓存
    sudogcache   []*sudog  // sudog 缓存
    goidcache    uint64    // 批量预取的 goid 段，摊薄全局 goidgen 访问
    goidcacheend uint64

    // ---- 定时器与 GC ----
    timers timers // 本 P 的定时器堆（time.Timer/Sleep）
    gcw    gcWork // GC 工作缓冲区
    wbBuf  wbBuf  // GC 写屏障缓冲区

    preempt bool // 置位表示该 P 应尽快进入调度器
}
```

**学习要点**

- **`runq` + `runnext` 是本地运行队列**：每个 P 有一个容量 256 的无锁环形队列，调度时优先从这里取 G，避免访问全局队列的锁竞争。`runnext` 是"插队"槽——当前 G 唤醒的新 G 放这里能连续执行，减少通信-等待型 goroutine 的调度延迟。
- **work-stealing（工作窃取）**：当某个 P 的 `runq` 空了，它会去"偷"其他 P 队列一半的 G 来跑，实现负载均衡。取 G 顺序：`runnext` → 本地 `runq` → 全局队列 → netpoller → 偷其他 P 的一半。
- **`mcache` 让内存分配无锁**：每个 P 独占一个 `mcache`，分配小对象时直接从本地缓存拿，不用加全局锁——调度器和内存分配器都以 P 为单位做缓存。
- **P 是各种缓存的宿主**：`deferpool`、`sudogcache`、`goidcache` 都挂在 P 上，把"全局竞争"降级为"每 P 本地操作"。
- **`status` 状态机**：`_Pidle`（空闲）、`_Prunning`（绑定 M 跑用户代码）、`_Psyscall`（其 M 陷入系统调用）、`_Pgcstop`、`_Pdead`（GOMAXPROCS 调小后多余的 P）。

## 三者关系小结

```
        ┌───────────── P (逻辑处理器, 数量 = GOMAXPROCS) ─────────────┐
        │  runnext │ runq[256] 本地队列 │ mcache │ timers │ 各种缓存    │
        └──────────────────────────┬──────────────────────────────────┘
                                    │ 绑定
                                    ▼
   G ── curg ──►  M (OS 线程) ── g0 调度栈 ── park 睡眠 / spinning 找活
   (任务)          (工人)
```

- **M 必须持有 P 才能执行 G**：P 是"许可证 + 资源包"，M 是"工人"，G 是"任务"。
- 全局最多 `GOMAXPROCS` 个 M 在并行跑用户代码（因为只有这么多 P）；但 M 总数可更多（阻塞 syscall 时 P 会 handoff 给别的 M）。
- 调度取 G 顺序：`runnext` → 本地 `runq` → 全局队列 → netpoller → 偷其他 P 的一半。