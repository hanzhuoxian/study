# Iter（range over func 与迭代器）

> 环境：`go version go1.26.3`。`range over func`（range-over-func）在 **Go 1.23 正式启用**（1.22 需要 `GOEXPERIMENT=rangefunc`），`iter` 包同版本引入。本文"底层原理"一节以 `cmd/compile/internal/rangefunc/rewrite.go`、`runtime/coro.go`、`iter/iter.go`、`internal/abi/rangefuncconsts.go` 源码为准。文中所有结论与压测数据均在该版本上实测。

## 一、基础使用

### 1.1 range over func：三种合法签名

`for ... range f` 中的 `f` 只能是下面三种函数类型之一（只看底层类型，命名类型也可以）：

```go
func(yield func() bool)          // 0 个迭代变量
func(yield func(V) bool)         // 1 个迭代变量        —— iter.Seq[V]
func(yield func(K, V) bool)      // 2 个迭代变量        —— iter.Seq2[K, V]
```

```go
for range tick { }        // f 是 func(func() bool)
for v := range seq { }    // f 是 func(func(V) bool)
for k, v := range seq2 { }// f 是 func(func(K, V) bool)
for k := range seq2 { }   // 合法：Seq2 可以只接收第一个值
// for i, v := range seq  // 编译错误：permits only one iteration variable
```

没有"三个及以上迭代变量"的形式；要传更多值就自己包一个 struct。

### 1.2 iter.Seq / iter.Seq2

标准库只给了两个类型别名级别的定义（`iter/iter.go`），没有任何"迭代器接口"：

```go
type Seq[V any]     func(yield func(V) bool)
type Seq2[K, V any] func(yield func(K, V) bool)
```

语义约定（这三条是整个机制的地基）：

1. 迭代器每产出一个元素，就调用一次 `yield`；
2. `yield` 返回 `true` 表示"继续"，返回 `false` 表示"调用方要提前结束，请立刻收尾并 return"；
3. **`yield` 返回 `false` 之后再调用它会 panic**。

对外暴露 API 时应使用 `iter.Seq` / `iter.Seq2`，而不是自定义的等价函数类型，这样才能和标准库的适配器（`slices.Collect`、`maps.Insert` 等）互相拼接。

### 1.3 写一个迭代器

```go
func Count(n int) iter.Seq[int] {
    return func(yield func(int) bool) {
        for i := range n {
            if !yield(i) { // 必须检查返回值！
                return     // 提前结束：这里是做清理的地方
            }
        }
    }
}

for v := range Count(5) {
    if v == 2 {
        break // break 会让 yield 返回 false
    }
}
```

带资源清理的写法，用 `defer` 保证 `break`/`return`/`panic` 三条退出路径都能释放：

```go
func Lines(name string) iter.Seq2[string, error] {
    return func(yield func(string, error) bool) {
        f, err := os.Open(name)
        if err != nil {
            yield("", err) // 错误也是序列的一部分
            return
        }
        defer f.Close()    // 无论 yield 返回 false 还是 panic，都会执行

        sc := bufio.NewScanner(f)
        for sc.Scan() {
            if !yield(sc.Text(), nil) {
                return
            }
        }
        if err := sc.Err(); err != nil {
            yield("", err)
        }
    }
}

for line, err := range Lines("a.txt") {
    if err != nil { /* 处理并 break */ }
}
```

### 1.4 给自定义容器提供迭代器

命名约定（`iter` 包文档给出的官方约定）：

| 名字                         | 含义                                 |
| ---------------------------- | ------------------------------------ |
| `All()`                      | 遍历容器全部元素，最常用的那一种顺序 |
| `Backward()`                 | 反向遍历                             |
| `Keys()` / `Values()`        | 只遍历键 / 只遍历值                  |
| `Preorder()` / `Postorder()` | 明确指定遍历次序                     |
| `Scan(min, max)`             | 需要额外参数时，用构造函数带参数     |

