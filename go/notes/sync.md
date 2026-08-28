# sync

> 环境：`go version go1.26.3`。这个包的实现在最近几个版本被大改过，网上绝大多数中文资料还停留在 1.20 之前，注意版本：
> - **1.9**：引入 `sync.Map`（read/dirty 双 map + misses 提升）。
> - **1.18**：`Mutex.TryLock`、`RWMutex.TryLock`、`RWMutex.TryRLock`。
> - **1.21**：`OnceFunc`、`OnceValue`、`OnceValues`。
> - **1.24**：`sync.Map` **整体换成 concurrent hash-trie**（`internal/sync.HashTrieMap`），read/dirty 那套设计已经不存在了；`Mutex`/`Map` 的实现下沉到 `internal/sync`，`sync` 包变成薄封装（为了让 `internal` 里的包也能用锁而不产生循环依赖）。
> - **1.25**：`WaitGroup.Go`；`WaitGroup` 的 state 里多了一位 synctest bubble 标记。
>
> 源码位置：`sync/{mutex,rwmutex,waitgroup,once,oncefunc,cond,map}.go`、`internal/sync/{mutex,hashtriemap}.go`、`runtime/sema.go`。配套代码：`notes/sync/`。

## 一、基础使用

### 1.1 Mutex

```go
type counter struct {
    mu sync.Mutex
    n  int
}

func (c *counter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.n++
}
```

几条容易被忽略的语义：

- **零值可用**，没有 `NewMutex`；也不能拷贝（见 3.1）。
- **锁不绑定 goroutine**：A 加锁、B 解锁是合法的（文档明确写了）。因此 Go 的 `Mutex` **不可重入**，同一个 goroutine 再 `Lock` 就是自死锁——运行时甚至不认为这是错误，只是永远等下去（单 goroutine 时会被死锁检测器抓到 `all goroutines are asleep - deadlock!`，多 goroutine 时通常不会）。
- **`Unlock` 一个没加锁的 mutex 是 `fatal error`，不是 `panic`**，`recover` 拦不住（见 3.2）。
- `TryLock` 存在但文档明确劝退：*correct uses of TryLock do exist, they are rare, and use of TryLock is often a sign of a deeper problem*。

### 1.2 RWMutex

```go
var rw sync.RWMutex

rw.RLock();  defer rw.RUnlock()  // 读：可并发
rw.Lock();   defer rw.Unlock()   // 写：独占
```

三条硬规则（都写在类型注释里）：

1. **写优先**：一旦有 goroutine 在 `Lock` 上等待，**后来的 `RLock` 会被挡住**，防止读者源源不断把写者饿死。
2. **因此 `RLock` 不可递归**。外层持读锁 → 中间来了写者 → 内层再 `RLock` 就死锁。这个 bug 的典型形态是"某个持读锁的方法调用了另一个也要读锁的方法"。
3. **不能升级/降级**：`RLock` 不能变 `Lock`，`Lock` 也不能变 `RLock`。

`readerCount` 上限 `rwmutexMaxReaders = 1<<30`，所以最多约 10.7 亿并发读者。

### 1.3 WaitGroup

```go
// Go 1.25+ 推荐写法
var wg sync.WaitGroup
for _, task := range tasks {
    wg.Go(func() { handle(task) })   // 内部就是 Add(1) + go func(){ defer Done(); f() }()
}
wg.Wait()

// 老写法：Add 必须在 go 之前
wg.Add(1)
go func() { defer wg.Done(); handle(task) }()
```

`wg.Go(f)` 有一个和手写不同的重要行为：**`f` 不允许 panic**。源码里 recover 到 panic 之后**故意不调 `Done`**，然后重新 panic：

```go
// sync/waitgroup.go:236
func (wg *WaitGroup) Go(f func()) {
    wg.Add(1)
    go func() {
        defer func() {
            if x := recover(); x != nil {
                // 不调 Done，直接重新 panic
                panic(x)
            }
            wg.Done()
        }()
        f()
    }()
}
```

原因写在注释里：如果调了 `Done`，`Wait` 会被解除阻塞，main goroutine 可能抢在 panic 打印完之前 `os.Exit(0)`，把崩溃现场吞掉。

### 1.4 Once 家族

```go
var once sync.Once
once.Do(initialize)                              // 最基础

var setup = sync.OnceFunc(initialize)            // 1.21+：包装成普通函数
var config = sync.OnceValue(loadConfig)          // 惰性单例，返回值被缓存
var conn, err = sync.OnceValues(dial)()          // 两个返回值
```

- 四者都保证 `f` **只执行一次，且返回时 `f` 已经执行完**（不是"已经开始执行"）。
- **`f` panic 之后不会重试**：`Once.Do` 里 `defer o.done.Store(true)` 在 panic 展开时也会执行。`OnceFunc`/`OnceValue` 更进一步——把 panic 值记下来，**之后每次调用都 panic 同一个值**。
- 需要"失败可重试的初始化"就别用 `Once`，用 `errgroup`/`singleflight` 或自己写"失败时重置状态"的逻辑。

### 1.5 Cond

