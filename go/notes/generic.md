# 泛型

> 环境：`go version go1.26.3`。泛型自 Go 1.18 引入，之后语言层面有几次重要演进，本文按 1.26 的行为书写，涉及版本差异处会单独标注：
> - **1.18**：类型参数、类型集约束、`any`/`comparable` 预声明标识符。
> - **1.20**：放宽 `comparable` 的约束满足规则（普通可比较类型/接口也能"满足"`comparable`，见 1.4）。
> - **1.21**：类型推断大幅增强；`cmp`、`slices`、`maps` 进入标准库。
> - **1.24**：支持**泛型类型别名**（`type A[T any] = B[T]`）。
> - **1.25**：规范中删除了"核心类型（core type）"这个概念，改为对 `range`/`make`/索引/收发等操作**逐条**定义合法性规则（见 2.5）。
>
> 实现细节以 `cmd/compile/internal/noder/{reader.go,writer.go}`、`cmd/compile/internal/typecheck/subr.go` 该版本源码为准。

## 一、基础使用

### 1.1 类型参数与实例化

```go
// T 是类型参数（type parameter），int | float64 | string 是它的约束（constraint）
func Max[T int | float64 | string](a, b T) T {
    if a > b {
        return a
    }
    return b
}

Max[int](1, 2)   // 显式实例化：把类型实参（type argument）int 填给 T
Max(1, 2)        // 类型推断：由实参推出 T = int
f := Max[int]    // 实例化后的函数是一个普通函数值
```

- `[T C]` 写在函数名之后、参数列表之前；多个类型参数用逗号分隔，共享约束时可写 `[K, V any]`。
- **泛型函数/类型必须先"实例化"成具体类型才能使用**，`f := Max` 这种写法是编译错误（见 3.9）。
- 实例化是**编译期**行为：`Max[int]` 和 `Max[float64]` 在编译后是两个不同的东西（具体共享到什么程度见 2.2）。

### 1.2 约束就是接口：从"方法集"到"类型集"

Go 1.18 把接口的含义从"方法集"推广为"**类型集（type set）**"：一个接口定义了一个类型的集合，普通接口的类型集就是"实现了这些方法的所有类型"，约束接口还可以直接列出类型。

```go
type Number interface {
    ~int | ~int64 | ~float64      // 类型项（type term）的并集（union）
}

type Stringish interface {
    ~string
    fmt.Stringer                  // 类型项和方法可以同时出现：交集
}
```

- **约束的作用是双向的**：既限制调用方能传哪些类型实参，也决定了泛型函数体内**能对该类型做哪些操作**——只有类型集里所有类型都支持的操作才允许写（见 2.5）。
- 只含方法的接口叫**基本接口（basic interface）**，它既能当约束、也能当普通类型；**一旦接口里出现类型项，就只能当约束**：

```go
type Num interface{ ~int | ~float64 }

var x Num // 编译错误：
// cannot use type Num outside a type constraint: interface contains type constraints
```

- 约束只有一个类型项时可以省略 `interface{}` 外壳：`[T int]`、`[T ~[]byte]` 都合法。

### 1.3 `~T`：底层类型近似

```go
type MyInt int

func AddStrict[T int](a, b T) T   { return a + b }  // 只接受 int 本身
func AddLoose[T ~int](a, b T) T   { return a + b }  // 接受所有底层类型是 int 的类型

AddLoose(MyInt(1), MyInt(2))  // OK
AddStrict(MyInt(1), MyInt(2)) // 编译错误：
// MyInt does not satisfy int (possibly missing ~ for int in int)
```

- `~T` 表示"底层类型（underlying type）为 `T` 的所有类型"，不带 `~` 就是类型本身，**具名类型不会自动匹配**。工程上写约束时**默认加 `~`**，除非你确实要排除具名类型。
- `~T` 里的 `T` 有两条硬性限制，违反直接编译错误：

```go
type MyInt int
type C1 interface{ ~MyInt } // invalid use of ~ (underlying type of MyInt is int)
type C2 interface{ ~error } // invalid use of ~ (error is an interface)
```

即 `~` 后面必须是"自己就是自己底层类型"的类型，且不能是接口。