```go
type Ring[V any] struct{ items []V }

// All returns an iterator over index-value pairs.
func (r *Ring[V]) All() iter.Seq2[int, V] {
    return func(yield func(int, V) bool) {
        for i, v := range r.items {
            if !yield(i, v) {
                return
            }
        }
    }
}

func (r *Ring[V]) Backward() iter.Seq[V] {
    return func(yield func(V) bool) {
        for i := len(r.items) - 1; i >= 0; i-- {
            if !yield(r.items[i]) {
                return
            }
        }
    }
}
```

注意方法本身返回的是"迭代器"，不是"迭代结果"，所以 `r.All()` 是廉价的（只是构造一个闭包），真正的遍历发生在 `range` 里。

### 1.5 循环体里的控制流

在 range-over-func 的循环体中，下面这些都能正常工作，而且语义和普通 `for` 完全一致：

```go
for v := range seq {
    if v < 0    { continue }        // → yield 返回 true
    if v == 10  { break }           // → yield 返回 false
    if v == 20  { return v, nil }   // → 先记录返回值，再让 yield 返回 false，函数真正返回
    if v == 30  { goto done }
}
done:

Outer:
for x := range f {
    for y := range g {
        if bad(x, y) { break Outer }    // 带标签的 break/continue 也支持
        if skip(x)   { continue Outer }
    }
}
```

代价是编译器要生成一套状态码来"逃出"闭包，见 2.2 / 2.4。

### 1.6 标准库里的迭代器 API

Go 1.23（随 range-over-func 一起）：

```go
// slices
slices.All(s)                  iter.Seq2[int, E]   // 正向 index-value
slices.Backward(s)             iter.Seq2[int, E]   // 反向
slices.Values(s)               iter.Seq[E]         // 只要值
slices.Chunk(s, n)             iter.Seq[[]E]       // 按 n 个一组切块
slices.Collect(seq)            []E                 // Seq → slice
slices.AppendSeq(s, seq)       []E                 // 追加到已有 slice
slices.Sorted(seq)             []E                 // Collect + Sort
slices.SortedFunc(seq, cmp)    []E
slices.SortedStableFunc(seq, cmp) []E

// maps
maps.All(m)                    iter.Seq2[K, V]
maps.Keys(m)                   iter.Seq[K]
maps.Values(m)                 iter.Seq[V]
maps.Collect(seq2)             map[K]V             // Seq2 → map
maps.Insert(m, seq2)                               // Seq2 → 写入已有 map

// go/ast
ast.Preorder(root)             iter.Seq[ast.Node]
```

Go 1.24 新增的字符串/字节切分迭代器（用来替代会一次性分配整个 `[]string` 的 `strings.Split`）：

```go
strings.Lines(s)               iter.Seq[string]    // 按行（保留行尾 \n）
strings.SplitSeq(s, sep)       iter.Seq[string]
strings.SplitAfterSeq(s, sep)  iter.Seq[string]
strings.FieldsSeq(s)           iter.Seq[string]
strings.FieldsFuncSeq(s, f)    iter.Seq[string]
bytes.Lines / SplitSeq / SplitAfterSeq / FieldsSeq / FieldsFuncSeq   // []byte 版本
// go/types: (*Interface).Methods()、(*Struct).Fields()、(*Scope).Children() 等
```

Go 1.26 新增：`reflect.Type.Fields/Methods/Ins/Outs`、`reflect.Value.Fields/Methods`。

最常用的组合式写法——按 key 有序遍历 map：

```go
for _, k := range slices.Sorted(maps.Keys(m)) {
    fmt.Println(k, m[k])
}
```

### 1.7 iter.Pull / Pull2：push 转 pull

`Seq` 是 **push 模型**：控制权在迭代器手里，它主动把值推给 `yield`。有些场景（归并两个序列、前瞻一个元素、由外部驱动）更适合 **pull 模型**：

```go
func Pull[V any](seq Seq[V]) (next func() (V, bool), stop func())
func Pull2[K, V any](seq Seq2[K, V]) (next func() (K, V, bool), stop func())
```

```go
next, stop := iter.Pull(seq)
defer stop()               // 约定俗成：一定要 defer stop
for {
    v, ok := next()
    if !ok {
        break
    }
    use(v)
}
```