```go
func (q *queue) Pop() (int, bool) {
    q.mu.Lock()
    defer q.mu.Unlock()
    for len(q.items) == 0 && !q.done {   // ① 必须是 for，不是 if
        q.cond.Wait()                    // ② Wait 内部：Unlock -> 挂起 -> 唤醒 -> Lock
    }
    ...
}
```

- `Wait` 必须**持锁调用**，它内部会 `Unlock`；返回时锁已重新持有。
- **一定写在 `for` 里**：`Signal`/`Broadcast` 只是"通知有事发生"，不保证你要的条件成立（虚假唤醒 + 被别人抢先消费）。
- `Signal` 唤醒一个，`Broadcast` 唤醒全部；**都不需要持锁**，但持锁调用更容易推理。
- **没有带超时的 Wait**。要超时就换 channel + `select`，或者用 `context`。
- 实践建议：Go 里 `Cond` 的使用场景比其他语言少得多，多数"等条件"都能用 channel 表达得更清楚。标准库自己只在 `net`、`database/sql` 等少数地方用它。

### 1.6 sync.Map

```go
var m sync.Map

m.Store("k", 1)
v, ok := m.Load("k")
actual, loaded := m.LoadOrStore("k", 2)   // 存在则返回旧值，不存在则存入
prev, loaded := m.Swap("k", 3)            // 无条件替换
ok = m.CompareAndSwap("k", 3, 4)          // 值必须可比较
v, loaded = m.LoadAndDelete("k")
ok = m.CompareAndDelete("k", 4)
m.Range(func(k, v any) bool { return true })
m.Clear()                                 // 1.23+
```

适用场景（文档原文）只有两类：

1. key 只写一次、读很多次（只增不减的 cache）；
2. 多个 goroutine 读写**互不相交**的 key 集合。

其他情况优先用 `map` + `RWMutex`，或者分片锁。注意 `sync.Map` **没有 `Len()`**——真要计数只能 `Range` 数，而且数出来的值天生过时。

### 1.7 选型：实测数字

`go test -bench . -benchmem ./sync`（go1.26.3 darwin/amd64, i5-1038NG7, GOMAXPROCS=8）：

```text
# 无竞争（单 goroutine 反复加解锁）
BenchmarkMutexUncontended-8              18.28 ns/op
BenchmarkRWMutexWriteUncontended-8       31.48 ns/op   ← 写锁比 Mutex 贵：内部还有一层 Mutex
BenchmarkRWMutexReadUncontended-8        13.57 ns/op
BenchmarkAtomicUncontended-8              7.06 ns/op
BenchmarkChanAsMutex-8                   42.00 ns/op   ← 用 chan 当锁：2 倍多的开销

# 有竞争（RunParallel，8 线程抢同一个东西）
BenchmarkMutexContended-8                59.59 ns/op
BenchmarkAtomicContended-8               17.83 ns/op
BenchmarkShardedCounter-8                 2.23 ns/op   ← 分片 + cache line padding
BenchmarkShardedCounterFalseSharing-8    17.24 ns/op   ← 只差一个 padding，慢 7.7 倍

# map：RWMutex+map vs sync.Map（1024 个 key，8 线程）
BenchmarkRWMapReadOnly-8                 54.69 ns/op
BenchmarkSyncMapReadOnly-8                7.13 ns/op   ← 快 7.7x：读路径全 atomic 无锁
BenchmarkRWMapMostlyRead-8               52.78 ns/op   （10% 写）
BenchmarkSyncMapMostlyRead-8             13.21 ns/op
BenchmarkRWMapHeavyWrite-8               67.44 ns/op   （50% 写）
BenchmarkSyncMapHeavyWrite-8             34.64 ns/op   31 B/op  1 allocs/op

# Once 的快路径基本免费
BenchmarkOnceDo-8                         0.93 ns/op
BenchmarkAtomicBoolCheck-8                0.67 ns/op
BenchmarkOnceValue-8                      4.05 ns/op
```

几个结论：

- **`RWMutex` 不是"读多就一定更快"**：无竞争时读锁 13.57ns vs `Mutex` 18.28ns，差距很小；写锁反而比 `Mutex` 贵近一倍（`RWMutex` 内部第一步就是 `rw.w.Lock()`）。临界区极短时 `Mutex` 往往更快，因为 `RWMutex` 的读路径也要动 `readerCount` 这个共享的原子变量，一样有 cache line 争抢。
- **换成 hash-trie 之后的 `sync.Map` 强了很多**：1.24 之前"写多就别用 sync.Map"是常识，现在这个实测里连 50% 写都还比 `RWMutex+map` 快一倍。但它仍然有 `any` 装箱开销（heavy write 那行有 1 allocs/op）和类型不安全的问题。
- **伪共享的代价是实打实的**：同样的分片计数器，加不加 64 字节 padding 差 7.7 倍。
- 结论顺序：**能用 atomic 就别用锁；能分片就别用全局锁；能不共享就别共享**。

## 二、底层原理

### 2.1 Mutex 的数据结构与状态位

Go 1.24 起 `sync.Mutex` 只是壳（`sync/mutex.go:30`）：

```go
type Mutex struct {
    _  noCopy
    mu isync.Mutex     // internal/sync.Mutex
}
```

真身在 `internal/sync/mutex.go:21`，一共 **8 字节**：

