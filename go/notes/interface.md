# 接口

> 环境：`go version go1.26.3`。接口内部结构（`eface`/`iface`/`itab`）以该版本源码（`runtime/iface.go`、`internal/abi/iface.go`）为准；1.23 之前 `itab` 定义在 `runtime` 包内、字段名是小写的 `inter`/`_type`/`hash`/`fun`，现已上移到 `internal/abi.ITab`（`Inter`/`Type`/`Hash`/`Fun`），语义不变，不同版本注意字段名差异。

## 一、基础使用

### 1.1 接口声明与隐式实现

```go
type Animal interface {
    Sound() string
}

type Dog struct{}
func (Dog) Sound() string { return "Woof" }

var a Animal = Dog{} // Dog 没有显式声明"implements Animal"，只要方法集匹配就自动满足
fmt.Println(a.Sound())
```

- Go 接口是**结构化类型（structural typing）**：没有 `implements` 关键字，只要一个类型的方法集包含接口要求的全部方法，就自动满足该接口，实现和接口定义可以分处不同包、互不知道对方存在。
- 接口本身只定义方法签名，不能有字段、不能有实现。

### 1.2 接口变量的赋值与调用

```go
var w io.Writer = os.Stdout // 具体类型 *os.File 赋给接口变量
n, err := w.Write([]byte("hi"))
```

- 接口变量在运行时持有**动态类型 + 动态值**两部分信息（详见 2.1），调用 `w.Write(...)` 时按动态类型分派到具体实现。

### 1.3 空接口 `interface{}` / `any`

```go
var x interface{} = 42
x = "hello"
x = struct{ N int }{1}

func Print(v any) { fmt.Println(v) } // any 是 interface{} 的别名（Go 1.18+）
```

- 空接口不要求任何方法，可以持有**任意类型**的值，常用于需要处理未知类型的通用容器/函数（如 `fmt.Println(args ...any)`、`encoding/json` 的反序列化目标）。
- Go 1.18 起优先用 `any` 书写，语义与 `interface{}` 完全相同，只是可读性更好；很多场景下**应优先考虑泛型**而不是空接口（见 3.5）。

### 1.4 类型断言（type assertion）

```go
var x interface{} = "hello"

s := x.(string)         // 不安全形式：断言失败直接 panic
s, ok := x.(string)     // 安全形式：断言失败 ok=false，s 为该类型零值，不 panic
n, ok := x.(int)        // ok=false, n=0
```

- `x.(T)` 用于把接口值还原成具体类型或另一个接口；两个参数返回值的形式是工程中的默认写法（见 3.3）。

### 1.5 类型选择（type switch）

```go
func describe(x interface{}) string {
    switch v := x.(type) {
    case int:
        return fmt.Sprintf("int: %d", v)
    case string:
        return fmt.Sprintf("string: %q", v)
    case nil:
        return "nil"
    default:
        return fmt.Sprintf("unknown: %T", v)
    }
}
```

- `switch v := x.(type)` 是多路类型断言的语法糖，每个 `case` 分支里 `v` 的静态类型就是该分支匹配到的类型；`case nil` 专门匹配接口值为 `nil` 的情况（见 2.5）。

### 1.6 接口的组合（interface embedding）

```go
type Reader interface{ Read(p []byte) (n int, err error) }
type Writer interface{ Write(p []byte) (n int, err error) }

type ReadWriter interface {
    Reader // 嵌入接口：把 Reader 的方法集并入 ReadWriter
    Writer
}
```

- 接口可以嵌入其它接口，等价于把被嵌入接口的方法集合并进来，是标准库里 `io.ReadWriter`、`io.ReadWriteCloser` 这类组合接口的构造方式。

### 1.7 面向接口编程：用接口解耦调用方与实现

```go
type Interface interface { // sort.Interface
    Len() int
    Less(i, j int) bool
    Swap(i, j int)
}

func Sort(data Interface) { /* 只依赖三个方法，不关心具体是什么类型 */ }
```

- 函数参数声明为接口类型时，只依赖接口定义的行为，不依赖具体类型，调用方传入任何满足接口的类型都能工作——这是 Go 里实现依赖倒置、便于测试打桩（mock）的核心手段。