规则（`iter/iter.go` 的文档注释）：

- `next()` 返回 `(值, true)`；序列结束后返回 `(零值, false)`，之后再调用还是 `(零值, false)`，不会 panic；
- **没有把序列消费完就不再用了，必须调用 `stop()`**，否则迭代器函数永远停在 `yield` 里，它持有的资源（以及背后那个协程）不会释放；
- `stop()` 可以重复调用，也可以在 `next()` 已经返回 `false` 之后调用；
- `next` / `stop` **不能被多个 goroutine 并发调用**；
- `seq` 内部的 panic 会在 `next()`（或 `stop()`）处以同样的值重新 panic 出来。

典型用途——Zip 两个序列：

```go
func Zip[A, B any](a iter.Seq[A], b iter.Seq[B]) iter.Seq2[A, B] {
    return func(yield func(A, B) bool) {
        nextB, stop := iter.Pull(b)
        defer stop()
        for va := range a {
            vb, ok := nextB()
            if !ok || !yield(va, vb) {
                return
            }
        }
    }
}
```

### 1.8 常用组合子

标准库刻意没有提供 `Map`/`Filter`/`Take`（社区还在讨论中），自己写只有几行，模板固定：**外层返回一个闭包，内层 range 上游 seq，转发 `yield` 的返回值**。

```go
func Map[A, B any](seq iter.Seq[A], f func(A) B) iter.Seq[B] {
    return func(yield func(B) bool) {
        for v := range seq {
            if !yield(f(v)) {
                return
            }
        }
    }
}

func Filter[V any](seq iter.Seq[V], keep func(V) bool) iter.Seq[V] {
    return func(yield func(V) bool) {
        for v := range seq {
            if keep(v) && !yield(v) {
                return
            }
        }
    }
}

func Take[V any](seq iter.Seq[V], n int) iter.Seq[V] {
    return func(yield func(V) bool) {
        if n <= 0 {
            return
        }
        i := 0
        for v := range seq {
            if !yield(v) {
                return
            }
            if i++; i >= n {
                return
            }
        }
    }
}

func Chain[V any](seqs ...iter.Seq[V]) iter.Seq[V] {
    return func(yield func(V) bool) {
        for _, s := range seqs {
            for v := range s {
                if !yield(v) {
                    return
                }
            }
        }
    }
}
```

因为是惰性的，无限序列 + `Take` 也能正常工作：

```go
func Naturals() iter.Seq[int] {
    return func(yield func(int) bool) {
        for i := 1; ; i++ {
            if !yield(i) {
                return   // 靠调用方 break 来终止
            }
        }
    }
}

sq := Map(Filter(Naturals(), func(n int) bool { return n%3 == 0 }),
          func(n int) int { return n * n })
fmt.Println(slices.Collect(Take(sq, 5))) // [9 36 81 144 225]
```

### 1.9 单次使用迭代器（single-use iterator）

大多数迭代器可以被 `range` 多次，每次都从头走一遍。但如果数据来自不可回退的流（网络、`bufio.Scanner`、`sql.Rows`），第二次遍历就什么都没有了：

```go
sc := bufio.NewScanner(r)
seq := iter.Seq[string](func(yield func(string) bool) {
    for sc.Scan() {
        if !yield(sc.Text()) { return }
    }
})
slices.Collect(seq) // [1 2 3]
slices.Collect(seq) // []   ← 第二次是空的
```

这类迭代器**必须在文档注释里写明 "It returns a single-use iterator."**，否则调用方会踩坑。

## 二、底层原理

### 2.1 编译期改写：range-over-func 没有运行时

range-over-func 完全是**前端语法改写**（`cmd/compile/internal/rangefunc/rewrite.go`，在 noder 之前跑，这样生成的函数还能被后端内联），运行时不参与调度。最朴素的情形：

```go
for x := range f {
    body
}
```

改写成：

```go
f(func(x T) bool {
    body
    return true
})
```

即：**循环体变成一个闭包，交给迭代器去回调**。控制权反转正是所有陷阱的来源。

如果 range 的变量不是用 `:=` 声明的，编译器会生成假参数再赋值：