```go
type Mutex struct {
    state int32     // 状态位 + 等待者计数
    sema  uint32    // 信号量：runtime 用它的地址做 key 去 semtable 找等待队列
}

const (
    mutexLocked      = 1 << 0   // 已加锁
    mutexWoken       = 1 << 1   // 已有等待者被唤醒，Unlock 不必再唤人
    mutexStarving    = 1 << 2   // 饥饿模式
    mutexWaiterShift = 3        // state >> 3 = 等待者数量
    starvationThresholdNs = 1e6 // 1ms
)
```

### 2.2 Lock 的三段路径

```go
func (m *Mutex) Lock() {
    // ① 快路径：一次 CAS，可被内联到调用点
    if atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) {
        return
    }
    m.lockSlow()   // ② 慢路径，单独函数以便快路径能内联
}
```

`lockSlow`（`internal/sync/mutex.go:95`）是一个大 for 循环，做三件事：

**② 自旋**。条件很严格（`runtime/proc.go:7938` `internal_sync_runtime_canSpin`）：

```go
if i >= active_spin ||                    // active_spin = 4，最多自旋 4 轮
   numCPUStartup <= 1 ||                  // 单核不自旋
   gomaxprocs <= npidle+nmspinning+1 {    // 没有其他正在跑的 P 就不自旋
    return false
}
if p := getg().m.p.ptr(); !runqempty(p) { // 本地队列有活就别自旋，去干活
    return false
}
```

每轮自旋执行 `active_spin_cnt = 30` 次 `PAUSE` 指令。**饥饿模式下不自旋**（所有权是交棒的，抢不到）。

**③ 排队**。CAS 更新 state（加上 `mutexLocked` / 等待者 +1 / 可能置 `mutexStarving`）之后：

```go
queueLifo := waitStartTime != 0                  // 老等待者插队头（LIFO），新来的排队尾
runtime_SemacquireMutex(&m.sema, queueLifo, 2)   // 挂起当前 G
starving = starving || nanotime()-waitStartTime > starvationThresholdNs
```

**④ 两种模式**（注释写得很清楚，值得原文理解）：

| | 正常模式 | 饥饿模式 |
| --- | --- | --- |
| 唤醒后 | 等待者和新来的 goroutine **公平竞争** | 所有权**直接交棒**给队首，不竞争 |
| 新来的 goroutine | 可以直接抢锁、可以自旋 | 必须排到队尾，不许自旋 |
| 优势 | 吞吐高（新来的已在 CPU 上，无需上下文切换） | 尾延迟可控 |
| 进入条件 | — | 某个等待者等待超过 **1ms** |
| 退出条件 | — | 拿到锁的是最后一个等待者，或它本次等待 < 1ms |

`notes/sync/` 里的实测（200ms、4 个 worker 抢同一把锁）：

```text
200ms 内各 worker 抢到锁的次数: [147796 180369 154564 154801]（合计 637530）
各自遇到的最长一次等待:         [6.5ms 5.7ms 6.6ms 12.3ms]
```

即使有饥饿模式兜底，单次等待仍可能到十几毫秒——**饥饿模式保证的是"不会无限期饿死"，不是"低延迟"**。

### 2.3 Unlock：为什么快路径是 Add 而不是 CAS

```go
func (m *Mutex) Unlock() {
    new := atomic.AddInt32(&m.state, -mutexLocked)  // 直接减掉 locked 位
    if new != 0 {
        m.unlockSlow(new)
    }
}
```

用 `Add` 而不是 CAS，是因为解锁方**一定**持有锁，`mutexLocked` 位一定是 1，无需比较。`new != 0` 说明还有等待者或处于饥饿模式，才进慢路径：

- 正常模式：CAS 抢到"唤醒别人的权利"（等待者 -1、置 `mutexWoken`），然后 `Semrelease(&m.sema, false, 2)`——**handoff = false**，被唤醒的 G 还要自己去抢。
- 饥饿模式：`Semrelease(&m.sema, true, 2)`——**handoff = true**，直接把所有权交给队首，并让出时间片。注意此时 `mutexLocked` 并没有被置上，靠 `mutexStarving` 挡住新来的。

`unlockSlow` 第一件事是检查 `(new+mutexLocked)&mutexLocked == 0` → `fatal("sync: unlock of unlocked mutex")`。

### 2.4 sema：等待队列长什么样

`m.sema` 只是一个 `uint32` 字段，真正的等待队列在 runtime 的全局表里（`runtime/sema.go:40`）：

```go
type semaRoot struct {
    lock  mutex
    treap *sudog        // 平衡树（treap），按 sudog.elem（也就是 &m.sema）排序
    nwait atomic.Uint32
}

const semTabSize = 251   // 质数，避免和用户的地址模式相关

var semtable [semTabSize]struct {
    root semaRoot
    pad  [cpu.CacheLinePadSize - unsafe.Sizeof(semaRoot{})]byte  // 每个桶独占 cache line
}

func (t *semTable) rootFor(addr *uint32) *semaRoot {
    return &t[(uintptr(unsafe.Pointer(addr))>>3)%semTabSize].root
}
```

要点：