### 1.4 预声明约束：`any` 与 `comparable`

```go
func Print[T any](v T)                       {}          // any = interface{}，无任何限制
func Index[T comparable](s []T, v T) int     { return 0 } // comparable：支持 == / != / 作 map key
```

`comparable` 有一个**容易踩的历史细节**：Go 1.20 起，"满足（satisfies）"约束和"实现（implements）"接口不再是同一件事。规范里的例外规则是：

> 若约束能写成 `interface{ comparable; E }`（`E` 是基本接口），只要类型实参**是可比较类型**且实现 `E`，就算满足该约束——即使它不是"严格可比较（strictly comparable）"。

```go
type Set[K comparable] map[K]struct{}

s := Set[any]{}          // Go 1.20+ 合法：any 可比较，"满足"但并不"实现" comparable
s[1] = struct{}{}        // OK
s[[]int{1}] = struct{}{} // 运行时 panic: hash of unhashable type []int
```

规范给出的判定表（节选自 `doc/go_spec.html`）：

| 类型实参 | 约束 | 是否满足 |
| --- | --- | --- |
| `string` | `comparable` | 满足（严格可比较） |
| `[]byte` | `comparable` | 不满足（切片不可比较） |
| `any` | `comparable` | **满足**（可比较且实现基本接口 `any`） |
| `struct{ f any }` | `comparable` | 满足 |
| `any` | `interface{ comparable; m() }` | 不满足（`any` 没实现 `m()`） |

**结论**：`comparable` 只保证编译期能写 `==`，**不保证运行时不 panic**（见 3.6）。

有序比较用标准库的 `cmp.Ordered`（Go 1.21+），不要自己重复造：

```go
import "cmp"

func Min[T cmp.Ordered](a, b T) T { if a < b { return a }; return b }
```

### 1.5 泛型类型及其方法

```go
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
    var zero T                      // 类型参数的零值只能这么拿
    if len(s.items) == 0 {
        return zero, false
    }
    v := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return v, true
}

s := &Stack[int]{} // 泛型类型必须实例化后才能使用
```

- 方法的接收者上要重复写出类型参数名（`*Stack[T]`），这里的 `T` 是**声明**而不是使用，名字可以和类型声明处不同（但不建议）。
- **方法不能引入自己的类型参数**，这是语言的硬性设计决定，不是暂时限制：

```go
func (l *List[T]) Map[U any](f func(T) U) []U { ... }
// syntax error: method must have no type parameters
```

需要额外类型参数时只能写成顶层函数：`func Map[T, U any](l *List[T], f func(T) U) []U`（见 3.3）。

- **泛型类型别名**（Go 1.24+）：

```go
type Pair[K comparable, V any] struct{ K K; V V }
type StrPair[V any] = Pair[string, V]   // 1.24 之前这行编译不过

fmt.Println(StrPair[int]{"a", 1}) // {a 1}
```

### 1.6 类型推断

能推断的主要是这几类信息：**函数实参的类型** → 类型参数；已确定的类型参数 → 约束里出现的其他类型参数（约束类型推断，见 1.7）。

```go
Max(1, 2)              // 由实参推出 T = int
Max(1, 2.5)            // 两个都是无类型常量，取默认类型 → T = float64，合法

var i int = 3
Max(i, 2.5)            // 编译错误：i 已定型为 int，2.5 无法转成 int
// cannot use 2.5 (untyped float constant) as int value in argument to Max (truncated)
```

**不能从返回值/赋值目标反推**：

```go
func Zero[T any]() T { var z T; return z }

var x int = Zero()     // 编译错误：
// in call to Zero, cannot infer T (declared at ...)
var y int = Zero[int]() // 只能显式实例化
```

### 1.7 约束类型推断：`S ~[]E` 惯用法

这是标准库 `slices` 里满屏都是的写法，目的是**保住具名类型**：

