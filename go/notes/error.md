# error

> 环境：`go version go1.26.3`。源码：`errors/{errors,wrap,join}.go`、`fmt/errors.go`。配套代码：`notes/errs/`（目录名避开 `error` 这个类型名）。
>
> 版本演进：
> - **1.13**：`errors.Is/As/Unwrap` + `fmt.Errorf` 的 `%w`。这是 Go 错误处理的分水岭：从"字符串 + `==`"变成"可编程的错误树"。
> - **1.20**：`errors.Join`；`fmt.Errorf` 支持**多个 `%w`**；`Unwrap() []error` 成为正式约定。
> - **1.21**：`errors.ErrUnsupported`（通用"不支持此操作"哨兵）。
> - **1.26**：**`errors.AsType[E error](err error) (E, bool)`** —— 泛型版 `As`，把运行时 panic 变成编译期检查。

## 一、基础使用

### 1.1 error 只是一个接口

```go
type error interface {
    Error() string
}
```

就这一行，定义在 `builtin` 里。任何实现了 `Error() string` 的类型都是 error。

```go
err := errors.New("something failed")
// 底层类型是 *errors.errorString —— 注意是指针
errors.New("x") == errors.New("x")   // false！两次 New 是两个不同的指针
```

`errors.New` 返回指针**正是为了**让两个同文本的错误不相等——错误的身份由变量决定，不由文本决定。

`fmt.Errorf` 不带 `%w` 时等价于 `errors.New(格式化结果)`（返回 `*errors.errorString`），`errors.Unwrap` 拿到 `nil`。

### 1.2 三种错误表达方式

| 方式 | 适用场景 | 调用方怎么判断 |
| --- | --- | --- |
| **哨兵**（sentinel）`var ErrNotFound = errors.New(...)` | 只需要区分"是哪种错误" | `errors.Is(err, ErrNotFound)` |
| **自定义类型** `type ValidationError struct{...}` | 调用方需要结构化字段（字段名、状态码、行号） | `errors.As(err, &ve)` / `errors.AsType[*ValidationError](err)` |
| **不透明错误**（opaque）只返回 `error` | 调用方无需区分，只需知道失败了 | 只看 `err != nil` |

标准库的哨兵：`io.EOF`、`io.ErrUnexpectedEOF`、`sql.ErrNoRows`、`fs.ErrNotExist`、`fs.ErrExist`、`context.Canceled`、`context.DeadlineExceeded`、`errors.ErrUnsupported`。

**默认选不透明错误**。哨兵和自定义类型都是**公开 API 承诺**，导出之后就不能改（见 3.8）。

### 1.3 包装：`%w`

```go
func layer1() error {
    if err := layer2(); err != nil {
        return fmt.Errorf("layer1: parse file: %w", err)
    }
    return nil
}
```

```text
最终错误: layer1: parse file: layer2: read header: unexpected EOF
Unwrap 链:
  [0] *fmt.wrapError      layer1: parse file: layer2: read header: unexpected EOF
  [1] *fmt.wrapError      layer2: read header: unexpected EOF
  [2] *errors.errorString unexpected EOF
Is(io.ErrUnexpectedEOF) = true
```

**`%w` 和 `%v` 唯一的区别**：`%w` 保留了可编程的关系（`Is`/`As` 能穿透），`%v` 只留下字符串。选哪个是个 API 设计决策，不是风格问题（见 3.4）。

### 1.4 Is 与 As

```go
errors.Is(err, fs.ErrNotExist)          // 找"值"：遍历错误树，看有没有等于 target 的
errors.As(err, &pathErr)                // 找"类型"：遍历错误树，看有没有能赋给 *fs.PathError 的
```

```text
os.Open 错误:       open /definitely/not/exist: no such file or directory
Is(fs.ErrNotExist): true
os.IsNotExist:      true    ← 老 API，不穿透 wrap，别再用
As(*fs.PathError):  Op=open Path=/definitely/not/exist Err=no such file or directory
```

判定规则（`errors/wrap.go`）：

- `Is`：`err == target`，或 `err` 实现了 `Is(error) bool` 且返回 true，然后递归 Unwrap；
- `As`：`err` 可赋值给 `*target` 的元素类型，或 `err` 实现了 `As(any) bool` 且返回 true，然后递归。

**`os.IsNotExist`/`os.IsPermission`/`os.IsTimeout` 这类老函数不穿透包装**，1.13 之后一律改用 `errors.Is`。

### 1.5 `errors.AsType`（1.26 新增）