- **按 `&m.sema` 的地址哈希到 251 个桶**，桶内是 treap（O(log n) 找到具体地址），**同一个地址上的多个等待者挂成 O(1) 的链表**（`sudog.waitlink`）。两层结构是为了修 `issue 17953`：大量不同 mutex 哈希到同一个桶时，退化成 O(n)。
- 挂起的是 **G 不是线程**：`sudog` 里存着 `g`，M 会去跑别的 G（和 channel 阻塞是同一套机制，见 chan.md 2.7）。
- 所以**锁竞争的真实代价 = 一次 G 切换 + 一次 semtable 加锁 + 可能的 cache line 弹跳**，这就是为什么 59ns 的争抢版本比 18ns 的无争抢版本贵 3 倍多。

### 2.5 RWMutex：一个原子变量搞定读写互斥

```go
type RWMutex struct {
    w           Mutex        // 写者之间互斥
    writerSem   uint32       // 写者等读者
    readerSem   uint32       // 读者等写者
    readerCount atomic.Int32 // 读者数；为负表示有写者 pending
    readerWait  atomic.Int32 // 写者需要等待的"离场读者"数
}
const rwmutexMaxReaders = 1 << 30
```

核心技巧是 `readerCount` 一个变量表达两种状态：

```go
func (rw *RWMutex) RLock() {
    if rw.readerCount.Add(1) < 0 {          // 负数 = 有写者在等
        runtime_SemacquireRWMutexR(&rw.readerSem, false, 0)
    }
}

func (rw *RWMutex) Lock() {
    rw.w.Lock()                                                  // ① 先和其他写者分胜负
    r := rw.readerCount.Add(-rwmutexMaxReaders) + rwmutexMaxReaders  // ② 减去 1<<30 宣告"我来了"
    if r != 0 && rw.readerWait.Add(r) != 0 {                     // ③ 等已在场的 r 个读者离场
        runtime_SemacquireRWMutex(&rw.writerSem, false, 0)
    }
}

func (rw *RWMutex) RUnlock() {
    if r := rw.readerCount.Add(-1); r < 0 {
        rw.rUnlockSlow(r)     // 最后一个离场的读者负责唤醒写者
    }
}
```

- **写优先**由步骤 ② 实现：`readerCount` 一旦变负，后续所有 `RLock` 都会去 `readerSem` 排队。
- `readerWait` 和 `readerCount` 必须分开：前者是"写者还要等几个"，后者是"当前有几个读者（含排队的）"。
- `Unlock` 里 `readerCount.Add(rwmutexMaxReaders)` 恢复，然后按数量 `Semrelease` 唤醒所有排队读者，最后才 `rw.w.Unlock()` 放行下一个写者——**读者优先于下一个写者被唤醒**，避免写者连续霸占。

### 2.6 WaitGroup：一个 uint64 装三样东西

```go
type WaitGroup struct {
    noCopy noCopy
    // bits[63:32] counter（任务数）
    // bits[32]    synctest bubble 标记（1.25 新增）
    // bits[31:0]  waiter count（Wait 的调用者数）
    state atomic.Uint64
    sema  uint32
}
```

- `Add(delta)` → `state.Add(uint64(delta) << 32)`：**一次原子操作同时读到 counter 和 waiter 数**，这是把两个计数塞进一个 64 位字的全部动机。
- counter 归零时，`Add` 负责把 `state` 清零并 `Semrelease` 唤醒所有 waiter。
- counter 变负 → `panic("sync: negative WaitGroup counter")`（**这个是 panic，可以 recover**，和 Mutex 的 fatal 不同）。
- `Wait` 返回前又被 `Add` → `panic("sync: WaitGroup misuse: Add called concurrently with Wait")`。

### 2.7 Once：为什么必须是"双检查 + 延后置位"

```go
func (o *Once) Do(f func()) {
    if !o.done.Load() {      // 快路径：一次原子读，能被内联
        o.doSlow(f)
    }
}

func (o *Once) doSlow(f func()) {
    o.m.Lock()
    defer o.m.Unlock()
    if !o.done.Load() {          // 锁内再判一次
        defer o.done.Store(true) // 关键：f 返回之后才置位
        f()
    }
}
```

源码注释里专门写了一个**反例**：

```go
if o.done.CompareAndSwap(false, true) { f() }   // ✗ 错的
```

它保证了"只执行一次"，但**不保证"Do 返回时 f 已执行完"**：并发时输家会立刻返回，然后用到还没初始化好的东西。这是 `Once` 语义中最容易被自己实现错的一点。

### 2.8 Cond：ticket 机制

`Cond` 的等待队列是 `runtime.notifyList`（`runtime/sema.go:547`）：

```go
type notifyList struct {
    wait   atomic.Uint32  // 下一个等待者的票号（锁外原子递增）
    notify uint32         // 下一个该被通知的票号（持锁写）
    lock   mutex
    head, tail *sudog
}
```

`Wait` 的三步：`runtime_notifyListAdd` 领票号 → `c.L.Unlock()` → `runtime_notifyListWait(t)` 挂起。票号机制解决的是"**领票之后、真正挂起之前，Signal 就来了**"这个竞态：`notifyListWait` 里先判 `less(t, l.notify)`，如果自己的票已经被叫过就直接返回，不会丢通知。

