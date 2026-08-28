# unsafe

> 环境：`go version go1.26.3 darwin/amd64`。源码：`unsafe/unsafe.go`（全是编译器内建，没有实现）、`runtime/checkptr.go`。配套代码：`notes/uns/`。
>
> 版本演进：
> - **1.17**：`unsafe.Add`、`unsafe.Slice`——指针算术和"指针+长度→slice"终于有了安全写法。
> - **1.20**：`unsafe.String`、`unsafe.StringData`、`unsafe.SliceData`；同时 **`reflect.SliceHeader`/`StringHeader` 标记为 deprecated**。
> - **1.23**：`//go:linkname` 收紧（`-checklinkname`，拉黑了一批 runtime 符号），大量老库因此失效。

## 一、基础

### 1.1 三个编译期常量函数

```go
unsafe.Sizeof(x)     // 类型占多少字节（不含指针指向的数据）
unsafe.Alignof(x)    // 对齐要求
unsafe.Offsetof(s.f) // 字段在结构体里的偏移
```

**它们是编译期常量**，可以用在 `const` 和数组长度里：

```go
const sz = unsafe.Sizeof(mixed{})   // 24
var arr [sz]byte                     // 合法
```

字段顺序对大小的影响（struct.md 2.1 的实测版）：

```text
type mixed  struct { a bool; b int64; c int16; d [3]byte }   // Sizeof=24
  a bool     offset=0   size=1
  b int64    offset=8   size=8    ← 前面填了 7 字节
  c int16    offset=16  size=2
  d [3]byte  offset=18  size=3    ← 后面填 3 字节到 24

type packed struct { b int64; c int16; d [3]byte; a bool }   // Sizeof=16 ← 省 8 字节
```

常见类型的大小（amd64）：

```text
bool/int8       1        string          16   （data + len）
int/指针         8        []byte          24   （data + len + cap）
map/chan/func   8        any/error       16   （type + data）
```

`Sizeof("很长的字符串")` **永远是 16**——它只算 header。

### 1.2 `unsafe.Pointer` 与 `uintptr` 的根本区别

文档说 `unsafe.Pointer` 有四种特殊转换能力：任意指针 ↔ `Pointer`、`uintptr` ↔ `Pointer`。

但真正要记住的只有一句：

| | 是什么 | GC 会不会保活 | 栈移动时会不会被修正 |
| --- | --- | --- | --- |
| `unsafe.Pointer` | **指针** | **会** | **会** |
| `uintptr` | **整数** | 不会 | 不会 |

所以 **`uintptr` 只能作为"某个表达式内部的临时算术中间值"存在**。把它存进变量、字段、map，就等于持有一个随时可能失效的地址（见 3.1）。

`go vet` 会对可疑的 `unsafe.Pointer(u)` 报 `possible misuse of unsafe.Pointer`——而它报得对，我在示例里想演示这个反例都被它拦下了。

### 1.3 1.17/1.20 的新 API

```go
// 1.17：指针算术
third := (*int32)(unsafe.Add(unsafe.Pointer(&arr[0]), 2*unsafe.Sizeof(arr[0])))
// 老写法：(*int32)(unsafe.Pointer(uintptr(unsafe.Pointer(&arr[0])) + 2*4))

// 1.17：指针 + 长度 -> slice
s := unsafe.Slice(&arr[0], len(arr))       // 和 arr 共享内存

// 1.20：[]byte <-> string 零拷贝
str := unsafe.String(unsafe.SliceData(b), len(b))
p := unsafe.StringData(str)                 // *byte
q := unsafe.SliceData(b)                    // *byte
```

这四个 API 的意义不只是"更短"，而是把最常见的三种 unsafe 用法变成了**编译器认识、`checkptr` 能检查**的形式。老写法（手拼 header、裸 uintptr 算术）编译器完全看不懂，只能靠人肉保证正确。

**1.20 之后 `reflect.SliceHeader`/`StringHeader` 已 deprecated，不要再用**（见 3.4）。

## 二、六种合法模式

`unsafe.Pointer` 的文档列出了**六种**合法模式，"不属于这些模式的代码今天很可能就是错的，或者将来会变错"。挑三种最常用的展开。

### 2.1 模式 ①：`*T1` → `Pointer` → `*T2`（类型重新解释）

```go
// 标准库自己就这么写（math.Float64bits）
func Float64bits(f float64) uint64 {
    return *(*uint64)(unsafe.Pointer(&f))
}
```

```text
float64 3.14 的 bit pattern = 0x40091eb851eb851f
和 math.Float64bits 一致: true
```

**前提（文档原文）：T2 不大于 T1，且两者内存布局等价。**

两个常见错误：

