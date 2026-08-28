# reflect

> 环境：`go version go1.26.3 darwin/amd64`。源码：`reflect/{type,value,deepequal}.go`。配套代码：`notes/refl/`。所有性能数字都是那份 benchmark 的真实输出。
>
> 版本演进（用 `$GOROOT/api/go1.*.txt` 可以精确核对）：
> - **1.13**：`Value.IsZero`。
> - **1.17**：`Value.UnsafePointer`、`reflect.VisibleFields`。
> - **1.18**：`Value.SetIterKey/SetIterValue`、`Value.Grow`。
> - **1.20**：`Value.Comparable`、`Value.Equal`、`Value.SetZero`。
> - **1.22**：**`reflect.TypeFor[T]()`**——不用造零值就能拿 Type。
> - **1.23**：`Value.Seq`/`Seq2`——让 `reflect.Value` 能直接 `range`。
> - **1.25**：`reflect.TypeAssert[T](v) (T, bool)`。
> - **1.26**：迭代器版反射 API——`Type.Fields()`/`Methods()`/`Ins()`/`Outs()`、`Value.Fields()`/`Methods()`。

## 一、基础

### 1.1 反射三定律

Rob Pike 那篇 *The Laws of Reflection* 的三条：

```go
// ① 从接口值可以反射出 reflect.Value
var x float64 = 3.4
v := reflect.ValueOf(x)          // type=float64 kind=float64

// ② 从 reflect.Value 可以还原成接口值
back := v.Interface().(float64)  // 3.4

// ③ 要修改一个 reflect.Value，它必须是"可设置的"（settable）
v.CanSet()                                  // false ← 传的是副本
p := reflect.ValueOf(&x).Elem()
p.CanSet()                                  // true
p.SetFloat(7.1)                             // x 变成 7.1
```

**第三条是所有反射代码的第一个坎**：`reflect.ValueOf(x)` 拿到的是 `x` 的**副本**（因为参数是 `any`，装箱时就拷贝了），所以永远改不了原值。要改必须传指针再 `.Elem()`。

### 1.2 Type 与 Value

```go
u := User{Name: "bob", Age: 30, Addr: Address{City: "SH"}}
t := reflect.TypeOf(u)
v := reflect.ValueOf(u)

t.Name()      // "User"
t.PkgPath()   // "main"
t.Kind()      // struct
t.Size()      // 72
t.NumField()  // 5
t.NumMethod() // 2 —— 只有值方法（Greet/String）

reflect.TypeOf(&u).NumMethod()   // 3 —— 值方法 + 指针方法（SetName）
```

字段遍历（注意 `Offset` 和 `IsExported`）：

```text
[0] Name     string       offset=0  exported=true  tag="json:\"name\" validate:\"required,min=2\""
[1] Age      int          offset=16 exported=true  tag="json:\"age\" ..."
[2] Email    string       offset=24 exported=true  tag="json:\"email,omitempty\""
[3] Addr     main.Address offset=40 exported=true  tag=""
[4] private  string       offset=56 exported=false tag=""
```

三种取字段的方式：

```go
v.Field(0)                       // 按索引，最快
v.FieldByName("Name")            // 按名字，慢 17 倍（见 3.4）
v.FieldByIndex([]int{3, 0})      // 按索引路径，可以穿透嵌套（Addr.City）
```

调用方法：

```go
out := v.MethodByName("Greet").Call(nil)               // out[0].String() == "hi bob"
reflect.ValueOf(&u).MethodByName("SetName").
    Call([]reflect.Value{reflect.ValueOf("alice")})     // 指针方法要用指针的 Value
```

`reflect.TypeFor[T]()`（1.22+）替代了 `reflect.TypeOf(T{})` 这种"为了拿类型而造一个零值"的写法，对不方便构造零值的类型（比如接口）尤其有用，而且**快 4 倍**（编译期就确定）。

### 1.3 Kind 与 Type 的区别

```text
int             Type=int             Kind=int
main.MyInt      Type=main.MyInt      Kind=int      ← 不同 Type，相同 Kind
[]int           Type=[]int           Kind=slice
main.MySlice    Type=main.MySlice    Kind=slice
*main.User      Type=*main.User      Kind=ptr
```

- **Kind** 是底层种类，一共 26 种（`Invalid`/`Bool`/.../`UnsafePointer`）；
- **Type** 是具体类型，无穷多。

**写通用代码时 switch 的是 Kind，判断具体类型才比 Type**。这也是所有序列化库的骨架：