```go
// 老写法：先声明、传指针、看 bool
var pathErr *fs.PathError
if errors.As(err, &pathErr) { use(pathErr.Path) }

// 新写法：一行，类型参数即目标类型
if pe, ok := errors.AsType[*fs.PathError](err); ok { use(pe.Path) }
```

签名：`func AsType[E error](err error) (E, bool)`。

价值不只是短：`errors.As` 的第二个参数是 `any`，传错了**只能在运行时 panic**（`errors: target must be a non-nil pointer`）；`AsType` 把这件事挪到了编译期。新代码优先用它。

### 1.6 `errors.Join`（1.20+）

```go
func validate(age int, name string) error {
    var errs []error
    if age < 0    { errs = append(errs, &ValidationError{Field: "age", Value: age}) }
    if name == "" { errs = append(errs, &ValidationError{Field: "name", Value: name}) }
    return errors.Join(errs...)   // 全 nil 时返回 nil
}
```

- `Error()` 用 **`\n`** 连接各子错误（`errors/join.go`）；
- 返回类型只实现 `Unwrap() []error`，**`errors.Unwrap()` 函数对它返回 nil**；
- `Is`/`As` 能正常穿透（深度优先，找到第一个匹配）；
- `Join(nil, nil)` → `nil`，所以可以无脑 `errors.Join(errs...)`。

典型场景：表单批量校验、批量任务汇总、多个 `defer Close()` 的错误合并（见 3.7）。

### 1.7 一个 `Errorf` 里多个 `%w`（1.20+）

```go
err := fmt.Errorf("both failed: %w and %w", ErrNotFound, ErrPermission)
errors.Is(err, ErrNotFound)    // true
errors.Is(err, ErrPermission)  // true
// 底层类型 *fmt.wrapErrors，实现 Unwrap() []error
```

`fmt/errors.go` 里按 `%w` 的个数分三条路：0 个 → `errors.New`；1 个 → `*wrapError`（`Unwrap() error`）；多个 → `*wrapErrors`（`Unwrap() []error`）。

## 二、原理

### 2.1 Unwrap 是一棵树，不是一条链

两种 Unwrap 方法：

```go
Unwrap() error      // 单个子错误：fmt.Errorf 一个 %w、自定义类型
Unwrap() []error    // 多个子错误：errors.Join、多个 %w
```

于是错误的结构是**树**。`Is`/`As` 的遍历顺序是**先序深度优先**（官方文档原文：*pre-order, depth-first traversal*）：先看自己，再依次深入每个子错误。

```text
root: %w(Join(leaf1, leaf2))
Is(leaf1) = true    Is(leaf2) = true
```

一个容易忘的点：`errors.Unwrap(err)` 这个**函数**只认 `Unwrap() error`，对 `Join` 的结果返回 `nil`。要遍历多分支得自己断言：

```go
if u, ok := err.(interface{ Unwrap() []error }); ok {
    for _, e := range u.Unwrap() { ... }
}
```

### 2.2 自定义 `Is` / `As` 方法

```go
type HTTPError struct{ Code int }

func (e *HTTPError) Error() string { return fmt.Sprintf("http status %d", e.Code) }

// 让所有 5xx 都匹配 ErrServerSide
func (e *HTTPError) Is(target error) bool {
    if target == ErrServerSide { return e.Code >= 500 }
    return false
}
```

```text
code=404 Is(ErrServerSide)=false
code=500 Is(ErrServerSide)=true
code=503 Is(ErrServerSide)=true
```

`Is` 方法用来表达"**一类**错误"；`As(target any) bool` 方法用来自己控制类型转换（复合错误如 `net.OpError` 常用）。这是把"内部错误映射成对外错误分类"的标准手法（见 3.8）。

### 2.3 `fmt.wrapError` 长什么样

```go
type wrapError struct {   // fmt/errors.go
    msg string
    err error
}
func (e *wrapError) Error() string { return e.msg }
func (e *wrapError) Unwrap() error { return e.err }
```

注意 `msg` 是**格式化时就算好的完整字符串**（包含子错误的文本）。所以：

- 打印一次错误 = 打印一个已拼好的字符串，`Error()` 不递归；
- 但**内存上**每层 wrap 都持有一份包含下层全文的字符串，深层嵌套会造成 O(n²) 的字符串占用。日志系统里对同一个错误反复 `fmt.Errorf` 包装是有实际成本的。

### 2.4 没有栈追踪，怎么办

标准库的错误**不带 stack trace**（设计取舍：错误应该是廉价的值）。三条现实路径：