## 二、底层原理

### 2.1 接口值的两种运行时表示：`eface` 与 `iface`

Go 编译器按接口是否为空接口，生成两种不同的底层结构（`internal/abi/iface.go`）：

```go
// 空接口 interface{} / any 的运行时布局
type EmptyInterface struct {
    Type *Type          // 动态类型信息
    Data unsafe.Pointer // 指向动态值的指针
}

// 非空接口（至少有一个方法）的运行时布局
type NonEmptyInterface struct {
    ITab *ITab           // 类型信息 + 方法表，见 2.2
    Data unsafe.Pointer  // 指向动态值的指针
}
```

- 两者都是"类型信息 + 数据指针"的二元组，区别在于第一个字：空接口没有方法要分派，直接存 `*Type` 就够了；非空接口需要额外的方法表来支持运行时调用，所以多了一层 `*ITab` 把 `Type` 包了一层。
- 这也解释了为什么空接口和非空接口**不能直接相互赋值转换**（除了都赋 `nil`/走类型断言）：它们是两种不同的内存布局，编译器要生成不同的转换代码。

### 2.2 itab：接口类型 + 动态类型 → 方法表

```go
type ITab struct {
    Inter *InterfaceType // 接口类型信息
    Type  *Type          // 动态（具体）类型信息
    Hash  uint32          // Type.Hash 的拷贝，用于 type switch 加速比较
    Fun   [1]uintptr      // 变长数组：按接口方法声明顺序，存具体类型对应方法的入口地址
}
```

- `itab` 由 `(接口类型, 动态类型)` 这一对唯一确定，作用是把"调用接口第 k 个方法"翻译成"调用 `itab.Fun[k]` 指向的具体函数"——这就是接口方法调用的动态派发（虚函数表的 Go 版本）。
- 运行时维护一张全局哈希表 `itabTable` 缓存所有已构建过的 `itab`（`runtime/iface.go` 的 `getitab`）：第一次发生某个 `(接口, 类型)` 组合的转换时，遍历接口的方法列表、逐个到具体类型的方法表里查找实现地址，构建 `itab` 并写入缓存；此后同样的组合直接查表命中，不再重新计算，因此"把值赋给接口"这个操作本身通常很轻量。
- 如果具体类型没有实现接口要求的某个方法，`itabInit` 会记录下第一个缺失的方法名；这正是类型断言失败时错误信息里 "missing method Xxx" 的来源。

### 2.3 接口赋值时的"装箱"与内存分配

```go
var i int = 42
var x interface{} = i // 把 i 的值拷贝进接口的 Data
```

- 把一个非指针值赋给接口变量时，运行时需要一份该值的**独立拷贝**供 `Data` 指针指向（接口存的是指针，不能直接存值本身），这个过程俗称"装箱（boxing）"。
- 若原值本来就是指针（如 `*T`），装箱几乎零开销，`Data` 直接存该指针；若是较大的值类型（如大 `struct`），通常需要在堆上分配一份拷贝，带来额外的内存分配和 GC 压力（见 3.5）。
- 运行时对小整数做了专门优化：`runtime/iface.go` 里的 `staticuint64s [256]uint64` 预先缓存了 0~255 的整数，把 `int`/`byte` 等小值类型装箱进接口时若值落在这个范围，直接复用这份只读的静态内存，不必每次都分配（`convT64` 等函数）。

### 2.4 接口值的比较

```go
var a, b interface{} = 1, 1
fmt.Println(a == b) // true：动态类型相同（int）且动态值相等

var c, d interface{} = 1, "1"
fmt.Println(c == d) // false：动态类型不同，直接判 false，不比较值
```

- 两个接口值相等，当且仅当**动态类型相同**且**动态值相等**（用 `==` 递归比较，规则与对应具体类型的 `==` 一致，见 struct.md 2.4）。
- 若动态类型是不可比较类型（slice、map、function），比较时运行时直接 `panic: comparing uncomparable type`（见 3.2）。

### 2.5 nil 接口值 vs "接口里装了 nil"

```go
var e EmptyInterface // 对应 var x interface{}
e.Type == nil && e.Data == nil // 接口变量真正等于 nil 的条件
```

