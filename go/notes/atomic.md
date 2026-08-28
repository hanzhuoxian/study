# atomic 与内存模型

> 环境：`go version go1.26.3 darwin/amd64`。源码：`sync/atomic/{doc,type,value}.go`、`runtime/internal/atomic/*.s`。配套代码：`notes/atomic/`。锁的部分见 sync.md。
>
> 版本演进：
> - **1.19**：**类型化 atomic**（`atomic.Int64`/`Bool`/`Pointer[T]` 等）；**Go 内存模型文档正式定稿**（`go.dev/ref/mem`），明确 atomic 是顺序一致的。
> - **1.20**：`atomic.Value` 补上 `Swap`/`CompareAndSwap`。
> - **1.23**：新增 **`And`/`Or`** 系列原子位操作。
> - 提案 `golang/go#68578` 讨论过引入 relaxed/acquire-release 弱原子，至今未进——理由是"太容易写错，收益有限"。

## 一、基础使用

### 1.1 优先用类型化 atomic（1.19+）

```go
type counters struct {
    requests atomic.Int64
    ok       atomic.Bool
    name     atomic.Pointer[string]
}

c.requests.Add(1)
c.ok.Store(true)
```

可用类型：`Int32` `Int64` `Uint32` `Uint64` `Uintptr` `Bool` `Pointer[T]` `Value`。

相比老的函数版 `atomic.AddInt64(&x, 1)`，三个实质好处：

1. **不可能忘记用原子操作访问它**——内部字段未导出，你没有办法绕过 `Load`/`Store` 直接读写（这是 3.1 那个 bug 的根治手段）；
2. **自动保证 64 位对齐**（内部带 `align64` 字段），32 位平台上函数版要自己操心（见 3.2）；
3. 带 `noCopy` 语义，`go vet` 能发现拷贝。

新代码应该**只用类型化 API**，函数版留给需要操作已有内存布局的场合（如 `unsafe` 互操作、老代码）。

### 1.2 五种操作 + 位操作

```go
var v atomic.Int64
v.Store(10)                      // 写
v.Load()                         // 读
v.Add(5)                         // 返回新值 = 15
v.Swap(100)                      // 返回旧值 = 15
v.CompareAndSwap(100, 7)         // true，值变 7
v.Add(-3)                        // 减法就是加负数

var f atomic.Uint32
f.Or(flagBusy)                   // 1.23+：置位，返回旧值
f.And(^flagBusy)                 // 清位，返回旧值
```

注意 **`Add` 返回新值、`Swap`/`And`/`Or` 返回旧值**，很容易记反。

没有原子的乘法、除法、浮点加——需要就自己写 CAS 循环（见 2.3）。

### 1.3 `atomic.Pointer[T]`：多字段一致读的标准解法

```go
type config struct {
    Timeout time.Duration
    Retries int
}
var cfg atomic.Pointer[config]

// 写者：整体替换，绝不原地改字段
cfg.Store(&config{Timeout: 2*time.Second, Retries: 5})

// 读者：一次原子读，拿到自洽的完整快照
c := cfg.Load()
use(c.Timeout, c.Retries)
```

这是**配置热更新 / 路由表热切换 / 特征开关**的教科书写法。两条铁律：

- **每次都换一个新对象**，绝不 `c.Retries = 3` 原地改（那又变回数据竞争了）；
- 读者拿到指针后可以随便读，因为那个对象已经不会再变（不可变快照）。

实测（8 线程并发读）：

```text
BenchmarkConfigAtomicPointer-8     1.993 ns/op
BenchmarkConfigAtomicValue-8       3.550 ns/op   ← 多一层 any 断言
BenchmarkConfigRWMutex-8          32.900 ns/op   ← 慢 16 倍
```

### 1.4 `atomic.Value`：1.19 之前的通用方案

```go
var v atomic.Value
v.Load()                  // 零值返回 nil
v.Store(config{...})      // 值类型也可以
v.Store("string")         // panic: store of inconsistently typed value
v.Store(nil)              // panic: store of nil value into Value
v.Store((*config)(nil))   // ✓ 想表达"空"就存 typed nil
```

三条限制：**类型必须前后一致、不能存 nil、Store 之后不能拷贝**。1.19 之后基本可以退休，用 `atomic.Pointer[T]` 替代。

