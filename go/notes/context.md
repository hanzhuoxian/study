# Context

> 环境：`go version go1.26.3`。内部结构以该版本源码 `src/context/context.go`（806 行）为准。版本沿革需注意：`WithCancelCause`/`Cause` 是 Go 1.20 加入；`WithoutCancel`/`AfterFunc`/`WithDeadlineCause`/`WithTimeoutCause` 是 Go 1.21 加入；Go 1.21 之前 `Background()`/`TODO()` 返回的是指向 `*emptyCtx` 全局变量的指针，现在改成了两个零大小值类型 `backgroundCtx{}`/`todoCtx{}`（都内嵌 `emptyCtx`），语义不变但打印结果和类型断言写法有差异。

context 只解决三件事：**传递取消信号**、**传递截止时间**、**传递请求域数据**。它不是"万能上下文对象"，也不是线程局部存储的替代品。

## 一、基础使用

### 1.1 Context 接口：只有四个方法

```go
type Context interface {
    Deadline() (deadline time.Time, ok bool) // 截止时间，没有则 ok == false
    Done() <-chan struct{}                   // 取消信号：被取消时 close；永不取消的 ctx 返回 nil
    Err() error                              // Done 未关闭返回 nil；否则 Canceled 或 DeadlineExceeded
    Value(key any) any                       // 按 key 沿链向上查找请求域数据
}
```

四个方法都必须能被多个 goroutine 并发调用，且**幂等**：`Done()` 多次调用返回同一个 channel，`Err()` 一旦返回非 nil 就永远返回同一个 error，`Deadline()` 多次调用结果相同。

包级只有两个错误值：

```go
var Canceled = errors.New("context canceled")          // 主动取消
var DeadlineExceeded error = deadlineExceededError{}   // 超时
```

`deadlineExceededError` 额外实现了 `Timeout() bool` 和 `Temporary() bool`（都返回 true），所以它能被 `net.Error` 那套超时判断识别——这是它不用 `errors.New` 的原因。

### 1.2 根 context：Background 与 TODO

```go
ctx := context.Background() // 永不取消、无值、无 deadline，用于 main、init、测试、请求入口
ctx := context.TODO()       // 语义等价，但表示"这里以后要换成真的 ctx"，供静态分析工具识别
```

两者底层完全一样（都是 `emptyCtx`：`Done()` 返回 nil、`Err()` 返回 nil、`Value()` 返回 nil），唯一区别是 `String()` 返回的名字。选 `TODO` 纯粹是给人和 linter 看的信号。

**永远不要传 nil context**：`WithCancel`/`WithValue` 等对 nil parent 直接 panic（见 3.4）。

### 1.3 WithCancel：手动取消

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel() // 必须调用，且建议 defer，否则泄漏（见 3.1）

go worker(ctx)
// ... 某个条件满足
cancel() // 关闭 ctx.Done()，并级联取消所有派生 context
```

worker 侧的标准写法：

```go
func worker(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err() // 返回 context.Canceled 或 context.DeadlineExceeded
        default:
        }
        // ... 干一小段活，保证能及时回到 select
    }
}
```

`cancel` 可以被多次调用、被多个 goroutine 并发调用，第一次之后都是空操作（`cancelCtx.cancel` 里检查 `c.err` 已设置就直接返回）。

### 1.4 WithTimeout / WithDeadline：超时控制

```go
// 相对时间
ctx, cancel := context.WithTimeout(parent, 500*time.Millisecond)
defer cancel()

// 绝对时间
ctx, cancel := context.WithDeadline(parent, time.Now().Add(500*time.Millisecond))
defer cancel()
```

`WithTimeout(parent, d)` 就是 `WithDeadline(parent, time.Now().Add(d))` 的一行封装。超时后 `ctx.Err() == context.DeadlineExceeded`，手动 cancel 则是 `context.Canceled`——谁先发生取谁。

**deadline 只会收敛不会放宽**：如果父 ctx 的 deadline 比要设的更早，`WithDeadline` 直接退化成 `WithCancel(parent)`（源码里的 `if cur, ok := parent.Deadline(); ok && cur.Before(d)` 分支），不会给你一个比父更晚的截止时间（见 3.8）。

即使超时了也要 `defer cancel()`：它负责 `timer.Stop()` 和从父节点摘除自己。

### 1.5 WithValue：请求域数据

```go
type ctxKey struct{} // 非导出类型做 key，天然避免跨包冲突

ctx := context.WithValue(parent, ctxKey{}, "req-123")
v, ok := ctx.Value(ctxKey{}).(string)
```

约定俗成的封装方式（标准库文档推荐）：

```go
package reqid

type key struct{}

func New(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, key{}, id)
}

func From(ctx context.Context) (string, bool) {
    id, ok := ctx.Value(key{}).(string)
    return id, ok
}
```

对外只暴露类型安全的 `New`/`From`，key 本身不导出。key 必须是**可比较**类型，否则 `WithValue` panic；用 `string` 做 key 能跑但容易撞（见 3.2）。

只放"跨越进程和 API 边界的请求域数据"：trace id、request id、认证身份、语言标签。不要放可选参数、配置、数据库连接、logger（见 3.11）。

### 1.6 WithCancelCause 与 Cause：区分"为什么取消"

`ctx.Err()` 只有两种取值，信息量太少。Go 1.20 加了原因：

```go
ctx, cancel := context.WithCancelCause(parent)
cancel(fmt.Errorf("upstream returned 503"))

ctx.Err()            // context.Canceled  ← 语义不变，兼容老代码
context.Cause(ctx)   // upstream returned 503  ← 真正的原因
```

超时也能带原因（Go 1.21）：

```go
var ErrSlowQuery = errors.New("query too slow")
ctx, cancel := context.WithTimeoutCause(parent, time.Second, ErrSlowQuery)
defer cancel()
<-ctx.Done()
ctx.Err()          // context.DeadlineExceeded
context.Cause(ctx) // ErrSlowQuery
```

三条规则：

1. `Cause` 沿链向上找最近的 `cancelCtx` 取它的 `cause`；没有显式 cause 时 `Cause(ctx) == ctx.Err()`。
2. 未取消时 `Cause(ctx) == nil`。
3. **第一个取消者胜出**：父先被 cause1 取消，则子的 `Cause` 也是 cause1；子先被 cause2 取消，则父是 cause1、子是 cause2（见 3.15）。
4. `CancelCauseFunc(nil)` 等价于把 cause 设为 `Canceled`。

`WithDeadlineCause`/`WithTimeoutCause` 返回的仍是普通 `CancelFunc`，**手动 cancel 不会设置那个 cause**——cause 只在超时触发时生效。

### 1.7 WithoutCancel：切断取消传播

Go 1.21 加入。典型场景：HTTP 请求已经返回、`r.Context()` 已取消，但还要用请求里的 trace id 去写审计日志。

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    // 保留 Value 链，丢掉取消信号和 deadline
    bg := context.WithoutCancel(ctx)
    go auditLog(bg) // handler 返回后 bg 不会被取消
}
```