`Cond` 还有个 `copyChecker`（`sync/cond.go:97`）：它把自己的地址存在自己里面，一旦被拷贝，地址就对不上，`panic("sync.Cond is copied")`。这是 sync 包里少见的**运行时**拷贝检测（`Mutex` 的 `noCopy` 只是给 `go vet` 看的静态标记）。

### 2.9 sync.Map：1.24 之后是 concurrent hash-trie

**老实现（1.9–1.23，很多面试题还在问这个）**：`read`（原子读、只读）+ `dirty`（加锁写）双 map，`misses` 攒够一次就把 dirty 提升为 read；删除是打 `expunged` 标记的延迟删除。缺点：写多时反复 promote 要**整表拷贝**，内存和延迟都不可控。

**新实现（1.24+，`internal/sync/hashtriemap.go`）**：

```go
type HashTrieMap[K comparable, V any] struct {
    inited   atomic.Uint32
    initMu   Mutex
    root     atomic.Pointer[indirect[K, V]]
    keyHash  hashFunc     // 从 map 类型的 rtype 里借来的哈希函数
    valEqual equalFunc
    seed     uintptr
}

const nChildrenLog2 = 4          // 16 路分支——注释说这是 load 性能的甜点
const nChildren     = 1 << 4

type indirect[K, V] struct {     // 内部节点
    node[K, V]
    dead     atomic.Bool
    mu       Mutex                              // 只保护这一个节点的 children
    parent   *indirect[K, V]
    children [nChildren]atomic.Pointer[node[K, V]]
}

type entry[K, V] struct {        // 叶子节点
    node[K, V]
    overflow atomic.Pointer[entry[K, V]]   // 哈希冲突链
    key      K
    value    V
}
```

`Load` 的路径（`hashtriemap.go:63`）：算一次哈希，每层取 4 位（`hash >> hashShift & 0xf`）往下走，遇到 `entry` 就在 overflow 链上比 key。**全程只有 `atomic.Pointer.Load`，一把锁都不加**——这就是实测里只读场景比 `RWMutex+map` 快 7.7 倍的原因（后者的 `RLock` 要动同一个 `readerCount`，8 个核在抢同一条 cache line）。

写路径只锁**目标叶子所在的那个 `indirect` 节点**（`i.mu`），因此不同 key 的写基本互不影响；`entry` 是不可变的，更新 value 靠**创建新 entry 替换指针**（copy-on-write），所以读者永远看到一个自洽的值。删空的子树用 `dead` 标记 + 自底向上收缩。`Clear()` 干脆直接换一个新根（`root.Store(newIndirectNode(nil))`），老树整棵交给 GC。

顺带一提：`unique` 包（1.23 的 `unique.Handle`）和 `weak` 指针也建立在这个 hash-trie 上，这才是它被写出来的初衷，`sync.Map` 是搭便车换掉了实现。

### 2.10 关键源码索引

| 位置 | 内容 |
| --- | --- |
| `sync/mutex.go:30` | `Mutex` 外壳（noCopy + isync.Mutex） |
| `internal/sync/mutex.go:21,95,187,202` | state 定义、`lockSlow`、`Unlock`、`unlockSlow` |
| `runtime/proc.go:7938` | `canSpin` 的四个条件 |
| `runtime/lock_spinbit.go:54` | `active_spin = 4`、`active_spin_cnt = 30` |
| `runtime/sema.go:40,49,164` | `semaRoot`、`semTabSize = 251`、`semacquire1` |
| `runtime/sema.go:547` | `notifyList`（Cond 的队列） |
| `sync/rwmutex.go:67,143,192` | `RLock`、`Lock`、`Unlock` |
| `sync/waitgroup.go:48,77,236` | state 布局、`Add`、`Go` |
| `sync/once.go:52,73` | `Do` 与 `doSlow`（含错误实现的反例注释） |
| `sync/oncefunc.go:11,46,80` | `OnceFunc`/`OnceValue`/`OnceValues` 的 panic 缓存 |
| `internal/sync/hashtriemap.go:21,63,530,541,564` | hash-trie 结构、`Load`、`nChildren`、`indirect`、`entry` |

## 三、常见陷阱

### 3.1 拷贝锁（最高频的并发 bug）

```go
type Config struct {
    mu   sync.Mutex
    data map[string]string
}

func (c Config) Get(k string) string {   // ✗ 值接收者：每次调用拷贝一份锁
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.data[k]
}
```

**运行时不会报任何错，只是锁完全失效**。靠 `go vet` 的 copylocks 检查拦：

```text
Get passes lock by value: Config contains sync.Mutex
assignment copies lock value to c2: sync.Mutex contains sync.noCopy
range var c copies lock value
```

`sync.Mutex` 里那个 `noCopy` 字段就是给 vet 看的标记（它实现了 `Lock`/`Unlock` 方法，因此被 vet 识别为"锁"），没有任何运行时作用。规则：**含锁的 struct 一律用指针传递**。

### 3.2 `fatal error` 不是 `panic`

```go
var mu sync.Mutex
defer func() { recover() }()   // 救不了
mu.Unlock()                    // fatal error: sync: unlock of unlocked mutex
```