```go
type Nums []int
func (n Nums) String() string { return fmt.Sprint([]int(n)) }

// 坏写法：返回值退化成 []int
func ScaleBad[E ~int](s []E, c E) []E { ... }

// 好写法：S 被推断为 Nums，返回值仍是 Nums
func ScaleGood[S ~[]E, E ~int](s S, c E) S { ... }

n := Nums{1, 2}
var a fmt.Stringer = ScaleGood(n, 2) // OK
var b fmt.Stringer = ScaleBad(n, 2)  // 编译错误：
// []int does not implement fmt.Stringer (missing method String)
```

推断过程：由实参 `n` 得到 `S = Nums`；再由约束 `S ~[]E` 反推 `E = int`——后一步就是"约束类型推断"。

### 1.8 标准库里的泛型

| 包 | 典型 API |
| --- | --- |
| `cmp` | `Ordered`、`Compare`、`Less`、`Or`（1.22） |
| `slices` | `Sort`/`SortFunc`、`Contains`、`Index`、`BinarySearch`、`Max`/`Min`、`Clone`、`Compact`、`Insert`/`Delete`、`Reverse`、`Chunk`、`All`/`Values`/`Collect`（配合 `iter`） |
| `maps` | `Keys`/`Values`/`All`（返回 `iter.Seq`）、`Clone`、`Copy`、`Equal`、`DeleteFunc`、`Collect`、`Insert` |
| `sync` | `OnceValue[T]`、`OnceValues[T1, T2]` |
| `sync/atomic` | `Pointer[T]`（类型安全的原子指针，替代 `atomic.Value` 的运行时类型检查） |

注意 `maps.Keys` 返回的是**迭代器** `iter.Seq[K]` 而不是切片，要切片用 `slices.Collect(maps.Keys(m))`（见 iter.md）。

### 1.9 什么时候用泛型，什么时候用接口

- **用泛型**：同一段算法逻辑要作用在多个具体类型上，且这些类型**编译期已知**——容器、算法、工具函数（`Map`/`Filter`/`Reduce`）、避免 `any` 装箱的热路径。
- **用接口**：要表达的是**行为契约 / 依赖倒置**，实现方在运行时才确定、可插拔、可 mock——`io.Reader`、存储层抽象、插件式架构。
- **判据**：如果你写完发现类型参数只在参数和返回值上出现、函数体内只调它的方法而不做任何和具体类型有关的操作，那这里其实一个接口就够了，用泛型反而多了一层字典开销（见 2.6、3.11）。

## 二、底层原理

### 2.1 三条实现路线与 Go 的选择

| 路线 | 代表 | 优点 | 缺点 |
| --- | --- | --- | --- |
| 完全单态化（monomorphization） | C++ 模板、Rust | 运行时零开销，可充分内联 | 代码膨胀、编译慢 |
| 完全装箱/擦除 | Java 泛型 | 只有一份代码 | 每次都要装箱、运行时开销 |
| **GC Shape Stenciling + Dictionaries** | **Go 1.18+** | 折中：按"形状"分组生成代码 | 形状内共享代码需要额外查字典 |

Go 的做法：把类型实参按 **GC shape（GC 形状）** 分组，**每组形状生成一份机器码**；每个具体实例化再额外生成一份只读的**字典（dictionary）**，把这份代码里"跟具体类型有关"的东西（类型描述符、itab、方法地址、子字典）作为数据传进去。

### 2.2 shape 类型：哪些实例共享同一份代码

规则在 `cmd/compile/internal/noder/reader.go` 的 `shapify()`：

```go
func shapify(targ *types.Type, basic bool) *types.Type {
    // basic 表示该类型参数的约束是不是"基本接口"（只有方法集，没有类型项）
    pointerShaping := basic && targ.IsPtr() && !targ.Elem().NotInHeap()
    under := targ.Underlying()
    if pointerShaping {
        under = types.NewPtr(types.Types[types.TUINT8]) // 所有指针统一成 *uint8
    }
    sym := types.ShapePkg.Lookup(under.LinkString())    // 放进虚拟包 go.shape
    ...
}
```

归纳成两条：

1. **默认情况：shape = 类型实参的底层类型**。所以 `int` 和 `type MyInt int` 共享代码，`int` 和 `int64`、`float64` 不共享。
2. **例外：约束是基本接口（只有方法、没有类型项）且类型实参是指针时，所有指针塌缩成 `go.shape.*uint8`**。因为这种情况下生成的代码根本不会直接操作指针指向的内容，方法调用一律走字典，指针指向什么完全无关紧要。

