# sync.Pool

> 环境：`go version go1.26.3 darwin/arm64`（Apple M4，`GOMAXPROCS=10`）。源码位置：`sync/pool.go`（318 行）、`sync/poolqueue.go`（302 行），GC 回调在 `runtime/mgc.go`。
>
> 一句话定位：**`sync.Pool` 是一个「per-P 分片 + 无锁队列 + 两级 GC 淘汰」的临时对象缓存，用来削减 GC 压力，不是资源池、不是连接池、不保证任何对象存活。**

## 一、基础使用

### 1.1 最小示例

```go
var bufPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) }, // 可选：池空时兜底构造
}

func handle(w io.Writer, data []string) {
    buf := bufPool.Get().(*bytes.Buffer) // 取（可能是复用的，也可能是 New 出来的）
    buf.Reset()                          // 关键：必须自己重置状态
    defer bufPool.Put(buf)               // 还

    for _, s := range data {
        buf.WriteString(s)
    }
    w.Write(buf.Bytes())
}
```

三个动作缺一不可：**Get → Reset → Put**。`Pool` 不会帮你清理对象，`Get` 拿到的可能是别人用过的脏对象。

### 1.2 New 的语义

```go
var p sync.Pool             // New == nil
fmt.Println(p.Get())        // <nil>：池空且没有 New，直接返回 nil

p.New = func() any { return 42 }
fmt.Println(p.Get())        // 42
```

- `New == nil` 时，池空 `Get()` 返回 `nil`，调用方必须自己判空。
- `New` 只在**所有取值路径都失败之后**才调用（源码 `pool.go:154`），所以它不参与任何锁竞争。
- `New` 不能与 `Get` 并发修改（文档明确要求），实践上都是在包级变量初始化时一次写定。

### 1.3 三种典型写法对比

```go
// 写法一：New 返回指针（推荐）
var p1 = sync.Pool{New: func() any { return new(bytes.Buffer) }}
buf := p1.Get().(*bytes.Buffer)

// 写法二：New 返回值类型（❌ Put 时会额外分配，见 4.1）
var p2 = sync.Pool{New: func() any { return make([]byte, 0, 4096) }}
b := p2.Get().([]byte)

// 写法三：值类型用指针包一层（✅ 正确的切片池写法）
var p3 = sync.Pool{New: func() any { s := make([]byte, 0, 4096); return &s }}
sp := p3.Get().(*[]byte)
*sp = (*sp)[:0]
```

### 1.4 泛型封装（消掉类型断言）

标准库没有泛型版 `Pool`，自己包一层很常见：

```go
type TypedPool[T any] struct {
    p   sync.Pool
    new func() *T
}

func NewTypedPool[T any](newFn func() *T) *TypedPool[T] {
    tp := &TypedPool[T]{new: newFn}
    tp.p.New = func() any { return tp.new() }
    return tp
}

func (tp *TypedPool[T]) Get() *T  { return tp.p.Get().(*T) } // 断言收敛到一处
func (tp *TypedPool[T]) Put(x *T) { tp.p.Put(x) }
```

注意：`sync.Pool` 内部存的是 `any`，泛型封装只能消除**调用方**的断言样板，无法消除装箱本身；所以池化对象类型仍应是指针（`*T`）。

### 1.5 收益有多大（实测）

```go
func work(b *bytes.Buffer) {
    for i := 0; i < 100; i++ { b.WriteString("hello world") }
}
// BenchmarkNoPool:   buf := new(bytes.Buffer); work(buf)
// BenchmarkWithPool: buf := pool.Get().(*bytes.Buffer); buf.Reset(); work(buf); pool.Put(buf)
```

`go test -bench . -benchmem -benchtime=300000x`（10 核并行，`b.RunParallel`）：

```
BenchmarkNoPool-10       574.5 ns/op    4032 B/op    6 allocs/op
BenchmarkWithPool-10      38.62 ns/op      0 B/op    0 allocs/op
```

约 15× 提速、0 分配。收益来自两处：省掉 `make` 本身，以及**省掉这些对象后续被 GC 扫描/回收的开销**（后者往往才是大头，benchmark 里体现为 GC 频率下降）。

关键前提：对象要**有一定构造成本**且**生命周期短、复用率高**。小对象（几十字节）池化后往往比直接 `new` 还慢，因为 `Get/Put` 自身有 `procPin`、原子操作、接口装箱的固定开销（见 4.7）。

### 1.6 标准库中的用法（都是很好的参考样本）

| 位置                                                                                   | 池化对象                          | 值得学的点                                                                                                           |
| -------------------------------------------------------------------------------------- | --------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `fmt/print.go:146` `ppFree`                                                            | `*pp`（printer 状态机）           | `free()` 里对 buffer 做 **64KB 硬上限**，超了就丢 buffer 只回收 printer（`golang.org/issue/23199`）                  |
| `encoding/json/encode.go:313` `encodeStatePool`                                        | `*encodeState`                    | 每次 Marshal 复用编码缓冲                                                                                            |
| `encoding/json/scanner.go:89` `scannerPool`                                            | `*scanner`                        | 解析器状态机复用                                                                                                     |
| `net/http/server.go:827` `bufioReaderPool` / `bufioWriter2kPool` / `bufioWriter4kPool` | `*bufio.Reader` / `*bufio.Writer` | **按容量分成多个池**（2k/4k），保证同一个池里对象大小均匀                                                            |
| `net/http/h2_bundle.go:9136` `http2bufPools [7]sync.Pool`                              | `*[]byte`                         | 按 16KB/32KB/…/512KB **分 7 档**，`bufPoolIndex(size)` 用 `bits.Len` 算档位；取出后还要校验 `len(*bp) >= scratchLen` |
| `net/http/header.go:160` `headerSorterPool`                                            | `*headerSorter`                   | 排序临时结构复用                                                                                                     |