```go
for expr1, expr2 = range f { ... }
// →
f(func(#p1 T1, #p2 T2) bool {
    expr1, expr2 = #p1, #p2
    ...
})
```

（生成的变量都以 `#` 开头，避免和用户变量重名，调试时一眼能认出来。）

### 2.2 `#next`：break / return / goto 怎么"逃出"闭包

`continue` → `return true`，`break` → `return false`，这两个最简单。但 `return`、`goto L`、带标签的 break/continue 要跳到闭包**外面**，闭包本身做不到，于是编译器引入一个整型状态码 `#next`：

```go
{
    var #next int
    f(func(x T1) bool {
        ...
        return true
    })
    ... 检查 #next ...
}
```

每个"困难语句"先给 `#next` 赋值，再 `return false` 停掉迭代器；迭代器返回后，外层代码按 `#next` 的值补做真正的控制流。

- 裸 `return` → `{#next = -1; return false}`，之后 `if #next == -1 { return }`；
- 带返回值的 `return a, b`：外层函数的返回值先被改写成命名返回值 `#rv1, #rv2`，在闭包里赋值，再 `#next = -1; return false`；
- 带标签的 `break L` / `continue L`：用**正数** `#next` 编码"要跳出第几层"，`perLoopStep*N` 表示 break 第 N 层，`perLoopStep*N-1` 表示 continue 第 N 层，逐层向外传播。

### 2.3 `#stateN` 状态机：运行时怎么发现"坏迭代器"

编译器给每个 range-over-func 循环生成一个 `#stateN` 变量，取值来自 `internal/abi`：

```go
RF_DONE      = 0 // 循环体已经以非 panic 的方式退出（yield 已返回过 false）
RF_READY     = 1 // 循环体尚未退出，且当前没在运行 —— 唯一允许调用 yield 的状态
RF_PANIC     = 2 // 循环体正在运行，或者已经 panic
RF_EXHAUSTED = 3 // 迭代器函数已经返回，整个序列结束
RF_MISSING_PANIC = 4 // 循环体 panic 了，却被迭代器 recover 掉且没有继续 panic
```

改写后的循环体长这样：

```go
var #state1 = abi.RF_READY
f(func(x T1) bool {
    if #state1 != abi.RF_READY {          // ← 入口检查
        #state1 = abi.RF_PANIC
        runtime.panicrangestate(#state1)
    }
    #state1 = abi.RF_PANIC                // 进入 body：标记"正在运行"
    ...
    if ... { #state1 = abi.RF_DONE; return false }   // break
    #state1 = abi.RF_READY                            // 正常结束一轮
    return true
})
if #state1 == abi.RF_PANIC {              // 迭代器 recover 掉了 body 的 panic
    panic(runtime.panicrangestate(abi.RF_MISSING_PANIC))
}
#state1 = abi.RF_EXHAUSTED
```

`runtime.panicrangestate`（`runtime/panic.go`）把状态翻译成四条错误信息，看到它们就知道是哪种误用：

| 状态               | panic 信息                                                                       |
| ------------------ | -------------------------------------------------------------------------------- |
| `RF_DONE`          | `range function continued iteration after function for loop body returned false` |
| `RF_PANIC`         | `range function continued iteration after loop body panic`                       |
| `RF_EXHAUSTED`     | `range function continued iteration after whole loop exit`                       |
| `RF_MISSING_PANIC` | `range function recovered a loop body panic and did not resume panicking`        |

注意状态机检查的是**调用时序**，不是"哪个 goroutine 在调用"。所以并发调用 `yield` 会命中 `RF_PANIC`（因为别人正在 body 里），而一个"严格串行、同步等待"的跨 goroutine 调用碰巧不会被抓到——但这不是规范保证的行为，不要依赖（见 3.3）。

### 2.4 嵌套循环

编译器用**一次遍历**同时改写最外层 range-over-func 循环及其内部所有 range-over-func 循环（否则重写自身生成的代码会带来嵌套深度的平方级开销）。内层循环退出时只做 `if #next < 0 { return false }`，把"该真正 return 了"的信号继续往外层抛，由最外层统一执行。

