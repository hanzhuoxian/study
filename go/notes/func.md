# 函数

> 环境：`go version go1.26.3`。部分底层实现细节（如调用约定、defer 内联阈值）以该版本为准，不同版本可能有差异。

## 一、基础使用

### 1.1 函数声明与调用

```go
func name(parameter-list) (result-list) {
    body
}
```

```go
func add(x, y int) int {      // 相同类型参数可合并
    return x + y
}

func swap(x, y string) (string, string) { // 多返回值
    return y, x
}

func noReturn() {             // 无返回值时省略 result-list
    fmt.Println("hello")
}
```

- 函数是**一等值（first-class value）**：可赋值给变量、作为参数传入、作为返回值返回。
- Go **没有默认参数**、**没有函数重载**，每个函数签名必须唯一。
- 参数传递全部是**值传递**：传入的是实参的副本（指针、slice header、map header 等本身是值，但它们"引用"了底层数据，通过它们可以修改原始数据）。

### 1.2 多返回值

```go
func divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

result, err := divide(10, 2)
if err != nil {
    log.Fatal(err)
}
```

- 多返回值是 Go 惯用的错误处理模式：最后一个返回值通常是 `error`。
- 调用方**必须处理所有返回值**，不想用的用 `_` 忽略：`result, _ := divide(10, 2)`。

### 1.3 命名返回值

```go
func minMax(s []int) (min, max int) {
    min, max = s[0], s[0]
    for _, v := range s[1:] {
        if v < min { min = v }
        if v > max { max = v }
    }
    return // 裸 return：自动返回已命名的 min, max
}
```

- 命名返回值在函数入口被初始化为零值，可在函数体内直接使用。
- **裸 return** 减少重复，但在长函数中会降低可读性，应谨慎使用。
- 命名返回值最重要的用途：**在 `defer` 中修改返回值**（详见 3.1）。

### 1.4 可变参数（variadic）

```go
func sum(nums ...int) int {
    total := 0
    for _, n := range nums {
        total += n
    }
    return total
}

sum(1, 2, 3)          // 直接传值
s := []int{1, 2, 3}
sum(s...)             // slice 展开传入
```

- `...T` 在函数内部被当作 `[]T` 使用，长度可以是 0。
- 只有**最后一个参数**可以是可变参数。
- 将 slice 展开传入（`s...`）**不会拷贝底层数组**，函数内部共享同一底层数组。

### 1.5 函数作为值与函数类型

```go
type BinaryOp func(int, int) int   // 定义函数类型

func apply(op BinaryOp, x, y int) int {
    return op(x, y)
}

add := func(x, y int) int { return x + y }  // 匿名函数赋值给变量
fmt.Println(apply(add, 3, 4))               // 7
fmt.Println(apply(func(x, y int) int { return x * y }, 3, 4)) // 12，匿名函数直接传入
```

### 1.6 闭包（Closure）

```go
func counter(start int) func() int {
    n := start
    return func() int {  // 内层函数捕获了外层变量 n
        n++
        return n
    }
}

c1 := counter(0)
fmt.Println(c1()) // 1
fmt.Println(c1()) // 2

c2 := counter(10)
fmt.Println(c2()) // 11（c2 与 c1 各自持有独立的 n）
```

- **闭包**：函数值 + 它所捕获的外层变量的引用（不是值拷贝）。
- 捕获的是**变量本身**（引用语义），不是调用时的快照，因此闭包能观察到外层变量后续的修改。
- 每次调用外层函数都会产生新的闭包实例，各自有独立的捕获变量（如 `c1` 和 `c2` 的 `n` 互不影响）。

### 1.7 defer

```go
func readFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()   // 注册一个"延迟调用"，函数返回时执行，无论是否出错
    return io.ReadAll(f)
}
```

**执行规则**：