`fmt` 和 `h2` 这两个例子回答了同一个问题的两种解法：**「大小不均匀的对象怎么池化」**——要么设上限丢弃超大的（`fmt`），要么按大小分级建多个池（`h2`）。

### 1.7 什么时候**不该**用 sync.Pool

`sync.Pool` 缺失所有资源池该有的能力，别把它当资源池：

| 需求                      | sync.Pool 是否支持 | 说明                                                                                  |
| ------------------------- | ------------------ | ------------------------------------------------------------------------------------- |
| 限制池内对象总数          | ❌                  | 没有任何容量上限，Put 多少存多少                                                      |
| 保证对象一定能复用        | ❌                  | GC 随时清空；文档原文 "may be removed automatically at any time without notification" |
| 空闲超时回收              | ❌                  | 只跟 GC 周期挂钩，与时间无关                                                          |
| 对象销毁钩子（`Close()`） | ❌                  | 对象被丢弃时不会通知你，`net.Conn`/`*sql.DB`/文件句柄放进去就是**句柄泄漏**           |
| 阻塞等待可用对象          | ❌                  | Get 永不阻塞                                                                          |
| 公平性 / FIFO             | ❌                  | Get 返回哪个对象完全不确定                                                            |

结论：**只放「纯内存、无外部资源、可被随时丢弃」的对象**。连接池请用 `database/sql` 或第三方（`puddle`、`grpc` 内建池）；需要限量/带 Close 的对象池，用 `chan *T` 自己实现。

---

## 二、底层原理

### 2.1 整体设计

三层结构，从快到慢：

```text
Get() 的取值路径（Put 只走前两级）
┌────────────────────────────────────────────────────────────┐
│ ① 当前 P 的 private        无锁、无原子操作，最快          │
│ ② 当前 P 的 shared 队头     popHead，单生产者，CAS         │
│ ③ 偷取其他 P 的 shared 队尾  popTail，多消费者，CAS        │
│ ④ victim（上个 GC 周期的 local），同样先 private 再偷      │
│ ⑤ 都失败 → New()（或 nil）                                │
└────────────────────────────────────────────────────────────┘
```

三个核心设计决策：

1. **per-P 分片**：每个 P（逻辑处理器）一份独立的 `poolLocal`，同一个 P 上的 Get/Put 天然串行（因为 `procPin` 禁止了抢占），所以本地路径完全不需要锁。这是 `sync.Pool` 高性能的根基。
2. **work-stealing**：本地空了去别的 P 偷。`shared` 是一个**单生产者多消费者**的双端队列——本地 P 从**头部**推入/弹出，其他 P 只能从**尾部**偷。头尾分离把「本地操作」和「偷取」的竞争降到最低。
3. **victim cache（Go 1.13 引入）**：GC 时不直接清空，而是把 `local` 降级为 `victim`，下个 GC 才真正丢弃。让对象有 **1~2 个 GC 周期**的存活期，避免每次 GC 后所有 Pool 集体冷启动造成的分配尖刺。

### 2.2 数据结构

```text
Pool
 ├── local      unsafe.Pointer ──▶ [GOMAXPROCS]poolLocal   // 本代缓存
 ├── localSize  uintptr                                     // = len(local)
 ├── victim     unsafe.Pointer ──▶ [oldGOMAXPROCS]poolLocal // 上一代缓存
 ├── victimSize uintptr
 └── New        func() any

poolLocal（128 字节，独占缓存行）
 ├── private any        // 只有本 P 能碰，容量恰好 1
 ├── shared  poolChain  // 本 P: pushHead/popHead；他 P: popTail
 └── pad     [96]byte   // 防伪共享

poolChain（动态扩容的双端队列 = poolDequeue 双向链表）
 ├── head *poolChainElt                  // 只有生产者访问，无需同步
 └── tail atomic.Pointer[poolChainElt]    // 消费者访问，必须原子

  head ────────────────────────────────▶ tail
  [dequeue 容量64] ⇄ [容量32] ⇄ [容量16] ⇄ [容量8]
   ↑最新，push 到这里                       ↑最旧，popTail 从这里偷

poolDequeue（固定大小无锁环形队列，SPMC）
 ├── headTail atomic.Uint64   // 高32位=head，低32位=tail，一次 CAS 同时改
 └── vals     []eface         // 容量必须是 2 的幂；eface{typ, val}
```

实测尺寸（`unsafe.Sizeof`）：`sync.Pool` = 40 字节，`poolLocalInternal` = 32 字节，`poolLocal` = **128 字节**。

### 2.3 为什么 poolLocal 要填充到 128 字节

`poolLocalInternal` 只有 32 字节，但源码强行 pad 到 128（`pool.go:77`）：

```go
pad [128 - unsafe.Sizeof(poolLocalInternal{})%128]byte  // 96 字节填充
```

因为 `local` 是一个**连续数组**，如果不填充，4 个 P 的 `poolLocal` 会挤在同一条缓存行里。P0 写自己的 `private`，会让 P1/P2/P3 缓存行失效，触发大量缓存一致性流量——这就是**伪共享（false sharing）**。填充到 128 字节（注释写的是「`128 mod cacheline == 0` 的平台」，覆盖 64 字节缓存行的 x86 和 128 字节缓存行的 Apple Silicon）后，每个 P 独占自己的缓存行，跨 P 干扰消失。