### 2.5 defer 的归属

循环体虽然被改写成了闭包，但里面写的 `defer` 仍然属于**外层函数**，不是这个闭包——运行时通过 `runtime.deferrangefunc` 把 defer 挂到原函数的链上。实测：

```go
func() {
    defer fmt.Println("outer func done")
    for v := range seq3() {
        defer fmt.Println("body defer", v)
    }
    fmt.Println("loop finished")
}()
// 输出：
//   [seq] deferred cleanup
//   loop finished
//   body defer 2 / body defer 1 / body defer 0
//   outer func done
```

也就是说在 range-over-func 循环体里写 `defer` 和在普通 `for` 里写一样危险：循环 100 万次就压 100 万个 defer，直到外层函数返回才释放。

### 2.6 iter.Pull 的实现：coro，不是普通 goroutine

`Pull` 需要"迭代器执行到一半，把控制权交还调用方，之后再从原地继续"——这是协程语义。实现靠 runtime 的三个内部函数（`iter` 包用 `//go:linkname` 拿到）：

```go
func newcoro(func(*coro)) *coro   // 创建一个 coro：一个阻塞着等待运行的 goroutine
func coroswitch(*coro)            // 切到 coro 里的 goroutine，并把当前 goroutine 阻塞在 coro 上
```

`runtime/coro.go` 的注释说得很清楚：coro 提供的是**额外的并发，而不是额外的并行**——可以把它想成"一个总是有 goroutine 阻塞在上面的特殊 channel"。任一时刻只有一方在跑，因此 `Pull` 出来的 `next`/`stop` **不需要加锁**，`Pull` 内部那个 `pull` 结构体是两个"goroutine"交替独占访问的。

调用链大致是：

```text
next()  →  pull.yieldNext = true  →  coroswitch(c) ─┐
                                                    ↓  切到 seq 所在的协程，从上次 yield 处继续
                                    seq 调用 yield(v) → 写 pull.v/pull.ok → coroswitch(c)
   ┌────────────────────────────────────────────────┘
   ↓  切回来
返回 (pull.v, pull.ok)
```

几个实现细节值得记：

- `stop()` 把 `pull.done = true` 再切一次，让 `yield` 返回 `false`，迭代器函数得以正常 return 并跑完自己的 `defer`；
- `seq` 里的 panic 被 `defer/recover` 捕获存进 `pull.panicValue`，在 `next()`/`stop()` 里重新 `panic`；连 `runtime.Goexit` 都做了转发（`goexitPanicValue`）；
- `yield` 被连续调用两次而中间没有 `next()`，会 panic `iter.Pull: yield called again before next`；`next()` 连续调用同理是 `iter.Pull: next called again before yield`（这是 race 检测之外的额外自检）；
- coro 的切换比普通 goroutine 调度轻（`coroswitch_m` 的注释：快路径只有 3 个 CAS，因为切换频率预期比普通调度高一个数量级以上），但依然远贵于一次函数调用。

### 2.7 内联与逃逸

因为改写发生在前端，生成的 body 函数对后端来说就是普通闭包，**可以被内联，一般不逃逸**：

```go
func sumAll(s []int) int {
    total := 0
    for v := range slices.Values(s) {
        total += v
    }
    return total
}
```

```console
$ go build -gcflags='-m' .
./esc.go:7:2:  can inline sumAll-range1
./esc.go:7:30: inlining call to slices.Values[go.shape.[]int,go.shape.int]
./esc.go:7:2:  inlining call to sumAll.Values[...].func1
./esc.go:7:2:  inlining call to sumAll-range1
./esc.go:7:30: func literal does not escape
```

循环体被命名为 `sumAll-range1`，迭代器闭包不逃逸、零分配。但"不分配"不等于"零成本"，见下。

### 2.8 性能：三种遍历方式的实测对比

1024 个 `int` 的 slice 求和（`go1.26.3`，Intel i5-1038NG7）：