## 二、内存模型

### 2.1 happens-before 的全部来源

Go 内存模型（`go.dev/ref/mem`，1.19 定稿）用的术语是 "**synchronizes before**"。能建立顺序关系的手段只有这些：

| 手段 | 保证 |
| --- | --- |
| `go` 语句 | `go f()` 之前的写，`f` 内部能看到 |
| channel 发送/接收 | 发送 synchronizes before 对应接收完成（chan.md 2.9） |
| `close(ch)` | close synchronizes before 从已关闭通道收到零值 |
| `Mutex` | 第 n 次 `Unlock` synchronizes before 第 m 次 `Lock`（n < m） |
| `RWMutex` | `Unlock` → `RLock`；`RUnlock` → 下一次 `Lock` |
| `WaitGroup` | 所有 `Done` synchronizes before `Wait` 返回 |
| `Once.Do` | `f()` 返回 synchronizes before 任何 `Do` 的返回 |
| **atomic** | 被观察到的原子写 synchronizes before 观察到它的原子读 |

**没有以上关系的并发读写就是 data race**。Go 内存模型明确说 race 是**未定义行为**（不像 Java 那样保证"至少读到某个曾经写入的值"）——实践中可能读到撕裂的值、编译器可能把整个循环优化掉。

### 2.2 Go 的 atomic 是顺序一致的（seq_cst）

官方文档原文：

> if the effect of an atomic operation A is observed by atomic operation B, then A "synchronizes before" B. Additionally, **all the atomic operations executed in a program behave as though executed in some sequentially consistent order**. This definition provides the same semantics as C++'s sequentially consistent atomics and Java's volatile variables.

也就是说 Go 的 atomic **不只是"不撕裂"**，还带完整的内存屏障。经典的 Dekker/store-buffer 测试：

```go
// goroutine 1: x.Store(1); r1 = y.Load()
// goroutine 2: y.Store(1); r2 = x.Load()
// 顺序一致性保证 r1 和 r2 不会同时为 0
```

实测 20000 轮，两边同时读到 0 的次数：**0**。

代价是没得选：Go **只有 seq_cst 一档**，没有 C++ 的 `memory_order_relaxed`/`acquire`/`release`。想要 relaxed 语义的高性能计数器，只能靠"分片 + 每片独占 cache line"来减少争抢（见 sync.md 3.11）。

实现上，amd64 的 `atomic.Store` 用 `XCHG`（自带 full barrier），`Load` 是普通 `MOV`（x86 的 TSO 模型下 load 天然是 acquire）；arm64 用 `LDAR`/`STLR`。

### 2.3 CAS 循环：补上 atomic 没有的操作

```go
// 原子地把 v 更新为 max(v, n)
func atomicMax(v *atomic.Int64, n int64) {
    for {
        old := v.Load()
        if old >= n { return }
        if v.CompareAndSwap(old, n) { return }
        // CAS 失败说明别人改了，重新读再试
    }
}

// 原子浮点加：借 Uint64 存 float64 的 bit pattern
func atomicAddFloat(v *atomic.Uint64, delta float64) float64 {
    for {
        oldBits := v.Load()
        newVal := math.Float64frombits(oldBits) + delta
        if v.CompareAndSwap(oldBits, math.Float64bits(newVal)) {
            return newVal
        }
    }
}
```

三段式：**读旧值 → 算新值 → CAS，失败重来**。

**但高竞争下 CAS 循环会输给锁**：

```text
BenchmarkAtomicContended-8      16.29 ns/op   ← 单条 LOCK XADD
BenchmarkMutexContended-8       60.45 ns/op
BenchmarkCASLoopContended-8     84.51 ns/op   ← 比锁还慢！
```

原因：`Add` 是一条硬件指令，无论多少核抢都是"排队一次成功"；CAS 循环在 8 核竞争下平均要重试好几次，每次失败都白付一次 cache line 争抢。**能用 `Add`/`Or` 表达的就别写 CAS 循环**。

### 2.4 ABA 问题

CAS 只比较值，不知道"中间被改过又改回来了"：

```text
T1: 读到 head=A，准备 CAS(A -> B)
T2: pop A、pop B、又 push A（此时 A 可能是另一块内存或另一种含义）
T1: CAS(A -> B) 成功——但语义已经错了
```