这也解释了为什么 `sync.Pool` 在核数越多的机器上相对优势越明显。

### 2.4 pin：为什么必须禁用抢占

`Get`/`Put` 第一件事都是 `p.pin()`，最后一件事都是 `runtime_procUnpin()`。`procPin` 的实现极简（`runtime/proc.go:7885`）：

```go
func procPin() int {
    gp := getg()
    mp := gp.m
    mp.locks++        // 只是给 m.locks 加 1，抢占检查会跳过 locks>0 的 M
    return int(mp.p.ptr().id)
}
```

必须 pin 的理由有三条：

1. **正确性**：整个 `Get`/`Put` 都建立在「我独占当前 P 的 `poolLocal`」这个假设上。如果中途被抢占、goroutine 被调度到另一个 P，两个 goroutine 就会同时读写同一个 `private`/同一个 `shared` 的头部——而 `private` 是**裸读裸写没有任何同步**的，`pushHead`/`popHead` 也只对「单生产者」安全。
2. **与 GC 互斥**：`poolCleanup` 在 STW 时执行。「world stopped」意味着没有任何 goroutine 处于 pin 区间内（注释：*in effect, this has all Ps pinned*），所以 cleanup 可以直接裸改 `p.local`/`p.victim` 而不需要原子操作。
3. **`localSize` / `local` 的读序**：pin 里先 load-acquire `localSize` 再 load `local`（与 `pinSlow` 的写序相反，`pool.go:211-216`）。因为 pin 期间 GC 不可能发生，就能保证观察到的 `local` 数组至少和 `localSize` 一样大。

`pin` 有个细节：`p == nil` 的检查放在 `procPin` **之前**（`pool.go:206`）。如果放在之后，nil 解引用会发生在 pinned 区间内，`m.locks > 0` 时的 panic 会被运行时升级成不可恢复的 `fatal error`。为了让用户看到正常的 panic 栈，特地提前判空。

### 2.5 poolChain / poolDequeue：无锁环形队列

`poolDequeue` 是一个**固定大小、单生产者多消费者**的双端环形队列。核心技巧是把 head 和 tail 打包进**一个** `uint64`：

```go
headTail atomic.Uint64   // 高 32 位 head，低 32 位 tail
```

这样一次 CAS 就能原子地「确认 head 没变 && tail 加 1」，避免了两个独立变量之间的 ABA/撕裂问题。head 放高位是刻意的（注释）：`Add(1 << dequeueBits)` 增加 head 时，溢出会自然回绕且不会污染低位的 tail。

三个操作的分工：

| 操作       | 调用者                 | 同步方式                                                                             |
| ---------- | ---------------------- | ------------------------------------------------------------------------------------ |
| `pushHead` | 只有本地 P（单生产者） | 先 load `headTail` 判满，写 slot，再 `Add` 推进 head。**推进 head 本身就是发布屏障** |
| `popHead`  | 只有本地 P（单生产者） | CAS 先把 head 减 1「抢回」slot 所有权，成功后才读值                                  |
| `popTail`  | 任意 P（多消费者）     | CAS 把 tail 加 1，成功者独占该 slot                                                  |

**槽位所有权的交接**是这里最精妙的部分。`vals[i].typ == nil` 表示空槽：

- `popTail` 取走值后，**先**写 `slot.val = nil`，**再**原子写 `slot.typ = nil`（`poolqueue.go:180-181`）。顺序不能反：`typ` 置 nil 是「我用完了，所有权还给生产者」的信号，必须最后发布。
- `pushHead` 即使算出队列没满，也要额外检查 `atomic.LoadPointer(&slot.typ) != nil`（`poolqueue.go:90-95`）——说明有消费者还在清理这个槽位，此时**队列实际上仍然是满的**，直接返回 false。

清空 slot（而不只是移动索引）也是有意的：否则环形数组里的悬空引用会让对象一直被 GC 视为存活，池就变成了内存泄漏。注释里点明这是「文献里通常不考虑但对 sync.Pool 很重要」的性质。

因为容量固定，用 `dequeueNil *struct{}` 做哨兵区分「用户 Put 了一个 nil 接口值」和「空槽」——不过 `Put` 入口已经拦掉了 `x == nil`，这个哨兵在 `sync.Pool` 语境下实际用不上，是 `poolDequeue` 作为独立数据结构的完备性设计。

`poolChain` 在此之上加了动态扩容：把多个 `poolDequeue` 串成双向链表，**每个新节点容量翻倍**（8 → 16 → 32 → …，上限 `dequeueLimit = 2^30`）。

```go
func (c *poolChain) pushHead(val any) {
    d := c.head
    if d == nil { /* 首次：分配容量 8 的 dequeue */ }
    if d.pushHead(val) { return }         // 常规路径
    newSize := len(d.vals) * 2            // 满了：新建一个双倍大的
    ...
    c.head = d2; d.next.Store(d2); d2.pushHead(val)
}
```

翻倍策略让「摊销分配次数」是对数级的，同时避免了一开始就分配大数组（大多数 P 上的池其实很小）。

`popTail` 里有一处必须注意顺序的地方（`poolqueue.go:277`）：**先** load `d.next`，**再**尝试 `d.popTail()`。因为 `d` 可能只是「暂时为空」（生产者正在写），只有当「pop 之前 next 已非 nil」且「pop 失败」时，才能断定 `d` **永久**为空，从而安全地把它从链上摘掉。反过来写就会误删还在被写入的节点。

