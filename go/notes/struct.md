# Struct

> 环境：`go version go1.26.3`。内存对齐、`unsafe.Sizeof`/`Alignof`/`Offsetof` 的结果与平台（这里是 `darwin/arm64`，字长 8 字节）相关，不同架构可能有细节差异。

## 一、基础使用

### 1.1 定义与初始化

```go
type Employee struct {
    ID        int
    Name      string
    Address   string
    DoB       time.Time
    Position  string
    Salary    int
    ManagerID int
}

var e1 Employee                                  // 零值：所有字段被置为各自类型的零值
e2 := Employee{}                                 // 同上，零值结构体
e3 := Employee{1, "Bob", "", t, "Dev", 0, 0}     // 顺序字面量：必须写全所有字段，且顺序一致
e4 := Employee{ID: 1, Name: "Bob", Position: "Dev"} // 具名字段：可只写部分，其余取零值（推荐）
```

- **顺序字面量**要求写出全部字段、顺序严格对应，字段增删会静默错位，脆弱，一般只用于 `Point{1, 2}` 这种字段极少且稳定的类型。
- **具名字段字面量**只需列出关心的字段，其余自动零值，字段顺序无关，是工程中的默认写法。

### 1.2 字段访问与指针

```go
e.Salary = 1000            // 通过点号访问/赋值字段
position := &e.Position    // 可以取某个字段的地址
*position = "CEO"          // 通过字段指针间接修改

var pe *Employee = &e
pe.Salary += 2000          // 结构体指针访问字段无需显式解引用，pe.Salary 等价于 (*pe).Salary
(*pe).Salary += 3000       // 显式写法，与上一行等价
```

- Go 对结构体指针访问字段做了语法糖：`pe.Field` 自动等价于 `(*pe).Field`，无需像 C 那样区分 `.` 和 `->`。

### 1.3 结构体作为函数返回值 / 可寻址性

```go
var employees []Employee

func GetEmployeeByID(id int) *Employee {
    for i := range employees {
        if employees[i].ID == id {
            return &employees[i] // 返回 slice 元素的地址，调用方可原地修改
        }
    }
    return nil
}

GetEmployeeByID(1).Position = "CTO" // 因为返回的是 *Employee，可以直接对字段赋值
```

- 只有**可寻址（addressable）**的结构体才能对其字段赋值。函数返回值若是**值**（`Employee` 而非 `*Employee`），则 `GetEmployeeByID(1).Position = "CTO"` 无法编译——返回的临时值不可寻址。
- `range` 时用 `employees[i]` 而不是循环变量 `v`，因为 `v` 是元素的拷贝，`&v` 拿到的是拷贝的地址（详见 3.1）。

### 1.4 匿名结构体

```go
config := struct {
    Host string
    Port int
}{
    Host: "localhost",
    Port: 8080,
}
```

- 无需 `type` 声明，直接定义并实例化，常用于临时数据（如一次性的 JSON 载荷、表驱动测试的用例结构、`map[string]struct{}` 里的集合值）。

### 1.5 结构体嵌入（组合）

```go
type Point struct{ X, Y int }

type Circle struct {
    Point      // 嵌入字段（匿名字段），只写类型名
    Radius int
}

type Wheel struct {
    Circle
    Spokes int
}

w := Wheel{Circle{Point{1, 2}, 3}, 4}
w.X = 10        // 字段提升（promotion）：可以直接访问 w.Circle.Point.X，无需写全路径
w.Point.X = 10  // 也可以写出中间层级
```

- 嵌入字段的**字段和方法都会被“提升”**到外层，可以像外层自己的成员一样直接访问 `w.X`、`w.Spokes`。
- 命名 vs 嵌入的区别：`Circle Circle`（有字段名）是普通具名字段，只能 `w.Circle.Radius`；`Circle`（无字段名，仅类型）才是嵌入，能享受提升。

### 1.6 方法与接收者

```go
func (p Point) Distance() int { ... }  // 值接收者：操作的是拷贝，不改变原对象
func (p *Point) Scale(f int)  { ... }  // 指针接收者：可修改原对象，且避免大结构体拷贝
```

- 值接收者拿到的是调用者的**副本**，方法内改字段不影响原对象；指针接收者共享同一实例，能改原对象。
- 嵌入类型的方法同样会被提升：若 `Point` 有 `Distance()`，则 `Circle`、`Wheel` 实例也能直接调用 `w.Distance()`，这是 Go 用组合替代继承的核心机制。

## 二、底层原理

### 2.1 内存布局与字段对齐

结构体在内存中是**各字段按声明顺序、连续排布**的一块内存，但字段之间/之后可能有编译器插入的**填充（padding）**，以满足每个字段的**对齐要求**：

- 每个类型有自己的对齐值（`unsafe.Alignof`），通常等于其大小，最大不超过字长（本机 8 字节）。字段的起始偏移必须是其对齐值的整数倍。
- 结构体整体的大小（`unsafe.Sizeof`）会被向上取整到结构体对齐值的整数倍（结构体对齐值 = 其最大字段对齐值），以保证在数组里连续排布时每个元素仍然对齐。