```text
BenchmarkPlainLoop-8               1958306      568.8 ns/op     0 B/op   0 allocs/op
BenchmarkRangeOverFunc-8            451736     2655   ns/op     0 B/op   0 allocs/op   // slices.All
BenchmarkRangeOverFuncIndirect-8    476984     2548   ns/op     0 B/op   0 allocs/op   // 变量类型是 iter.Seq[int]
BenchmarkPull-8                      14074    85627   ns/op   256 B/op   7 allocs/op   // iter.Pull
```

换算成每元素：直接 `for range` 约 0.56ns，range-over-func 约 2.5ns（≈4.5×），`iter.Pull` 约 84ns（≈150×）。

结论：

- range-over-func 零分配，但每个元素多一次（可能无法内联掉的）间接调用 + 状态机读写，热路径上比裸循环慢几倍；
- `iter.Pull` 每个元素要做两次协程切换，量级完全不同——**只在确实需要 pull 语义时才用**（归并、前瞻、外部驱动），不要拿它当"更好写的 for"；
- 抽象层数会叠加：`Map(Filter(...))` 每层都是一次闭包调用，链路长了开销线性增长。

## 三、常见陷阱

### 3.1 忽略 `yield` 的返回值

最高频的错误。写迭代器时必须检查：

```go
// 错误
func Bad(n int) iter.Seq[int] {
    return func(yield func(int) bool) {
        for i := range n {
            yield(i)          // ← 没检查返回值
        }
    }
}

for v := range Bad(5) {
    if v == 1 { break }
}
// panic: runtime error: range function continued iteration after
//        function for loop body returned false
```

`break` 让 `yield` 返回 `false`，但迭代器不理会，继续调用 `yield` → 状态机在 `RF_DONE` 状态上抓到 → panic。正确写法永远是 `if !yield(v) { return }`。

### 3.2 迭代器把循环体的 panic recover 掉了

```go
func swallow() iter.Seq[int] {
    return func(yield func(int) bool) {
        defer func() { recover() }()   // ← 想"保护"一下，结果吞掉了 body 的 panic
        for i := range 3 {
            yield(i)
        }
    }
}

for v := range swallow() {
    if v == 1 { panic("boom") }
}
// panic: runtime error: range function recovered a loop body panic
//        and did not resume panicking
```

循环体的 panic 在语义上属于**调用方**，迭代器无权吞掉。要么不 recover，要么 recover 之后重新 panic。迭代器里的 `recover()` 只应该用于处理迭代器自己产生的 panic。

### 3.3 把 `yield` 存起来 / 并发调用 / 传给别的 goroutine

`yield` 只在当次调用期间有效。存下来以后再用、或多个 goroutine 同时调用，都会被状态机抓住：

```go
func parallel() iter.Seq[int] {
    return func(yield func(int) bool) {
        var wg sync.WaitGroup
        for i := range 4 {
            wg.Add(1)
            go func() { defer wg.Done(); yield(i) }()  // ← 并发 yield
        }
        wg.Wait()
    }
}
// panic: range function continued iteration after loop body panic
//        （RF_PANIC：别的 goroutine 正在 body 里跑）
```

更坏的情况是：这个 panic 发生在**迭代器自己创建的 goroutine** 里，如果那个 goroutine 又 recover 掉，主循环会静默地少收到几个元素，什么错误都看不到（上面的例子实测只收到 3/4 个值）。

反过来，如果是"启动一个 goroutine 调 yield，然后同步等它结束"，时序上没有重叠，实测**不会**触发检查，甚至循环体里的 `return` 也能正常工作——但这是实现细节，规范上不保证，也和 `iter` 文档明确禁止的用法冲突。结论：**`yield` 只在调用迭代器的那个 goroutine 里、串行地调用**。

### 3.4 `iter.Pull` 忘记 `stop()` → 泄漏一个 goroutine

`Pull` 背后的 coro 就是一个阻塞着的 goroutine，**它不会被 GC 回收**（`newcoro` 创建的 g 一直可达且处于阻塞态）：

```go
next, _ := iter.Pull(Count(1000000))
next()                       // 只取一个就不要了
runtime.GC(); runtime.GC()
runtime.NumGoroutine()       // 依然是 2，泄漏了
```