摘链用 `c.tail.CompareAndSwap(d, d2)`，赢的那个把 `d2.prev` 置 nil——既让空节点可被 GC 回收，也防止 `popHead` 沿 prev 链回溯得过远。

### 2.6 victim cache 与 GC 的关系

`sync.Pool` 的生命周期完全由 GC 驱动。`init` 里向运行时注册回调（`pool.go:296`），GC 在 `gcStart` 的 STW 阶段调用 `clearpools()`（`runtime/mgc.go:875`、`2159`）：

```go
// clearpools before we start the GC. If we wait the memory will not be
// reclaimed until the next GC cycle.
clearpools()
```

`poolCleanup` 只做两件事（都是 O(池数量)，不遍历池内对象，所以 STW 增量极小）：

```go
for _, p := range oldPools {          // 上一代的 victim：彻底丢弃
    p.victim = nil; p.victimSize = 0
}
for _, p := range allPools {          // 本代的 local：降级为 victim
    p.victim, p.victimSize = p.local, p.localSize
    p.local, p.localSize = nil, 0
}
oldPools, allPools = allPools, nil
```

所以一个对象的存活窗口是 **1~2 个 GC 周期**：

```text
Put ──▶ local ──[GC #1]──▶ victim ──[GC #2]──▶ 被丢弃，交给 GC 回收
```

实测（`runtime.GC()` 手动触发，Put 100 个对象后循环 Get 直到 nil）：

```
put 100
存活数量 after 1 GC: 100     // 全在 victim 里，还能捞回来
存活数量 after 2 GC: 0       // 已被彻底丢弃
```

对比 Go 1.13 之前：GC 直接清空所有池，导致每轮 GC 之后所有 Pool 同时冷启动、分配量骤增（典型的锯齿状 CPU/内存曲线）。victim cache 用「多留一代」把这个尖刺抹平了。

**推论**：Pool 的实际缓存效果与 GC 频率强相关。GC 越频繁（堆小、`GOGC` 低），对象越容易被淘汰，池的命中率越低。压测时如果 Pool 收益不明显，可以先看 `GODEBUG=gctrace=1` 确认 GC 频率。

### 2.7 getSlow 的两个细节

```go
func (p *Pool) getSlow(pid int) any {
    // ① 先把所有 P 的 local.shared 偷一遍
    for i := 0; i < int(size); i++ {
        l := indexLocal(locals, (pid+i+1)%int(size))   // 从 pid+1 开始，避开自己
        if x, _ := l.shared.popTail(); x != nil { return x }
    }
    // ② 全部失败才动 victim
    ...
    if x := l.private; x != nil { l.private = nil; return x }  // 只看自己 pid 的 victim.private
    for i := 0; i < int(size); i++ { ... popTail ... }
    atomic.StoreUintptr(&p.victimSize, 0)   // ③ 标记 victim 已空
    return nil
}
```

**细节一：为什么 victim 放在偷取之后？** 注释说得很直白：*we want objects in the victim cache to age out if at all possible*。victim 里的对象已经「老」了一代，优先消耗新鲜的 local，让 victim 尽快自然清空，避免老对象被无限期续命。

**细节二：一次失败的扫描会让整个 victim 作废。** 最后那行 `atomic.StoreUintptr(&p.victimSize, 0)` 是个短路优化——既然扫了一圈没货，后续的 Get 就不用再白跑一遍了。代价是：victim 里各个 P 的 `private` 槽位**只有跑在对应 P 上的 goroutine 才能取到**（源码只查 `indexLocal(locals, pid).private` 这一个），偷取环节只覆盖 `shared` 队列。所以那些留在别的 P 的 `victim.private` 里的对象，会被这次 `victimSize = 0` 一并放弃。

这个行为可以直接观测到：只 Put **1** 个对象（它会落到 `private`），GC 一次，然后 `Get()` —— 大概率返回 `nil`，因为调用 Get 的 goroutine 未必还在原来那个 P 上。丢弃量上限是 `GOMAXPROCS - 1` 个对象，实现者认为这个精度损失换取快速路径是划算的。

---

## 三、关键源码解读

### ① Pool 与 poolLocal（pool.go:51、67）

```go
type Pool struct {
    noCopy noCopy                       // go vet copylocks 检查用的空结构体

    local     unsafe.Pointer // 实际类型 *[P]poolLocal
    localSize uintptr

    victim     unsafe.Pointer
    victimSize uintptr

    New func() any
}
```

`local` 用 `unsafe.Pointer` 而不是 `[]poolLocal`，是为了能用一条原子指令替换整个数组（`atomic.StorePointer`），并配合 `indexLocal` 做手工索引：

```go
func indexLocal(l unsafe.Pointer, i int) *poolLocal {
    lp := unsafe.Pointer(uintptr(l) + uintptr(i)*unsafe.Sizeof(poolLocal{}))
    return (*poolLocal)(lp)
}
```

`noCopy` 让 `go vet` 能报出拷贝错误——实测：

```
copy/main.go:8:7: assignment copies lock value to q: sync.Pool contains sync.noCopy
```

注意这只是 vet 的静态检查，编译**不会**失败。拷贝一个已使用的 Pool，会复制 `local` 指针，两个 Pool 共享同一份 `poolLocal` 数组，而 `poolCleanup` 只认识 `allPools` 里注册过的那一个 → 悬空引用 + 双重管理。

