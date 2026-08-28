# 内存分配与栈

> 环境：`go version go1.26.3 darwin/amd64`。源码：`runtime/{malloc,mcache,mcentral,mheap,mspan,mpagealloc,stack}.go`、`internal/runtime/gc/sizeclasses.go`。配套代码：`notes/mem/`。回收部分见 gc.md。
>
> 版本相关：
> - **1.12**：`mheap` 的 span 管理从 treap 起步演进；`GODEBUG=madvdontneed` 引入。
> - **1.14**：page allocator 换成**基数树**（`mpagealloc.go`），替换旧的 treap，大幅降低分配延迟。
> - **1.21**：`unsafe.SliceData`/`unsafe.String` 等让"零拷贝转换"有了合法写法。
> - **1.22**：引入 **malloc header**（`MinSizeForMallocHeader = 512`），≥512B 的含指针对象在对象头存类型指针，缩小 span 位图。
> - **1.24**：`sizeclasses.go` 移到 `internal/runtime/gc`。
> - **1.26**：`mcache` 新增 `reusableNoscan`——把回收的 noscan 小对象串成自由链表就地复用（链表指针存在对象第一个字里）。

## 一、分配器

### 1.1 三层结构

| 层 | 归属 | 作用 |
| --- | --- | --- |
| `mcache` | **每个 P 一个**，无锁 | 按 `spanClass` 各持有一个可分配的 `mspan`；小对象分配的快路径全在这里 |
| `mcentral` | 每个 `spanClass` 一个，全局 | `partial[2]`/`full[2]` 两组 `spanSet`，mcache 用完了来这里换 span |
| `mheap` | 全局唯一，加锁 | 管理所有 page（基数树）；mcentral 没货就来这里申请，不够就 `sysAlloc` 找 OS |

```text
mallocgc
 ├─ tiny  (<16B 且 noscan) → mcache.tiny 就地切一刀
 ├─ small (≤32KB)          → mcache.alloc[spanClass]
 │                            └ 空 → mcentral.cacheSpan
 │                                    └ 空 → mheap.alloc → sysAlloc(OS)
 └─ large (>32KB)          → 直接 mheap.alloc（按 8KB 页取整）
```

关键常量（`internal/runtime/gc/sizeclasses.go`）：

```go
MaxSmallSize   = 32768   // 小对象上限
NumSizeClasses = 68      // size class 档数
PageShift      = 13      // 页大小 8KB
TinySize       = 16      // tiny 阈值
heapArenaBytes = 64MB    // amd64 非 Windows（runtime/malloc.go:252）
```

`mcache`/`mcentral`/`mheap` 都带 `_ sys.NotInHeap`，即**它们自己不在 GC 堆里**，GC 不会扫描这些元数据。

### 1.2 size class：68 个档位

```text
前 20 档: 8 16 24 32 48 64 80 96 112 128 144 160 176 192 208 224 240 256 288 320
后 10 档: 14336 16384 18432 19072 20480 21760 24576 27264 28672 32768
```

- 小对象档位密（差 8/16 字节），大对象档位疏；设计目标是**最坏浪费不超过约 12.5%**。
- **一个 span 只放同一 size class 的对象**，所以"分配"退化成"在位图里找一个空位"，O(1) 且无需 header。
- 三张元数据表：`SizeClassToSize`（档位→字节）、`SizeClassToNPages`（档位→占几页）、`SizeClassToDivMagic`（用乘法+移位代替除法，算"地址→span 内下标"）。
- `spanClass = sizeclass<<1 | noscan`，所以有 **136 个 spanClass**。

实测取整浪费（`notes/mem/`）：

```text
请求     实际占用   浪费    浪费率
1        8         7      87.5%
9        16        7      43.8%
33       48        15     31.2%
100      112       12     10.7%
1000     1024      24     2.3%
17000    18432     1432   7.8%
```

benchmark 里能直接看到取整：

```text
BenchmarkAlloc89-8   30.34 ns/op   96 B/op   1 allocs/op   ← 89 和 96 一样贵
BenchmarkAlloc96-8   31.57 ns/op   96 B/op   1 allocs/op
BenchmarkAlloc97-8   33.34 ns/op   112 B/op  1 allocs/op   ← 多 1 字节，跨档
```