而且迭代器函数停在 `yield` 里，它的 `defer f.Close()` 也永远不会执行——文件句柄一起泄漏。规矩：**`next, stop := iter.Pull(seq)` 的下一行就写 `defer stop()`**。

### 3.5 `next` / `stop` 不能并发调用

`Pull` 内部没有任何锁（靠 coro 的独占语义），并发调用 `next` 是数据竞争，`-race` 下会直接报出来（`iter` 内部特意埋了 `race.Acquire/Release` 和一个 `racer` 字段来让竞争可被检测）。需要多消费者就自己在外面加锁，或者改用 channel。

### 3.6 单次使用迭代器被消费两次

`iter.Seq` 只是个函数值，看起来像"集合"，但它可能只能走一遍（见 1.9）。常见事故：先 `slices.Collect(seq)` 统计一下数量，再 `for range seq` 处理——第二次是空的。要复用就先 `Collect` 成 slice。

### 3.7 把 `iter.Seq` 当集合用

`Seq` 上没有 `len`、没有下标、不能随机访问，也不缓存结果：**每一次 `range` 都会把整条链路重新执行一遍**。

```go
seq := Map(slices.Values(hugeSlice), expensive)
fmt.Println(len(slices.Collect(seq)))  // 执行了一遍 expensive
for v := range seq { ... }             // 又执行了一遍
```

需要多次使用就 `Collect` 一次，把惰性序列固化成 slice。

### 3.8 在遍历过程中修改底层容器

迭代器只是普通函数，没有 `ConcurrentModificationException` 这种保护。`maps.Keys(m)` 底层就是 `for k := range m`，遍历中删 key 是安全的（Go map 的规则），但**新增** key 是否被遍历到是未定义的；`slices.Values(s)` 拿的是 range 开始时的 slice 头，遍历中 `append` 触发扩容后，迭代器看到的还是旧数组。要在遍历中修改，就按 `iter` 文档建议的做法：暴露一个"位置类型"的迭代器（`Seq[*Pos[V]]`），把 `Delete`/`Set` 定义成位置上的方法。

### 3.9 只是为了好看而套一层迭代器

```go
// 没有任何收益：多了一层闭包调用，还慢了几倍
for i, v := range slices.All(s) { ... }
// 直接写
for i, v := range s { ... }
```

`slices.All` 的价值在于**把 slice 适配成 `iter.Seq2` 传给通用函数**，而不是替代 `for range s`。同理，热路径上不要把 `for _, v := range s` 改写成 `for v := range slices.Values(s)`。

### 3.10 忘了迭代器是"惰性"的：错误与 context 要显式传

`iter.Seq` 的签名里没有 `error`，也没有 `context.Context`。约定的做法：

```go
// 错误随值一起产出
func Rows(...) iter.Seq2[Row, error]

// 提前取消：把 ctx 交给迭代器构造函数
func Rows(ctx context.Context, ...) iter.Seq2[Row, error]
```

不要把错误藏在闭包捕获的变量里让调用方"循环结束后自己去查"——很容易被忽略。如果一定要这么设计（类似 `bufio.Scanner.Err()`），必须在文档里写清楚。

### 3.11 循环体里的 `defer` 堆积

见 2.5：range-over-func 循环体里的 `defer` 归属外层函数，不是"每轮结束就执行"。需要每轮清理就显式调用，或者包一层函数字面量。

### 3.12 迭代器函数被当成"值"多次传递

```go
seq := Count(3)   // 只是构造闭包，什么都没执行
```

初学者常以为 `Count(3)` 已经产生了序列。实际上直到 `range` / `Collect` / `Pull` 才开始跑。这意味着：迭代器构造函数里做的参数校验、资源打开会**延迟**到遍历时才发生；如果要"立刻失败"，应该在构造函数里就返回 `(iter.Seq[T], error)`。

## 四、常见面试题

**Q1：range over func 是运行时特性还是编译期特性？**
纯编译期。前端 `rangefunc.Rewrite` 把 `for x := range f { body }` 改写成 `f(func(x T) bool { body; return true })`，运行时只提供 `panicrangestate` 这种错误报告和 `deferrangefunc` 这种 defer 归属修正。因此循环体可被内联，一般零分配。