- 把 `*[2]byte` 当 `*int64` 读——越界读到别人的内存；
- 忽略字节序——同一份代码在大端机器上得到不同结果。

一个真实用途是零拷贝地把数值切片当字节流写文件：

```go
nums := []int32{1, 2, 3}
raw := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(nums))), len(nums)*4)
// [1 0 0 0 2 0 0 0 3 0 0 0]   ← 小端
```

### 2.2 模式 ③：`Pointer` → `uintptr` 做算术 → **立刻**转回 `Pointer`

```go
// ✓ 两次转换必须在同一个表达式里，中间只能有算术
agePtr := (*int32)(unsafe.Pointer(uintptr(unsafe.Pointer(&r)) + unsafe.Offsetof(r.Age)))

// ✓ 1.17+ 更好的写法
agePtr := (*int32)(unsafe.Add(unsafe.Pointer(&r), unsafe.Offsetof(r.Age)))
```

文档明确列出的三条禁忌：

```go
// ✗ 中间存进变量
u := uintptr(p)
p = unsafe.Pointer(u + offset)

// ✗ 指到分配区之外（C 里合法的 one-past-the-end 在 Go 里非法！）
end = unsafe.Pointer(uintptr(unsafe.Pointer(&s)) + unsafe.Sizeof(s))

// ✗ 对 nil 做算术
u := unsafe.Pointer(nil)
p := unsafe.Pointer(uintptr(u) + offset)
```

**为什么尾后指针非法**：Go 的 GC 要根据指针值判断它属于哪个对象（span 查找），一个刚好指向对象末尾之后的地址会被误判成"下一个对象的开头"，导致错误的保活甚至错误的标记。

### 2.3 模式 ④：syscall 参数

```go
// ✓ 转换必须出现在调用的参数列表里
syscall.Syscall(SYS_READ, uintptr(fd), uintptr(unsafe.Pointer(p)), uintptr(n))

// ✗ 存进变量：编译器不会为它保活 p
u := uintptr(unsafe.Pointer(p))
syscall.Syscall(SYS_READ, uintptr(fd), u, uintptr(n))
```

编译器**只在参数列表里**识别这个模式，并保证被引用的对象在调用期间不被回收/移动。

### 2.4 模式 ⑤：`reflect.Value.UnsafeAddr` / `Pointer` 必须立刻转换

```go
// ✓
real := reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()

// ✗ 存进变量
u := v.UnsafeAddr()
p := unsafe.Pointer(u)
```

`reflect` 让这两个方法返回 `uintptr` 而不是 `unsafe.Pointer`，是为了逼调用方**显式 import unsafe**。1.18 起有 `Value.UnsafePointer()` 直接返回 `unsafe.Pointer`，优先用它。

配合模式 ⑤ 可以读写未导出字段（reflect.md 3.3）：

```text
读未导出字段: 42
写未导出字段: 100
```

### 2.5 剩下两种

- **模式 ②**：`Pointer` → `uintptr`（且**不转回来**）——唯一正当用途是打印地址；
- **模式 ⑥**：`reflect.SliceHeader`/`StringHeader` 的 `Data` 字段 ↔ `Pointer`——**1.20 起已废弃**，用 `unsafe.Slice`/`String` 代替。

## 三、常见陷阱

### 3.1 把 `uintptr` 存进变量：最经典的 unsafe bug

```go
// ✗
u := uintptr(unsafe.Pointer(obj))
// ... 中间发生了 GC 或栈增长 ...
p := (*T)(unsafe.Pointer(u))     // u 指向的可能已经是别的东西
```

**两个独立的失效原因**：

1. **GC 回收**：`uintptr` 不是引用，不保活对象（gc.md 2.2）；
2. **栈移动**：goroutine 栈增长时会整体拷贝并按指针位图**修正所有指针**，但 `uintptr` 是整数，不会被修正。mem.md 3.2 的实测里，一次 400 层递归就发生了 **4 次栈搬家**。

隐蔽变体：把 `unsafe.Pointer` 存进只有 `uintptr` 字段的结构体、存进 `map[uintptr]T`、存进 `atomic.Uintptr`——都不保活。

正确形态只有一种：**转换和算术在同一个表达式里，结果立刻变回 `Pointer`**。

### 3.2 `checkptr`：给 unsafe 加的运行时检查

`-race` 或 `-gcflags=all=-d=checkptr` 会插入两类检查（`runtime/checkptr.go`）：

| 检查 | 抓什么 |
| --- | --- |
| `checkptrAlignment` | 转换后的指针不满足目标类型的对齐要求 |
| `checkptrArithmetic` | 算出来的指针不在原对象的分配范围内 |

实测（`notes/uns/` 会起一个 `-race` 子进程验证）：