```go
switch v.Kind() {
case reflect.Struct: ...
case reflect.Slice, reflect.Array: ...
case reflect.Map: ...
case reflect.Ptr, reflect.Interface: ...
default: // 基础类型
}
```

`Elem()` 的含义**随 Kind 变化**：`Ptr` 是指向的类型、`Slice`/`Array` 是元素类型、`Map` 是 value 类型、`Chan` 是元素类型，其他 Kind 调用会 **panic**。

### 1.4 可设置性（settability）

```text
                                       CanAddr  CanSet  CanInterface
ValueOf(u)                             false    false   true
ValueOf(u).Field(0)                    false    false   true
ValueOf(&u)                            false    false   true      ← 指针本身不可寻址
ValueOf(&u).Elem()                     true     true    true
ValueOf(&u).Elem().Field(0)            true     true    true
ValueOf(&u).Elem().Field(4)  未导出      true     false   false     ← 可寻址但不可设置
```

规则：**`CanSet = CanAddr && 字段是导出的`**。

容器的差异很值得记：

```go
// map 的元素不可寻址（因为 map 会重哈希搬移），只能整体替换
mv.MapIndex(k).CanSet()          // false
mv.SetMapIndex(k, newVal)         // ✓

// slice 的元素可寻址（底层数组固定在堆上）
sv.Index(0).CanSet()              // true
sv.Index(0).SetInt(99)            // ✓ 真的改了原 slice
```

### 1.5 struct tag

```go
f := t.Field(0)
f.Tag.Get("json")                  // "name"，找不到返回 ""
jsonTag, ok := f.Tag.Lookup("json") // 能区分"没有这个 key"和"值是空串"
```

- tag 就是一个字符串，约定格式 `key:"value" key2:"v2"`（**空格分隔，值用双引号**）；
- tag 里逗号后面的选项（`omitempty` 之类）是**各个库自己解析的**，`reflect` 完全不管；
- **写错了编译器不报错，只会静默失效**：

```go
type bad struct {
    A string `json: "a"`   // 冒号后有空格 -> Get("json") 返回 ""
}
```

`go vet` 的 `structtag` 检查能抓到常见错误（我在示例里想演示这个坑，结果被 vet 直接拦下，只能用 `reflect.StructOf` 动态构造）。

### 1.6 1.26 的迭代器 API

```go
for f := range reflect.TypeFor[User]().Fields() { ... }        // StructField
for f, v := range reflect.ValueOf(u).Fields() { ... }          // StructField + Value
for m := range reflect.TypeFor[*User]().Methods() { ... }      // Method
for t := range reflect.TypeFor[func(int, string) bool]().Ins() { ... }
for t := range ft.Outs() { ... }
```

比 `for i := range t.NumField() { f := t.Field(i) }` 短，语义也更清楚。1.23 的 `Value.Seq`/`Seq2` 则是**完全对齐 `range` 那个类型时的行为**——容易被坑的一点：

```go
s := reflect.ValueOf([]int{10, 20, 30})
for v := range s.Seq()  { ... }   // 给的是**下标** 0 1 2（对齐 for i := range s）
for i, v := range s.Seq2() { ... } // 0=10 1=20 2=30
```

## 二、原理

### 2.1 `reflect.Value` 的内部结构

```go
type Value struct {
    typ_ *abi.Type      // 类型描述符（和 eface 里的 _type 是同一个东西）
    ptr  unsafe.Pointer // 数据指针
    flag                // kind + 是否可寻址 + 是否只读 + 是否间接寻址...
}
// Sizeof(reflect.Value) = 24
```

`reflect.ValueOf(x any)` 做的三件事：

1. **调用方先把具体值装箱进 `eface`**——这一步就可能有一次逃逸和堆分配（mem.md 2.2）；
2. 从 eface 里拆出 `(type, data)` 填进 `typ_`/`ptr`；
3. 按类型算出 `flag`（kind、`flagIndir`、`flagAddr`、`flagRO`）。

两个推论：

- **反射慢的根源**是"每次进出反射边界都要装箱/拆箱 + 查表 + 位运算"，而不是某个操作本身多复杂；
- `Value` 只有 24 字节，所以**按值传递没问题**（标准库到处这么用）。

`flagRO` 就是"未导出字段"的标记——`Interface()` 检查这个位，为真就 panic（见 3.3）。

### 2.2 `MakeFunc`：动态造函数