返回的 ctx：`Done()` 返回 nil、`Err()` 返回 nil、`Deadline()` 的 ok 为 false、`Cause()` 返回 nil，但 `Value()` 照样能穿透到父链（见 2.10）。

注意它也切断了 deadline，所以后台任务要自己重新设一个超时，否则可能永远跑：

```go
bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
defer cancel()
```

### 1.8 AfterFunc：取消后回调

Go 1.21 加入。以前要"ctx 取消时做点清理"，得自己起一个 goroutine 守着 `<-ctx.Done()`；现在：

```go
stop := context.AfterFunc(ctx, func() {
    conn.Close() // ctx 取消后在独立 goroutine 里执行
})
defer stop() // 不再需要时注销；返回 true 表示成功阻止了 f 执行
```

要点：

- 如果 ctx 已经取消，`AfterFunc` 立即在新 goroutine 里跑 f。
- 多次 `AfterFunc` 互相独立，不覆盖。
- `stop()` 返回 false 说明 f 已经开始跑了或已被 stop 过；`stop()` **不等待** f 结束。
- 相比手写守护 goroutine，它在 ctx 从未取消的常见路径上不额外创建 goroutine（挂在父的 children map 里而已，见 2.11）。

标准库用它实现了 `Context` 到 `sync.Cond`、到 `net.Conn` deadline 之类的桥接。

### 1.9 select 中的标准用法

```go
select {
case <-ctx.Done():
    return ctx.Err()
case v := <-ch:
    use(v)
case out <- v:
}
```

两个注意点：

- **两个 case 同时就绪时 select 随机选**，所以"ctx 已取消"不保证一定走 Done 分支。要严格优先取消，得二次检查（见 3.7）。
- `ctx.Done()` 可能是 **nil channel**（`Background()`、`WithoutCancel()` 的结果）。nil channel 永久阻塞，在 select 里等价于该 case 不存在——这刚好是想要的语义，但如果整个 select 只有这一个 case，就是死锁（见 3.6、chan.md 3.3）。

### 1.10 常用并发模式

**超时调用外部服务**

```go
func fetch(ctx context.Context, url string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, err
    }
    resp, err := http.DefaultClient.Do(req) // ctx 取消会中断连接和读取
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
```

**扇出后统一取消（第一个出错就停掉全部）**

```go
g, ctx := errgroup.WithContext(ctx) // golang.org/x/sync/errgroup
for _, u := range urls {
    g.Go(func() error {
        return fetch(ctx, u) // 任一返回非 nil error → ctx 被取消 → 其余全部收到信号
    })
}
if err := g.Wait(); err != nil {
    return err
}
```

**优雅退出：把信号变成 ctx**

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

srv := &http.Server{Addr: ":8080"}
go func() { _ = srv.ListenAndServe() }()

<-ctx.Done() // 收到 SIGINT/SIGTERM

shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = srv.Shutdown(shutdownCtx) // 注意用新的根 ctx，不能用已取消的那个
```

**服务端：客户端断开自动取消**

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // 客户端断连、或 ServeHTTP 返回时被取消
    rows, err := db.QueryContext(ctx, "SELECT ...") // 取消会传到驱动、真正掐掉查询
    ...
}
```

**"第一个 context 参数"约定**

```go
func DoSomething(ctx context.Context, arg Arg) error
```

ctx 永远是第一个参数、名字就叫 ctx，不要塞进 struct（见 3.3）。

## 二、底层原理

### 2.1 全局结构：查值是链表，取消是树

context 的实现只有五种具体类型（外加两个辅助壳），全部在 `context/context.go` 里，没有 runtime 支持：

| 类型 | 构造函数 | 作用 |
|---|---|---|
| `backgroundCtx` / `todoCtx`（内嵌 `emptyCtx`） | `Background()` / `TODO()` | 根节点，什么都不做 |
| `*valueCtx` | `WithValue` | 携带一对 kv |
| `*cancelCtx` | `WithCancel` / `WithCancelCause` | 可取消 |
| `*timerCtx`（内嵌 `cancelCtx`） | `WithDeadline` / `WithTimeout`(`Cause`) | 可取消 + 定时 |
| `withoutCancelCtx` | `WithoutCancel` | 只保留 Value 链 |
| `*afterFuncCtx`（内嵌 `cancelCtx`） | `AfterFunc` | 取消时跑回调 |
| `stopCtx` | 内部使用 | 记住如何注销 AfterFunc |

每个 `WithXxx` 都是**新建一个节点并把 parent 存进去**（`valueCtx`/`cancelCtx` 内嵌 `Context` 接口字段），从来不修改 parent。所以：

- **向上是单向链表**：`Value` 从当前节点沿 parent 一路往上找，找不到就到根返回 nil。
- **向下是树**：`cancelCtx` 用 `children map[canceler]struct{}` 记住所有可取消的子节点，取消时递归下推。

一句话：**数据向上查，信号向下推**。

```
Background
   └─ valueCtx{traceID}          ← Value 查找方向：子 → 父
        └─ cancelCtx  ────────┐
             ├─ timerCtx      │  取消推送方向：父 → 子（children map）
             └─ cancelCtx     │
                  └─ valueCtx ┘
```

### 2.2 emptyCtx：零大小的根

```go
type emptyCtx struct{}

func (emptyCtx) Deadline() (deadline time.Time, ok bool) { return }
func (emptyCtx) Done() <-chan struct{}                   { return nil }
func (emptyCtx) Err() error                              { return nil }
func (emptyCtx) Value(key any) any                       { return nil }

type backgroundCtx struct{ emptyCtx }
func (backgroundCtx) String() string { return "context.Background" }
type todoCtx struct{ emptyCtx }
func (todoCtx) String() string { return "context.TODO" }
```

零字段结构体，值接收者，装进接口时不需要堆分配（interface.md 2.3）。`Done()` 返回 nil 是关键设计：**"永不取消"用 nil channel 表示**，配合 select 的 nil channel 永久阻塞语义（chan.md 3.3），既省掉一个 channel 对象，又让 `propagateCancel` 有一条极快的短路径（见 2.5）。

### 2.3 valueCtx：单向链表 + 线性查找

```go
type valueCtx struct {
    Context   // 内嵌接口 = parent 指针
    key, val any
}

func (c *valueCtx) Value(key any) any {
    if c.key == key {
        return c.val
    }
    return value(c.Context, key)
}
```

只重写 `Value`，`Deadline`/`Done`/`Err` 全靠内嵌接口自动转发到 parent（方法提升，method.md 1.5）。查找是 `value()` 里的一个 for 循环：

```go
func value(c Context, key any) any {
    for {
        switch ctx := c.(type) {
        case *valueCtx:
            if key == ctx.key { return ctx.val }
            c = ctx.Context
        case *cancelCtx:
            if key == &cancelCtxKey { return c }
            c = ctx.Context
        case withoutCancelCtx:
            if key == &cancelCtxKey { return nil } // 让 Cause 返回 nil
            c = ctx.c
        case *timerCtx:
            if key == &cancelCtxKey { return &ctx.cancelCtx }
            c = ctx.Context
        case backgroundCtx, todoCtx:
            return nil
        default:
            return c.Value(key) // 自定义实现，交回给它自己
        }
    }
}
```