1. `defer` 语句执行时，**函数和参数立即求值**，但调用推迟到外层函数返回。
2. 多个 `defer` 按 **LIFO（后进先出）** 顺序执行（栈）。
3. `defer` 在 `return` 语句执行**之后**、函数真正返回**之前**触发（因此可修改命名返回值）。

```go
func deferOrder() {
    defer fmt.Println("first")
    defer fmt.Println("second")
    defer fmt.Println("third")
}
// 输出：third → second → first
```

### 1.8 panic 与 recover

```go
func safeDiv(a, b int) (result int, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("recovered: %v", r)
        }
    }()
    return a / b, nil  // b==0 时触发 panic
}
```

- `panic` 立即终止当前函数的正常执行流，沿调用栈向上传播，沿途执行各层的 `defer`。
- `recover` **只在直接被 `defer` 调用的函数内有效**，用来捕获并阻止 `panic` 继续传播。
- 只用 `recover` 来处理**无法预料的异常**（如第三方库 panic），正常错误处理用 `error` 返回值。

## 二、底层原理

### 2.1 调用约定与栈帧

Go 使用**寄存器调用约定**（Go 1.17 起，AMD64 最多使用 9 个整型寄存器 + 14 个浮点寄存器传参/返回）：

- 参数和返回值优先放在**寄存器**中，寄存器不够时才溢出到栈上。
- 每次函数调用都会在栈上创建一个**栈帧（stack frame）**，保存：局部变量、被保存的寄存器、返回地址等。
- Go 的 goroutine 从一个**小栈（默认 2-8 KB）** 开始，运行时按需动态扩容（最大默认 1 GB），不像线程那样预分配大栈，这使得创建大量 goroutine 成为可能。

### 2.2 闭包的内部实现

闭包在运行时被表示为一个**带有捕获变量指针的结构体**：

```
closure = { funcptr, *capturedVar1, *capturedVar2, ... }
```

- 捕获变量被**分配到堆上**（通过逃逸分析，编译器判断其生命周期超出了外层函数栈帧，因此堆分配）。
- 闭包持有这些堆上变量的**指针**，多个闭包共享同一捕获变量时，修改是互相可见的。

```go
func makeAdders() (func(int) int, func(int) int) {
    n := 0                       // n 逃逸到堆
    inc := func(x int) int { n += x; return n }
    dec := func(x int) int { n -= x; return n }
    return inc, dec              // inc 和 dec 共享同一个 n
}

add, sub := makeAdders()
add(5) // n=5
sub(2) // n=3（共享的 n）
```

### 2.3 defer 的实现

- **小函数内少量 defer**：编译器在 Go 1.14+ 会做**内联优化（open-coded defer）**，直接展开为普通代码，无运行时开销。
- **循环中的 defer / 无法静态分析数量**：退化为运行时 `_defer` 链表，每个 `defer` 在运行时分配一个 `_defer` 结构体并挂到 goroutine 的链表上，函数返回时遍历执行。

```
// 伪代码：defer f(args) 的运行时语义
args_snapshot := eval(args)   // 参数立即求值，形成快照
defer_list.push({ f, args_snapshot })

// 函数返回时
for d := defer_list.pop(); d != nil; d = defer_list.pop() {
    d.f(d.args_snapshot)
}
```

### 2.4 逃逸分析

编译器通过**逃逸分析（escape analysis）**决定变量分配在栈还是堆：

- 若变量的生命周期不超过当前函数（如只被局部使用，没有被指针"带出去"），分配在**栈上**，函数返回自动回收，无 GC 压力。
- 若变量被引用传递到函数外（如闭包捕获、返回指针、赋值给接口等），**逃逸到堆上**，由 GC 管理。

```go
func noEscape() *int {
    x := 42     // x 逃逸：函数返回后仍被引用
    return &x   // 编译器将 x 堆分配
}

func stackAlloc() int {
    x := 42     // x 不逃逸：值返回，栈分配
    return x
}
```

用 `go build -gcflags='-m'` 可以查看逃逸分析结果。

## 三、常见陷阱

### 3.1 defer 修改命名返回值