```go
type Bad struct {
    a bool  // 1 字节 + 7 填充
    b int64 // 8 字节
    c bool  // 1 字节 + 7 填充
}           // Sizeof = 24

type Good struct {
    b int64 // 8
    a bool  // 1
    c bool  // 1 + 6 填充
}           // Sizeof = 16
```

**要点**：把大字段排前面、相同/相近大小的字段聚在一起，能减少 padding。可用 `unsafe.Sizeof / Alignof / Offsetof` 观察实际布局。

### 2.2 结构体是值类型

- 结构体变量的**赋值、函数传参、作为返回值、作为 map/slice 元素**都是**整体按值拷贝**——所有字段（包括嵌入字段）逐字节复制一份。
- 因此传递大结构体会有拷贝开销；需要共享或修改原对象时，传 `*Struct` 指针。
- 结构体里若含指针/slice/map 字段，拷贝的是这些 header（浅拷贝），底层数据仍共享——这是很多“改了拷贝却影响了原对象”问题的根源。

### 2.3 嵌入的本质

嵌入不是继承，而是**编译期的字段/方法查找规则**：

- `w.X` 的解析过程：先在 `Wheel` 自身找 `X`，没有则到嵌入的 `Circle` 里找，再到 `Circle` 嵌入的 `Point` 里找，按“最浅深度优先”命中。
- 内存上，嵌入字段就是一个以类型名为字段名的普通字段，`Wheel` 里实实在在地包含一个完整的 `Circle`（再包含完整的 `Point`），没有任何虚表/间接层。
- 若外层和嵌入层有**同名字段/方法**，外层的会“遮蔽”内层的（shadowing），此时访问内层必须写全路径 `w.Circle.X`；若**同一深度**有两个嵌入字段同名，则该名字变为“有歧义”，直接访问会编译报错，必须显式限定。

### 2.4 结构体的可比较性

```go
w := Wheel{Circle{Point{1, 2}, 3}, 4}
w1 := Wheel{Circle{Point{1, 2}, 3}, 4}
fmt.Println(w == w1) // true：逐字段递归比较
```

- 结构体是否可用 `==` 比较，取决于**所有字段是否都可比较**。全部字段可比较时，`==` 逐字段递归比较，全等才返回 `true`。
- 若结构体**含有不可比较字段**（slice、map、function），则整个结构体不可比较，`==` 无法编译，也不能作为 map 的 key。
- 可比较的结构体可以直接作为 map 的 key（如 `map[Point]int`），这是 struct 相比 slice 的一个重要区别。

### 2.5 空结构体 `struct{}`

- `struct{}` 不占内存，`unsafe.Sizeof(struct{}{}) == 0`。运行时所有零大小值都指向同一个全局地址 `runtime.zerobase`。
- 常见用途：`map[string]struct{}` 实现集合（value 不占空间）、`chan struct{}` 做纯信号量（只关心“发生了”而不关心值）。

## 三、常见陷阱

### 3.1 range 循环变量是拷贝，取地址会踩坑

```go
type T struct{ V int }
s := []T{{1}, {2}, {3}}

var ptrs []*T
for _, v := range s {
    ptrs = append(ptrs, &v) // 错误：&v 是循环变量的地址
}
```

- 在 Go 1.22 之前，`v` 是被**复用**的同一个变量，循环结束后 `ptrs` 里三个指针全指向它、且值都是最后一个元素——经典 bug。
- Go 1.22 起循环变量**每次迭代都是新变量**，`&v` 各不相同，但它仍是**元素的拷贝的地址**，改 `*ptrs[i]` 不会影响 `s[i]`。要拿到原元素地址应写 `&s[i]`。

### 3.2 值接收者方法改不动原对象

```go
func (e Employee) Raise() { e.Salary += 1000 } // 改的是拷贝，调用后原对象 Salary 不变
func (e *Employee) Raise() { e.Salary += 1000 } // 指针接收者才能真正修改
```

- 需要修改接收者、或结构体较大不想拷贝时，用指针接收者。**同一类型的方法集最好统一用指针接收者**，避免值/指针混用带来的方法集不一致。

### 3.3 含指针/引用字段的浅拷贝

```go
type Box struct {
    Items []int
}
a := Box{Items: []int{1, 2, 3}}
b := a          // 拷贝了 Box，但 b.Items 与 a.Items 共享同一底层数组
b.Items[0] = 99 // a.Items[0] 也变成 99
```

- 结构体值拷贝是**浅拷贝**：slice/map/指针字段拷贝的是 header/地址，底层数据仍共享。需要独立副本时要对这些字段单独深拷贝。

### 3.4 map 中的结构体不可寻址，不能直接改字段