1. **靠包装文本形成"人造调用链"**：`"layer1: parse file: layer2: read header: unexpected EOF"`——这也是为什么惯例是 `"pkg: op: detail"`。
2. **用 `github.com/pkg/errors`（已归档）或 `cockroachdb/errors`** 等第三方库抓栈。
3. **在日志层补**：`log/slog` 记录时带上 `source`（1.21 的 `slog.HandlerOptions{AddSource: true}`），配合 wrap 文本定位。

Go 官方多次讨论过在标准库加栈追踪（`errors.StackTrace` 提案），至今未进。

## 三、常见陷阱

### 3.1 nil 接口陷阱（最经典的一道题）

```go
type MyErr struct{}
func (e *MyErr) Error() string { return "my error" }

func badReturn(fail bool) *MyErr {   // ✗ 返回具体类型
    if fail { return &MyErr{} }
    return nil
}

var err error = badReturn(false)
err == nil   // false！
```

```text
err = badReturn(false)  -> err == nil ? false （类型 *main.MyErr）
err = goodReturn(false) -> err == nil ? true
```

原因：接口值是 `(type, data)` 两个字，这里 `type=*main.MyErr`、`data=nil`，**只有两者都为 nil 接口才等于 nil**（见 interface.md 2.1）。

**铁律：函数返回错误一律声明为 `error`，绝不返回具体的错误指针类型**。同理，中间变量也要用 `error` 类型接。`go vet` 的 nilness 分析和 staticcheck SA4023 能抓到一部分。

### 3.2 用 `==` 比较错误

```go
wrapped := fmt.Errorf("read config: %w", os.ErrNotExist)
wrapped == os.ErrNotExist              // false —— 包过就不相等
errors.Is(wrapped, os.ErrNotExist)     // true
```

- 唯一还能用 `==` 的：`io.EOF`（约定俗成，标准库承诺不包装它）——即便如此也建议写 `errors.Is`。
- **最糟的写法是比较错误字符串** `err.Error() == "not found"`：文案随时会改，还可能被上层包装。

### 3.3 `errors.As` 的参数陷阱

```go
var ve *ValidationError
errors.As(err, ve)    // ✗ panic: errors: target must be a non-nil pointer
errors.As(err, &ve)   // ✓
```

`go vet` 能静态抓到大部分（*second argument to errors.As must be a non-nil pointer...*），但通过接口/函数值间接调用时抓不到。**1.26 起用 `errors.AsType` 彻底避开这个坑**。

### 3.4 包装还是不包装

| 写法 | 什么时候用 |
| --- | --- |
| `fmt.Errorf("...: %w", err)` | 调用方确实需要 `Is`/`As` 判断底层原因；模块内部的层级传递 |
| `fmt.Errorf("...: %v", err)` | 底层是实现细节（今天 MySQL 明天 Redis）；不想让调用方依赖第三方错误类型 |
| `return err` | 这一层没有任何新信息可加 |

两个反模式：

- **每层都包一次**，日志里出现十层重复前缀：`"handler: service: repo: dao: query: ..."`。只在**跨越有意义的边界**时加上下文。
- **重复动词**：`"failed to open file: failed to open: no such file or directory"`。加的信息应该是"这一层特有的"（哪个文件、哪个用户、哪个请求 ID）。

### 3.5 错误文案规范

```go
errors.New("user: not found")     // ✓ 小写开头、无标点、带包/操作前缀
errors.New("User not found.")     // ✗
```

原因：错误常被嵌进更长的句子（`"read config: user: not found"`）。例外是以专有名词/缩写开头（`"HTTP request failed"`、`"TLS handshake timeout"`）。对应 lint 规则 `ST1005`。

### 3.6 panic 还是 error

| 用 `error` | 用 `panic` |
| --- | --- |
| 一切**可预期**的失败：IO、网络、解析、校验、业务规则 | 程序员的 **bug**，继续执行只会更糟 |
| | 不可能到达的分支：`default: panic("unreachable")` |
| | 初始化期的致命错误：`regexp.MustCompile`、`init` 里 |
| | 违反不变量（内部状态自相矛盾） |

- **库的边界要 recover**：HTTP handler、goroutine 入口、插件调用点——一个 goroutine 里的 panic 会带走整个进程。
- `recover` 只在**直接 defer 的函数**里有效（见 func.md 3.4）。
- **别用 panic/recover 做控制流**。可以在包内部用 panic 简化深递归的错误传递（`encoding/json` 就这么干），但必须在**包的导出边界**转成 error：