```go
func wrapWithLog(fn any) any {
    fv := reflect.ValueOf(fn)
    return reflect.MakeFunc(fv.Type(), func(in []reflect.Value) []reflect.Value {
        out := fv.Call(in)
        log.Printf("call(%v) -> %v", in, out)
        return out
    }).Interface()
}

logged := wrapWithLog(add).(func(int, int) int)   // 类型完全保持
logged(3, 4)                                       // 7，并打日志
```

用途：RPC 客户端桩、ORM 的动态查询、测试 mock、依赖注入容器。

代价：**每次调用都要构造 `[]Value`（分配）+ 反射调用**，实测 470ns vs 直接调用 0.64ns——**慢 700 倍**。所以它只适合"每次调用本身就很贵"的场景（网络 RPC），不适合热路径。

### 2.3 `DeepEqual` 的规则与坑

```text
reflect.DeepEqual([]int(nil), []int{})              false ← nil slice ≠ 空 slice
reflect.DeepEqual(map[string]int(nil), map[...]{})  false
reflect.DeepEqual(NaN, NaN)                         false ← 浮点规则
reflect.DeepEqual(f, f)  （同一个函数）                false ← 函数只有都是 nil 才相等
reflect.DeepEqual(int(1), int64(1))                 false ← 类型必须一致
循环引用的两个等价结构                                  true  ← 有 visited 记录，不会栈溢出
```

三条实践建议：

1. **`DeepEqual` 会比较未导出字段**，所以对 `time.Time`（含单调时钟和 `*Location`）、`sync.Mutex`、带缓存字段的结构体容易误判；
2. **测试里优先用 `google/go-cmp`**（`cmp.Diff`）：能配置忽略字段、能自定义比较器、失败时打印可读的 diff；
3. 简单场景用 `slices.Equal`/`maps.Equal`（1.21+），**不走反射，快一个数量级**。

## 三、常见陷阱

### 3.1 反射把编译期错误变成了运行时 panic

```text
改一个不可设置的值           -> reflect.Value.SetString using unaddressable value
Elem() 一个非指针            -> call of reflect.Value.Elem on int Value
Int() 一个 string           -> call of reflect.Value.Int on string Value
Field() 越界                -> Field index out of range
Call 参数个数不对             -> Call with too few input arguments
读未导出字段的 Interface()    -> cannot return value obtained from unexported field or method
用零值 Value                -> call of reflect.Value.Int on zero Value
```

这是使用反射**最大的成本**：类型系统的保护全部失效。三条纪律：

1. **反射代码要有充分的单元测试**（覆盖每个 Kind 分支）；
2. **在边界处校验 Kind**，别指望上游传对；
3. **把反射收敛在一个包/一层里**，对外暴露类型安全的 API。

### 3.2 nil 与零值

```go
reflect.ValueOf(nil).IsValid()      // false ← 零值 Value，调任何方法都 panic
reflect.ValueOf(nilPtr).IsValid()   // true，IsNil()=true，Kind()=ptr
reflect.ValueOf(nilSlice).IsNil()   // true，Len()=0
```

- **`IsNil` 只对 chan/func/interface/map/ptr/slice 合法**，其他 Kind 会 panic；
- **`IsZero`（1.13+）对所有类型合法**，判断是否等于类型零值；
- 处理 `any` 参数的反射代码，**第一件事就是 `if !v.IsValid()`**。

### 3.3 未导出字段

```go
f := reflect.ValueOf(u).Field(4)   // private 字段
f.Type()          // string ✓ 能读到类型
f.CanInterface()  // false
f.Interface()     // panic!
f.String()        // "secret" ← 居然能拿到值！
```

`String()` 是个"后门"——它不走 `Interface()`，所以只对 string kind 有效。别依赖这个。

真要突破（**测试和序列化库里偶尔用，生产代码里出现基本等于设计有问题**）：

```go
pv := reflect.ValueOf(pu).Elem().Field(4)
real := reflect.NewAt(pv.Type(), unsafe.Pointer(pv.UnsafeAddr())).Elem()
real.CanSet()          // true
real.SetString("changed")
```

### 3.4 性能：实测数据