- 接口变量 `== nil` 当且仅当**动态类型指针和数据指针都为 nil**——也就是这个接口变量从未被赋过任何值。
- 如果把一个类型化的 `nil` 指针（如 `var p *T = nil`）赋给接口变量，`Type` 会被设成 `*T` 的类型信息（非 nil），只有 `Data` 是 nil，此时接口变量本身 **`!= nil`**——这是 Go 里最著名的陷阱之一（见 3.1）。

## 三、常见陷阱

### 3.1 "nil 接口" 陷阱：类型化的 nil 不等于接口 nil

```go
type MyError struct{}
func (*MyError) Error() string { return "boom" }

func doSomething() *MyError { return nil } // 显式返回 nil 指针

func run() error {
    var err *MyError = doSomething()
    return err // 危险：把 (*MyError)(nil) 赋给 error 接口
}

func main() {
    if err := run(); err != nil { // 恒为 true！
        fmt.Println("failed:", err) // 输出 failed: <nil>
    }
}
```

- 根源见 2.5：`err` 是 `*MyError` 类型的 `nil`，赋给 `error` 接口后 `Type = *MyError`、`Data = nil`，接口整体不等于 `nil`。
- 正确写法：函数直接返回 `error` 接口类型，成功路径显式 `return nil`（此时是真正的接口 `nil`），而不是返回具体的指针类型再赋给接口；或者在 `run()` 里手动判断 `if err == nil { return nil }` 后再返回。

### 3.2 不可比较的动态类型放进接口比较会 panic

```go
var a, b interface{} = []int{1}, []int{2}
fmt.Println(a == b) // panic: runtime error: comparing uncomparable type []int
```

- 接口类型本身声明为"可比较"是**静态编译期**放行的（`interface{}` 支持 `==` 语法），但**运行时**才真正检查动态类型是否可比较（见 2.4），装的是 slice/map/function 时必然 panic。
- 用作 `map` 的 key、或放进需要比较的容器时要格外小心动态类型，必要时先用 `reflect.DeepEqual` 或自定义比较逻辑。

### 3.3 类型断言用了不安全形式，断言失败直接 panic

```go
var x interface{} = "hello"
n := x.(int) // panic: interface conversion: interface {} is string, not int
```

- 除非能**确定**断言一定成功（如刚做过 `type switch` 分支判断），否则一律用 `v, ok := x.(T)` 的双返回值形式，`ok == false` 时优雅处理而不是让程序崩溃（见 1.4）。

### 3.4 接口频繁转换带来的堆分配开销

```go
func LogValue(v interface{}) { /* ... */ }

for i := 0; i < 1e6; i++ {
    LogValue(bigStruct{...}) // 每次调用都触发一次装箱拷贝
}
```

- 见 2.3：把非指针的大值类型反复装箱进空接口传参（常见于早期日志库、`fmt.Sprintf` 类可变参数函数），会产生大量堆分配，是热路径上容易被忽视的性能问题。
- 常见优化：改用指针传参（`*bigStruct` 装箱几乎零开销）、或用泛型（`func LogValue[T any](v T)`，在编译期单态化，避免接口装箱）代替 `interface{}` 参数。

### 3.5 用空接口代替泛型，丢失类型安全

```go
func Max(a, b interface{}) interface{} { // Go 1.18 前的常见写法
    switch a.(type) {
    case int:
        if a.(int) > b.(int) { return a }
        return b
    // ... 每种类型都要写一遍 case，且调用方传错类型只能运行时发现
    }
    return nil
}

func Max[T cmp.Ordered](a, b T) T { // 现在应优先用泛型
    if a > b { return a }
    return b
}
```

- `interface{}` 参数把类型检查从编译期推迟到运行时，需要大量类型断言/`type switch` 兜底，调用方传错类型编译器也不会报错；Go 1.18 引入泛型后，这类"要处理多种具体类型但逻辑一致"的场景应优先用泛型，只有真正需要"运行时不确定类型、需要动态判断"时才用接口（如 1.5 的 `type switch` 场景、插件式架构）。

## 四、常见面试题