**Go 里为什么很少中招**：① 有 GC，只要还有指针引用节点内存就不会被复用成别的对象（C/C++ 的 ABA 主要来自内存重用）；② 无锁数据结构在业务代码里本来就少见。

仍然会出现在"**带版本语义**"的场合。解法是把版本号一起 CAS：

```go
type state struct {
    version uint64
    value   string
}
var st atomic.Pointer[state]

cur := st.Load()
next := &state{version: cur.version + 1, value: "B"}
st.CompareAndSwap(cur, next)   // 指针不同 -> 版本不同 -> ABA 不成立
```

或者把 (指针, 计数) 打包进一个 64 位字（`WaitGroup` 就是这么干的，见 sync.md 2.6）。

### 2.5 性能全景

```text
# 单线程
BenchmarkPlainAdd-8               2.289 ns/op   ← 普通 ++（未优化掉是因为是全局变量）
BenchmarkAtomicAdd-8              6.280 ns/op   ← LOCK XADD，约 3x
BenchmarkAtomicLoad-8             0.644 ns/op   ← 就是一条 MOV，几乎免费
BenchmarkMutexAdd-8              12.880 ns/op   ← 无竞争的锁也要 2 次原子操作

# 8 线程竞争同一个变量
BenchmarkAtomicContended-8       16.29 ns/op
BenchmarkMutexContended-8        60.45 ns/op
BenchmarkCASLoopContended-8      84.51 ns/op

# 8 线程只读
BenchmarkAtomicLoadContended-8    0.403 ns/op   ← 纯读不会互相干扰
BenchmarkRWMutexReadContended-8  34.52 ns/op    ← RLock 也要改 readerCount，一样抢 cache line
```

三条结论：

1. **原子读几乎免费**（amd64 上就是一条 MOV），所以"读多写少"用 `atomic.Pointer` 完胜 `RWMutex`（86 倍差距）；
2. **原子写有固定成本**（LOCK 前缀锁总线/cache line），约是普通写的 3 倍；
3. **竞争下 atomic ≈ 锁的 1/4 成本**，但两者都受 cache line 争抢支配——真正的解法是**别共享**（分片计数器，见 sync.md 3.11）。

## 三、常见陷阱

### 3.1 混用原子和普通访问

```go
// goroutine A
atomic.AddInt64(&n, 1)
// goroutine B
fmt.Println(n)         // ✗ 普通读 -> data race
```

"一半原子一半不原子"等于没有原子。`-race` 能抓到（`notes/atomic/` 里 3.1 会真的起一个 `-race` 子进程验证，输出 `DATA RACE`）。

**根治手段是用类型化 atomic**：`atomic.Int64` 的内部字段未导出，语法上就不可能绕过 `Load`/`Store`。

### 3.2 32 位平台的 64 位对齐

在 386 / arm / 32 位 mips 上：

- 函数版 `atomic.AddInt64(&s.count, 1)` **要求地址 8 字节对齐**，否则运行时 panic；
- 文档只承诺"分配对象/全局变量/切片的第一个字"是 8 字节对齐的；
- 老代码的经典 hack：把 `int64` 字段放结构体第一位，或者手工加 padding。

用 `atomic.Int64` 类型完全不用操心（内部有 `align64` 字段）。这是类型化 API 最实际的好处之一。

### 3.3 多个原子变量 ≠ 一致快照

```go
avg := total.Load() / count.Load()   // ✗ 两次 Load 之间别人可能改了任意一个
```

三种解法：

1. 用锁保护这一组变量；
2. **打包进 struct 用 `atomic.Pointer[T]` 整体替换**（推荐，见 1.3）；
3. 塞进同一个 64 位里，一次 `Add` 同时更新两个计数：

```go
pack := func(hi, lo uint32) uint64 { return uint64(hi)<<32 | uint64(lo) }
packed.Add(pack(1, 5))   // count += 1 且 total += 5，一次原子操作
```

第 3 种是 runtime 的常用手法（`WaitGroup.state`、`Mutex.state`），但要注意低位溢出会污染高位。

### 3.4 `atomic.Value` 的四条规则