注意两点：

- 写成 `for` + `type switch` 而不是递归调用 `parent.Value(key)`，是为了**避免每层一次接口方法调用**，把已知的内置类型直接展开成迭代。只有遇到用户自定义的 Context 实现才走回接口调用。
- **查找是 O(链长) 的线性扫描**，key 比较是接口比较（interface.md 2.4）。挂 10 个 value 就要最多比 10 次。所以不要把 context 当 map 用（见 3.14）。
- `c.key == key` 是接口相等比较，如果 key 的动态类型不可比较会 panic——所以 `WithValue` 在入口就用 `reflectlite.TypeOf(key).Comparable()` 挡住了。

### 2.4 cancelCtx：懒创建 done + 原子 err

```go
type cancelCtx struct {
    Context // parent

    mu       sync.Mutex            // 保护下面的字段
    done     atomic.Value          // chan struct{}，懒创建，第一次 cancel 时关闭
    children map[canceler]struct{} // 第一次 cancel 后置 nil
    err      atomic.Value          // 第一次 cancel 时写入，非 nil 即已取消
    cause    error                 // 第一次 cancel 时写入
}
```

`done` 用 `atomic.Value` + 懒创建：

```go
func (c *cancelCtx) Done() <-chan struct{} {
    d := c.done.Load()
    if d != nil { return d.(chan struct{}) }   // 快路径，无锁
    c.mu.Lock()
    defer c.mu.Unlock()
    d = c.done.Load()
    if d == nil {                              // double-check
        d = make(chan struct{})
        c.done.Store(d)
    }
    return d.(chan struct{})
}
```

**没人调用 `Done()` 就不会创建 channel**——很多 context 只是链路中转，从头到尾没人 select 它，省下一次 hchan 分配（chan.md 2.1）。

`Err()` 也走原子读：

```go
func (c *cancelCtx) Err() error {
    if err := c.err.Load(); err != nil {
        <-c.Done() // 确保 done 一定已经关闭再返回非 nil
        return err.(error)
    }
    return nil
}
```

源码注释写明"原子读比加锁快约 5 倍，在紧凑循环里有意义"。那句 `<-c.Done()` 是为了修正一个可见性问题：`cancel` 里先 `c.err.Store(err)` 再 `close(d)`，中间存在一个窗口——此时 `Err()` 已能返回非 nil，但 `Done()` 还没关闭。加上这行阻塞等待后，**"`Err()` 返回非 nil" 就严格蕴含 "`Done()` 已关闭"**，`select` 与 `Err()` 之间不会出现自相矛盾的观测。

`cancelCtx.Value` 的特殊之处：它对自己内部的私有 key 返回自己。

```go
var cancelCtxKey int // 用 &cancelCtxKey 做 key，地址唯一，外部拿不到

func (c *cancelCtx) Value(key any) any {
    if key == &cancelCtxKey { return c }
    return value(c.Context, key)
}
```

这是一个巧妙的复用：**用 Value 查找机制来实现"找到链上最近的可取消节点"**，不需要再维护第二套指针（见 2.6）。

### 2.5 propagateCancel：把子节点挂到父节点上的三条路径

所有可取消 context 的构造最终都调 `propagateCancel`：

```go
func (c *cancelCtx) propagateCancel(parent Context, child canceler) {
    c.Context = parent

    done := parent.Done()
    if done == nil {
        return // 父永不取消 → 什么都不用挂
    }

    select {
    case <-done:
        child.cancel(false, parent.Err(), Cause(parent)) // 父已取消 → 子立即取消
        return
    default:
    }

    if p, ok := parentCancelCtx(parent); ok {
        // 路径 A：父链上有真正的 *cancelCtx → 直接登记到它的 children map
        p.mu.Lock()
        if err := p.err.Load(); err != nil {
            child.cancel(false, err.(error), p.cause) // 刚好在这一瞬取消了
        } else {
            if p.children == nil { p.children = make(map[canceler]struct{}) }
            p.children[child] = struct{}{}
        }
        p.mu.Unlock()
        return
    }

    if a, ok := parent.(afterFuncer); ok {
        // 路径 B：父自己实现了 AfterFunc 方法 → 注册回调，并把 parent 换成 stopCtx
        c.mu.Lock()
        stop := a.AfterFunc(func() {
            child.cancel(false, parent.Err(), Cause(parent))
        })
        c.Context = stopCtx{Context: parent, stop: stop}
        c.mu.Unlock()
        return
    }

    // 路径 C：兜底，为自定义 Context 实现起一个守护 goroutine
    goroutines.Add(1)
    go func() {
        select {
        case <-parent.Done():
            child.cancel(false, parent.Err(), Cause(parent))
        case <-child.Done():
        }
    }()
}
```

关键结论：

- **绝大多数情况走路径 A**，代价是往父的 map 里插一个 entry，**不创建任何 goroutine**。网上"每个 WithCancel 都会起一个 goroutine"的说法是早期版本（Go 1.8 之前）的记忆，现在只有路径 C 才起。
- 路径 C 只在父是**用户自定义的 Context 实现**（既不是标准库类型、又不提供 `AfterFunc`）时触发。所以自己包装 Context 时若想省掉这个 goroutine，就实现 `AfterFunc(func()) func() bool`（见 3.10、3.16）。
- 包级 `var goroutines atomic.Int32` 就是给测试统计"到底起了多少个兜底 goroutine"用的。
- `parent.Done() == nil` 的短路径解释了为什么 `WithCancel(context.Background())` 极便宜：父永不取消，连挂载都不需要。

实测（在同一个父 ctx 下连续创建 100 个 `WithCancel` 子 context，观察 `runtime.NumGoroutine()` 增量）：

| 父 context 形态 | 走的路径 | goroutine 增量 |
|---|---|---|
| `context.Background()` | `done == nil` 短路 | +0 |
| `*cancelCtx` | A | +0 |
| `struct{ context.Context }{cancelCtx}`（内嵌未覆写 `Done`） | A | +0 |
| 覆写 `Done()` 返回自己 channel 的自定义类型 | C | **+100** |
| 上者再实现 `AfterFunc` 方法 | B | +0 |

### 2.6 parentCancelCtx：为什么要比对 done channel

```go
func parentCancelCtx(parent Context) (*cancelCtx, bool) {
    done := parent.Done()
    if done == closedchan || done == nil {
        return nil, false
    }
    p, ok := parent.Value(&cancelCtxKey).(*cancelCtx)
    if !ok {
        return nil, false
    }
    pdone, _ := p.done.Load().(chan struct{})
    if pdone != done {
        return nil, false // 父链上的 cancelCtx 被人用自定义实现包过一层
    }
    return p, true
}
```

第一步通过 `Value(&cancelCtxKey)` 找到链上最近的 `*cancelCtx`；第二步**校验 `parent.Done()` 是否就是那个 cancelCtx 的 done channel**。