**工程含义**：结构体字段排布（struct.md 2.1）不只影响 `Sizeof`，还决定落进哪个 size class。把 96 字节优化到 89 字节毫无意义，优化到 80 字节才真省 16 字节。

### 1.3 tiny allocator

`mcache` 里三个字段专门服务它：

```go
tiny       uintptr   // 当前 tiny 块的起始地址
tinyoffset uintptr   // 块内已用偏移
tinyAllocs uintptr   // 计数
```

条件：**大小 < 16 字节 且 不含指针**。多个这样的对象被塞进同一个 16 字节块里。实测：

```text
200000 个 *int8（1 字节 noscan）:        平均 9.0 B/个
200000 个 struct{p *int8}（8 字节含指针）: 平均 16.0 B/个
```

为什么含指针的不能合并？① 一个对象存活就会拖住整块（放大内存占用）；② GC 的指针位图是按对象粒度表达的，混合内容无法描述。

**副作用**：tiny 块里只要有一个对象活着，整块（含其他已死对象）都回收不了。极端情况下大量 `*byte` 会造成看不见的内存滞留。

### 1.4 大对象

```text
make([]byte,  32768) -> 堆增长 32768 字节（4.00 页）
make([]byte,  32769) -> 堆增长 40960 字节（5.00 页）  ← 多 1 字节，多 8KB
make([]byte,  65536) -> 堆增长 65536 字节（8.00 页）
make([]byte,  66561) -> 堆增长 73728 字节（9.00 页）
```

- >32KB 绕过 mcache/mcentral，**直接找 mheap 按页分配**，每次都要拿 `mheap.lock`。
- 页的管理是**基数树 `pageAlloc`**（1.14 起），按 chunk 维护位图，支持快速找连续页。
- 实测并发下大对象分配更贵：

```text
BenchmarkAlloc32KB-8            3986 ns/op    BenchmarkAlloc32KBParallel-8   4683 ns/op
BenchmarkAlloc33KB-8            4797 ns/op    BenchmarkAlloc33KBParallel-8   5734 ns/op
```

一个容易搞混的点：**"大对象"说的是堆分配路径，不代表一定在堆上**。只要长度是编译期常量且不逃逸，40KB 的 slice 也会待在栈上：

```text
不逃逸的 make([]byte, 40<<10)：堆增长 0 字节（在栈上！）
```

### 1.5 noscan：有没有指针决定 GC 扫不扫

```go
type noPtr  struct{ a, b, c, d int64 }   // 32B，noscan
type hasPtr struct{ a, b, c *int64; d int64 }  // 32B，scan
```

- `noscan` 对象所在的 span **完全不参与标记扫描**。
- 1.22 起，含指针且 ≥ `MinSizeForMallocHeader`（512B）的对象在**对象头存一个类型指针**（malloc header），小对象仍用 span 级位图。这样既省位图空间又能精确扫描。
- **这是最容易被忽视的性能杠杆**：`[]*Item`（1000 万个指针要逐个扫）换成 `[]Item` 或 `[]int32` 索引，GC 标记时间能降一个数量级（见 gc.md 3.10）。

## 二、逃逸分析

### 2.1 唯一的判定原则

> 编译器能否证明这个值的生命周期**不超过**它所在的函数栈帧？能 → 栈上；不能（或不确定）→ 堆上。

**逃逸分析是编译期的静态分析，和"值有多大""是不是 new 出来的"都无关**。`new(T)` 完全可能在栈上，字面量 `&T{}` 也可能在堆上。

```go
func stackOnly() int      { p := point{1, 2}; return p.x + p.y }   // 栈
func escapeReturn() *point { p := point{1, 2}; return &p }          // moved to heap: p
func noEscapeParam(p *point) int { return p.x }                     // p does not escape
func escapeToGlobal(p *point)    { global = p }                     // &p escapes to heap
```

查看命令：

```bash
go build -gcflags='-m' ./mem          # 逃逸 + 内联决策
go build -gcflags='-m -l' ./mem       # 关掉内联，结论更直白
go build -gcflags='-m -m' ./mem       # 打印完整的逃逸推导链（很长但能定位到具体哪一步）
```

两类输出要分清：

- `escapes to heap`：**这个值**逃逸了（通常是被存到了别处）；
- `moved to heap: x`：**变量 x 整体**从栈搬到堆（因为它的地址逃逸了）。

### 2.2 常见的逃逸原因