```text
$ UNS_DEMO=checkptr go run -race .
fatal error: checkptr: pointer arithmetic result points to invalid allocation
runtime.checkptrArithmetic(...)
    /usr/local/go/src/runtime/checkptr.go:69
```

**但它只能抓一部分**。同一台机器上，把 `[]byte` 的第 1 个字节当 `*int64` 读（明显未对齐）**没有**被拦住，读出来是 0。文档原话：

> Running "go vet" can help find uses of Pointer that do not conform to these patterns, but **silence from "go vet" is not a guarantee** that the code is valid.

另一个实测细节：**结果必须真的被用到**（比如存进包级变量），否则整段可能被优化掉、检查也就不会插入。这意味着 checkptr 能不能发现问题，还取决于优化——又一条"别指望工具兜底"的理由。

### 3.3 写只读内存 = SIGSEGV（不是 panic）

```go
s := "hello world"
b := unsafe.Slice(unsafe.StringData(s), len(s))
b[0] = 'H'    // 💥
```

```text
unexpected fault address 0x247f61c
fatal error: fault
```

- 字符串字面量在 **`.rodata` 只读段**，写入触发 SIGSEGV；
- 这是**信号级错误，不是 panic**：`recover` 救不了，进程直接死；
- 所以 `unsafe.Slice(unsafe.StringData(s), len(s))` 拿到的 `[]byte` **只能读**。

对称地，`unsafe.String(unsafe.SliceData(b), len(b))` 之后**不能再改 `b`**——否则"不可变"的字符串会在别人眼前变化（string.md 2.2 有实测）。

### 3.4 别自己拼 header

```go
// ✗ 老代码常见写法
type sliceHeader struct{ Data uintptr; Len, Cap int }
h := (*sliceHeader)(unsafe.Pointer(&b))
```

三个理由：

1. **Go 从未承诺 slice/string 的内存布局**，理论上随时能改；
2. `Data` 是 `uintptr`——一旦你构造了一个 header **变量**，里面的 `Data` 就不再保活对象（3.1 的坑）；
3. **1.20 起 `reflect.SliceHeader`/`StringHeader` 明确 deprecated**。

正确写法：`unsafe.Slice` / `unsafe.String` / `unsafe.SliceData` / `unsafe.StringData`。

### 3.5 `//go:linkname`

```go
import _ "unsafe"   // 必须导入，否则编译报错

//go:linkname runtimeNanotime runtime.nanotime
func runtimeNanotime() int64    // 只声明不实现，由链接器绑定
```

```text
runtime.nanotime() 两次调用差 79 ns（单调时钟，比 time.Now 便宜）
```

这是**完全绕过类型系统和包边界**的后门。三条现实：

- runtime 内部函数**没有任何兼容性承诺**，升级 Go 随时可能崩；
- **Go 1.23 起收紧了限制**（`-checklinkname`，拉黑了一批符号），历史上大量库靠它偷 `runtime.g`、`fastrand`、`memhash`，1.22+ 陆续失效；
- runtime 源码里能看到官方的抱怨清单——`Notable members of the hall of shame`（sync.md 里那段 `sync_runtime_canSpin` 的注释就是一例）。

自己写库时**不要**用它。真的需要 runtime 能力，优先找导出 API（`runtime/metrics`、`runtime/debug`、`sync/atomic`）。

### 3.6 `unsafe` 与逃逸分析

`unsafe.Pointer` 会让编译器的逃逸分析变保守：一旦一个对象的地址流经 `unsafe.Pointer`，编译器通常**无法证明它不逃逸**，于是搬到堆上。这意味着"为了性能用 unsafe"有时反而变慢——**必须实测**。

### 3.7 `unsafe` 与 cgo

cgo 的规则和 unsafe 交织在一起，两条最容易违反的：

1. **C 不得长期保存 Go 指针**——Go 的栈会移动、GC 不扫描 C 内存；
2. **传给 C 的 Go 指针，其指向的内存里不能再包含 Go 指针**。

需要长期共享就用 `C.malloc` 分配（不在 Go 堆里），或者用 handle 表（`runtime/cgo.Handle`，1.17+）把 Go 对象换成一个整数交给 C。

## 四、什么时候真的该用

**值得用**：

- 热路径上 KB 级以上的 `[]byte` ↔ `string` 零拷贝（string.md 2.2 实测 1650 倍）；
- 类型重新解释：`math.Float64bits` 这类位操作；
- 和 C / 系统调用 / `mmap` 内存互操作；
- 序列化库里按偏移直接读写字段（避开 reflect 开销）；
- 测试里读写未导出字段。

**不该用**：