sync 包里区分得很清楚：

| 错误 | 机制 | 能否 recover |
| --- | --- | --- |
| unlock of unlocked mutex | `fatal()` | **不能**，进程必死 |
| RUnlock of unlocked RWMutex | `fatal()` | **不能** |
| inconsistent mutex state | `throw()` | **不能** |
| negative WaitGroup counter | `panic()` | 能（但别依赖） |
| WaitGroup misuse | `panic()` | 能 |
| sync.Cond is copied | `panic()` | 能 |

用 `fatal` 的理由：锁状态已经不自洽了，继续跑只会产生更难查的数据竞争。

### 3.3 RWMutex 递归读锁

```go
func (s *Store) Get(k string) string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.getLocked(k)     // 这个方法里又 RLock 了 -> 有写者时必死锁
}
```

只要在两次 `RLock` 之间挤进一个 `Lock`，就是死锁：内层 `RLock` 等写者，写者等外层 `RUnlock`。**这个 bug 在低并发下几乎不复现**。解法是约定"带 `Locked` 后缀的方法假设调用者已持锁"，或者干脆用 `Mutex`。

### 3.4 锁粒度：`defer Unlock` 会把慢操作圈进临界区

```go
func (c *cache) Get(k string) string {
    c.mu.Lock()
    defer c.mu.Unlock()      // 锁一直持到函数返回
    v := c.m[k]
    return slowRPC(v)        // ✗ RPC 期间别人全在等
}
```

实测（8 个并发，临界区里 sleep 1ms）：`defer` 版 10ms，手动提前 `Unlock` 版 1ms。`defer Unlock` 很安全，但**函数里有 IO / 大量计算 / 调用外部代码时必须手动缩小临界区**（或者拆成两个函数，让持锁的那个足够短）。

另一个变体是**在锁里调用可能反过来加锁的回调**：

```go
c.mu.Lock()
for _, cb := range c.callbacks { cb() }   // ✗ 回调里再操作 c 就死锁
c.mu.Unlock()
```

正确做法：锁内只拷一份回调列表，锁外再调。

### 3.5 Once 相关

```go
var once sync.Once
once.Do(func() { once.Do(func() {}) })   // ✗ 自死锁：doSlow 已持 o.m
```

以及"初始化失败无法重试"：

```go
var db *sql.DB
var once sync.Once
func DB() *sql.DB {
    once.Do(func() {
        var err error
        db, err = sql.Open(...)
        if err != nil { log.Println(err) }   // ✗ 失败了，但 done 已置位，永远不会再试
    })
    return db
}
```

要重试就别用 `Once`：自己写 `mu + inited bool`，失败时不置位。

### 3.6 WaitGroup 的四种误用

```go
// ✗ ① Add 在 goroutine 内部：Wait 可能先跑完
go func() { wg.Add(1); defer wg.Done(); work() }()

// ✗ ② 按值传递
func run(wg sync.WaitGroup) { defer wg.Done() }   // vet: passes lock by value

// ✗ ③ 循环里 Add(len) 但某条分支提前 return，Done 没配平 -> Wait 永久阻塞
// ✗ ④ Wait 还没返回就复用同一个 wg -> panic: WaitGroup misuse
```

1.25 之后前两个基本被 `wg.Go(f)` 消灭了，优先用它。

### 3.7 goroutine 泄漏 vs 锁泄漏

`Lock` 之后在中间 `return`（或 panic）却没有 `Unlock`，锁就永久被持有——**比 goroutine 泄漏更致命**，因为所有后续访问者全部挂死，而 `pprof` 的 goroutine profile 里只会看到一堆 `semacquire`，看不出谁是元凶。

排查手段：

```bash
GODEBUG=asyncpreemptoff=1 go run ...      # 排除抢占干扰
kill -QUIT <pid>                          # 打印全部 goroutine 栈，找持锁者
go tool pprof http://.../debug/pprof/mutex   # 需要先 runtime.SetMutexProfileFraction(n)
go tool pprof http://.../debug/pprof/block   # 需要 runtime.SetBlockProfileRate(n)
```

这也是"能用 `defer Unlock` 就用"和"临界区要短"之间的真实张力：**优先保证正确性（defer），再靠拆函数来缩小临界区**。

### 3.8 sync.Map 的 `LoadOrStore` 会白算一次

```go
m.LoadOrStore("k", expensive())   // ✗ expensive() 是实参，先求值再传进去
```

实测 8 个 goroutine 并发调用，`expensive()` 被调了 8 次，只有 1 次的结果被存下来。想"只构造一次"：先 `Load`，miss 了再 `LoadOrStore`；或者存 `*sync.OnceValue` 之类的惰性值；或者上 `singleflight`。

### 3.9 把 sync.Map 当普通 map 用

- **没有 `Len()`**：`Range` 数出来的值天生过时。
- **`Range` 不是快照**：遍历期间的修改可能被看到、也可能看不到，且顺序不保证。
- **key/value 是 `any`**：装箱开销 + 类型不安全。自己包一层泛型 wrapper（内部仍是 `sync.Map`，只是把断言收敛到一处）。
- **不能拷贝**（`noCopy`）。

### 3.10 用 channel 当锁