实测（`//go:noinline` 防止被内联掉，再用 `go tool nm` 看符号）：

```text
# 场景一：约束含类型项（~int | ~int64 | ~float64），按底层类型分组
main.Sum[go.shape.int]              <- Sum[int] 和 Sum[MyInt] 共用这一份代码
main.Sum[go.shape.int64]
main.Sum[go.shape.float64]
main..dict.Sum[int]                 <- 但字典是每个实例一份
main..dict.Sum[main.MyInt]
main..dict.Sum[int64]
main..dict.Sum[float64]

# 场景二：约束是基本接口（Stringer），指针实参全部塌缩
main.Show[go.shape.*uint8]          <- Show[*A] 和 Show[*B] 共用一份代码
main..dict.Show[*main.A]
main..dict.Show[*main.B]

# 场景三：约束 any，非指针实参各自成形
main.Id[go.shape.*uint8]            <- Id[*int]、Id[*string] 共用
main.Id[go.shape.[]int]
main.Id[go.shape.map[int]int]
```

补充：为避免超长类型名（protobuf 大结构体）撑爆符号表，`shapify` 会在名字过长时改用 hash（`-d=maxshapelen`）。

### 2.3 字典里装了什么

字典的内容在 `objDictIdx()` 里按顺序写出，共四类（`readerDict`）：

```go
dict.typeParamMethodExprs // 类型参数上的方法：具体类型对应方法的函数地址
dict.subdicts             // 本函数内部再调用其他泛型函数时要用的"子字典"
dict.rtypes               // 派生类型的运行时类型描述符 *runtime._type
dict.itabs                // (具体类型, 接口) 对应的 itab，供接口转换用
```

实测 dump（`go build -gcflags=-S`）：

```text
main..dict.Show[main.A] SRODATA dupok size=24
    rel 0+8 t=R_ADDR   main.A.String+0        <- 槽 0：方法地址
    rel 8+8 t=R_ADDR   type:main.A+0          <- 槽 1：A 的 rtype
    rel 0+0 t=R_USEIFACE type:interface {}+0  <- 函数体里 var s any = v 要用到
```

几个直接推论：

- 字典是 `SRODATA dupok`，即**只读数据、可跨编译单元去重**，运行时不会修改，也不参与 GC 扫描。
- **`reflect` 看到的是真类型，不是 shape 类型**，因为 rtype 是从字典里取的：

```go
func TypeOf[T any](v T) string {
    return fmt.Sprintf("%v / %v", reflect.TypeOf(v), reflect.TypeFor[T]())
}
TypeOf(MyInt(1)) // main.MyInt / main.MyInt   （和 TypeOf(int(1)) 共用同一份代码）
TypeOf(int(1))   // int / int
```

### 2.4 一次泛型方法调用的汇编

字典作为**隐藏的第一个参数**传入（amd64 regabi 下放在 `AX`，真实参数顺延到 `BX`…）：

```text
# 调用方 main：
LEAQ  main..dict.Show[main.A](SB), AX   ; 字典地址 → AX
CALL  main.Show[go.shape.struct { main.n int }](SB)

# 被调方 Show，函数体里的 v.String()：
MOVQ  (AX), CX     ; 从字典槽 0 取出 main.A.String 的地址
MOVQ  AX, DX
MOVQ  BX, AX       ; 真正的参数 v
CALL  CX           ; 间接调用
```

**这是理解泛型性能的关键**：类型参数上的方法调用编译成 `CALL CX` 这种**间接调用**，和接口的动态派发是同一个量级，**无法内联、无法去虚化**。编译器只能内联"泛型函数整体"（因为调用点上 shape 和字典都是静态已知的），一旦函数体超过内联预算就彻底失去优化机会：

```text
./a.go:11:6: cannot inline sumGeneric[main.Cnt]: function too complex: cost 86 exceeds budget 80
```

### 2.5 操作合法性：从"核心类型"到逐条规则