```go
func safeDivide(a, b int) (result int, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("safeDivide: recovered: %v", r)
        }
    }()
    return a / b, nil
}
// safeDivide(1, 0) -> safeDivide: recovered: runtime error: integer divide by zero
```

注意有些错误 **recover 不了**：`fatal error: concurrent map writes`、`unlock of unlocked mutex`、`stack overflow`、OOM（见 sync.md 3.2、mem.md 3.3）。

### 3.7 `defer` 里的错误被吞掉

```go
// ✗ Close 的错误被忽略——写文件时可能丢数据（缓冲区没 flush 成功）
func writeBad(name string, data []byte) error {
    f, err := os.Create(name)
    if err != nil { return err }
    defer f.Close()
    _, err = f.Write(data)
    return err
}

// ✓ 命名返回值 + errors.Join
func writeGood(name string, data []byte) (err error) {
    f, err := os.Create(name)
    if err != nil { return err }
    defer func() { err = errors.Join(err, f.Close()) }()
    _, err = f.Write(data)
    return err
}
```

只读场景忽略 `Close` 错误没问题；**写场景必须检查**。1.20 之前这里要手写 `if cerr := f.Close(); err == nil { err = cerr }`，现在 `errors.Join` 一行搞定，而且两个错误都保留。

用 `errcheck`（或 golangci-lint 的 errcheck）扫被忽略的错误。

### 3.8 哨兵与包装都是 API 契约

```go
// 你的库里写了这一行
return fmt.Errorf("query user: %w", pqErr)
```

调用方就可能写 `errors.Is(err, pq.ErrSSLNotSupported)`——**于是你再也不能换掉那个数据库驱动**。这是隐式的 API 泄漏。

对外的包应该：

1. 只暴露自己定义的哨兵/错误类型；
2. 内部错误用 `%v` 转成文字（保留可读性，切断可编程依赖）；
3. 需要分类时定义自己的层级（`ErrTimeout`/`ErrConflict`/`ErrInvalidInput`），用**自定义 `Is` 方法**把底层错误映射上去。

### 3.9 `errors.Is(err, err)` 与自引用

`errors.Is` 第一步就是 `err == target`，所以 `Is(err, err)` 恒为 true（哪怕 err 是 Join 出来的）。这在写"错误分类表"时容易造成误判——写循环遍历 target 列表时注意别把 `err` 自己混进去。

### 3.10 忘了 `%w` 的动词写成了 `%s`

```go
fmt.Errorf("read: %s", err)   // 编译通过、打印正常，但 Is/As 全部失效
```

没有编译错误、没有 vet 警告（`%s` 对 error 是合法的），只有在"为什么我的 `errors.Is` 不生效"时才会发现。**统一约定：包装错误只用 `%w` 或 `%v`，不用 `%s`**，这样 grep 就能审计。

### 3.11 在循环里累积 error 却只留最后一个

```go
for _, item := range items {
    if err := process(item); err != nil {
        lastErr = err       // ✗ 前面的错误全丢了
    }
}
```

用 `errors.Join`：

```go
var errs []error
for _, item := range items {
    if err := process(item); err != nil {
        errs = append(errs, fmt.Errorf("item %s: %w", item.ID, err))
    }
}
return errors.Join(errs...)
```

### 3.12 `context` 错误的判断

```go
if errors.Is(err, context.DeadlineExceeded) { /* 超时 */ }
if errors.Is(err, context.Canceled)         { /* 主动取消 */ }
```

注意：`net/http`、`database/sql` 等会把 context 错误包装在自己的错误里，**必须用 `Is` 而不是 `==`**。另外 `context.Cause(ctx)`（1.20+）能拿到 `WithCancelCause` 传入的原因（见 context.md）。

## 四、常见面试题

**1. Go 的错误处理为什么是返回值而不是异常？**
显式优于隐式：每个调用点都能看到失败的可能性和处理方式，控制流不会被非局部跳转打断。代价是啰嗦（`if err != nil` 满屏）。Go 团队多次讨论过 `try`/`check` 语法糖（2019 年的 `try` 提案），全部否决，理由是"错误处理应该显式、可读，语法糖会掩盖控制流"。

**2. `errors.New("x") == errors.New("x")` 是 true 还是 false？为什么？**
false。`errors.New` 返回 `*errorString` 指针，两次调用是两个不同对象。这正是设计意图：错误的身份由变量（哨兵）决定，而不是文本（见 1.1）。