| 原因 | 例子 | 说明 |
| --- | --- | --- |
| 返回局部变量地址 | `return &p` | 最直白的一种 |
| 存入全局/更长生命周期的结构 | `global = p`、`m[k] = &p` | 生命周期超出栈帧 |
| **装箱进接口** | `takeAny(v)`、`fmt.Println(x)` | 接口 data 是指针，值必须有个可取地址的家 |
| 闭包逃出函数 | `return func(){ n++ }` | 捕获变量 `n` 被搬到堆 |
| `go func(){...}()` 捕获 | 新 goroutine 生命周期不受本帧约束 | 必然逃逸 |
| 长度非常量的 `make` | `make([]int, n)` | 编译器不知道要在栈上留多少 |
| 单个栈对象过大 | `make([]int, 10000)` | 超过约 64KB 就上堆（`implicit variable too large`） |
| `append` 扩容 | — | 新底层数组一定在堆上 |
| `map` | 任何大小 | `hmap` 永远在堆上 |
| 通过接口/函数指针间接调用 | 参数被"当作可能逃逸"处理 | 编译器看不到被调方，只能保守 |

实测接口装箱的三种结果（`notes/mem/bench_test.go`）：

```text
BenchmarkPassInt-8              1.28 ns/op   0 B/op  0 allocs/op   // 直接传 int
BenchmarkPassIntAsAny-8        13.95 ns/op   8 B/op  1 allocs/op   // 变量装箱 -> convT64 真分配
BenchmarkPassSmallIntAsAny-8    3.19 ns/op   0 B/op  0 allocs/op   // 0-255 命中 staticuint64s
BenchmarkPassConstAsAny-8       1.89 ns/op   0 B/op  0 allocs/op   // 常量装箱 -> 编译期进只读段
```

三个 0 alloc 的原因各不相同，写 benchmark 时特别容易被这三种优化骗到：**测装箱开销必须用变量，且值要大于 255**。

### 2.3 栈上分配到底快多少

```text
BenchmarkStackAlloc-8    1.330 ns/op   0 B/op   0 allocs/op
BenchmarkHeapAlloc-8    20.280 ns/op  16 B/op   1 allocs/op
```

15 倍。而且这只是**分配**的差价，堆对象后续还要被 GC 标记、清扫、可能被 scavenger 归还——总成本远不止 19ns。

### 2.4 预分配的价值

```text
BenchmarkSliceNoPrealloc-8    4621 ns/op   25208 B/op   12 allocs/op
BenchmarkSlicePrealloc-8       689 ns/op       0 B/op    0 allocs/op   ← 快 6.7x
BenchmarkMapNoPrealloc-8     68065 ns/op   74264 B/op   20 allocs/op
BenchmarkMapPrealloc-8       18344 ns/op   36944 B/op    5 allocs/op   ← 快 3.7x
```

`SlicePrealloc` 甚至是 0 alloc——因为 `make([]int, 0, 1000)` 长度是常量、`s` 不逃逸，整个 slice 待在栈上。

字符串拼接同理：

```text
BenchmarkConcatPlus-8           5025 ns/op   9744 B/op   99 allocs/op   ← 每次 += 都新建
BenchmarkConcatBuilder-8         746 ns/op    504 B/op    6 allocs/op
BenchmarkConcatBuilderGrow-8     650 ns/op    320 B/op    1 allocs/op
BenchmarkConcatBytes-8           624 ns/op    192 B/op    1 allocs/op   ← 最省
```

## 三、栈

### 3.1 尺寸与增长

```go
stackMin     = 2048          // Go 代码最小栈（runtime/stack.go:78）
fixedStack   = 2048          // amd64：stackMin + stackSystem 向上取整到 2 的幂
maxstacksize = 1000000000    // 1GB（64 位，runtime/proc.go:160）；32 位是 250MB
```

- **每个 goroutine 起步 2KB**，这是"能开百万 goroutine"的直接原因（对比 pthread 默认 8MB）。
- 增长路径：函数入口的栈检查发现 `SP < stackguard0` → `morestack` → `newstack` → **`newsize = oldsize * 2`**（不够就继续翻倍）→ `copystack` 把整个栈拷到新空间。
- 收缩：GC 期间 `shrinkstack`，**使用量不足 1/4 就减半**（也走 `copystack`）。
- 栈内存**也是从 mheap 拿的**，走 `stackpool`（全局，按 order 分级）+ `mcache.stackcache`（每 P 缓存）两级；计入 `MemStats.StackSys`。