### ② Put：两级写入（pool.go:99）

```go
func (p *Pool) Put(x any) {
    if x == nil { return }              // nil 直接丢弃，不占坑
    if race.Enabled {
        if runtime_randn(4) == 0 { return }   // race 模式下 25% 概率随机丢弃！
        race.ReleaseMerge(poolRaceAddr(x))
        race.Disable()
    }
    l, _ := p.pin()
    if l.private == nil {
        l.private = x                  // 优先塞 private（最快路径）
    } else {
        l.shared.pushHead(x)           // private 占了就进队头
    }
    runtime_procUnpin()
    ...
}
```

`Put` 全程只有 `pushHead` 里一次 CAS，`private` 路径连原子操作都没有。

值得单独说的是 **race 模式下 25% 随机丢弃**（`pool.go:104`）：这是故意的，用来主动暴露「代码错误地依赖了 Get 一定能拿回刚 Put 的对象」这类 bug。所以 `-race` 下测 Pool 命中率是没有意义的。

`race.ReleaseMerge` / `race.Acquire`（在 Get 里）建立起了文档承诺的 happens-before：*a call to Put(x) "synchronizes before" a call to Get returning that same value x*。

### ③ Get：分级取值（pool.go:131）

```go
func (p *Pool) Get() any {
    l, pid := p.pin()
    x := l.private                     // ① private
    l.private = nil
    if x == nil {
        x, _ = l.shared.popHead()      // ② 本地队头（注释：prefer head for temporal locality）
        if x == nil {
            x = p.getSlow(pid)         // ③ 偷 + ④ victim
        }
    }
    runtime_procUnpin()
    ...
    if x == nil && p.New != nil {
        x = p.New()                    // ⑤ 兜底，注意在 unpin 之后
    }
    return x
}
```

两处设计取舍：

- **本地取队头而非队尾**：注释 *we prefer the head over the tail for temporal locality of reuse* —— 队头是最近 Put 进来的对象，更可能还在 CPU 缓存里。同时头/尾分离让本地 `popHead` 和远程 `popTail` 打不到同一个槽位。
- **`New()` 在 unpin 之后调用**：`New` 是用户代码，可能分配大对象、可能触发 GC、甚至可能阻塞。放在 pin 区间内会拖长禁止抢占的窗口，也可能与 STW 死锁。

### ④ pin / pinSlow：延迟初始化（pool.go:202、223）

`pinSlow` 处理两种情况：Pool 首次使用，以及 `GOMAXPROCS` 变大。

```go
func (p *Pool) pinSlow() (*poolLocal, int) {
    runtime_procUnpin()          // 不能在 pinned 状态下加锁
    allPoolsMu.Lock()
    defer allPoolsMu.Unlock()
    pid := runtime_procPin()     // 重新 pin，double-check
    s := p.localSize
    l := p.local
    if uintptr(pid) < s { return indexLocal(l, pid), pid }

    if p.local == nil { allPools = append(allPools, p) }   // 首次注册到全局表
    size := runtime.GOMAXPROCS(0)
    local := make([]poolLocal, size)
    atomic.StorePointer(&p.local, unsafe.Pointer(&local[0]))  // store-release
    runtime_StoreReluintptr(&p.localSize, uintptr(size))      // store-release
    return &local[pid], pid
}
```

要点：

- 先 unpin 再加锁（`pool.go:226`）：`procPin` 期间不能阻塞，否则持锁的 M 无法被抢占会拖垮调度。加锁后重新 pin 并 double-check，是标准的慢路径模式。
- **写序与读序相反**：这里先 store `local` 再 store `localSize`，`pin` 里先 load `localSize` 再 load `local`。这个 acquire/release 配对保证读到某个 `localSize` 时，对应的 `local` 数组一定已经完整初始化。
- 注释 *"If GOMAXPROCS changes between GCs, we re-allocate the array and lose the old one"*：`GOMAXPROCS` 变大时整个 `local` 数组被换掉，**里面缓存的对象全部丢失**。Go 1.25 起 `GOMAXPROCS` 默认变成容器感知且会**运行时自动调整**（`runtime/proc.go` 的 `updatemaxprocs`，每秒检查 cgroup 配额），所以在容器里 CPU limit 变化时，Pool 会经历一次静默的全量清空。反过来 `GOMAXPROCS` 变小则不会触发重建（`pid < localSize` 恒成立），只是尾部若干 `poolLocal` 从此不再被本地访问，只能被偷取。

### ⑤ poolCleanup（pool.go:257）

```go
//go:linkname poolCleanup
func poolCleanup() {
    // This function is called with the world stopped, at the beginning of a GC.
    // It must not allocate and probably should not call any runtime functions.
    ...
}
```

两个约束值得注意：

1. **不能分配内存**——它运行在 STW 期间的 GC 启动阶段，分配会死锁。所以整个函数只做指针搬移。
2. `//go:linkname poolCleanup` + 注释里的 "hall of shame"（`bytedance/gopkg`、`songzhibin97/gkit`）：这些库通过 linkname 直接引用了这个私有函数（用于实现自己的 Pool 或提前清理），导致标准库不能改它的签名（`go.dev/issue/67401`）。这是个很好的反面教材：linkname 私有符号会绑住上游的手。

---

## 四、常见陷阱

### 4.1 Put 值类型会额外分配一次