**Q2：`yield` 返回 `false` 代表什么？迭代器该怎么响应？**
代表调用方要提前结束（`break`/`return`/`goto`/外层要跳出）。迭代器必须立刻停止产出并 return，跑完自己的清理逻辑。继续调用 `yield` 会 panic `range function continued iteration after function for loop body returned false`。

**Q3：循环体里的 `return` 是怎么实现的？**
编译器给外层函数的返回值起名（`#rv1, #rv2`），在闭包里赋值，然后置 `#next = -1` 并 `return false` 结束迭代；迭代器函数返回后，外层代码 `if #next == -1 { return }` 执行真正的返回。带标签的 break/continue 用正数 `#next` 编码层级，逐层向外传播。

**Q4：编译器怎么防止"坏迭代器"？**
每个循环维护一个 `#stateN`（`RF_READY/RF_PANIC/RF_DONE/RF_EXHAUSTED`）。进入 body 前检查是否为 `RF_READY`，不是就 `panicrangestate`；body 运行期间置 `RF_PANIC`；迭代器返回后若状态仍是 `RF_PANIC`，说明它 recover 掉了 body 的 panic，同样 panic。四条错误信息对应四种误用。

**Q5：`iter.Pull` 是怎么实现的？和开一个 goroutine + channel 有什么区别？**
基于 runtime 的 coro：`newcoro` 创建一个阻塞的 goroutine，`coroswitch` 在调用方和迭代器之间**互斥地**转移控制权。任一时刻只有一方运行，所以不需要锁、没有并行、没有 channel 的发送/接收开销和调度延迟，切换快路径只有 3 个 CAS。相比之下 goroutine+channel 每个元素至少两次调度 + 内存屏障，并且需要额外机制来传播 panic 和做取消。不过 coro 切换依然比函数调用贵得多，实测每元素约 84ns。

**Q6：为什么 `Pull` 一定要 `stop()`？不 `stop` 会怎样？**
迭代器停在 `yield` 里，它所在的 goroutine 永远阻塞在 coro 上，不会被 GC 回收，它的 `defer`（关文件、解锁、断连接）也不会执行 → goroutine 泄漏 + 资源泄漏。`stop()` 会置 `done` 并切回迭代器，让 `yield` 返回 `false`，迭代器正常收尾。

**Q7：push 迭代器和 pull 迭代器的取舍？**
push（`iter.Seq`）：控制权在迭代器，写起来最简单、性能最好、能配合 `range` 和 `defer` 做清理，是对外 API 的默认选择。pull（`iter.Pull`）：控制权在调用方，适合需要"同时推进多个序列"（归并、zip）、前瞻、或由外部事件驱动的场景，代价是协程切换。**API 一律暴露 `Seq`，需要 pull 时在内部自己 `Pull`。**

**Q8：`iter.Seq` 上为什么没有 `Map`/`Filter`/`Reduce`？**
Go 团队刻意只在标准库放了类型定义和最小的适配器（`slices.Collect`、`maps.Insert` 等），组合子还在讨论（`x/exp` 里有实验）。自己实现只有几行，模板固定：返回闭包 → range 上游 → 转发 `yield` 的返回值。

**Q9：`for i, v := range slices.All(s)` 和 `for i, v := range s` 有什么区别？**
语义相同，前者慢几倍（每元素一次间接调用 + 状态机读写），实测 1024 元素求和 2655ns vs 568ns。`slices.All` 的用途是把 slice 适配成 `iter.Seq2` 传给接受迭代器的通用函数，不是替代原生 range。

**Q10：怎么给自己的容器加迭代器？有哪些约定？**
方法返回 `iter.Seq`/`iter.Seq2` 而不是切片：`All()`（全部元素）、`Backward()`（反向）、`Keys()`/`Values()`、`Preorder()` 等按顺序命名；需要参数就用构造函数带参数（`Scan(min, max)`）；只能遍历一次的要在文档里注明 single-use；要支持遍历中修改，暴露"位置类型"的迭代器而不是直接给出可变引用。