为什么要校验？考虑有人这么包装：

```go
type myCtx struct {
    context.Context
    done chan struct{} // 自己的 done，比父的更早关闭
}
func (c *myCtx) Done() <-chan struct{} { return c.done }
```

此时 `Value(&cancelCtxKey)` 仍能穿透内嵌字段找到里层的 `*cancelCtx`，但如果直接挂到它的 children 上，就会**绕过 `myCtx` 自己的取消逻辑**——`myCtx.done` 关闭时子节点收不到信号。channel 比对不相等，于是退回路径 C 起 goroutine 守着 `myCtx.Done()`，语义正确性优先于性能。

### 2.7 cancel：一次关闭 + 递归下推

```go
func (c *cancelCtx) cancel(removeFromParent bool, err, cause error) {
    if err == nil { panic("context: internal error: missing cancel error") }
    if cause == nil { cause = err }

    c.mu.Lock()
    if c.err.Load() != nil {
        c.mu.Unlock()
        return // 已经取消过，幂等
    }
    c.err.Store(err)
    c.cause = cause

    d, _ := c.done.Load().(chan struct{})
    if d == nil {
        c.done.Store(closedchan) // 从没人调 Done()，直接塞一个全局已关闭 channel
    } else {
        close(d)
    }

    for child := range c.children {
        child.cancel(false, err, cause) // 持父锁的同时拿子锁
    }
    c.children = nil
    c.mu.Unlock()

    if removeFromParent {
        removeChild(c.Context, c)
    }
}
```

细节：

- **`closedchan` 是包级复用的已关闭 channel**（`init` 里 `close(closedchan)`）。如果一个 ctx 被取消时还没人调用过 `Done()`，就不分配 channel 了，直接把这个全局已关闭 channel 塞进去。后来的 `Done()` 调用拿到它，一读就返回。
- **递归取消是在持有父锁的情况下拿子锁**，方向严格是"父 → 子"，不存在反向加锁，所以不会死锁。但这也意味着**取消一棵大树时会持锁走完整棵子树**。
- `children = nil` 之后父不再持有子的引用，子树可以被 GC。
- `removeFromParent` 只在"取消的源头"是 true：外部 `cancel()` 调用传 true（要从父的 map 里摘掉自己），而级联下推给子节点时传 false（父马上就要把整个 map 置 nil，一个个 delete 是浪费）。
- 关闭 channel 会一次性唤醒所有等待者（chan.md 2.6），这正是"一次取消、所有 select 同时醒"的实现。

### 2.8 removeChild：为什么必须调 cancel

```go
func removeChild(parent Context, child canceler) {
    if s, ok := parent.(stopCtx); ok {
        s.stop() // 注销当初注册的 AfterFunc
        return
    }
    p, ok := parentCancelCtx(parent)
    if !ok { return }
    p.mu.Lock()
    if p.children != nil { delete(p.children, child) }
    p.mu.Unlock()
}
```

这就是"忘记 cancel 会泄漏"的机制层解释：**子节点被登记在父的 `children` map 里，只有 `cancel`（或父自己被取消）才会把它删掉**。父 ctx 活多久，这个 entry 和它挂着的整条子链就活多久。在一个长生命周期的父 ctx 下循环创建子 ctx 而不 cancel，`children` map 会单调增长——内存泄漏（见 3.1）。

### 2.9 timerCtx：定时器 + deadline 收敛

```go
type timerCtx struct {
    cancelCtx
    timer    *time.Timer // 受 cancelCtx.mu 保护
    deadline time.Time
}
```

`WithDeadlineCause` 的完整逻辑：

```go
if cur, ok := parent.Deadline(); ok && cur.Before(d) {
    return WithCancel(parent) // 父的 deadline 更早 → 退化成纯 cancelCtx，连 timer 都不建
}
c := &timerCtx{deadline: d}
c.cancelCtx.propagateCancel(parent, c)
dur := time.Until(d)
if dur <= 0 {
    c.cancel(true, DeadlineExceeded, cause) // 已经过期，立即取消
    return c, func() { c.cancel(false, Canceled, nil) }
}
c.mu.Lock()
defer c.mu.Unlock()
if c.err.Load() == nil {
    c.timer = time.AfterFunc(dur, func() { c.cancel(true, DeadlineExceeded, cause) })
}
return c, func() { c.cancel(true, Canceled, nil) }
```

- `Deadline()` 直接返回自己的字段，`Done()`/`Err()`/`Value()` 全部由内嵌的 `cancelCtx` 提供。
- **`Deadline()` 返回的是"申请的时间"，不是"实际取消的时间"**：一个被提前手动 cancel 的 timerCtx，`Deadline()` 依然返回原来那个未来时刻。
- `timerCtx.cancel` 覆写了父方法：先调 `c.cancelCtx.cancel(false, ...)` 走完常规取消，再 `removeChild`，最后 `timer.Stop(); timer = nil`。**不调 cancel 就不会 Stop，timer 会一直挂在运行时定时器堆里直到 deadline 触发**，而它的闭包引用着整个 ctx，所以这条链在超时前无法回收。
- 创建 timer 前再检查一次 `c.err.Load() == nil`，是因为 `propagateCancel` 可能已经因为父已取消而把它取消了，这时不必再建 timer。

### 2.10 withoutCancelCtx：只保留 Value 链

```go
type withoutCancelCtx struct{ c Context } // 注意：字段名是 c，不是内嵌

func (withoutCancelCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (withoutCancelCtx) Done() <-chan struct{}       { return nil }
func (withoutCancelCtx) Err() error                  { return nil }
func (c withoutCancelCtx) Value(key any) any         { return value(c, key) }
```

它**不内嵌** `Context` 而是用具名字段 `c`，这样就不会意外继承父的任何方法，三个取消相关方法全部返回"空"，只有 `Value` 显式往上走。

`value()` 里对它有一条特判：

```go
case withoutCancelCtx:
    if key == &cancelCtxKey { return nil } // 截断 cancelCtx 查找
    c = ctx.c
```

这保证了 `Cause(WithoutCancel(ctx)) == nil`：找不到 cancelCtx，`Cause` 就只能返回 `ctx.Err()`，而它的 `Err()` 是 nil。同时也保证在它之下再 `WithCancel`，不会被错误地挂到上游那个 cancelCtx 上。

### 2.11 afterFuncCtx 与 stopCtx

```go
type afterFuncCtx struct {
    cancelCtx
    once sync.Once // 二选一：要么启动 f，要么阻止 f 启动
    f    func()
}

func (a *afterFuncCtx) cancel(removeFromParent bool, err, cause error) {
    a.cancelCtx.cancel(false, err, cause)
    if removeFromParent { removeChild(a.Context, a) }
    a.once.Do(func() { go a.f() })
}
```

`AfterFunc(ctx, f)` 就是造一个 `afterFuncCtx` 挂到 ctx 上（走 2.5 的三条路径），取消时在新 goroutine 里跑 f。返回的 `stop` 也去抢同一个 `once`：抢到就说明 f 还没启动，随即调 `a.cancel(true, Canceled, nil)` 把自己从父节点摘掉。`sync.Once` 保证 f 与 stop **恰好只有一个生效**。