```go
// 想要在出错时总是返回 -1，但写法错了：
func wrongWay() (result int) {
    defer func() {
        result = -1  // 这里的 result 是命名返回值，修改会生效！
    }()
    return 42        // return 先把 result 设为 42，再执行 defer，defer 把 result 改成 -1
}
// 实际返回 -1，不是预期的 42

// 正确利用这个特性来增强错误处理：
func openFile(path string) (f *os.File, err error) {
    f, err = os.Open(path)
    if err != nil {
        return
    }
    defer func() {
        if err != nil {  // 如果后续操作出错，关闭文件并确保 err 传递出去
            f.Close()
            f = nil
        }
    }()
    // ... 对 f 做一些操作
    return
}
```

**规则**：`return expr` 等价于"先把 `expr` 赋给命名返回值，再执行 defer，最后返回"。defer 中对命名返回值的赋值会**覆盖** `return` 语句的值。

### 3.2 闭包捕获循环变量

```go
// 错误：所有闭包共享同一个 i 变量（Go 1.22 之前）
funcs := make([]func(), 5)
for i := 0; i < 5; i++ {
    funcs[i] = func() { fmt.Println(i) }
}
for _, f := range funcs {
    f() // 全部输出 5（循环结束后 i==5）
}

// 修复方法1：每次迭代创建新的局部变量
for i := 0; i < 5; i++ {
    i := i  // 遮蔽外层 i，新变量 i 被闭包独立捕获
    funcs[i] = func() { fmt.Println(i) }
}

// 修复方法2：通过参数传值
for i := 0; i < 5; i++ {
    funcs[i] = func(n int) func() {
        return func() { fmt.Println(n) }
    }(i)
}
```

- **Go 1.22 起**：`for` 循环变量每次迭代都是新变量，不再有这个问题（`for i := range n` 的 `i` 与 `for i := 0; i < n; i++` 的 `i` 均如此）。
- 但若代码需要在旧版本运行，或为了明确表达意图，仍应显式创建局部变量或通过参数传值。

### 3.3 defer 参数立即求值

```go
x := 10
defer fmt.Println(x)  // 参数 x 立即被求值为 10，之后 x 改变不影响这里
x = 20
// 函数返回时输出：10，不是 20

// 若想延迟求值，用闭包：
defer func() { fmt.Println(x) }()  // x 是捕获引用，输出 20
```

### 3.4 recover 只在直接的 defer 函数中有效

```go
// 错误：recover 在间接调用中无法捕获 panic
defer func() {
    helper()  // helper 内部的 recover 捕获不到外层的 panic
}()

func helper() {
    if r := recover(); r != nil { ... }  // 无效，r 永远是 nil
}

// 正确：recover 必须直接在 defer 的函数里调用
defer func() {
    if r := recover(); r != nil {
        fmt.Println("caught:", r)
    }
}()
```

### 3.5 值传递陷阱：误以为能修改原始数据

```go
func doubleWrong(n int) {
    n *= 2  // 操作的是副本，原始值不变
}

func doubleRight(n *int) {
    *n *= 2  // 通过指针修改原始值
}

x := 10
doubleWrong(x)  // x 仍为 10
doubleRight(&x) // x 变为 20
```

- slice 和 map 作为参数时，函数内**可以修改其元素**（共享底层数据），但**不能让调用方感知到 `append` 导致的扩容**（header 是值传递，扩容后新 header 调用方看不到），和 struct 一样需要返回或用指针。

## 四、常见面试题

**1. Go 函数参数是值传递还是引用传递？**
全部是**值传递**。传递 int/struct 是副本；传递 slice/map/channel 是"引用类型 header 的副本"，header 本身被复制，但 header 内的指针指向同一底层数据，所以函数内可以修改元素，但无法让调用方感知到 header 本身的变化（如 append 导致的扩容）。

