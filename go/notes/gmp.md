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

## 各要素的诞生顺序（从进程到 goroutine）

把 m0/g0、P、goroutine、工作线程按「谁先出现、谁触发谁」串成一条时间线。

### 主干：进程 → 主线程 → 绑定 P → 首个 goroutine

```
① 进程诞生
   OS exec() 加载可执行文件 → 内核创建【主线程】(主 OS 线程)
   这个主线程在 Go 里就是 m0

② 主线程跑引导 (rt0_go 汇编)
   划出 g0 栈  →  绑定 m0.g0 = g0 / g0.m = m0
   （m0 和 g0 都是全局变量，进程唯一）

③ schedinit：把 runtime 拉起来
   mallocinit / gcinit / ...
   读 GOMAXPROCS
   procresize(n)  →  一次性创建 n 个【P】(P0..Pn-1)
                     m0 绑定 P0，其余 P 进空闲列表(_Pidle)

④ newproc(runtime.main)
   创建【主 goroutine】(goid=1)，入 P0 的运行队列
   —— 此刻只入队，还没跑

⑤ mstart → schedule()
   m0 在自己的 g0 上跑调度循环，捞到主 goroutine → 开始执行 runtime.main
```

到 ⑤ 为止，`m0—P0—主goroutine` 三者就位。

### 运行期：按需新建线程 / 新建 goroutine

```
⑥ runtime.main 内部
   newm(sysmon)  →  新建一条【监控线程】(不绑 P，独立跑)
   gcenable      →  启动后台 GC
   跑各种 init …
        │
        ▼
⑦ 调用用户 main.main
        │
        ▼
⑧ 用户写的 go func(){...}
   编译成 → newproc(fn)
        ├─ 新建/复用一个【g】(goroutine)，状态 _Grunnable
        ├─ runqput：放进当前 P 的 runnext / 本地队列
        └─ wakep()：看有没有空闲 P + 需要干活
              │
              ├─ 有空闲 P 且没有空闲 M → newm() 新建【工作线程】
              │        新线程绑定那个空闲 P，跑起自己的 schedule()
              │        去队列里捞这个新 g 来执行 ← 实现并行
              │
              └─ 已有空闲 M 睡着 → 直接唤醒它(而不是新建)
```

### 谁触发谁

| 事件              | 由谁触发                     | 产物                           |
| ----------------- | ---------------------------- | ------------------------------ |
| 主线程(m0)        | OS 创建进程                  | 唯一，程序入口执行流           |
| P 数组            | `schedinit → procresize`     | GOMAXPROCS 个，一次建好        |
| 主 goroutine      | `rt0_go → newproc`           | 入口 `runtime.main`            |
| sysmon 线程       | `runtime.main → newm`        | 后台监控，不占 P               |
| **新工作线程(M)** | **`newproc → wakep → newm`** | **有空闲 P 且没空闲 M 时才建** |
| 用户 goroutine    | 用户 `go` 语句 → `newproc`   | 入 P 本地队列等调度            |

**三个容易混的点**

1. **P 是「批量、提前」建的**：启动时 `procresize` 一把建满 `GOMAXPROCS` 个；而 **M（线程）是「懒惰、按需」建的**，只有有活干、有空闲 P、又没有空闲 M 时才 `newm`，空闲后会复用不销毁。
2. **新建 goroutine ≠ 新建线程**：`go func()` 绝大多数情况只是往当前 P 队列塞个 g，由现有 M 顺手执行，**不会**新建线程。只有「有富余 P 没人用 + 确实需要并行」时，`wakep` 才可能拉起新 M。所以百万 goroutine 也就跑在 GOMAXPROCS 量级的线程上。
3. **绑定方向是 M 去绑 P**（工人领许可证），不是 P 绑 M。M 拿到 P 才能跑用户 g；进系统调用时 M 会把 P handoff 给别的 M，自己带着 g 陷入内核。