Go 1.25 之前，规范用"核心类型（core type）"统一描述"什么操作能作用在类型参数上"；1.25 起该概念被删除（`doc/go_spec.html` 里已搜不到 "core type"），改为对 `range`、`make`、索引、`len`、通道收发等**逐个操作**给出规则。实际效果和以前基本一致：**类型集里所有类型的底层类型必须相同**（通道另有方向上的宽松规则）。

```go
// ✗ 底层类型不同：range / make 都报错
func Count[T ~[]byte | ~string](v T) int { for range v { } ; return 0 }
// cannot range over v (variable of type T constrained by ~[]byte | ~string):
//   []byte and string have different underlying types

func Make[T ~[]int | ~map[string]int](n int) T { return make(T, n) }
// invalid argument: cannot make T: []int and map[string]int have different underlying types

// ✓ 底层类型相同的多个具名类型
type IntSlice []int
type Nums []int
func Sum[S IntSlice | Nums](s S) int { total := 0; for _, v := range s { total += v }; return total }

// ✓ 通道的方向例外：元素类型相同即可
func Recv[T ~chan int | ~<-chan int](c T) int { return <-c }
```

### 2.6 性能实测：泛型 ≠ 更快

环境：`go1.26.3 darwin/amd64`，Intel i5-1038NG7。

**场景一：类型参数上的方法调用（1000 个元素求和）**

```text
BenchmarkConcrete-8    2564491     469.3 ns/op    0 B/op   0 allocs/op   # 具体类型，方法被内联
BenchmarkGeneric-8      765192    1589   ns/op    0 B/op   0 allocs/op   # 泛型，走字典间接调用
BenchmarkIface-8        693045    1769   ns/op    0 B/op   0 allocs/op   # 接口，走 itab 动态派发
```

泛型比具体类型慢 **3.4x**，和接口基本持平——因为两者都是间接调用且都无法内联（见 2.4）。

**场景二：避免 `any` 装箱（128 字节结构体逃逸）**

```text
BenchmarkBoxAny-8         33932595    34.98 ns/op    128 B/op    1 allocs/op
BenchmarkNoBoxGeneric-8  276205976     4.34 ns/op      0 B/op    0 allocs/op
```

泛型快 **8x**，且零分配——因为值以原本的形状传递，不需要装箱到接口里（interface.md 2.3）。

**结论**：泛型的性能收益来自**消除装箱**，不来自"消除动态派发"。约束里只有方法的泛型，性能和接口一样；约束里有类型项（可以直接做运算/索引）的泛型，才真正省掉了装箱和断言。

## 三、常见陷阱

### 3.1 含类型项的接口当普通类型用

```go
type Num interface{ ~int | ~float64 }

var x Num              // ✗ cannot use type Num outside a type constraint
func f(n Num) {}       // ✗ 同上
func g[T Num](n T) {}  // ✓ 只能当约束
```

**原因**：类型集里的类型没有共同的方法表，运行时无法为它构造 itab，这种接口纯粹是编译期概念。

### 3.2 忘了 `~`，具名类型不满足约束

```go
type Celsius float64 // 实际项目里满地都是这种具名类型：Duration、UserID、Score...

func Sum[T int | float64](xs []T) T { ... }

Sum([]Celsius{1, 2}) // ✗ Celsius does not satisfy float64 (possibly missing ~ for float64)
```

**正确写法**：约束一律写 `~int | ~float64`，或者直接用 `cmp.Ordered`（它内部就是 `~` 形式）。

### 3.3 方法不能有类型参数

```go
type List[T any] struct{ items []T }
func (l *List[T]) Map[U any](f func(T) U) []U { ... } // ✗ method must have no type parameters
```

**原因**：允许参数化方法会破坏接口的可实现性判定（一个带类型参数的方法对应无穷多个签名，无法建 itab）。

**正确写法**：提为顶层函数。

```go
func Map[T, U any](l *List[T], f func(T) U) []U { ... }
```

这也是为什么标准库是 `slices.Map` 风格而不是 `s.Map()` 风格。

### 3.4 不能直接对类型参数做类型断言 / type switch