**3. `%w` 和 `%v` 的区别？**
`%w` 生成 `*fmt.wrapError`（实现 `Unwrap() error`），保留可编程关系，`Is`/`As` 能穿透；`%v` 只把子错误格式化成字符串，关系断掉。选哪个是 API 设计决策：`%w` 意味着你把底层错误当成了对外承诺（见 1.3、3.4、3.8）。

**4. `errors.Is` 和 `errors.As` 的区别？各自怎么判断？**
`Is` 找**值**：`err == target`，或 `err` 的 `Is(error) bool` 方法返回 true，然后递归 Unwrap。`As` 找**类型**：`err` 可赋值给 target 指向的类型，或 `err` 的 `As(any) bool` 返回 true。遍历都是先序深度优先（见 1.4、2.1）。

**5. `errors.AsType` 相比 `errors.As` 好在哪？（1.26 新增）**
签名 `AsType[E error](err error) (E, bool)`：不需要预声明变量、不需要取地址、**参数类型错误在编译期就报**。`errors.As` 的第二个参数是 `any`，传错只能运行时 panic（见 1.5、3.3）。

**6. `errors.Join` 的错误怎么遍历？`errors.Unwrap` 能拿到吗？**
`Join` 返回的类型只实现 `Unwrap() []error`，`errors.Unwrap()` 函数（只认 `Unwrap() error`）对它返回 nil。要遍历得断言 `interface{ Unwrap() []error }`。`Is`/`As` 是能正常穿透的（见 1.6、2.1）。

**7. 为什么 `err != nil` 但 `err` 打印出来是 `<nil>`？**
典型的 nil 接口陷阱：函数返回了具体的错误指针类型（值为 nil），装进 `error` 接口后 type 字段非 nil，所以 `err != nil`，但 `%v` 打印 data 得到 `<nil>`。修法：返回类型一律写 `error`（见 3.1）。

**8. `os.IsNotExist(err)` 和 `errors.Is(err, fs.ErrNotExist)` 有什么区别？**
前者是 1.13 之前的 API，**不会穿透 `%w` 包装**，只检查最外层。后者遍历整棵错误树。所有 `os.IsXxx` 系列都应该换成 `errors.Is`（见 1.4）。

**9. 什么时候用哨兵错误，什么时候用自定义错误类型？**
只需要区分种类 → 哨兵 + `Is`；调用方需要结构化信息（字段名、状态码、重试建议）→ 自定义类型 + `As`。都不需要 → 不透明错误（只返回 error）。默认选最后一种，因为前两种都是不可撤回的 API 承诺（见 1.2、3.8）。

**10. Go 的 error 为什么没有栈追踪？怎么补？**
设计取舍：error 是**廉价的值**，抓栈要付分配和遍历的代价，而且大多数错误只需要"哪一步失败了"而非完整栈。补法：靠 wrap 文本形成人造调用链（`"pkg: op: detail"`）、用第三方库抓栈、或在日志层加 `AddSource`（见 2.4）。

**11. panic/recover 和 error 的边界在哪？**
可预期失败用 error；程序员 bug、不变量被破坏、初始化致命错误用 panic。库内部可以用 panic 简化深递归（`encoding/json` 就是），但必须在导出边界 recover 成 error。goroutine 入口、HTTP handler 一定要有 recover。注意 `fatal error`（并发写 map、解锁未加锁的 mutex、栈溢出）**recover 不了**（见 3.6）。

**12. `defer f.Close()` 有什么问题？**
错误被丢掉。读文件无所谓，**写文件可能意味着数据没落盘**。正确写法是命名返回值 + `defer func(){ err = errors.Join(err, f.Close()) }()`（见 3.7）。

**13. 为什么不建议对第三方错误用 `%w`？**
一旦包装，调用方就能 `errors.Is(err, thirdparty.ErrXxx)`，你的实现细节变成了 API 契约，之后换库就是破坏性变更。对外包装用 `%v`，需要分类时映射到自己定义的错误上（见 3.8）。

**14. 错误信息应该怎么写？**
小写开头、不带标点、带 `包名: 操作:` 前缀，让多层包装自然拼成一条可读的链。原因是错误常被嵌进更长的句子。对应 lint 规则 ST1005（见 3.5）。

**15. 一个 `fmt.Errorf` 里能有多个 `%w` 吗？**
1.20 起可以。多个 `%w` 会生成 `*fmt.wrapErrors`（实现 `Unwrap() []error`），`Is`/`As` 对每个都能匹配（见 1.7）。
