# 方法

> 定位：接收者、方法集与方法值。
> 前置知识：[函数](func.md)、[Struct](struct.md)。
> 配套示例：[method/main.go](method/main.go)（`go run ./method`）；命令均在 notes 根目录执行。

**阅读路线**：先读 [基础使用](#topic-1) → [常见陷阱](#topic-3)；深入实现或进阶用法时读 [底层原理](#topic-2)。

**篇内导航**

- [基础使用](#topic-1)
- [底层原理](#topic-2)
- [常见陷阱](#topic-3)
- [常见面试题](#topic-4)

> 环境：`go version go1.26.3`。方法集、接口 itab 等运行时结构以该版本源码（`runtime/iface.go`）为准，不同版本可能有细节差异。

<a id="topic-1"></a>

## 一、基础使用

<a id="section-1-1"></a>

### 1.1 方法声明

```go
type Point struct{ X, Y float64 }

func (p Point) Distance(q Point) float64 {
    return math.Hypot(q.X-p.X, q.Y-p.Y)
}

p := Point{1, 2}
q := Point{4, 6}
fmt.Println(p.Distance(q)) // 方法调用：p 作为接收者
```

- `func (p Point) Distance(...)` 中 `(p Point)` 是**接收者（receiver）**，把方法绑定到 `Point` 类型上；调用时用 `p.Distance(q)`，`p` 隐式传入。
- 接收者类型必须是**当前包内定义的具名类型**（不能是内置类型、接口类型，也不能是指针的指针），同一类型上方法名不能重复。

<a id="section-1-2"></a>

### 1.2 值接收者与指针接收者

```go
type Counter struct{ n int }

func (c Counter) IncByValue()    { c.n++ } // 操作副本，不影响原对象
func (c *Counter) IncByPointer() { c.n++ } // 操作原对象

var c Counter
c.IncByValue()
c.IncByPointer() // 等价于 (&c).IncByPointer()，编译器自动取址
fmt.Println(c.n) // 1
```

- 值接收者拿到调用者的拷贝，方法内修改不影响原对象；指针接收者共享同一实例（详见 [方法与接收者](struct.md#section-1-6)、[值接收者方法改不动原对象](struct.md#section-3-2)，规则相通）。
- **可寻址（addressable）的值**调用指针接收者方法时，编译器会自动插入取址操作（`c.IncByPointer()` → `(&c).IncByPointer()`），无需手写 `&`；但不可寻址的值（如函数返回的临时值、map 的 value）无法这样做（见 [不可寻址的值无法调用指针接收者方法](#section-3-4)）。

<a id="section-2-2"></a>

### 1.3 方法集（method set）规则

方法集决定了一个类型的变量能调用哪些方法，也决定了它能满足哪些接口：

| 类型 | 方法集包含                            |
| ---- | ------------------------------------- |
| `T`  | 所有以 `T` 为接收者声明的方法         |
| `*T` | 所有以 `T` 或 `*T` 为接收者声明的方法 |

- `*T` 的方法集更大，因为持有指针总能解引用得到一个可寻址的 `T`；反过来 `T` 的值不一定可寻址（如接口里存的值、map 的 value），无法安全地取址去调用指针方法。
- 这条规则直接决定接口满足关系：一个方法用指针接收者声明，那么只有 `*T` 满足对应接口，`T` 的值不满足（见 [值类型无法满足指针接收者方法的接口](#section-3-2)）。

<a id="section-1-3"></a>

### 1.4 方法值（method value）

```go
p := Point{1, 2}
distance := p.Distance   // 方法值：绑定了接收者 p
fmt.Println(distance(q)) // 等价于 p.Distance(q)，类型为 func(Point) float64
```

- 方法值在**取值的那一刻**就把接收者绑定进去了，之后即使 `p` 改变，`distance` 仍使用绑定时的那份接收者（值接收者时是拷贝，指针接收者时是共享的指针，见 [方法值绑定接收者的时机](#section-3-3)）。

<a id="section-1-4"></a>

### 1.5 方法表达式（method expression）

```go
distanceExpr := Point.Distance  // 方法表达式：不绑定接收者
fmt.Println(distanceExpr(p, q)) // 等价于 p.Distance(q)，类型为 func(Point, Point) float64
```

- 方法表达式把接收者变成了函数的**第一个显式参数**，调用时必须自己传入，常用于把某个类型的方法当作普通高阶函数传递。

<a id="section-1-5"></a>

### 1.6 通过嵌入获得方法

```go
type Point struct{ X, Y float64 }
func (p Point) Distance(q Point) float64 { /* ... */ }

type ColoredPoint struct {
    Point
    Color string
}

cp := ColoredPoint{Point{1, 2}, "red"}
cp.Distance(Point{4, 6}) // 直接调用被提升的 Distance 方法
```

- 嵌入字段的方法会被提升到外层类型的方法集中（与字段提升规则一致，见 [结构体嵌入（组合）](struct.md#section-1-5)、[嵌入的本质](struct.md#section-2-3)），这是 Go 用**组合**代替继承来复用方法的核心手段。

<a id="section-1-6"></a>

### 1.7 nil 接收者

```go
type IntList struct {
    Value int
    Tail  *IntList
}

func (list *IntList) Sum() int {
    if list == nil {
        return 0
    }
    return list.Value + list.Tail.Sum()
}

var list *IntList       // nil
fmt.Println(list.Sum()) // 0，合法调用
```

- 指针接收者的方法可以在 `nil` 指针上调用，只要方法体在访问字段前显式处理了 `nil` 情况（这是 Go 里链表、树等递归结构常见的写法）。

<a id="topic-2"></a>

## 二、底层原理

<a id="section-2-1"></a>

### 2.1 方法只是语法糖：接收者是隐式的第一个参数

编译器把方法**降级（desugar）**为一个普通函数，接收者作为显式的第一个参数：

```go
func (p Point) Distance(q Point) float64 { /* ... */ }
// 编译器内部大致生成：
func Point_Distance(p Point, q Point) float64 { /* ... */ }

p.Distance(q)        // 被编译为 Point_Distance(p, q)
Point.Distance(p, q) // 方法表达式，本质就是拿到这个降级后的函数
```

- 这解释了 [方法表达式（method expression）](#section-1-4) 的方法表达式为什么类型是 `func(Point, Point) float64`——它本来就是普通函数，只是多了一层 `p.Method(...)` 的调用语法糖。
- 方法值（[方法值（method value）](#section-1-3)）则是对这个函数做了一次**局部应用（partial application）**：编译器生成一个闭包，把接收者固化，返回 `func(Point) float64`。

<a id="section-2-3"></a>

### 2.2 接口值与动态方法派发（itab）

接口方法调用通常通过方法表找到具体实现；编译器也可能在能确定具体类型时消除间接调用。这里重点区分两件事：**方法集决定接口是否满足，接口的动态类型决定实际调用哪个实现**。

`eface`、`iface`、`ITab` 的结构和构建路径统一见 [接口值的两种运行时表示：`eface` 与 `iface`](interface.md#section-2-1)、[itab：接口类型 + 动态类型 → 方法表](interface.md#section-2-2)，不在方法篇重复维护源码结构。

<a id="section-2-4"></a>

### 2.3 嵌入方法提升的编译期实现

[通过嵌入获得方法](#section-1-5) 的方法提升不是运行时查找，而是编译器在外层类型上**自动生成一个转发方法（forwarding method）**：

```go
// cp.Distance(q) 能编译通过，是因为编译器自动生成了：
func (c ColoredPoint) Distance(q Point) float64 {
    return c.Point.Distance(q) // 转发给内嵌字段的同名方法
}
```

- 这也是为什么“提升”只在**编译期**生效：`ColoredPoint` 实实在在多了一个 `Distance` 方法（进入它的方法集），可以满足要求 `Distance` 方法的接口；不存在任何运行时的间接层。
- 若内嵌字段是指针类型（如嵌入 `*Point`），转发调用前内嵌指针若为 `nil`，在内层解引用同样会崩溃，需和 [nil 接收者](#section-1-6)、[nil 指针调用方法未做判空](#section-3-5) 一样小心处理。

<a id="topic-3"></a>

## 三、常见陷阱

<a id="section-3-1"></a>

### 3.1 值接收者改不动原对象

同 [值接收者方法改不动原对象](struct.md#section-3-2)：值接收者操作的是拷贝，需要修改原对象或避免大结构体拷贝时应统一使用指针接收者。

<a id="section-3-2"></a>

### 3.2 值类型无法满足指针接收者方法的接口

```go
type Setter interface{ Set(string) }

type T struct{ v string }
func (t *T) Set(s string) { t.v = s } // 指针接收者

var s Setter = T{}  // 编译错误：T does not implement Setter (Set method has pointer receiver)
var s Setter = &T{} // 正确
```

- 根源是 [方法集（method set）规则](#section-2-2) 的方法集规则：`Set` 只在 `*T` 的方法集里，`T` 的值不满足 `Setter`。
- 注意这与 [值接收者与指针接收者](#section-1-2) 的“可寻址值自动取址调用”不是一回事：`t.Set("x")`（`t` 是局部变量，可寻址）能编译通过，是直接方法调用的语法糖；但把 `T{}` **赋值给接口变量**时，编译器只做静态方法集检查，不会因为“理论上可以取址”就放行。

<a id="section-3-3"></a>

### 3.3 方法值绑定接收者的时机

```go
type Counter struct{ n int }
func (c Counter) Get() int { return c.n } // 值接收者

c := Counter{n: 1}
get := c.Get // 方法值：此刻拷贝了 c 的当前值（n=1）
c.n = 100
fmt.Println(get()) // 仍输出 1，不是 100

// 换成指针接收者：
func (c *Counter) GetP() int { return c.n }
getP := c.GetP // 绑定的是指针
c.n = 200
fmt.Println(getP()) // 输出 200，因为共享同一实例
```

- 值接收者的方法值在**取值那一刻**就固化了接收者的拷贝；指针接收者的方法值绑定的是指针，后续通过指针看到的修改都可见。混用时很容易对“方法值到底绑定了什么时刻的状态”产生误判。

<a id="section-3-4"></a>

### 3.4 不可寻址的值无法调用指针接收者方法

```go
type T struct{ v int }
func (t *T) Inc() { t.v++ }

m := map[string]T{"a": {v: 1}}
m["a"].Inc() // 编译错误：cannot call pointer method on m["a"]，m["a"] 不可寻址

func newT() T { return T{} }
newT().Inc() // 编译错误：函数返回值是临时值，不可寻址
```

- map 的 value（同 [map 中的结构体不可寻址，不能直接改字段](struct.md#section-3-4)）、函数调用的返回值等都不可寻址，[值接收者与指针接收者](#section-1-2) 提到的“自动取址”语法糖在这些场景失效，必须先取出赋给局部变量、或把 value 类型改成指针。

<a id="section-3-5"></a>

### 3.5 nil 指针调用方法未做判空

```go
type T struct{ Name string }
func (t *T) String() string { return t.Name } // 没有处理 nil

var p *T
_ = p.String() // panic: runtime error: invalid memory address or nil pointer dereference
```

- [nil 接收者](#section-1-6) 的 `IntList.Sum` 能安全处理 `nil` 是因为方法体**显式判断了 `list == nil`**；调用本身在 `nil` 指针上总是合法的（方法降级为函数后，`nil` 只是一个普通的参数值），真正 panic 的是方法体内对 `nil` 指针字段的解引用，这两者不要混为一谈。

<a id="topic-4"></a>

## 四、常见面试题

**1. Go 里方法和函数的本质区别是什么？**
没有本质区别，方法是编译器把接收者当作隐式第一个参数的普通函数（见 [方法只是语法糖：接收者是隐式的第一个参数](#section-2-1)）。方法表达式 `T.Method` 直接暴露了这个降级后的函数形式，接收者变成显式的第一个参数。

**2. 方法集（method set）的规则是什么？为什么 `*T` 的方法集比 `T` 大？**
`T` 的方法集只包含以 `T` 为接收者声明的方法；`*T` 的方法集包含以 `T` 或 `*T` 为接收者声明的所有方法。因为持有指针总能解引用得到可寻址的 `T`，反之 `T` 的值不一定可寻址，无法安全地取址调用指针方法（见 [方法集（method set）规则](#section-2-2)）。

**3. 为什么 `var i Interface = T{}` 在 `Set` 是指针接收者时会编译错误？**
接口赋值只检查静态方法集：指针接收者方法只在 `*T` 的方法集里，`T` 的值不满足接口。这与“可寻址值调用指针方法时编译器自动取址”是两条不同的规则，后者只在直接方法调用语法 `t.Method()` 时生效，接口赋值不会隐式取址（见 [值类型无法满足指针接收者方法的接口](#section-3-2)）。

**4. 接口方法调用在运行时是怎么执行的？**
接口值是 `(动态类型, 数据)` 的二元组，方法调用通过 `itab`（interface table）查表：`itab.fun[k]` 存着该接口第 k 个方法在具体类型上的函数地址，调用即“查表取地址→调用”，是运行时的动态派发（见 [接口值与动态方法派发（itab）](#section-2-3)）。

**5. 嵌入类型的方法提升是运行时虚表机制吗？**
不是。编译器在外层类型上**静态生成转发方法**，把调用委托给内嵌字段的同名方法，编译期就完成，没有任何运行时间接层，这和接口的 itab 动态派发是两种完全不同的机制（见 [嵌入方法提升的编译期实现](#section-2-4)）。

**6. 方法值和方法表达式有什么区别？**
方法值（`p.Method`）在取值时就绑定了接收者，得到的函数不再需要传接收者；方法表达式（`T.Method`）不绑定接收者，接收者变成返回函数的第一个显式参数，调用时必须自己传入（见 [方法值（method value）](#section-1-3)、[方法表达式（method expression）](#section-1-4)）。

**7. `nil` 指针可以调用方法吗？什么时候会 panic？**
可以。方法调用本质是把 `nil` 当作一个普通参数传给降级后的函数，只要方法体在访问字段/解引用之前显式判断了 `nil`（如链表 `Sum()`），调用是安全的；一旦方法体在未判空的情况下解引用 `nil` 接收者的字段，才会 panic（见 [nil 接收者](#section-1-6)、[nil 指针调用方法未做判空](#section-3-5)）。

**8. 什么情况下不能对指针接收者方法用“隐式取址”语法糖？**
调用者必须是**可寻址的值**。map 的 value、函数调用的返回值、接口里存储的值等都不可寻址，此时编译器不会自动取址，必须先赋值给局部变量拿到地址，或者把类型改成指针存储（见 [不可寻址的值无法调用指针接收者方法](#section-3-4)）。

**9. 同一个类型的方法应该统一用值接收者还是指针接收者？**
一般建议统一：只要类型里有任何一个方法需要指针接收者（修改状态、避免大结构体拷贝），其余方法也用指针接收者，避免方法集不一致导致部分方法不满足接口、或值/指针混用带来的心智负担（同 [值接收者方法改不动原对象](struct.md#section-3-2) 的建议）。

**10. 方法能定义在哪些类型上？有什么限制？**
接收者类型必须是在**当前包内定义**的具名类型（不能是内置类型如 `int`，也不能是其他包定义的类型、接口类型、或指针类型本身，接收者不能是指针的指针）。同一类型上方法名不能重复，但不同类型可以有同名方法，因为方法本质上是和类型绑定的（见 [方法只是语法糖：接收者是隐式的第一个参数](#section-2-1)）。