```go
func F[T any](v T) {
    switch v.(type) { case int: } // ✗
    // cannot use type switch on type parameter value v (variable of type T constrained by any)
}
```

**正确写法**：先转成接口。

```go
func F[T any](v T) {
    switch x := any(v).(type) {
    case int:    fmt.Println("int", x)
    case string: fmt.Println("string", x)
    }
}
```

但要警惕：一旦写出 `any(v).(type)`，说明你在为不同类型写不同逻辑，**这时候泛型往往是错的抽象**，应该考虑接口 + 多态，或者干脆写多个函数。

### 3.5 `v == nil` 非法；零值判断需要 `comparable`

```go
func IsNil[T any](v T) bool  { return v == nil } // ✗ mismatched types T and untyped nil
func IsZero[T any](v T) bool { var z T; return v == z } // ✗ incomparable types in type set
```

**正确写法**：

```go
func IsZero[T comparable](v T) bool { var z T; return v == z } // 约束收紧
func IsZeroAny[T any](v T) bool     { return reflect.ValueOf(&v).Elem().IsZero() } // 兜底方案
```

想表达"可能为 nil"就别用 `T any`，用 `*T` 或 `T ~*E | ~[]E`（能表达就表达在类型里）。

### 3.6 `comparable` 不保证运行时不 panic

```go
type Set[K comparable] map[K]struct{}
s := Set[any]{}
s[[]int{1}] = struct{}{} // panic: runtime error: hash of unhashable type []int
```

Go 1.20 起 `any` 能"满足"`comparable`（见 1.4）。**如果你的容器绝对不能在运行时崩，别把 key 类型参数暴露成 `comparable` 就完事**，要么在文档里写清楚，要么约束成具体的类型集（`~string | ~int`）。

### 3.7 返回 `[]E` 丢失具名类型

```go
func Filter[E any](s []E, f func(E) bool) []E   // ✗ 传 Nums 进去，出来是 []int
func Filter[S ~[]E, E any](s S, f func(E) bool) S // ✓ 传 Nums 进去，出来还是 Nums
```

具名切片类型上挂的方法（`String()`、`Len()`…）会在第一种写法里全部丢掉，调用方拿到的值不再实现原来的接口。**凡是"输入什么切片类型、输出就该是什么切片类型"的函数，都用 `S ~[]E` 双参数形式**——这就是 `slices` 包全员这么写的原因。

### 3.8 不能从返回值推断类型参数

```go
var m map[string]int = Make()      // ✗ cannot infer T
var m = Make[map[string]int]()     // ✓
```

**推论**：设计泛型 API 时，尽量让类型参数出现在**参数**里，否则调用方每次都得手写实例化，可读性很差。工厂函数可以改成"传一个零值/指针进去"：`func New[T any](proto T) *Box[T]`。

### 3.9 未实例化的泛型函数不能当值用

```go
g := Max            // ✗ cannot use generic function Max without instantiation
g := Max[int]       // ✓
var h func(int, int) int = Max[int] // ✓
```

同理，泛型函数不能直接作为 `func` 类型的参数传递，必须先实例化。

### 3.10 实例化循环（instantiation cycle）

```go
func F[T any](x T) { F([]T{x}) } // ✗ instantiation cycle: T instantiated as []T
```

每次递归调用都产生一个新类型（`int` → `[]int` → `[][]int` …），编译期无法收敛。编译器会直接报错而不是无限编译下去。**递归泛型函数的类型参数必须保持不变**。

### 3.11 把泛型当性能银弹

见 2.6 的实测：约束里只有方法时，泛型的方法调用走字典间接调用，和接口一样快（慢），还额外多了一次字典寻址；能内联的具体类型版本比它快 3 倍以上。

**什么时候泛型真的更快**：热路径上原本用 `any` 装箱大结构体（省下堆分配）、原本用 `interface{}` + 类型断言做数值运算（省下断言和装箱）。**什么时候不会更快**：原本就用小接口做方法派发的地方。

### 3.12 代码膨胀与编译变慢