1. 类型必须前后一致 → 否则 `panic: sync/atomic: store of inconsistently typed value`；
2. 不能 `Store(nil)` → `panic: store of nil value into Value`；想表达"空"存 typed nil；
3. **Store 之后不能拷贝**（内部是 `(typ, data)` 两个字，拷贝会破坏一致性）；
4. `CompareAndSwap` 要求值可比较，存 slice/map/func 会 panic。

### 3.5 race detector 能做什么、不能做什么

**原理**：ThreadSanitizer 的 vector clock。每个内存字（8 字节粒度）记录最近若干次访问的 `(goroutine, 时钟, 读/写)`，新访问来了就检查是否存在 happens-before 关系。

| 能 | 不能 |
| --- | --- |
| 竞态**真的发生了**就基本 100% 报出，几乎无误报 | 没走到的代码路径检测不到（动态分析，不是静态分析） |
| 精确指出两个访问的栈 | 内存 ~5-10x、CPU ~2-20x 开销，不能生产常开 |
| 覆盖所有 sync/atomic/channel 原语 | 检测不到**逻辑竞态**（如 3.3：每次 Load 都是原子的，组合起来却是错的） |

实践：

```bash
go test -race ./...                       # CI 的底线
GORACE="halt_on_error=1" go test -race    # 第一个竞态就失败
GORACE="history_size=7" ...               # 增大历史记录（默认较小，深栈可能丢信息）
```

### 3.6 用 atomic 做自旋等待

```go
for !done.Load() {}                          // ✗ 吃满一个核
for !done.Load() { runtime.Gosched() }       // ✗ 稍好，仍在烧 CPU
<-ch                                          // ✓ goroutine 真正挂起，0 CPU
```

Go 不给用户代码提供 `PAUSE`/spin hint（runtime 自己的 `procyield` 不导出）。**要等就用 channel / WaitGroup / Cond 让 goroutine 挂起**，让调度器去干正事。

### 3.7 以为 `atomic` 能替代锁

atomic 保证的是**单个变量的单次操作**原子。以下场景必须用锁：

- 多个变量要一致（见 3.3）；
- "检查再修改"的复合逻辑（除非能写成一次 CAS）；
- 需要保护一段临界区（比如往 map 里写、调用有副作用的函数）。

反过来，能用 atomic 表达的别用锁：计数器、开关、配置指针、序号生成。

### 3.8 `atomic.Pointer[T]` 存的对象被原地修改

```go
c := cfg.Load()
c.Retries = 3        // ✗ 所有持有这个指针的读者都看到了未同步的修改
```

原子指针只保证"指针本身的读写是原子的"，**不保证指向的对象内容安全**。铁律：**指针指向的对象一律视为不可变**，要改就换新对象。

### 3.9 忘了 `atomic` 也不能防止逻辑上的丢失更新

```go
n.Store(n.Load() + 1)   // ✗ 两步之间可能被插入，等价于非原子的 ++
n.Add(1)                // ✓
```

这个错误在 code review 里非常常见，尤其是从 `Load`/`Store` 组合出复杂逻辑时。判断标准：**任何"读-改-写"都必须是一条 `Add`/`Swap`/`CAS`，或者一个 CAS 循环**。

## 四、常见面试题

**1. Go 的 atomic 提供什么内存序保证？**
只有一档：**顺序一致性（seq_cst）**。文档明确说等价于 C++ 的 seq_cst 原子和 Java 的 volatile：被观察到的原子写 synchronizes before 观察到它的原子读，且所有原子操作表现得像存在一个全局一致的顺序。没有 relaxed/acquire/release 可选（见 2.2）。

**2. atomic 和 mutex 怎么选？各自开销多少？**
单变量单操作用 atomic（实测无竞争 6.3ns vs 锁 12.9ns，8 线程竞争 16.3ns vs 60.5ns，只读 0.4ns vs RWMutex 34.5ns）；多变量一致性、复合临界区、要保护函数调用用锁（见 2.5、3.7）。

**3. 类型化 atomic（`atomic.Int64`）比函数版好在哪？**
① 内部字段未导出，语法上不可能绕过 `Load`/`Store` 直接访问（消灭"一半原子一半不原子"的 bug）；② 自动 64 位对齐，32 位平台不用手工 padding；③ 有 `noCopy`，`go vet` 能查拷贝（见 1.1、3.1、3.2）。