> 一句话：**进程给你一个主线程 → 主线程建好一批 P 并绑住一个 → 用「首个 goroutine」启动 runtime.main → 用户每个 `go` 只是造一个 g 入队，线程只在需要并行且有空闲 P 时才被 `wakep` 按需拉起。** 完整的进程启动汇编链路见 [exec.md](exec.md)。

## 调度是如何运转的

理解 GMP 的最后一块拼图：**调度不是某个中心线程的专属工作，而是去中心化地跑在每个 M 各自的 `g0` 上。** 这里澄清两个高频误解。

### 误解一：`g0` 是全局唯一的

不是。`g0` 是**每个 M 都有一份**的「系统栈/调度栈」。无论是引导阶段的 `m0`，还是运行期 `newm` 创建的工作线程、sysmon 线程，创建时都各自带一个 `g0`：

```
m0   ── g0(m0)      ← 主线程的调度栈（负责引导）
m1   ── g0(m1)      ← 工作线程1的调度栈
m2   ── g0(m2)
sysmon线程 ── g0    （不绑 P，基本一直在 g0 上跑监控）
...
```

- **普通 g（goroutine）**：跑用户代码，栈可增长（初始 2KB）。
- **g0**：固定大小，专门承载 runtime 自己的代码——`schedule()`、GC、栈扩容（`morestack`）、`newproc`、系统调用切换等。

### 误解二：`m0`/`g0` 负责协调整个 GMP

`m0` 的特殊只体现在**启动阶段**：`rt0_go → schedinit → 创建主 goroutine → 第一次 schedule()` 都发生在 `m0` 的 `g0` 上，因为那时还没有别的线程。**引导一结束，m0 就退化成一个普通 M**，和 m1/m2 一样去抢 P、跑 G、在自己 g0 上做调度，不比别人多管什么。（少数平台相关操作要求主线程执行时，m0 才再次体现特殊性。）

真正「协调」调度的是 **P（队列与资源的枢纽）+ 每个 M 在自己 g0 上跑的 `schedule()`**——N 个 M 通过共享的 P、全局队列、work-stealing 互相协作，没有总指挥。

### 一次调度切换的流程

```
某个 M 上：
  用户 goroutine (gA) 运行中
        │  时间片用完 / 阻塞 / 主动让出(Gosched) / 系统调用返回
        ▼
  切换到 本 M 的 g0 栈            ← mcall / systemstack
        │
        ▼
  在 g0 上执行 schedule()：
     - 按顺序找一个可运行的 G：
       runnext → 本地 runq → 全局队列 → netpoller → 偷其他 P 的一半
     - 找到 gB 后 gogo(gB) 切过去
        │
        ▼
  切回 gB 的用户栈继续执行
```

关键点：**用户栈 ↔ g0 栈的来回切换**就是调度的物理动作。`gA` 让出时把执行现场存进它自己的 `g.sched (gobuf)`，切到 g0 选出 `gB`，再从 `gB.sched` 恢复现场跳过去——全程在用户态完成，不进内核，这就是 goroutine 切换比线程切换快的根源。

### goroutine 从生到死

```
go func(){...}
    │  编译成 runtime.newproc
    ▼
newproc → newproc1：gfget 复用 或 malg(2KB) 新建一个 g
    │  设置 g.sched.pc = goexit，再用 gostartcallfn 塞入真正的 fn
    ▼
runqput(P, g)：放进当前 P 的 runnext / 本地队列，状态 _Grunnable
    │  （必要时 wakep 唤醒或新建 M 来并行执行）
    ▼
被某个 M 的 schedule() 选中 → gogo → 状态 _Grunning，执行 fn
    │  fn 返回后自动回到 goexit
    ▼
goexit0：清理 g，状态置 _Gdead，放回 P 的 gFree 缓存池等待复用
```

> 因为新 g 的返回地址被设成了 `goexit`，goroutine 执行完函数会「自动」回到 runtime 手里做清理和回收，用户无需关心生命周期。

> 完整的进程启动 → 首个 goroutine → 首次调度的链路，见 [exec.md](exec.md)。