shape 只按**底层类型**合并，所以 `Sum[int]`、`Sum[int64]`、`Sum[float64]`、`Sum[uint32]`… 每种底层类型都是一份独立机器码，加上每个实例一份字典。约束写得越宽、实例化的类型越多，二进制越大、编译越慢。

**缓解**：热点之外优先接口；类型参数数量控制在 1–2 个；不要为"可能将来会用"的类型提前放宽约束。

### 3.13 union 里不能放带方法的接口

```go
type Bad interface{ ~int | fmt.Stringer }
// ✗ cannot use fmt.Stringer in union (fmt.Stringer contains methods)
```

**原因**：类型项的并集是"类型集求并"，而带方法的接口的类型集是无穷的、且需要 itab 才能调用，两者语义无法合并。

**正确写法**：方法和类型项用**交集**（分行并列）而不是并集：

```go
type Good interface {
    ~int
    fmt.Stringer  // 含义：底层类型是 int 且实现了 Stringer
}
```

### 3.14 不同底层类型的 union 上什么操作都做不了

```go
func Len[T ~[]byte | ~string](v T) int { return len(v) } // len 恰好合法
func Cnt[T ~[]byte | ~string](v T) int { for range v {}; return 0 } // ✗ range 不行
```

写出一个"看起来很通用"的 union 之后，会发现函数体里几乎什么都不能写（见 2.5）。**约束不是越宽越好，宽到无法操作就没意义了**——这种情况正确的解法通常是重载成两个函数，或者接受一个转换函数。

### 3.15 用泛型模拟函数重载

```go
// ✗ 反模式
func Process[T int | string | []byte](v T) {
    switch x := any(v).(type) { ... } // 每个分支逻辑完全不同
}
```

类型参数只在签名上"统一"，函数体里立刻分叉，等于用泛型伪装重载：既没有类型安全收益，也没有性能收益，还让调用方看不清函数到底干什么。**逻辑不同就写不同的函数**（`ProcessInt`、`ProcessString`），这在 Go 里不是缺点。

## 四、常见面试题

**1. Go 泛型是怎么实现的？和 C++ 模板、Java 泛型有什么区别？**
Go 走的是折中路线 **GC Shape Stenciling + Dictionaries**：按类型实参的 GC 形状（基本上就是底层类型）分组，每组生成一份机器码；每个具体实例化再生成一份只读字典，把类型描述符、itab、方法地址、子字典作为隐藏参数传进去。C++ 是完全单态化（零运行时开销但代码膨胀），Java 是类型擦除（一份代码但全程装箱），Go 介于两者之间（见 2.1）。

**2. `Sum[int]` 和 `Sum[MyInt]`（`type MyInt int`）会生成几份代码？**
机器码只有一份 `main.Sum[go.shape.int]`（因为底层类型都是 `int`），但字典有两份 `..dict.Sum[int]` 和 `..dict.Sum[main.MyInt]`。而 `Sum[int64]` 底层类型不同，会另生成一份代码（见 2.2）。

**3. 字典（dictionary）里装了什么？运行时会变吗？**
四类内容：类型参数上方法的函数地址、内层泛型调用要用的子字典、派生类型的 `*runtime._type`、需要的 itab。它是 `SRODATA dupok` 只读数据，编译期就完全确定，运行时不修改也不需要 GC 扫描（见 2.3）。

**4. 泛型函数里 `reflect.TypeOf(v)` 拿到的是 shape 类型还是真实类型？**
真实类型。虽然 `TypeOf[int]` 和 `TypeOf[MyInt]` 共用一份代码，但 rtype 是从各自的字典里取的，所以分别返回 `int` 和 `main.MyInt`（见 2.3）。

**5. 泛型比接口快吗？**
看约束。约束只有方法时**不快**：类型参数上的方法调用编译成从字典取地址再 `CALL`，和接口的 itab 派发同量级，都无法内联，实测比具体类型慢 3 倍以上。约束含类型项、能直接做运算时**明显快**：省掉了装箱和类型断言，实测传 128 字节结构体比 `any` 快 8 倍且零分配（见 2.6）。

**6. 为什么方法不能有类型参数？**
带类型参数的方法等价于无穷多个签名，编译器无法为它建 itab、也无法判断一个类型是否实现了某接口。需要额外类型参数时提为顶层函数（见 1.5、3.3）。