```text
当前 StackInuse=384KB（1 个 goroutine）
100 个深递归 goroutine 后: StackInuse=992KB（101 个 goroutine）
```

### 3.2 copystack：栈上变量的地址会变

实测：递归 400 层，采样 11 个局部变量地址，**检测到 4 次地址大跳变**——每一次都是一轮 `copystack`。

三个直接推论：

1. **栈必须是精确可扫描的**：`copystack` 要遍历每个栈帧，按 `funcdata` 里的指针位图逐个修正指针。这也是 Go 要求"栈上不能有无法识别的指针"的原因。
2. **`unsafe.Pointer` 转成 `uintptr` 之后不能存起来**：`uintptr` 只是个整数，`copystack` 不会修正它，栈一搬就成了野指针（见 unsafe.md）。
3. **不能把 Go 栈上对象的指针长期交给 C**：cgo 规则里"C 不得保存 Go 指针"，栈地址不稳定是原因之一（另一个原因是 GC 不扫描 C 的内存）。

### 3.3 栈溢出

```text
runtime: goroutine stack exceeds 8388608-byte limit
runtime: sp=0x1776a29e0388 stack=[0x1776a29e0000, 0x1776a31e0000]
fatal error: stack overflow
...（后面还有 275 行栈帧，其中一句是 "...349379 frames elided..."）
```

- 超过 `maxstacksize` 直接 `throw("stack overflow")`，是 **fatal error，无法 recover**。
- `debug.SetMaxStack(n)` 可以调小（上面子进程设成 8MB），用来让"疯狂递归"快速失败而不是把机器内存吃光。
- 注意区分：**栈溢出**（递归太深）是 fatal error；**空指针解引用**是可 recover 的 panic。

### 3.4 nosplit 与栈检查

```go
stackNosplit = abi.StackNosplitBase * sys.StackGuardMultiplier
stackGuard   = stackNosplit + stackSystem + abi.StackSmall
```

`//go:nosplit` 标记的函数**不插入栈增长检查**（因为它可能运行在没法增长栈的时刻，比如 `morestack` 自己）。链接器会遍历所有 nosplit 调用链，确保总帧大小不超过预留区，否则报 `nosplit stack overflow`。这解释了为什么 runtime 里很多函数写得那么克制。

## 四、优化实践与陷阱

### 4.1 分配次数 > 分配字节数

```text
131072 个 8 字节 []byte: 65582 次分配（比对象数还少：tiny allocator 合并了）
1 个 1MB []byte:        1 次分配
```

标记成本 ∝ **对象个数 + 指针个数**。所以：

- `pprof` 看 **`-alloc_objects`**，不是只看 `-alloc_space`；
- 合并小对象（一个大 `[]byte` + 偏移索引）比压缩单个对象大小有效得多；
- `runtime/metrics` 里 `/gc/heap/allocs:objects` 是最该盯的一条曲线。

### 4.2 优化清单（按性价比）

1. **预分配容量**：`make([]T, 0, n)`、`make(map[K]V, n)`、`strings.Builder.Grow(n)`。
2. **避免装箱**：热路径不要过 `any`；日志用条件判断包起来（`if lvl >= Debug`），因为参数求值和装箱发生在调用前。
3. **减少指针**：`[]T` 优于 `[]*T`；用 `int32` 索引代替指针；`map[string]int` + `[]T` 优于 `map[string]*T`。
4. **复用对象**：`sync.Pool`（见 pool.md），只对"构造成本高 + 本来就逃逸"的对象划算。
5. **控制 struct 大小**：字段按大小降序排（struct.md 2.1）；注意 size class 边界。
6. **`[]byte` 与 `string` 转换**：注意编译器的零拷贝特例（见 string.md），必要时用 `unsafe.String`/`unsafe.SliceData`。

### 4.3 陷阱：以为 `new` 就是堆分配

```go
func f() int { p := new(int); *p = 1; return *p }   // 栈上，0 alloc
```

`new`/`&T{}`/`make` 都只是"分配"的语法，**在哪分配由逃逸分析决定**。反过来，`var x T` 也可能在堆上。

### 4.4 陷阱：`-benchmem` 的 B/op 不是"内存占用"