**4. `atomic.Value` 有哪些限制？为什么现在推荐 `atomic.Pointer[T]`？**
Value 的限制：类型必须一致、不能存 nil、Store 后不能拷贝、CAS 要求可比较。`Pointer[T]` 编译期类型安全、没有 `any` 装箱、实测快一倍（1.99ns vs 3.55ns）（见 1.4、1.3）。

**5. 怎么实现"原子地读取多个字段"？**
打包成 struct，用 `atomic.Pointer[T]` 整体替换：写者每次造一个新对象再 `Store`，读者一次 `Load` 拿到自洽快照。关键约束是**指向的对象必须视为不可变**（见 1.3、3.8）。

**6. CAS 是什么？CAS 循环有什么问题？**
`CompareAndSwap(old, new)`：值等于 old 才写入 new。CAS 循环用来实现 atomic 没提供的操作（max、浮点加）。问题是**高竞争下疯狂重试**——实测 8 线程下 CAS 循环 84.5ns，比 mutex 的 60.5ns 还慢，因为 `Add` 是一条硬件指令而 CAS 循环要反复争抢 cache line（见 2.3）。

**7. 什么是 ABA 问题？Go 里严重吗？**
CAS 只比较值，无法区分"没变"和"变了又变回来"。Go 里较轻：GC 保证只要有引用节点就不会被复用成别的对象。但"带版本语义"的场景仍需防护——把版本号和数据一起 CAS（新对象指针天然不同），或者把 (指针, 计数) 塞进一个 64 位字（见 2.4）。

**8. Go 内存模型里，哪些操作能建立 happens-before？**
`go` 语句、channel 发送/接收/close、Mutex 的 Unlock→Lock、RWMutex 的 Unlock→RLock 和 RUnlock→Lock、WaitGroup 的 Done→Wait、Once.Do 的 f() 返回→Do 返回、以及 atomic 的写→读。除此之外的并发读写就是 data race，属于**未定义行为**（见 2.1）。

**9. `n.Store(n.Load() + 1)` 和 `n.Add(1)` 有区别吗？**
有本质区别。前者是两个独立的原子操作，中间可以被插入，等价于非原子的 `++`，会丢更新。**任何"读-改-写"都必须是单条 `Add`/`Swap`/`CAS` 或一个 CAS 循环**（见 3.9）。

**10. race detector 的原理是什么？为什么它报的都是真 bug？**
ThreadSanitizer 的 vector clock：每个内存字记录最近若干次访问的 goroutine 和逻辑时钟，新访问来了检查是否存在 happens-before 关系。它是**动态分析**——只在竞态实际发生（两个访问都执行到）时报，因此几乎无误报，但覆盖不到没走到的路径。开销 CPU 2-20x、内存 5-10x（见 3.5）。

**11. 为什么 32 位平台上 `atomic.AddInt64` 会 panic？**
硬件要求 64 位原子操作的地址 8 字节对齐，32 位平台上结构体字段可能只 4 字节对齐。文档只承诺"分配对象/全局变量/切片的第一个字"对齐。用 `atomic.Int64` 类型可以彻底避免（内部有 align64 字段）（见 3.2）。

**12. `for !flag.Load() {}` 有什么问题？**
纯自旋会吃满一个核，而且没有让出调度点（1.14 异步抢占之后不会完全卡死调度器，但依然是纯浪费）。要等就用 channel/WaitGroup/Cond 让 goroutine 挂起（见 3.6）。

**13. `atomic.Pointer[T]` 保证指向的对象也是线程安全的吗？**
不保证。它只保证指针本身的读写原子。指向的对象必须被当作**不可变快照**——要修改就构造新对象再 `Store`（见 3.8）。

**14. Go 为什么不提供弱内存序（relaxed atomics）？**
提案 `golang/go#68578` 讨论过。核心理由：seq_cst 在 x86 上本来就几乎免费（Load 是普通 MOV），弱序主要收益在 arm64 等弱内存模型平台，但**极易写错且难以测试**，与 Go "简单可靠"的取向冲突。需要极致性能的场合，正确方向是消除共享（分片）而不是放松内存序（见 2.2）。