`Put(x any)` 的参数是接口。放入非指针类型时，值必须装箱到堆上（因为它要逃逸进池里），这就多了一次分配——池化的目的直接被抵消了一部分。

```go
// ❌ 每次 Put 都装箱分配
var bad = sync.Pool{New: func() any { return make([]byte, 0, 1024) }}
s := bad.Get().([]byte)
bad.Put(s)                    // []byte 是 24 字节的 header，装箱 → 1 alloc

// ✅ 指针本身能直接塞进 eface 的 data 字段，零分配
var good = sync.Pool{New: func() any { s := make([]byte, 0, 1024); return &s }}
p := good.Get().(*[]byte)
good.Put(p)
```

实测差异：

```
BenchmarkPutSliceValue-10    16.35 ns/op    24 B/op    1 allocs/op
BenchmarkPutSlicePtr-10       7.302 ns/op    0 B/op    0 allocs/op
```

`staticcheck` 有对应检查 **SA6002**（*argument should be pointer-like to avoid allocations*）。

### 4.2 忘记 Reset：脏数据与信息泄漏

```go
buf := pool.Get().(*bytes.Buffer)
// 忘了 buf.Reset()
buf.WriteString(userInput)
resp.Write(buf.Bytes())     // ⚠️ 前面残留的内容也一起发出去了
```

在 HTTP handler 里这是**真实的跨请求信息泄漏**：上一个用户的响应内容可能被拼到下一个用户的响应里。

`Reset` 放在 Get 之后还是 Put 之前，两种流派都有人用，但**必须二选一并全局统一**：

```go
// 流派 A（推荐）：Get 后立刻 Reset —— 防御性更强，不依赖所有 Put 点都守规矩
buf := pool.Get().(*bytes.Buffer); buf.Reset()

// 流派 B：Put 前 Reset —— 池内对象始终是干净的，但漏掉任何一个 Put 点就出问题
buf.Reset(); pool.Put(buf)
```

注意 struct 的重置要**逐字段**考虑，尤其是 slice/map 字段：`s = s[:0]` 保留了底层数组（这是我们要的），但如果元素是指针，`s[:0]` **不会**清掉底层数组里的指针引用，会让被引用对象一直存活。需要彻底清时用 `clear(s)` 再 `s = s[:0]`。

### 4.3 Put 之后继续使用对象

```go
pool.Put(buf)
buf.WriteString("x")   // ⚠️ 数据竞争：buf 可能已被别的 goroutine Get 到
```

`Put` 是所有权转移。常见的隐蔽版本是 `defer pool.Put(buf)` 配合返回了 `buf.Bytes()`：

```go
func render() []byte {
    buf := pool.Get().(*bytes.Buffer)
    defer pool.Put(buf)
    buf.WriteString("...")
    return buf.Bytes()      // ⚠️ 返回的 slice 指向池内 buffer 的底层数组
}
```

返回值必须拷贝：`return bytes.Clone(buf.Bytes())`。这类 bug 在低并发下几乎不复现，上线才炸。

### 4.4 大对象回池导致内存不降

`Pool` 没有容量上限。如果偶发的超大对象（比如一次 100MB 的响应体）被 Put 回池，它会一直占着内存直到两轮 GC 之后——而只要池还在被高频使用，这块内存的常驻概率就很高。

标准库的两种解法（见 1.6）：

```go
// 解法一：设上限，超了就丢（fmt 的做法）
func put(buf *bytes.Buffer) {
    if buf.Cap() > 64*1024 { return }   // 直接不回池，交给 GC
    buf.Reset()
    pool.Put(buf)
}

// 解法二：按大小分级（net/http2 的做法）
var pools [7]sync.Pool                   // 16K/32K/64K/128K/256K/512K/∞
func idx(size int) int {
    if size <= 16384 { return 0 }
    i := bits.Len(uint(size-1)) - 14
    return min(i, len(pools)-1)
}
```

### 4.5 对象大小不均匀 → 命中即浪费

文档原文：*Proper usage of a Pool requires each entry to have approximately the same memory cost.* 如果一个池里同时有 1KB 和 1MB 的 buffer，需要 1KB 时可能拿到 1MB（浪费内存），需要 1MB 时可能拿到 1KB（还得扩容，池化收益归零）。所以要么统一规格，要么分级建池。

### 4.6 把 Pool 当资源池用

```go
// ❌ 严重错误
var connPool = sync.Pool{New: func() any { c, _ := net.Dial("tcp", addr); return c }}
```

被 GC 丢弃的 `net.Conn` 不会被 `Close()`（`sync.Pool` 没有任何销毁钩子），fd 会一直泄漏到对象被 GC 回收、finalizer（如果有）才可能关闭——而 `net.Conn` 的 finalizer 行为不该被依赖。同理适用于文件句柄、goroutine、锁、带外部状态的 client。详见 1.7 的能力对照表。

### 4.7 小对象池化反而更慢

`Get`/`Put` 的固定成本：`procPin`（一次 `m.locks++`）+ 至少一次原子 load + 可能的 CAS + 接口装箱/断言。实测量级在 5~15ns。如果对象本身 `new` 出来只要 2ns（比如一个 16 字节的小 struct），池化就是纯亏。

判断标准：**构造成本 ≫ Get/Put 开销，且对象够大到值得省 GC 扫描**。不确定就写 benchmark，别凭感觉。

### 4.8 不能拷贝 Pool

```go
var p sync.Pool
p.Put(1)
q := p          // go vet: assignment copies lock value to q
```