`B/op` 是**通过分配器分配的字节数**，不包含：栈上分配、`mmap` 的内存、cgo 分配的内存。所以一个 0 B/op 的 benchmark 完全可能吃掉几 MB 栈。要看真实占用得看 `runtime/metrics` 或 RSS。

### 4.5 陷阱：大 slice 截小仍持有整个底层数组

```go
data := make([]byte, 10<<20)   // 10MB
small := data[:10]             // 仍然引用那 10MB
```

必须 `bytes.Clone(data[:10])` 或者 `append([]byte(nil), data[:10]...)` 才能断开。同理适用于 `s[:0]` 复用时元素是指针的情况——`clear(s)` 再截断才能真正释放引用（见 slice.md 3.3、pool.md 1.4）。

### 4.6 陷阱：碎片

```text
HeapInuse=1.6MB HeapIdle=13.9MB HeapReleased=4.6MB HeapSys=15.5MB
内部碎片指标: HeapInuse - HeapAlloc = 0.4MB
```

- **内部碎片**：size class 取整的浪费（最坏 ~12.5%）；
- **外部碎片**：span 专属某个 size class，某 class 的空 span **不能直接**给别的 class 用（要先整个 span 释放回 mheap）；
- Go 是**非移动 GC**，不做内存整理，碎片只能靠 size class 设计 + 对象复用控制。

典型症状：程序按 1KB 分配了很久，突然改成大量 200B 对象，RSS 不降反升——因为 1KB 那档的空 span 还留着。

### 4.7 有用的 GODEBUG

| 选项 | 用途 |
| --- | --- |
| `gctrace=1` | 每轮 GC 一行日志（见 gc.md 2.4） |
| `scavtrace=1` | 归还 OS 的节奏 |
| `madvdontneed=0` | 改用 `MADV_FREE`（RSS 降得更慢，但重用更快） |
| `harddecommit=1` | 归还时真正 decommit，暴露 use-after-free |
| `clobberfree=1` | 释放对象时填垃圾值，抓悬垂指针 |
| `efence=1` | 每个对象独占一页，抓越界（极慢） |
| `invalidptr=1` | **默认开**：发现非法指针值立刻崩，别关 |
| `inittrace=1` | 每个包 `init` 的耗时和分配量（查启动慢） |
| `asyncpreemptoff=1` | 关异步抢占，排查抢占相关问题 |

## 五、常见面试题

**1. Go 的内存分配器是什么结构？借鉴了什么？**
三层：per-P 的 `mcache`（无锁快路径）→ per-sizeclass 的 `mcentral`（全局，两组 spanSet）→ 全局 `mheap`（页管理 + 找 OS）。设计源自 **TCMalloc**（thread-caching malloc），核心思想是"线程本地缓存 + 按大小分类的空闲链表"，Go 把"线程"换成了 P（见 1.1）。

**2. 小对象、大对象、微小对象的分配路径有什么区别？**
`<16B 且 noscan` 走 tiny allocator（多个对象共用一个 16B 块）；`≤32KB` 走 mcache 的对应 spanClass；`>32KB` 直接找 mheap 按 8KB 页分配并加 `mheap.lock`（见 1.1、1.3、1.4）。

**3. size class 是什么？为什么要 68 档？**
把分配尺寸归一到 68 个固定档位（8B 到 32KB），一个 span 只放同一档的对象，于是分配 = 位图找空位、无需 per-object header、也便于批量清扫。档位分布经过设计使**最坏内部碎片约 12.5%**。分配 89 字节和 96 字节的实际成本完全相同（见 1.2）。

**4. 什么是 span？spanClass 为什么是 136 个？**
`mspan` 是一组连续页（`SizeClassToNPages`），被切成同一 size class 的对象。`spanClass = sizeclass<<1 | noscan`，68 × 2 = 136——同一尺寸的"含指针"和"不含指针"要分开，因为后者完全不用被 GC 扫描（见 1.2、1.5）。

**5. 逃逸分析的判定标准是什么？`new` 一定在堆上吗？**
唯一标准：编译器能否证明该值的生命周期不超过当前栈帧。`new(int)` 完全可以在栈上；反之 `var x T` 也可能被搬到堆。用 `go build -gcflags='-m -l'` 查看（见 2.1、4.3）。