`stopCtx` 是路径 B 留下的痕迹：

```go
type stopCtx struct {
    Context
    stop func() bool
}
```

当子节点通过 `parent.AfterFunc(...)` 挂上去时，子的 `c.Context` 被替换成 `stopCtx{parent, stop}`。这样后续 `removeChild` 看到 parent 是 `stopCtx`，就知道该调 `s.stop()` 注销回调，而不是去 map 里 delete。**Value 查找不受影响**——`stopCtx` 内嵌 `Context`，`Value` 自动转发。

### 2.12 内存模型保证

- `cancel` 里 `close(done)` 之前的所有写（`err`、`cause`）都 happens-before 任何 `<-ctx.Done()` 的返回（channel 关闭的 happens-before 保证，chan.md 2.9）。所以 `<-ctx.Done()` 之后读 `ctx.Err()` / `Cause(ctx)` 永远能看到确定的非 nil 值，不需要额外同步。
- 反过来，`Err()` 内部的 `<-c.Done()` 补齐了另一个方向：`Err()` 返回非 nil ⇒ done 已关闭（见 2.4）。
- `Value` 链是**只读不可变**的：所有节点在构造后不再被修改，因此并发读天然安全，不需要锁。这是 context 敢把接口文档写成"可被多 goroutine 并发调用"的根本原因。
- `WithValue` 的 kv 本身不做保护：如果 val 是可变对象（如 `*bytes.Buffer`、map），并发读写它仍然是数据竞争。context 只保证**链结构**安全，不保证**值内容**安全（见 3.11）。

### 2.13 关键源码索引

全部在 `$GOROOT/src/context/context.go`（go1.26.3，806 行）：

| 位置 | 内容 |
|---|---|
| `Context` 接口 | 四方法定义与详尽注释 |
| `Canceled` / `deadlineExceededError` | 两个错误值，后者带 `Timeout()`/`Temporary()` |
| `emptyCtx` / `backgroundCtx` / `todoCtx` | 根节点 |
| `WithCancel` / `WithCancelCause` / `withCancel` | 取消构造 |
| `Cause` | 沿链找 `cancelCtx.cause` |
| `AfterFunc` / `afterFuncCtx` / `stopCtx` | 取消回调 |
| `parentCancelCtx` / `removeChild` | 挂载点定位与摘除 |
| `canceler` 接口 / `closedchan` | 可取消抽象、复用的已关闭 channel |
| `cancelCtx` 及其 `Value`/`Done`/`Err`/`propagateCancel`/`cancel` | 取消核心 |
| `WithoutCancel` / `withoutCancelCtx` | 切断取消 |
| `WithDeadline` / `WithDeadlineCause` / `timerCtx` | 超时 |
| `WithValue` / `valueCtx` / `value` | 值链与查找循环 |

## 三、常见陷阱

### 3.1 忘记调用 cancel：两种泄漏

```go
// 错误
func handler(ctx context.Context) {
    ctx, _ = context.WithTimeout(ctx, time.Second) // cancel 被丢掉
    doWork(ctx)
}

// 正确
func handler(ctx context.Context) {
    ctx, cancel := context.WithTimeout(ctx, time.Second)
    defer cancel()
    doWork(ctx)
}
```

不调 cancel 的后果有两层：

1. **子节点一直挂在父的 `children` map 里**（2.8），父活着它就活着。在长生命周期父 ctx 下高频创建子 ctx，map 单调增长。
2. **timerCtx 的定时器不会 Stop**（2.9），直到 deadline 到达才释放，闭包持有整条链。

`go vet` 的 `lostcancel` 检查专门抓这个，把它接进 CI：

```bash
go vet ./...   # "the cancel function returned by context.WithTimeout should be called..."
```

即使确信"函数返回前一定会超时"，也照样 `defer cancel()`——它是幂等的，多调一次没有代价。

### 3.2 WithValue 用内置类型做 key

```go
// 错误：不同包都用 "user" 这个 string，后写的覆盖先写的，且互相看不见
ctx = context.WithValue(ctx, "user", u)

// 正确：非导出类型，跨包不可能冲突
type userKey struct{}
ctx = context.WithValue(ctx, userKey{}, u)
```

`struct{}` 做 key 还有个附带好处：零大小值装进 `any` 不需要堆分配（interface.md 2.3）。用 `type key int; var userKey key` 也可以，但要注意同一个包里多个 key 必须取不同的值（`iota`），否则互相覆盖。

另外 key 必须可比较，`WithValue(ctx, []string{"a"}, v)` 会 panic：`key is not comparable`。

### 3.3 把 context 存进 struct

```go
// 错误
type Service struct {
    ctx context.Context // 生命周期错配：Service 是长期的，ctx 是单次请求的
}

// 正确：每个方法显式接收
type Service struct{ db *sql.DB }
func (s *Service) Get(ctx context.Context, id int) (*User, error)
```