**2. 什么是闭包？闭包捕获的是变量的值还是引用？**
闭包是函数值 + 它所引用的外层作用域变量的组合。Go 闭包捕获的是**变量本身（引用语义）**，不是调用时的值快照；被捕获的变量会逃逸到堆上，所有引用它的闭包共享同一份内存，任何一方修改都对其他方可见（见 1.6、2.2）。

**3. defer 的执行时机和顺序是什么？**
`defer` 在外层函数返回前（`return` 赋值之后）以 **LIFO** 顺序执行。多个 `defer` 像栈一样，最后注册的最先执行。`panic` 时也会执行（沿调用栈向上逐层执行 defer），直到被 `recover` 截获或进程崩溃（见 1.7）。

**4. `defer` 里修改返回值，调用方能看到吗？**
要看有没有用**命名返回值**。有命名返回值时，`return expr` 先赋值给命名返回值变量，再执行 defer，所以 defer 中对命名返回值的修改**会反映到调用方**。没有命名返回值时，`return` 创建的是临时拷贝，defer 无法修改（见 1.3、3.1）。

**5. `panic` 和 `recover` 的使用场景？**
`panic` 用于表示程序进入了**不可恢复的状态**（如数组越界、nil 指针解引用、不变量被违反）。`recover` 用于边界层（如 HTTP handler、goroutine 顶层）**捕获并记录 panic**，防止整个进程崩溃，通常和 `defer` 配合；但正常的错误流程应该用 `error` 返回值，不应滥用 panic/recover 模拟异常控制流。

**6. `defer` 的参数什么时候被求值？**
**立即求值**（在 `defer` 语句执行时），而不是延迟到调用时。若想让参数在执行时才求值，应把逻辑包在闭包里（`defer func() { use(x) }()`），这样 `x` 是通过引用捕获，会在执行时读取当前值（见 3.3）。

**7. 匿名函数和普通函数有什么区别？**
匿名函数没有名字，不能被其他包引用，但可以赋值给变量、立即调用（IIFE）、或作为参数/返回值传递。匿名函数可以形成**闭包**，捕获外层作用域的变量；普通命名函数只能通过参数获取外部数据。

**8. 可变参数 `...T` 和传入 `[]T` 有什么区别？**
在函数内部没有区别，都是 `[]T`。区别在调用侧：`...T` 允许零个或多个单独值传入；`[]T` 必须显式传 slice；把已有的 `[]T` 传给 `...T` 参数时，需要写 `s...` 展开，**不会拷贝底层数组**。

**9. 函数值（function value）是什么类型？可以比较吗？**
函数值是**不可比较类型**，只能和 `nil` 比较，不能作为 map key，两个函数变量之间不能用 `==`（类似 slice/map）。函数类型包含参数列表和返回值列表，参数/返回值类型完全相同的函数才属于同一类型。

**10. 如何让函数的局部变量不逃逸到堆上？**
不返回其地址、不被闭包捕获、不赋值给接口、不通过 `chan` 发送——让编译器的逃逸分析确认变量生命周期限于当前栈帧，就会分配在栈上。用 `go build -gcflags='-m'` 验证，减少堆分配能降低 GC 压力，提升性能（见 2.4）。

**11. `defer` 在循环中使用有什么问题？**
循环中的 `defer` 在**函数返回时**才执行，而不是每次迭代结束时。如果在循环里打开资源并 defer 关闭，资源会在函数退出前一直累积，可能导致文件描述符耗尽等问题。解决办法：把每次迭代的资源操作封装到独立函数中，让 defer 在该函数返回时立即执行。

**12. Go 的函数调用是否有开销？defer 有额外开销吗？**
普通函数调用的主要开销是参数/返回值的寄存器/栈操作。`defer` 在 Go 1.14 后对**数量固定的简单场景**做了内联优化（open-coded defer），几乎无额外开销；在循环或运行时才知道数量的情况下退化为链表操作，有分配和遍历开销。`panic/recover` 路径因为需要展开调用栈，开销较大，不应用于正常控制流。