```go
sem := make(chan struct{}, 1)
sem <- struct{}{}
// 临界区
<-sem
```

能跑，但实测 42ns vs `Mutex` 18ns，**贵一倍多**，而且没有饥饿模式保证、没有 `go vet` 支持、栈上看不出是在等锁。channel 适合传递**所有权和数据**，不适合当纯互斥量。反过来，"限制并发数"这种带计数的场景 channel 才是对的（`sync` 里没有 Semaphore，官方在 `golang.org/x/sync/semaphore`）。

### 3.11 伪共享（false sharing）

```go
type stats struct {
    hits   atomic.Int64   // 8 字节
    misses atomic.Int64   // 紧挨着，落在同一条 64 字节 cache line 上
}
```

两个字段被不同核高频写时，cache line 在核间反复弹跳。实测 64 个分片计数器，加 padding 2.23ns/op、不加 17.24ns/op，**差 7.7 倍**。解法是 padding 到 cache line 大小（runtime 里 `semtable` 和 `sync.Pool` 的 `poolLocal` 都是这么干的，见 pool.md 2.3）。

### 3.12 该用锁的地方用了 atomic

```go
var total atomic.Int64
var count atomic.Int64
// 想算平均值
avg := total.Load() / count.Load()   // ✗ 两次 Load 之间可能被改，得到的不是一致快照
```

atomic 只保证**单个变量**的单次操作原子，**多个变量的一致性必须靠锁**（或者把它们塞进一个 struct 用 `atomic.Pointer` 整体换）。

### 3.13 `sync` 之外的常用件

标准库 `sync` 故意保持很小，下面这些在 `golang.org/x/sync`：

| 组件 | 用途 |
| --- | --- |
| `errgroup.Group` | WaitGroup + 首个 error + `context` 取消；`SetLimit(n)` 限并发 |
| `semaphore.Weighted` | 带权重的信号量，支持 `Acquire(ctx, n)` 阻塞等待 |
| `singleflight.Group` | 相同 key 的并发请求合并成一次（缓存击穿的标准解法） |

`errgroup` 几乎是"并发调 N 个下游"的默认答案，比手写 `WaitGroup + errChan` 少一半代码。

## 四、常见面试题

**1. `sync.Mutex` 的两种模式是什么？为什么需要饥饿模式？**
正常模式下等待者被唤醒后要和新来的 goroutine 竞争，新来的已经在 CPU 上跑、还能自旋，所以经常赢——吞吐高，但等待者可能被反复插队。某个等待者等待超过 1ms 就把 mutex 切进饥饿模式：`Unlock` 直接把所有权交棒给队首（`Semrelease` handoff=true），新来的一律排队尾且不许自旋，从而限制尾延迟。拿到锁的是最后一个等待者、或本次等待不足 1ms 时退出饥饿模式（见 2.2）。

**2. `Mutex` 的 state 里有什么？为什么 `Unlock` 用 `Add` 而 `Lock` 用 `CAS`？**
`state int32`：bit0 locked、bit1 woken、bit2 starving、bit3+ 等待者数。`Lock` 不知道当前状态，必须 CAS 才能保证"只有一个赢家"；`Unlock` 的调用者一定持锁，`mutexLocked` 位必然是 1，直接 `Add(-1)` 更便宜，之后判 `new != 0` 再决定是否唤人（见 2.1、2.3）。

**3. Go 的 Mutex 是可重入的吗？为什么不设计成可重入？**
不可重入，同一 goroutine 二次 `Lock` 自死锁。原因：Go 的锁不记录持有者（文档明确"锁不与特定 goroutine 关联"，允许 A 锁 B 解），要支持重入就得存 goroutine 标识 + 计数，`Mutex` 会从 8 字节膨胀、快路径也不再是一次 CAS。Russ Cox 的观点更根本：需要重入通常意味着临界区划分错了。

**4. goroutine 阻塞在 Mutex 上时，OS 线程也被阻塞了吗？**
没有。`runtime_SemacquireMutex` 把 G 包进 `sudog` 挂到 `semtable` 的队列上并让出 M，M 去跑别的 G。等待队列是全局 251 个桶的哈希表，桶内 treap 按 `&m.sema` 排序，同地址的等待者串成链表（见 2.4）。

**5. `RWMutex` 一定比 `Mutex` 快吗？**
不一定。实测无竞争时 `RLock/RUnlock` 13.6ns vs `Lock/Unlock` 18.3ns，差距很小；写锁 31.5ns 反而比 `Mutex` 贵近一倍（内部第一步就是 `rw.w.Lock()`）。而且读路径也要原子改同一个 `readerCount`，多核下同样有 cache line 争抢。只有**临界区足够长、读远多于写**时 `RWMutex` 才明显划算（见 1.7）。

**6. `RWMutex` 怎么防止写者被饿死？为什么读锁不能递归？**
`Lock` 里 `readerCount.Add(-1<<30)` 把计数变负，后续 `RLock` 看到负数就去 `readerSem` 排队，即"写者 pending 时新读者一律等待"。副作用就是读锁不可递归：外层持读锁、中间来了写者、内层 `RLock` 排队 → 内层等写者、写者等外层，死锁（见 1.2、2.5）。