**7. `~int` 和 `int` 作为约束有什么区别？`~` 后面能写什么？**
`int` 只接受 `int` 本身，`~int` 接受所有底层类型为 `int` 的类型（含 `type MyInt int`）。`~T` 中的 `T` 必须"自己就是自己的底层类型"，且不能是接口，否则报 `invalid use of ~`（见 1.3）。

**8. 约束接口和普通接口有什么区别？含类型项的接口能声明变量吗？**
Go 1.18 把接口推广为"类型集"。只有方法的叫基本接口，既能当约束也能当类型；一旦出现类型项（`~int`、union），就只能当约束，声明变量会报 `cannot use type X outside a type constraint`——因为这类接口没有统一的方法表，运行时无法表示（见 1.2、3.1）。

**9. `comparable` 能保证运行时比较不 panic 吗？**
不能。Go 1.20 起区分了"满足（satisfies）"和"实现（implements）"：`any`、`struct{ f any }` 这类"可比较但不严格可比较"的类型也能满足 `comparable`，把切片装进去再比较/做 map key 会 panic `hash of unhashable type`（见 1.4、3.6）。

**10. 泛型的类型推断能做到什么程度？不能做什么？**
能从函数实参推类型参数，能通过约束（如 `S ~[]E`）从已知类型参数推未知类型参数；**不能从返回值或赋值目标反推**，报 `cannot infer T`。无类型常量按默认类型参与推断，但和已定型的实参混用时会失败（见 1.6、3.8）。

**11. `func Filter[E any](s []E) []E` 和 `func Filter[S ~[]E, E any](s S) S` 有什么区别？**
前者返回值退化成 `[]E`，具名切片类型上挂的方法全部丢失，调用方拿到的值不再实现原接口；后者靠约束类型推断把 `S` 推成调用方传入的具名类型，原样返回。标准库 `slices` 全部采用后者（见 1.7、3.7）。

**12. 能对类型参数做 type switch 吗？**
不能直接做，报 `cannot use type switch on type parameter value`。要先 `any(v)` 转成接口。但如果你需要这么做，通常说明泛型不是这里正确的抽象（见 3.4）。

**13. `union` 里能放 `fmt.Stringer` 吗？方法和类型项怎么组合？**
不能，报 `cannot use fmt.Stringer in union (contains methods)`。方法和类型项要用**交集**表达：接口里分行写 `~int` 和 `fmt.Stringer`，含义是"底层类型是 int 且实现了 Stringer"（见 3.13）。

**14. 什么是"核心类型（core type）"？现在还有吗？**
Go 1.18–1.24 的规范用它来判断能不能对类型参数做 `range`/`make`/索引等操作。Go 1.25 起该概念被删除，改为逐操作定义规则，实际效果仍是"类型集里所有类型的底层类型必须相同"，通道的方向另有宽松规则（见 2.5）。

**15. 什么时候该用泛型，什么时候该用接口？**
编译期已知的多个具体类型上跑同一段算法（容器、算法、避免装箱的热路径）→ 泛型；要表达运行时可替换的行为契约、依赖倒置、可 mock → 接口。如果类型参数只用来调方法、函数体里没有任何和具体类型相关的操作，那用接口就够了，泛型只是多了一层字典（见 1.9、2.6）。

**16. 泛型对二进制体积和编译速度有什么影响？怎么控制？**
每种底层类型一份机器码 + 每个实例一份字典，约束越宽、实例化越多，膨胀越明显。控制手段：热点之外优先接口、类型参数控制在 1–2 个、不要提前放宽约束（见 3.12）。

**17. `g := Max` 为什么编译不过？**
泛型函数必须实例化后才是一个值。`Max` 只是模板，`Max[int]` 才有确定的签名和字典，才能赋给变量或作为参数传递（见 3.9）。

**18. 什么是实例化循环？**
递归泛型调用中类型参数不断变化（`T` → `[]T` → `[][]T`…），编译期无法收敛，编译器直接报 `instantiation cycle`。递归泛型函数的类型参数必须保持不变（见 3.10）。