**1. Go 接口是如何实现"隐式实现"的？和 Java/C++ 的接口有什么本质区别？**
Go 用结构化类型：类型不需要声明"我实现了某接口"，只要方法集包含接口要求的全部方法就自动满足，接口和实现可以分处不同、互不知情的包中；Java/C++ 是名义类型（nominal typing），必须显式 `implements`/继承。这让 Go 里可以先定义一个小接口，再让已有类型"事后满足"它，无需修改已有代码（见 1.1）。

**2. 接口值在运行时是什么结构？空接口和非空接口一样吗？**
不一样。接口值本质是"类型信息 + 数据指针"的二元组，但空接口（`EmptyInterface`）第一个字直接是 `*Type`；非空接口（`NonEmptyInterface`）第一个字是 `*ITab`（内部再包一层 `Type`），因为非空接口还需要一张方法表支持动态派发（见 2.1）。

**3. itab 是什么？什么时候构建、要不要每次都重新计算？**
`itab` 是 `(接口类型, 动态类型)` 这一对信息对应的方法表，把"调用接口的第 k 个方法"翻译成"具体类型上第 k 个方法的函数地址"。运行时用全局哈希表缓存所有构建过的 `itab`，同一对 `(接口, 类型)` 只在第一次转换时构建一次，之后都是查表复用，转换本身开销很小（见 2.2）。

**4. 为什么 `var err error = (*MyError)(nil)` 之后 `err != nil`？**
接口变量等于 `nil` 要求类型指针和数据指针都为 `nil`。把一个类型化的 `nil` 指针赋给接口后，接口的类型部分被设成了 `*MyError`（非 nil），只有数据部分是 `nil`，所以接口整体不等于 `nil`（见 2.5、3.1）。这是函数返回值类型不应该用具体错误类型指针、而应该直接用 `error` 接口类型的原因之一。

**5. 两个接口值用 `==` 比较，规则是什么？什么情况下会 panic？**
先比较动态类型是否相同，不同直接 `false`；相同再递归比较动态值是否相等。如果动态类型是不可比较类型（slice、map、func），运行时会 panic（见 2.4、3.2）。

**6. 类型断言 `x.(T)` 和类型选择 `switch x.(type)` 有什么关系？**
`type switch` 本质是多路类型断言的语法糖，编译器会把每个 `case` 展开为一次类型断言（内部实际是按 `itab`/`Type` 做比较，用 `Hash` 字段加速），只是把多次判断合并到一个结构里，同时能拿到每个分支里正确静态类型的变量 `v`（见 1.4、1.5、2.2）。

**7. 把一个 `struct` 值赋给 `interface{}` 会发生什么？**
会触发"装箱"：运行时在堆上（或利用逃逸分析优化在栈上，视情况而定）分配一份该值的拷贝，接口的 `Data` 指针指向这份拷贝；如果原值本身已经是指针，则直接存指针，几乎零开销。小整数（0~255）有专门的静态缓存优化，不必每次分配（见 2.3）。

**8. 为什么说"频繁把大结构体传给 `interface{}` 参数"是性能陷阱？怎么优化？**
每次传参都要重新装箱拷贝一份数据到堆上，产生额外分配和 GC 压力。优化手段：改传指针（`*T` 装箱几乎无开销）；或者用泛型替代 `interface{}` 参数，编译期单态化生成具体类型版本的函数，完全避免装箱（见 2.3、3.4、3.5）。

**9. 什么时候该用接口，什么时候该用泛型？**
需要"同一段逻辑处理多种已知具体类型、编译期就能确定"时优先用泛型（类型安全、无装箱开销）；需要"运行时才知道具体类型、按类型做不同分支处理，或者要对外暴露一组行为契约、隐藏具体实现（依赖倒置/插件式架构）"时用接口。两者不是互斥的，标准库里两者常常配合使用（见 1.7、3.5）。

**10. 接口嵌入和结构体嵌入是同一种机制吗？**
不是。结构体嵌入（method.md 1.5、struct.md 1.5）是把字段/方法"提升"到外层类型，编译器生成转发方法；接口嵌入只是把被嵌入接口的方法签名**合并**进外层接口的方法集合，接口本身不含任何实现，没有"提升"和转发的概念，纯粹是方法列表在编译期的拼接（见 1.6）。