- 只为省一次拷贝就在业务代码里散落 unsafe；
- 绕过未导出字段做生产逻辑（这是设计问题）；
- 自己拼 slice/string header；
- 通过 `linkname` 依赖 runtime 内部。

**用了之后的三条纪律**：

1. **收敛**到一个包/一个文件，函数名带 `Unsafe` 前缀，注释写清前提条件（"调用方保证 b 在返回的 string 使用期间不被修改"）；
2. **测试必须跑 `-race`**（顺带开 checkptr）；
3. **每次升级 Go 版本都完整跑一遍测试**——unsafe 代码是唯一会被 Go 版本升级悄悄搞坏的代码。

包文档开头那句话值得抄在每个用了 unsafe 的文件顶部：

> Package unsafe contains operations that step around the type safety of Go programs. Packages that import unsafe may be non-portable and are not protected by the Go 1 compatibility guidelines.

## 五、常见面试题

**1. `unsafe.Pointer` 和 `uintptr` 有什么区别？**
`unsafe.Pointer` 是**指针**：GC 认它、会保活指向的对象、栈移动时会被修正。`uintptr` 是**整数**：三者全不成立。所以 `uintptr` 只能作为表达式内部的临时算术中间值（见 1.2、3.1）。

**2. 为什么不能把 `uintptr` 存进变量再转回 `Pointer`？**
两个独立原因：① GC 不认 `uintptr`，对象可能已被回收；② 栈增长时 runtime 会拷贝栈并修正所有指针，但不会修正整数。唯一正确形态是"转换 + 算术在同一个表达式里"（见 3.1）。

**3. `unsafe.Sizeof` 是运行时调用吗？**
不是，它和 `Alignof`/`Offsetof` 都是**编译期常量**，可以用在 `const` 和数组长度里。`Sizeof(string)` 永远是 16（只算 header，不含数据）（见 1.1）。

**4. Go 里为什么不能构造"尾后指针"（one-past-the-end）？**
GC 需要根据指针值定位它属于哪个对象（span 查找）。刚好指向对象末尾之后的地址会被误判成"下一个对象的开头"，造成错误保活或错误标记。C 里这是合法的，Go 里不是（见 2.2）。

**5. 怎么零拷贝地把 `[]byte` 转成 `string`？有什么前提？**
1.20 起 `unsafe.String(unsafe.SliceData(b), len(b))`。前提：**转换后绝不修改 `b`**，且 `b` 的生命周期覆盖字符串的使用期。反方向（`string → []byte`）更危险：写字面量的内存会 SIGSEGV，且 recover 不了（见 1.3、3.3、string.md 2.2）。

**6. `checkptr` 是什么？能完全依赖它吗？**
`-race`/`-d=checkptr` 插入的运行时检查，抓"未对齐的转换"和"越界的指针算术"。**不能完全依赖**：实测未对齐读 `[]byte` 的第 1 字节当 `int64` 没被拦住；而且结果必须真的被用到，否则可能被优化掉、检查也不插入（见 3.2）。

**7. `reflect.SliceHeader` 为什么被废弃了？**
它的 `Data` 是 `uintptr`——一旦构造成变量，就不再保活对象。而且 Go 从未承诺 slice/string 的内存布局。1.20 起用 `unsafe.Slice`/`String`/`SliceData`/`StringData` 代替（见 3.4）。

**8. `//go:linkname` 是什么？为什么不该用？**
绕过包边界直接绑定符号（需要 `import _ "unsafe"` 且函数只声明不实现）。runtime 内部符号无兼容承诺，1.23 起有 `-checklinkname` 拉黑了一批，历史上一大批库因此在 1.22+ 失效（见 3.5）。

**9. 用了 `unsafe` 一定更快吗？**
不一定。一旦对象地址流经 `unsafe.Pointer`，逃逸分析通常变保守，对象被搬到堆上——可能反而更慢。必须实测（见 3.6）。

**10. `unsafe.Pointer` 的六种合法模式分别是什么？**
① `*T1`→`Pointer`→`*T2`（布局等价、T2 不更大）；② `Pointer`→`uintptr`（不转回来，只用于打印）；③ `Pointer`→`uintptr` 算术→`Pointer`（同一表达式内）；④ syscall 参数列表里的转换；⑤ `reflect.Value.Pointer`/`UnsafeAddr` 的结果立刻转换；⑥ `reflect.SliceHeader`/`StringHeader` 的 `Data` 字段（已废弃）（见第二节）。

**11. cgo 里传 Go 指针给 C 有什么限制？**
C 不得长期保存 Go 指针（栈会移动、GC 不扫 C 内存）；传出去的 Go 指针所指内存里不能再含 Go 指针。长期共享用 `C.malloc` 或 `runtime/cgo.Handle`（见 3.7）。