**7. `WaitGroup` 为什么用一个 uint64 存状态？**
`state` 高 32 位是 counter、低 31 位是 waiter 数（1.25 起中间 1 位是 synctest 标记）。一次 `atomic.Add` 就能同时更新/读到两个计数，避免"counter 归零"和"有新 waiter 进来"之间的竞态。counter 变负 panic，`Wait` 未返回就复用 panic `WaitGroup misuse`（见 2.6）。

**8. `wg.Go(f)` 和手写 `wg.Add(1); go func(){defer wg.Done(); f()}()` 有什么区别？**
语义基本一致，但 `wg.Go` 里如果 `f` panic，它**故意不调 `Done`**而是重新 panic——防止 `Wait` 被解除阻塞后 main 抢先 `os.Exit(0)` 把崩溃现场吞掉。所以用 `wg.Go` 的前提是 `f` 内部自己兜住 panic（见 1.3）。

**9. `sync.Once` 为什么不能用 `CompareAndSwap(false, true)` 实现？**
那样只保证"f 只执行一次"，不保证"`Do` 返回时 f 已执行完"：并发时 CAS 的输家会立刻返回，拿到未初始化的状态。所以实现是"原子读 done 的快路径 + 锁内双检查 + `defer o.done.Store(true)` 延后置位"（源码注释里就写着这个反例，见 2.7）。

**10. `Once.Do` 里 panic 了会怎样？**
`done` 仍然被置位（`defer` 在展开时执行），**初始化不会重试**，后续 `Do` 直接返回。`OnceFunc`/`OnceValue` 更明确：记住 panic 值，之后每次调用都 panic 同一个值。需要可重试的初始化不能用 `Once`（见 1.4、3.5）。

**11. `sync.Cond` 的 `Wait` 为什么必须写在 for 循环里？**
`Signal`/`Broadcast` 只表示"有事发生"，被唤醒时条件可能已被别的 goroutine 消费掉（虚假唤醒）。另外 `Wait` 必须持锁调用，内部会 `Unlock` 再挂起、返回前重新 `Lock`。实现上用 ticket（`notifyList.wait/notify`）解决"领票后、挂起前收到 Signal"的竞态（见 1.5、2.8）。

**12. `sync.Map` 的底层实现是什么？（注意版本）**
Go 1.24 起是 **concurrent hash-trie**（`internal/sync.HashTrieMap`）：16 路分支，`Load` 全程只做 `atomic.Pointer.Load` 完全无锁；写只锁目标叶子所在的那个内部节点；`entry` 不可变，更新靠 copy-on-write 换指针；`Clear` 直接换新根。1.23 及之前才是 read/dirty 双 map + misses 提升 + expunged 延迟删除那套（见 2.9）。

**13. 什么场景该用 `sync.Map`，什么场景该用 `map + RWMutex`？**
文档给的两类：key 写一次读多次的只增 cache；多 goroutine 操作互不相交的 key。其他情况优先 `map + RWMutex` 或分片锁——`sync.Map` 的 `any` 装箱、缺少 `Len()`、`Range` 无快照语义都是实际成本。不过 1.24 换成 hash-trie 之后它的写性能已经好很多，实测 50% 写仍快于 `RWMutex+map`（见 1.7、3.9）。

**14. 拷贝一个 `sync.Mutex` 会发生什么？为什么运行时不报错？**
拷贝出来的是两把独立的锁，互斥完全失效，运行时**不会有任何提示**。防线只有编译期的 `go vet` copylocks（靠 `noCopy` 字段识别）。`sync.Cond` 是唯一有运行时拷贝检测的（`copyChecker`，`panic("sync.Cond is copied")`，见 3.1、2.8）。

**15. `Unlock` 一个未加锁的 mutex 会 panic 吗？**
不是 panic，是 `fatal error: sync: unlock of unlocked mutex`，`recover` 无效、进程必死。sync 包里锁状态类错误都用 `fatal()`/`throw()`，只有 WaitGroup 计数、Cond 拷贝这类"用法错误"才用 `panic()`（见 3.2）。

**16. 什么是伪共享？Go 里哪些地方在防它？**
不同核高频写同一条 cache line 上的不同变量，导致 cache line 反复失效。实测分片计数器加/不加 64 字节 padding 差 7.7 倍。runtime 里 `semtable` 的每个桶、`sync.Pool` 的 `poolLocal` 都填充到 cache line 大小来规避（见 3.11、pool.md 2.3）。

**17. 只有一个变量要保护，用 atomic 还是 Mutex？多个变量呢？**
单变量单操作用 atomic（实测 7ns vs 18ns，竞争下 17.8ns vs 59.6ns）。多个变量要**一致快照**时必须用锁，或者把它们打包进一个 struct 用 `atomic.Pointer[T]` 整体替换。`total/count` 两次 `Load` 算平均值是典型错误（见 3.12）。

**18. `errgroup` 比 `WaitGroup` 多了什么？**
收集第一个非 nil error、内置 `context` 取消（`WithContext`：任一子任务出错就 cancel 其余）、`SetLimit(n)` 限制并发。它在 `golang.org/x/sync`，不在标准库，但基本是"并发调 N 个下游"的默认写法（见 3.13）。