同理不能把 `sync.Pool`（值，非指针）作为字段放进会被拷贝的 struct，也不能用值接收者的方法去传递它。总是用 `*sync.Pool` 或包级变量。

### 4.9 Get 拿到的不一定是你 Put 的

文档：*Callers should not assume any relation between values passed to Put and the values returned by Get.* 可能来自其他 P（偷取）、来自 victim、或者干脆是 `New()` 现造的。任何「Put 进去 Get 一定拿得到」的逻辑都是错的——`-race` 模式的 25% 随机丢弃（见 ②）就是专门用来打这类假设的。

### 4.10 单个对象 + GC 后大概率丢失

如 2.7 所述，落在 `victim.private` 里的对象只有原 P 上的 goroutine 能取回，且一次失败扫描会作废整个 victim。对于**低频使用**的 Pool（池里长期只有寥寥几个对象），命中率会很难看。`sync.Pool` 是为高频并发场景设计的；低频场景直接 `new` 更简单也更可预测。

---

## 五、常见面试题

**1. sync.Pool 解决什么问题？为什么能提升性能？**
缓存「已分配但暂时不用」的临时对象，摊薄高频分配的开销。收益有两部分：省掉分配本身，以及**省掉这些对象后续被 GC 标记/清扫的开销**——后者通常是主要收益。实测 `bytes.Buffer` 场景 574ns/6allocs → 38ns/0allocs。

**2. sync.Pool 的内部结构是怎样的？**
`Pool` 持有 `local`（长度 = `GOMAXPROCS` 的 `poolLocal` 数组）和 `victim`（上一个 GC 周期的 local）。每个 `poolLocal` = 一个 `private` 槽（容量 1，仅本 P 可访问）+ 一个 `shared`（`poolChain`，本 P 从头 push/pop，其他 P 从尾偷），并 pad 到 128 字节防伪共享。`poolChain` 是容量逐级翻倍（8→16→32…）的 `poolDequeue` 双向链表，`poolDequeue` 是把 head/tail 打包进一个 `uint64` 的无锁环形队列。

**3. 为什么 sync.Pool 几乎不需要加锁？**
两点。① **per-P 分片 + `procPin`**：pin 期间当前 goroutine 不会被抢占、不会换 P，所以同一个 P 上的 Get/Put 天然串行，`private` 可以裸读裸写。② **单生产者多消费者的头尾分离**：本地只碰队头，偷取只碰队尾，`headTail` 打包成一个 `uint64` 让「判断 + 移动索引」一次 CAS 完成。全局锁 `allPoolsMu` 只在 `pinSlow`（首次使用 / `GOMAXPROCS` 变大）时才用到。

**4. Get 的完整取值顺序是什么？**
① 本 P 的 `private` → ② 本 P `shared` 的**队头**（`popHead`，时间局部性最好） → ③ 从其他 P 的 `shared` **队尾**偷（`popTail`，从 `pid+1` 开始遍历） → ④ victim（先本 pid 的 `private`，再遍历偷 victim 的 shared） → ⑤ `New()`（若为 nil 则返回 nil）。`New()` 在 unpin 之后调用。

**5. 为什么本地取队头、偷取取队尾？**
队头是最近 Put 的对象，CPU 缓存更热（源码注释：*temporal locality of reuse*）；同时头尾分离让本地 `popHead` 和远程 `popTail` 操作不同槽位，把 CAS 竞争降到最低。

**6. victim cache 是什么？为什么要引入？**
Go 1.13 引入。GC 时不直接清空池，而是把 `local` 降级为 `victim`，`victim` 里上一代的内容才真正丢弃，于是对象有 1~2 个 GC 周期的存活期。引入前每轮 GC 后所有 Pool 同时冷启动，造成分配量和延迟的周期性尖刺；victim 把这个尖刺抹平了。实测：Put 100 个对象，1 次 GC 后全部还能取回，2 次 GC 后全部消失。

**7. sync.Pool 里的对象什么时候被回收？能撑多久？**
只跟 GC 周期挂钩，与时间无关。GC 在 `gcStart` 的 STW 阶段调 `clearpools()` → `poolCleanup()`，`local` → `victim` → 丢弃。所以存活 1~2 个 GC 周期。GC 越频繁池命中率越低，这也意味着 **Pool 的效果依赖具体的 GC 行为，不可控**。

**8. poolCleanup 为什么可以不加锁地改 local/victim？**
它在 STW 期间执行。world stopped 意味着不存在处于 pin 区间内的 goroutine（等价于所有 P 都被 pin 住了），不会有任何并发访问。同时它被严格限制**不能分配内存**（STW 期间分配会死锁），所以只做指针搬移，STW 增量是 O(池数量) 而非 O(对象数量)。

**9. 为什么 poolLocal 要填充到 128 字节？**
`poolLocalInternal` 只有 32 字节，`local` 是连续数组，不填充的话多个 P 的 `poolLocal` 会落在同一条缓存行上，P0 写 `private` 会让其他 P 的缓存行失效（**伪共享**），产生大量缓存一致性流量。pad 到 128 字节让每个 P 独占缓存行——128 同时覆盖 64 字节缓存行的 x86 和 128 字节的 Apple Silicon。

**10. 为什么 Put 一个 `[]byte` 不好，要 Put `*[]byte`？**
`Put(x any)` 参数是接口，非指针值需要装箱到堆上，多一次分配，部分抵消池化收益。指针能直接放进 eface 的 data 字段，零分配。实测 16.35ns/1alloc vs 7.30ns/0alloc。`staticcheck SA6002` 会报这个问题。