**6. 有哪些常见的逃逸原因？**
返回局部变量地址、存入全局或更长命的结构、**装箱进接口**（含 `fmt.Println`）、闭包逃出函数、`go func()` 捕获变量、`make` 长度非常量、单个栈对象过大（>64KB）、`append` 扩容、`map`、通过接口间接调用导致编译器保守（见 2.2）。

**7. 为什么 `fmt.Println(x)` 会让 x 逃逸？**
参数类型是 `...any`，装箱需要把值放到一个可取地址的位置，而 `fmt` 内部还会通过反射持有它，编译器无法证明它不会活过本帧。所以热路径上一行日志就能让 `allocs/op` 从 0 变成若干（见 2.2）。

**8. 栈上分配比堆上快多少？为什么？**
实测同一个 16 字节 struct，栈 1.33ns / 堆 20.28ns，约 15 倍。栈分配只是移动 SP（甚至编译器直接复用栈槽），堆分配要走 mcache 查空位、更新位图、记账，还要付后续的 GC 标记/清扫成本（见 2.3）。

**9. goroutine 的栈初始多大？怎么增长？会缩小吗？**
初始 2KB（`stackMin`）。函数入口检查 `SP < stackguard0` → `morestack` → `newstack` → 容量**翻倍**（不足则继续翻倍）→ `copystack` 拷贝整个栈。GC 时 `shrinkstack` 在使用量 < 1/4 时减半。上限 1GB（64 位），超了是 fatal error（见 3.1、3.3）。

**10. Go 的栈是连续栈还是分段栈？为什么换了？**
现在是**连续栈**（1.4 起）。之前是分段栈（segmented stack），问题是"**hot split**"：如果一个函数调用恰好卡在段边界上被反复调用，就会不断地分配/释放栈段，性能断崖。连续栈用"整体拷贝 + 指针修正"换掉了这个问题（见 3.1）。

**11. 栈拷贝的时候指针怎么办？**
`copystack` 遍历每个栈帧，按编译器生成的 `funcdata` 指针位图，把所有指向老栈的指针逐个加上偏移。这要求栈**精确可扫描**——也正因如此，`uintptr` 不会被修正（它不是指针），把 `unsafe.Pointer` 转成 `uintptr` 保存下来是错的（见 3.2、unsafe.md）。

**12. 为什么 Go 能开百万 goroutine，线程不行？**
① 栈起步 2KB 且按需增长，线程默认 8MB 且一次性预留虚拟地址；② goroutine 切换在用户态（只换几个寄存器 + SP/PC），线程切换要陷入内核；③ 调度器是 M:N，不需要为每个 goroutine 占一个内核对象（见 3.1、gmp.md）。

**13. `HeapAlloc`、`HeapInuse`、`HeapIdle`、`HeapSys` 的关系？**
`HeapAlloc` = 存活对象字节；`HeapInuse` = 正在使用的 span 总字节（≥ HeapAlloc，差额就是内部碎片）；`HeapIdle` = 空闲 span（其中 `HeapReleased` 已还 OS）；`HeapSys` = 从 OS 拿到的总量 ≈ Inuse + Idle（见 4.6、gc.md 2.6）。

**14. 什么是内部碎片和外部碎片？Go 怎么处理？**
内部碎片 = size class 取整浪费（最坏 12.5%）；外部碎片 = span 专属某个 size class，空 span 不能直接给别的 class 用。Go 是非移动 GC，不做整理，只能靠 size class 设计和对象复用来控制（见 4.6）。

**15. tiny allocator 有什么副作用？**
一个 16 字节的 tiny 块里只要有一个对象存活，整块都无法回收。大量小的 `*byte`/`*int8` 场景下可能造成看不见的内存滞留（见 1.3）。

**16. 为什么把 `[]*T` 改成 `[]T` 能显著降低 GC 时间？**
`[]*T` 里每个指针都要被 GC 追踪（且指向的对象散落各处，cache 极不友好）；`[]T`（T 不含指针）落在 `noscan` span，**GC 完全不扫**。1000 万元素的量级下这是一个数量级的差别（见 1.5、gc.md 3.10）。

**17. `-benchmem` 的 `B/op` 为 0 就代表不占内存吗？**
不是。`B/op` 只统计经由分配器的堆分配，不含栈分配、`mmap`、cgo。一个 0 B/op 的函数可能用掉几 MB 栈（见 4.4）。