```text
u.Name 直接取字段                0.83 ns/op   0 allocs
Value.Field(0).String()         3.50 ns/op   0 allocs    ~4x
ValueOf(u).Field(0).String()    4.43 ns/op   0 allocs（含 ValueOf）
Value.FieldByName("Name")      59.50 ns/op   0 allocs   ~72x  ← 真凶
FieldByIndex(缓存的 []int)      14.45 ns/op   0 allocs         ← 缓存索引的效果

u.Age = 1 直接设                0.63 ns/op
Value.SetInt(1)                 3.04 ns/op                ~5x

直接调普通函数                    0.64 ns/op   0 allocs
u.Greet() 直接调                27.39 ns/op   1 alloc
Value.Method.Call(nil)         315.6  ns/op   4 allocs
MakeFunc 包装后调用              470.2  ns/op   5 allocs

reflect.TypeOf(u)               3.62 ns/op
reflect.TypeFor[User]()         0.88 ns/op                 ← 快 4 倍
Value.Interface()               4.93 ns/op
v.Interface().(User)           10.35 ns/op
reflect.TypeAssert[User](v)    20.00 ns/op                 ← 实测反而更慢

a1 == a2（struct 直接比）         6.87 ns/op   0 allocs
reflect.DeepEqual(a1, a2)      198.0  ns/op   2 allocs     ~29x
```

**四个和流传说法不符的点**：

1. **取字段只慢 4 倍**，不是"反射慢 100 倍"——真正的性能杀手是 **`FieldByName`（72 倍）**，因为它要遍历所有字段并比较字符串；
2. **`TypeFor[T]` 比 `TypeOf(v)` 快 4 倍**：前者编译期确定类型，后者要装箱再读 eface；
3. **`TypeAssert[T]`（1.25+）实测比 `Interface().(T)` 慢一倍**——它的价值在类型安全和"避免装箱分配"（本例两者都没分配），不在速度；
4. **`DeepEqual` 有分配**（2 allocs），热路径上别用。

### 3.5 四条优化手法

标准库和主流库都在用：

1. **缓存 Type 级别的解析结果**——`encoding/json` 的 `cachedTypeFields`（`sync.Map`，见 json.md 2.1）、`reflect.Type` 本身也可以做 map key（同一类型是同一个指针）；
2. **用 `[]int` 索引路径代替 `FieldByName`**——实测 14.45ns vs 59.50ns；
3. **走一次反射生成闭包，之后走闭包**——`sqlx`、`gorm` 的做法：第一次用反射为每个字段生成 `func(dst, src)`，之后调闭包；
4. **代码生成**——`easyjson`、`protobuf-go`、`sqlc`：编译期生成零反射代码，快 5-10 倍，代价是要跑 generator。

### 3.6 反射与接口的往返

```go
var s fmt.Stringer = User{Name: "bob"}

reflect.ValueOf(s).Type()             // main.User ← 接口这层没了！
reflect.ValueOf(&s).Elem().Type()     // fmt.Stringer ← 这才是接口类型
reflect.ValueOf(&s).Elem().Elem()     // main.User（动态值）
```

`reflect.ValueOf` 会**自动穿透接口**拿到动态类型。这是写序列化/校验库时最容易搞错的地方。

判断类型是否实现接口用 `Type.Implements`：

```go
stringerType := reflect.TypeFor[fmt.Stringer]()
reflect.TypeFor[User]().Implements(stringerType)     // true
reflect.TypeFor[*User]().Implements(stringerType)    // true（值方法被指针继承）
reflect.TypeFor[Address]().Implements(stringerType)  // false
```

### 3.7 什么时候不该用反射

反射的正当用途其实很窄：

| 场景 | 是否该用反射 |
| --- | --- |
| 序列化/反序列化通用库 | ✓（或代码生成） |
| ORM / 配置绑定 / 校验器 | ✓（或代码生成） |
| 测试辅助（深比较、构造随机值） | ✓ |
| 依赖注入容器 | ✓（启动期一次性） |
| **业务逻辑里"通用地"处理多种类型** | ✗ **用泛型或接口**（generic.md 1.9） |
| 绕过未导出字段 | ✗ 设计问题 |
| 性能敏感的热路径 | ✗ |

**1.18 之后很多老的反射用法应该换成泛型**：如果类型集在编译期是已知的、有限的，泛型能给你同样的复用度 + 类型安全 + 好几倍的性能。

## 四、常见面试题

**1. 反射三定律是什么？**
① 从接口值可以反射出 `reflect.Value`；② 从 `reflect.Value` 可以还原成接口值；③ 要修改 `reflect.Value`，它必须是"可设置的"（可寻址 + 导出）。第三条的实践后果是 `ValueOf` 必须传指针再 `.Elem()`（见 1.1）。

**2. `reflect.Value` 里存了什么？为什么反射慢？**
`typ_ *abi.Type` + `ptr unsafe.Pointer` + `flag`，共 24 字节。慢的根源是每次进出反射边界都要装箱/拆箱（可能触发堆分配）+ 查类型表 + 位运算判断，而不是单个操作复杂（见 2.1）。