```go
m := map[string]Employee{"a": {Salary: 100}}
m["a"].Salary = 200 // 编译错误：m["a"] 不可寻址
```

- map 的 value 不可寻址（map 扩容会搬迁元素，地址不稳定），因此不能对 `m["a"]` 的字段直接赋值。解决办法：
  - 取出、改、放回：`e := m["a"]; e.Salary = 200; m["a"] = e`；
  - 或把 value 存成指针：`map[string]*Employee`，然后 `m["a"].Salary = 200`。

### 3.5 嵌入字段同名遮蔽 / 歧义

```go
type A struct{ Name string }
type B struct{ Name string }
type C struct {
    A
    B
}
var c C
// c.Name        // 编译错误：ambiguous selector，A.Name 和 B.Name 同深度冲突
c.A.Name = "x"   // 必须写全路径
```

- 同一深度多个嵌入字段同名时，直接访问会因歧义报错，必须显式限定；不同深度时浅层遮蔽深层。

## 四、常见面试题

**1. Go 里 struct 是值类型还是引用类型？**
值类型。赋值、传参、返回都整体按值拷贝所有字段。要共享或修改原对象、或避免大结构体拷贝，就传 `*Struct`。

**2. struct 的内存布局是怎样的？为什么字段顺序会影响 struct 大小？**
字段按声明顺序连续排布，但为满足每个字段的对齐要求会插入 padding，结构体总大小还要按最大字段对齐值向上取整。字段顺序不同，padding 数量不同，因此大小可能不同——把大字段排前、相近大小聚拢可减少填充（见 2.1）。

**3. 两个 struct 什么时候能用 `==` 比较？能作为 map key 吗？**
当且仅当所有字段都是可比较类型时，struct 可用 `==` 逐字段递归比较，也能作为 map key。只要含有 slice、map、function 这类不可比较字段，整个 struct 就不可比较、不能作 key（见 2.4）。

**4. 嵌入（embedding）和继承有什么区别？**
嵌入是组合，本质是编译期的字段/方法查找规则（提升 promotion），内存上外层真实包含内层的完整副本，没有虚表/多态。它不建立“is-a”关系，也没有运行时动态派发；Go 用“嵌入 + 接口”替代传统继承。

**5. 字段提升的查找规则？同名字段怎么处理？**
按“最浅深度优先”查找：先外层，再逐层进入嵌入字段。浅层同名会遮蔽深层；同一深度多个嵌入字段同名则产生歧义，直接访问编译报错，必须写全路径限定（见 2.3、3.5）。

**6. 方法用值接收者还是指针接收者？**
需要修改接收者、或结构体较大想避免拷贝、或类型的其它方法已用指针接收者（保持方法集一致）——用指针接收者；否则小的、不可变语义的类型可用值接收者。同一类型建议统一，不要混用（见 3.2）。

**7. 为什么 `m["key"].Field = x`（m 是 map）编译不过？**
map 的元素不可寻址（扩容搬迁导致地址不稳定），不能对其字段直接赋值。要么“取出-修改-放回”，要么把 value 类型改成指针 `map[K]*V`（见 3.4）。

**8. 空结构体 `struct{}` 有什么用？占多少内存？**
大小为 0，不占内存（所有零大小值共享 `runtime.zerobase`）。常用于 `map[K]struct{}` 实现集合、`chan struct{}` 做纯信号通知（见 2.5）。

**9. struct 拷贝是深拷贝还是浅拷贝？**
浅拷贝。值字段逐字节复制，但 slice/map/指针字段复制的是 header/地址，底层数据仍与原对象共享，改一个会影响另一个。要独立副本需对这些字段单独深拷贝（见 3.3）。

**10. 顺序字面量和具名字段字面量的区别，工程上用哪个？**
顺序字面量要求写全所有字段且顺序对应，字段增删易静默错位，仅适合 `Point{1,2}` 这类极稳定的小类型；具名字段字面量只写关心的字段、其余零值、与顺序无关，是工程默认写法（见 1.1）。

**11. `for _, v := range structSlice` 中对 `&v` 取地址有什么问题？**
`v` 是元素的拷贝。Go 1.22 前循环变量还会被复用，导致收集到的指针全指向同一变量、值为最后一个元素；1.22 后每轮是新变量但仍是拷贝的地址，改它不影响原 slice。要原元素地址应用 `&s[i]`（见 3.1）。

**12. 匿名结构体在什么场景用？**
临时、一次性的数据结构，不值得单独 `type` 声明的场合：一次性 JSON 请求/响应体、表驱动测试的用例集合、`map[string]struct{}` 的集合值等（见 1.4）。

**13. 如何观察一个 struct 的实际内存占用和字段偏移？**
用 `unsafe.Sizeof(x)`（总大小）、`unsafe.Alignof(x)`（对齐值）、`unsafe.Offsetof(x.Field)`（字段偏移）。这些都是编译期常量，能直观看到 padding 分布，指导字段重排优化（见 2.1）。