例外：**代表一次运行、且本身就是一次性对象**的结构体可以存（如 `http.Request` 内部就存了 ctx，通过 `r.Context()`/`r.WithContext()` 暴露）。但业务里的 service/client/repository 一律走参数。官方专门写过一篇 [context-and-structs](https://go.dev/blog/context-and-structs) 讨论这件事。

### 3.4 nil parent 直接 panic

```go
context.WithValue(nil, k, v)          // panic: cannot create context from nil parent
context.WithCancel(nil)               // 同上
context.WithTimeout(nil, time.Second) // 同上
```

`WithValue`/`withCancel`/`WithDeadlineCause`/`WithoutCancel` 入口都有 `if parent == nil { panic(...) }`。不确定用什么 ctx 时用 `context.TODO()`，不要传 nil。

反过来，如果自己写的函数收到 nil ctx，取决于是否要防御性处理——标准库的做法是不检查、直接让它在第一次方法调用时 nil 接口 panic。

### 3.5 判断错误用 == 而不是 errors.Is

```go
// 脆弱：错误一旦被 %w 包装过就判不出来
if err == context.DeadlineExceeded { ... }

// 健壮
if errors.Is(err, context.DeadlineExceeded) { ... }
```

`ctx.Err()` 本身返回的一定是裸的 `Canceled`/`DeadlineExceeded`，`==` 没问题；但业务层拿到的 err 常常已经被多层 `fmt.Errorf("...: %w", err)` 包过（interface.md 3.9）。统一用 `errors.Is` 更稳。

判"是不是超时"还有第三种写法，因为 `DeadlineExceeded` 实现了 `Timeout() bool`：

```go
var ne net.Error
if errors.As(err, &ne) && ne.Timeout() { ... } // context 超时和网络超时都命中
```

### 3.6 Done() 可能是 nil

```go
ctx := context.Background()
<-ctx.Done() // 永久阻塞！fatal error: all goroutines are asleep - deadlock

select {
case <-ctx.Done(): // 在多 case 的 select 里等价于该分支不存在，这是正常语义
case v := <-ch:
}
```

`Background()`、`TODO()`、`WithoutCancel()` 的 `Done()` 都返回 nil。只在多 case select 里用它，不要单独裸读。写库函数时如果必须等取消，先判断：

```go
if done := ctx.Done(); done != nil {
    <-done
}
```

### 3.7 select 中 Done 与业务 case 同时就绪时是随机的

```go
// ctx 已取消，且 ch 里有数据 → 有 50% 概率走 ch 分支
select {
case <-ctx.Done():
    return ctx.Err()
case v := <-ch:
    process(v) // 明明已经该停了，还是处理了一条
}
```

select 在多个就绪 case 间随机选（chan.md 2.8）。如果"取消必须严格优先"，加一次前置检查：

```go
for {
    if err := ctx.Err(); err != nil { // 先检查
        return err
    }
    select {
    case <-ctx.Done():
        return ctx.Err()
    case v := <-ch:
        process(v)
    }
}
```

多数业务里"多处理一条"无害，不必都这么写；但涉及计费、下单、发消息这类有副作用的操作时要注意。

### 3.8 子 context 的 deadline 无法比父更长

```go
parent, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

child, cancel2 := context.WithTimeout(parent, time.Hour) // 想放宽到 1 小时
defer cancel2()

d, _ := child.Deadline() // 仍然是 1 秒后（实际上 child 退化成了 WithCancel(parent)）
```

`WithDeadline` 明确检查 `cur.Before(d)` 并退化成 `WithCancel`（2.9）。要真的放宽，必须先 `WithoutCancel` 断开：

```go
child, cancel2 := context.WithTimeout(context.WithoutCancel(parent), time.Hour)
```

但这么做也就同时放弃了父的取消传播，要清楚自己在做什么。

### 3.9 cancel 不等待，只是通知

```go
ctx, cancel := context.WithCancel(context.Background())
go worker(ctx)
cancel()
// 此刻 worker 可能还在跑！cancel 只关闭了 channel
```

`CancelFunc` 文档第一句就是 "A CancelFunc does not wait for the work to stop."。要等 goroutine 真的退出，配 `sync.WaitGroup` 或 `errgroup`：

```go
var wg sync.WaitGroup
wg.Add(1)
go func() { defer wg.Done(); worker(ctx) }()

cancel()
wg.Wait() // 现在才是真的停了
```

`AfterFunc` 的 `stop()` 同理不等待 f 完成。

### 3.10 自定义 Context 包装会引入兜底 goroutine

```go
type myCtx struct{ context.Context } // 只想加个方法

ctx, cancel := context.WithCancel(myCtx{parent}) // 走路径 A 还是 C？
```

这个例子里 `myCtx` 内嵌了 `Context`，`Done()` 被自动提升、返回的就是父的 done channel，所以 `parentCancelCtx` 的 channel 比对通过，仍走高效路径 A。

但只要**覆写了 `Done()` 并返回自己的 channel**，就一定退到路径 C，每个子 context 多一个 goroutine（2.5、2.6）。这时应该实现 `AfterFunc` 走路径 B：

```go
func (c *myCtx) AfterFunc(f func()) func() bool {
    return context.AfterFunc(c.Context, f) // 或自己的注册/注销逻辑
}
```

### 3.11 用 context 传"隐式依赖"

```go
// 反模式：把必需依赖藏进 ctx
func Handle(ctx context.Context) {
    db := ctx.Value(dbKey{}).(*sql.DB) // 编译期无法检查、断言可能 panic、可测性差
    logger := ctx.Value(logKey{}).(*slog.Logger)
}
```

这套写法把编译期错误变成了运行时 panic，还让函数签名彻底失去表达力。判断标准：**这个东西是"这次请求的属性"，还是"这个组件的依赖"？** 前者（trace id、用户身份、语言、超时预算）放 ctx；后者（DB、logger、配置、client）走构造函数或参数。

另外 context 只保证链结构并发安全，放进去的值如果自身可变（map、buffer），并发读写它照样是竞态（2.12）。

### 3.12 循环里 defer cancel

```go
// 错误：所有 cancel 堆到函数返回才执行，循环期间 context 全都活着
for _, item := range items {
    ctx, cancel := context.WithTimeout(ctx, time.Second)
    defer cancel()
    process(ctx, item)
}

// 正确：包一层函数，或显式调用
for _, item := range items {
    func() {
        ctx, cancel := context.WithTimeout(ctx, time.Second)
        defer cancel()
        process(ctx, item)
    }()
}
```

注意上面错误版本还有第二个 bug：`ctx, cancel :=` 里的 `ctx` 遮蔽了外层变量，导致每次迭代都在**上一次的 ctx** 基础上再套一层超时，deadline 会越来越紧（因为 deadline 只收敛，3.8），链也越来越长。循环里派生 ctx 时用不同的变量名：

```go
for _, item := range items {
    itemCtx, cancel := context.WithTimeout(ctx, time.Second)
    process(itemCtx, item)
    cancel()
}
```

### 3.13 handler 返回后继续用 r.Context()

```go
func handler(w http.ResponseWriter, r *http.Request) {
    go func() {
        // handler 一返回，r.Context() 就被取消了 → 这个后台任务立刻失败
        _ = auditLog(r.Context(), event)
    }()
    w.WriteHeader(http.StatusOK)
}
```

`net/http` 在 `ServeHTTP` 返回时（以及客户端断连时）取消请求 ctx。要做后台工作用 `WithoutCancel` 保留 value 链、再自己设超时（1.7）。Go 1.21 之前只能手工重建一个 `context.Background()` 并把需要的值一个个拷过去。

### 3.14 Value 链过长的性能问题

```go
for i := 0; i < 100; i++ {
    ctx = context.WithValue(ctx, keys[i], vals[i]) // 100 层链表
}
ctx.Value(keys[99]) // 命中第一层，很快
ctx.Value(keys[0])  // 走完 100 层，每层一次接口比较
```

`Value` 是线性查找（2.3），层数多了每次取值都是 O(n)，而且是在**热路径**上（每个中间件、每次日志都可能取）。实践做法：把一次请求需要的所有元数据装进**一个 struct**，只挂一层：

```go
type reqMeta struct {
    TraceID string
    UserID  int64
    Locale  string
}
type metaKey struct{}
ctx = context.WithValue(ctx, metaKey{}, &reqMeta{...})
```

注意如果传指针并在下游修改字段，并发下需要自己加锁（2.12）。

### 3.15 CancelCauseFunc 的两个易错点

```go
ctx, cancel := context.WithCancelCause(parent)
cancel(nil)          // 等价于 cancel(context.Canceled)，不是"不设原因"
context.Cause(ctx)   // context.Canceled

// 已取消后再设 cause 无效：第一个取消者胜出
cancel(errA)
cancel(errB)
context.Cause(ctx)   // errA
```

还有一个容易忽略的：`WithTimeoutCause(parent, d, cause)` 返回的是普通 `CancelFunc`，**手动调它不会写入 cause**，只有超时触发时才写（源码里 `func() { c.cancel(true, Canceled, nil) }`）。

### 3.16 用了自定义 Context 后 Cause 拿不到原因

```go
type myCtx struct {
    context.Context
    done chan struct{}
}
func (c *myCtx) Done() <-chan struct{} { return c.done }
func (c *myCtx) Err() error            { return errors.New("my cancel") }

context.Cause(&myCtx{...}) // 返回 Err()，不是上游的 cause
```

`Cause` 依赖 `Value(&cancelCtxKey)` 找到 `*cancelCtx` 再读它的 `cause` 字段；自定义实现如果自己造 done/err，`Cause` 只能退回返回 `c.Err()`（源码注释写着 "so c must have been canceled in some custom context implementation"）。想让包装类型支持 cause，就不要覆写 `Done`/`Err`，让它们转发给内部的 cancelCtx。

### 3.17 不要用 ctx.Done() 做业务事件通知

```go
// 反模式：把 context 当通用事件总线
ctx, cancel := context.WithCancel(context.Background())
go func() { waitForConfigChange(); cancel() }() // "配置变了"也用 cancel 通知
```

`Done()` 是**一次性**的、语义固定为"该停下来了"。用它表达"配置更新""数据就绪"这类可重复事件，会让下游无法区分"停止"和"有新事件"，也无法重置。需要广播型通知就用自己的 channel + close，或 `sync.Cond`、`atomic.Pointer` 之类（chan.md 1.8）。

## 四、常见面试题

**1. context 到底解决什么问题？为什么不用全局变量或者传一个 chan？**
解决三件事：跨 API 边界传递取消信号、传递截止时间、传递请求域数据。相比裸传 `chan struct{}`，context 提供了**树形级联取消**（父取消自动传播到任意深度的子）、**deadline 自动收敛**、**错误原因区分**（`Canceled` vs `DeadlineExceeded` vs `Cause`）和统一的接口约定，使得任意第三方库都能接进同一套取消链路。全局变量做不到"每次请求一份、随请求生命周期结束"（见 1.1、2.1）。

**2. context 的内部结构是什么？为什么说"查值是链表、取消是树"？**
每个 `WithXxx` 都新建一个节点、把 parent 存进节点内嵌的 `Context` 字段，从不修改 parent。`Value` 从当前节点沿 parent 指针向上线性查找，所以向上看是**单向链表**；`cancelCtx` 用 `children map[canceler]struct{}` 记住所有可取消的子节点，取消时递归向下推，所以向下看是**树**。数据向上查、信号向下推（见 2.1、2.3、2.7）。

**3. `WithCancel` 会不会为每个 context 起一个 goroutine？**
不会。`propagateCancel` 有三条路径：父链上能找到真正的 `*cancelCtx` 就直接登记到它的 `children` map（**路径 A，绝大多数情况，零 goroutine**）；父实现了 `AfterFunc` 方法就注册回调（路径 B）；只有父是**自定义 Context 实现**且不提供 `AfterFunc` 时，才起一个守护 goroutine 兜底（路径 C）。"每个 WithCancel 一个 goroutine"是 Go 1.8 之前的老印象。另外父的 `Done()` 返回 nil（如 `Background()`）时连挂载都直接跳过（见 2.5）。

**4. `parentCancelCtx` 为什么找到 `*cancelCtx` 之后还要比对 `Done()` channel？**
防止绕过用户的自定义取消逻辑。有人可能内嵌一个标准 context 但覆写 `Done()` 返回自己的 channel，此时 `Value(&cancelCtxKey)` 仍能穿透找到里层的 `*cancelCtx`，如果直接挂到它的 children 上，那个包装层自己的取消就永远传不到子节点。channel 比对不相等时退回路径 C 起 goroutine 守着，牺牲一点性能换语义正确（见 2.6、3.10）。

**5. `cancelCtxKey` 是什么？为什么用它的地址做 key？**
它是包级 `var cancelCtxKey int`，`cancelCtx.Value` 对 `&cancelCtxKey` 这个 key 返回自己。这是**复用 Value 查找机制来实现"定位链上最近的可取消节点"**，省掉再维护一套父指针。用变量地址而不是值，是因为地址全局唯一且外部包拿不到，不可能被误撞（见 2.4、2.6）。

**6. `done` channel 为什么懒创建？`closedchan` 是干什么的？**
很多 context 只是链路中转，从头到尾没人 select 它的 `Done()`，懒创建能省掉一次 hchan 分配。对称地，`cancel` 时如果发现 `done` 还没创建，就直接把包级已关闭的 `closedchan` 存进去，也不分配——之后的 `Done()` 调用拿到它一读就返回。两个优化配合起来，"创建了但从未监听、最后被取消"的 context 全程零 channel 分配（见 2.4、2.7）。

**7. `cancelCtx.Err()` 里为什么有一行 `<-c.Done()`？**
修正可见性窗口。`cancel` 里是先 `c.err.Store(err)` 再 `close(done)`，中间存在一个瞬间：`Err()` 已能返回非 nil，但 `Done()` 还没关闭。加上这行阻塞等待后，"`Err()` 返回非 nil" 就严格蕴含 "`Done()` 已关闭"，避免出现 `Err() != nil` 但 select 的 Done 分支不就绪这种自相矛盾的观测。同时 `Err()` 用 `atomic.Value` 而不是加锁读，源码注释说明原子读比加锁快约 5 倍，在紧凑循环里有意义（见 2.4）。

**8. 递归取消整棵子树会不会死锁？**
不会。`cancel` 在持有父锁的情况下调 `child.cancel`，加锁顺序严格是"父 → 子"，不存在反向获取，所以没有环。代价是取消一棵大树时会持锁走完整棵子树。另外级联时传 `removeFromParent=false`，因为父马上要把整个 `children` map 置 nil，逐个 delete 是浪费（见 2.7）。

**9. 为什么必须调用 cancel？不调会怎样？**
两层泄漏。一是子节点被登记在父的 `children` map 里，只有 `cancel`（或父自己被取消）才会 `removeChild` 把它删掉，父活多久这条子链就活多久，高频派生会让 map 单调增长。二是 `timerCtx` 的 `time.AfterFunc` 定时器不会 `Stop()`，一直挂在运行时定时器堆里直到 deadline 触发，闭包引用着整个 ctx。`go vet` 的 `lostcancel` 专门检查这个。cancel 是幂等的，多调无害（见 2.8、2.9、3.1）。

**10. `ctx.Err()` 和 `context.Cause(ctx)` 有什么区别？**
`Err()` 只有 `Canceled`/`DeadlineExceeded` 两种取值，是稳定的机器可判定语义；`Cause`（Go 1.20）沿链找最近 `cancelCtx` 的 `cause` 字段，返回业务给出的**具体原因**，没设置时退化为等于 `Err()`，未取消时返回 nil。规则是"第一个取消者胜出"：父先被 cause1 取消则子的 Cause 也是 cause1；子先被 cause2 取消则父是 cause1、子是 cause2。另外 `cancel(nil)` 等价于把 cause 设成 `Canceled`，而 `WithTimeoutCause` 返回的 CancelFunc 手动调时不写 cause（见 1.6、3.15）。

**11. `WithoutCancel` 解决什么问题？它是怎么实现的？**
解决"请求已结束但还要用请求里的元数据做后台工作"。`withoutCancelCtx` 用具名字段 `c Context` 而非内嵌，避免继承父的任何方法，`Deadline`/`Done`/`Err` 全返回空值，只有 `Value` 显式往上走。`value()` 里对它有特判：查 `&cancelCtxKey` 时直接返回 nil，从而保证 `Cause` 返回 nil，也保证在它之下再 `WithCancel` 不会被错误挂到上游的 cancelCtx（见 1.7、2.10）。注意它同时切断了 deadline，后台任务需自己重设超时。

**12. `AfterFunc` 比自己起个 goroutine 守 `<-ctx.Done()` 好在哪？**
在"ctx 从未取消"这条常见路径上不创建 goroutine——它只是把一个 `afterFuncCtx` 挂进父的 children map。内部用 `sync.Once` 保证 f 的执行与 `stop()` 的取消**恰好只有一个生效**。当子 context 是通过父的 `AfterFunc` 挂上去时，子的 parent 会被替换成 `stopCtx{parent, stop}`，这样 `removeChild` 看到 `stopCtx` 就知道该调 `stop()` 注销而不是去 map 里 delete（见 1.8、2.11）。

**13. 子 context 能设置比父更长的超时吗？**
不能。`WithDeadline` 里 `if cur, ok := parent.Deadline(); ok && cur.Before(d)` 直接退化成 `WithCancel(parent)`，连 timer 都不创建——deadline 只收敛不放宽。要真的放宽必须先 `context.WithoutCancel(parent)` 断开，但同时也放弃了父的取消传播。另外 `timerCtx.Deadline()` 返回的是"申请的时刻"，被提前手动 cancel 也不变（见 1.4、2.9、3.8）。

**14. `select { case <-ctx.Done(): ...; case v := <-ch: ... }`，ctx 已取消且 ch 有数据时走哪个分支？**
随机。select 在多个就绪 case 间伪随机选择。如果要求取消严格优先，得在 select 之前加一次 `if err := ctx.Err(); err != nil { return err }` 的前置检查。涉及有副作用的操作（计费、下单）时必须这么写（见 1.9、3.7）。

**15. `context.Background().Done()` 返回什么？直接 `<-` 它会怎样？**
返回 **nil channel**。裸读会永久阻塞，主 goroutine 上会触发 `fatal error: all goroutines are asleep - deadlock`。用 nil 表示"永不取消"是刻意设计：在多 case select 里 nil channel 等价于该分支不存在，正好是想要的语义，同时省掉一个 channel 对象，还让 `propagateCancel` 得到一条 `done == nil` 直接返回的极快短路径（见 1.9、2.2、3.6）。`WithoutCancel` 的结果同理。

**16. `WithValue` 的 key 为什么要用非导出类型？为什么很多人写 `struct{}`？**
非导出类型保证跨包不可能撞 key（对方拿不到这个类型，构造不出相等的值）。用 `struct{}` 而不是 `int`：零大小值装进 `any` 不需要堆分配。key 还必须**可比较**，`WithValue` 入口用 `reflectlite.TypeOf(key).Comparable()` 挡住 slice/map/func，否则后面 `c.key == key` 会 panic（见 1.5、2.3、3.2）。

**17. 为什么说 context 不该用来传 logger、DB 连接？**
判断标准是"这是本次请求的属性，还是这个组件的依赖"。放进 ctx 意味着编译期检查失效、取值时要类型断言（可能 panic）、函数签名失去表达力、测试要构造完整 ctx。trace id、用户身份、语言标签这类随请求流动并可能跨进程传递的元数据适合放；DB、logger、配置、client 走构造函数注入。另外 context 只保证**链结构**并发安全（所有节点构造后不可变），放进去的值如果自身可变，并发读写它依然是数据竞争（见 2.12、3.11）。

**18. 为什么 context 里的 Value 链不用加锁就能并发读？**
因为所有节点在构造完成后**再也不被修改**：`valueCtx` 的 key/val、`cancelCtx` 的 `Context` 父指针、`timerCtx` 的 deadline 全是只读的。会变的只有 `cancelCtx` 的 `done`/`err`/`cause`/`children`，它们分别用 `atomic.Value` 和 `mu` 保护。不可变结构 + 局部同步，这是接口文档敢写"可被多 goroutine 并发调用"的根本原因（见 2.12）。

**19. `cancel()` 返回之后，worker goroutine 一定已经退出了吗？**
不一定。文档明确写 "A CancelFunc does not wait for the work to stop."，cancel 只是关闭 channel 发出信号。要确认真正退出得配 `sync.WaitGroup` 或 `errgroup.Wait()`。`AfterFunc` 的 `stop()` 同样不等待 f 完成。而且 `Done()` 的关闭本身也允许异步发生在 cancel 返回之后（接口注释里写了 "The close of the Done channel may happen asynchronously, after the cancel function returns."）（见 3.9）。

**20. `DeadlineExceeded` 为什么不是简单的 `errors.New`？**
它是 `deadlineExceededError{}`，额外实现了 `Timeout() bool` 和 `Temporary() bool`（都返回 true），因此满足 `net.Error` 接口，能被网络库那一套"是不是超时、能不能重试"的判断统一识别。这也让 `errors.As(err, &netErr) && netErr.Timeout()` 一次覆盖 context 超时和网络超时两种情况（见 1.1、3.5）。

**21. 判断 `err` 是不是 context 取消，用 `==` 还是 `errors.Is`？**
`ctx.Err()` 直接返回的一定是裸值，`==` 能用；但业务代码拿到的 err 通常已经被多层 `fmt.Errorf("...: %w", err)` 包装，动态类型变成 `*fmt.wrapError`，`==` 判不出来。统一用 `errors.Is(err, context.Canceled)` / `errors.Is(err, context.DeadlineExceeded)`（见 3.5、interface.md 3.9）。

**22. HTTP handler 里 `go func(){ use(r.Context()) }()` 有什么问题？**
`net/http` 在 `ServeHTTP` 返回时（以及客户端断连时）取消请求 ctx，所以 handler 一返回这个后台任务立刻就收到取消。正确做法是 `context.WithoutCancel(r.Context())` 保留 value 链、丢掉取消信号，再自己套一个 `WithTimeout` 防止无限期运行（见 1.7、3.13）。

**23. 在 for 循环里 `ctx, cancel := context.WithTimeout(ctx, d); defer cancel()` 有几个 bug？**
两个。一是 `defer` 全部堆积到函数返回才执行，循环期间所有 context 都活着；二是 `ctx, cancel :=` 遮蔽了外层变量，每次迭代都在**上一轮的 ctx** 上再套一层，链越来越长、deadline 越来越紧（因为只收敛不放宽）。正确写法是用不同变量名，并在每次迭代结束时显式 `cancel()`，或用一个立即执行的闭包把 defer 的作用域缩到单次迭代（见 3.12）。