**11. sync.Pool 能当连接池用吗？**
不能。缺三样关键能力：没有容量上限、没有对象销毁钩子（被丢弃的 `net.Conn` 不会 `Close`，直接 fd 泄漏）、没有空闲超时/阻塞获取。它只适合放「纯内存、可被随时丢弃」的对象。需要限量 + Close 就用 `chan *T` 自己实现，或用 `database/sql` 这类真正的资源池。

**12. Get 到的对象需要手动重置吗？不重置会怎样？**
必须手动重置，`Pool` 不做任何清理。不重置在 HTTP 场景下就是**跨请求信息泄漏**（上个用户的响应内容拼进下个用户的响应）。约定「Get 后立刻 Reset」比「Put 前 Reset」更安全，因为不依赖所有 Put 点都守规矩。

**13. 为什么 race 模式下 sync.Pool 会随机丢弃对象？**
`Put` 里有 `if runtime_randn(4) == 0 { return }`，25% 概率直接丢掉。这是**故意的**，用来暴露「代码错误地依赖 Get 一定能拿回刚 Put 的对象」这类假设。所以 `-race` 下测命中率没有意义。

**14. `poolDequeue` 为什么把 head 和 tail 打包进一个 uint64？**
这样「校验 head 未变 + 移动 tail」可以用**一次 CAS** 原子完成，不用处理两个独立变量之间的 ABA/撕裂。head 放在高 32 位是刻意的：`Add(1<<32)` 推进 head 时溢出会自然回绕，不会污染低位的 tail。

**15. `popTail` 取走值后为什么要把槽位清零？顺序有讲究吗？**
清零是为了不留下悬空引用——否则环形数组会一直让对象对 GC 可见，池就变成内存泄漏（源码注释说这是「文献里通常不考虑但对 sync.Pool 很重要」的性质）。顺序必须是**先** `slot.val = nil` **后**原子写 `slot.typ = nil`：`typ` 置 nil 是「所有权交还生产者」的信号，必须最后发布。对应地，`pushHead` 即使算出队列未满也要检查 `slot.typ != nil`，非 nil 说明消费者还在清理，队列实际仍满。

**16. `poolChain.popTail` 为什么必须先读 `next` 再 pop？**
因为一个 dequeue 可能只是「暂时为空」（生产者正在往里写）。只有当「pop 之前 `next` 已非 nil」且「pop 失败」时，才能断定它**永久**为空，从而安全地从链上摘掉。顺序反了会误删还在被写入的节点。

**17. `pin` 里为什么先判 `p == nil` 再 `procPin`？**
如果放在 pin 之后，nil 解引用发生在 `m.locks > 0` 的区间内，运行时会把 panic 升级成不可恢复的 `fatal error`，用户看不到正常的 panic 栈。提前判空就能给出正常的 `panic("nil Pool")`。

**18. `GOMAXPROCS` 变化会对 Pool 有什么影响？**
变**大**时，`pinSlow` 会重新分配整个 `local` 数组，**原有缓存对象全部丢失**（源码注释明确承认这点）。变**小**不会触发重建（`pid < localSize` 恒成立），只是尾部若干 `poolLocal` 不再被本地访问、只能被偷取。Go 1.25 起 `GOMAXPROCS` 默认容器感知且会**运行时自动调整**（每秒检查 cgroup 配额），所以容器 CPU limit 变动时 Pool 会静默清空一次——这是新版本才需要留意的行为。

**19. 只 Put 一个对象，GC 一次之后还能 Get 到吗？**
大概率**不能**。单个对象会落在某个 P 的 `private` 槽里，GC 后进入 `victim`；而 `getSlow` 的 victim 路径只检查**自己 pid** 的 `private`（偷取环节只覆盖 `shared` 队列），调用 Get 的 goroutine 未必还在原来那个 P 上。更狠的是一次失败扫描末尾会执行 `atomic.StoreUintptr(&p.victimSize, 0)`，把整个 victim 直接作废。最多损失 `GOMAXPROCS-1` 个对象，实现者认为这个精度损失换快速路径是划算的。

**20. sync.Pool 和自己用 `chan *T` 实现的对象池，各自适合什么场景？**
`sync.Pool`：追求极致吞吐、对象是纯内存、可以接受随时丢失、需要自动随负载伸缩（高峰扩、空闲缩）。`chan *T`：需要**硬性容量上限**、需要**阻塞等待**、对象带外部资源需要 `Close`、需要确定的存活语义。两者不是替代关系——`sync.Pool` 是 GC 优化工具，`chan` 是资源管控工具。

**21. Pool 里的对象大小不一样会有什么问题？标准库怎么解决的？**
需要小对象时可能拿到大对象（浪费内存且难以释放），需要大对象时拿到小的（还得扩容，收益归零）。标准库两种解法：`fmt` 给 buffer 设 **64KB 硬上限**，超了就不回池（`issue/23199`）；`net/http2` 按 16KB~512KB **分 7 档建 7 个池**，用 `bits.Len` 算档位，取出后还校验 `len >= 需求`。

**22. `//go:linkname poolCleanup` 和源码里的 "hall of shame" 注释说明了什么？**
`bytedance/gopkg`、`songzhibin97/gkit` 等库通过 linkname 直接引用了这个私有函数，导致标准库为了兼容不能再改它的签名（`go.dev/issue/67401`）。反面教材：linkname 私有符号会把上游的手绑死，同时自己也随时可能被上游重构打断。