**3. `Kind` 和 `Type` 有什么区别？**
`Kind` 是底层种类（26 种），`Type` 是具体类型（无穷）。`type MyInt int` 的 Kind 是 `int` 但 Type 是 `main.MyInt`。写通用代码 switch Kind，判断具体类型比 Type（见 1.3）。

**4. 什么是 settable？为什么 `reflect.ValueOf(x).Field(0).Set(...)` 会 panic？**
`CanSet = CanAddr && 字段导出`。`ValueOf(x)` 拿到的是装箱时产生的**副本**，不可寻址，所以改不了。必须 `reflect.ValueOf(&x).Elem()`（见 1.1、1.4）。

**5. map 的元素和 slice 的元素，哪个可以通过反射修改？**
slice 元素可以（`sv.Index(0).SetInt(99)`，底层数组固定可寻址）；map 元素不行（map 会重哈希搬移，元素不可寻址），只能 `SetMapIndex` 整体替换（见 1.4）。

**6. 反射能读写未导出字段吗？**
能读类型和 Kind；`Interface()` 会 panic（`flagRO` 标记）；`String()` 是个后门能拿到 string 值。要真正读写得用 `reflect.NewAt(t, unsafe.Pointer(v.UnsafeAddr()))`——测试和序列化库里偶尔用，生产代码里出现基本是设计问题（见 3.3）。

**7. `reflect.DeepEqual` 有哪些坑？**
`nil slice ≠ 空 slice`、`NaN != NaN`、函数只有都是 nil 才相等、类型必须完全一致、**会比较未导出字段**（`time.Time`、`sync.Mutex` 容易误判）、有 2 次分配。测试里用 `go-cmp`，简单场景用 `slices.Equal`（见 2.3）。

**8. 反射到底慢多少？瓶颈在哪？**
实测：取字段 4x（3.5ns vs 0.83ns），**`FieldByName` 72x（59.5ns）**，方法调用 ~11x 且 4 次分配，`DeepEqual` 29x。瓶颈是 `FieldByName` 的字符串遍历和 `Call` 的 `[]Value` 构造，不是"反射本身"（见 3.4）。

**9. 怎么优化反射代码？**
① 缓存 Type 级别的解析结果（`encoding/json` 的做法）；② 用 `FieldByIndex([]int)` 代替 `FieldByName`（4 倍差距）；③ 反射一次生成闭包，之后走闭包（sqlx/gorm）；④ 代码生成，彻底消灭反射（easyjson/protobuf）（见 3.5）。

**10. `reflect.TypeOf` 和 `reflect.TypeFor[T]` 有什么区别？**
`TypeOf(v)` 要先把 v 装箱成 `any` 再读 eface；`TypeFor[T]()`（1.22+）编译期就确定，**快 4 倍**，而且对不方便造零值的类型（接口）更方便（见 1.2、3.4）。

**11. `reflect.ValueOf(接口变量)` 拿到的是接口类型还是动态类型？**
动态类型——`ValueOf` 会自动穿透接口。要拿接口类型本身得 `reflect.ValueOf(&iface).Elem()`。判断"是否实现某接口"用 `Type.Implements(reflect.TypeFor[I]())`（见 3.6）。

**12. `MakeFunc` 是什么？代价多大？**
用运行时构造的函数值填充任意函数类型，签名 `MakeFunc(typ Type, fn func([]Value) []Value) Value`。实测比直接调用慢 700 倍（470ns vs 0.64ns）且 5 次分配，只适合"调用本身就很贵"的场景，如 RPC 桩（见 2.2）。

**13. 有了泛型，反射还有必要吗？**
有，但用途变窄了。**类型集编译期已知**就用泛型（类型安全 + 快几倍）；**只有运行时才知道类型**（解析任意 JSON、ORM 映射任意 struct、依赖注入）才用反射。1.18 之后很多老的 `any + 反射` 代码应该换成泛型（见 3.7、generic.md 1.9）。

**14. 1.26 的迭代器反射 API 有什么用？**
`Type.Fields()`/`Methods()`/`Ins()`/`Outs()`、`Value.Fields()`/`Methods()` 把 `for i := range t.NumField()` 的样板换成 `for f := range t.Fields()`。注意 1.23 的 `Value.Seq()` 语义对齐 `range`——slice 上单变量给的是**下标**不是元素（见 1.6）。
