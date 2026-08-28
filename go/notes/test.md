# 测试

> 环境：`go version go1.26.3`。配套代码：`notes/tst/`（可运行的完整示例，包含单元测试、benchmark、fuzz、synctest、TestMain、httptest 六个文件）。
>
> 版本演进：
> - **1.7**：子测试 `t.Run`。
> - **1.14**：`t.Cleanup`。
> - **1.17**：`t.Setenv`。
> - **1.18**：**模糊测试**（`f.Fuzz`）。
> - **1.20**：`-coverpkg` 支持集成测试覆盖率（`go build -cover`）。
> - **1.22**：循环变量每轮新建，`tc := tc` 那行样板可以删了。
> - **1.24**：**`b.Loop()`** 取代 `for range b.N`；`t.Chdir`；`t.Context`；`testing/synctest`（实验性）。
> - **1.25**：`testing/synctest` 转正（`synctest.Test`）。
> - **1.26**：`t.Attr`（结构化标签）、`t.ArtifactDir`（测试产物目录）。

## 一、基本形态

### 1.1 表驱动 + 子测试

这是 Go 测试的**默认形态**，标准库里到处都是：

```go
func TestParseRange(t *testing.T) {
    tests := []struct {
        name    string
        in      string
        wantLo  int
        wantHi  int
        wantErr error       // 用 errors.Is 比较，不比字符串
    }{
        {name: "单个数字", in: "5", wantLo: 5, wantHi: 5},
        {name: "lo 大于 hi", in: "9-2", wantErr: ErrBadRange},
    }

    for _, tc := range tests {           // 1.22+ 不需要 tc := tc
        t.Run(tc.name, func(t *testing.T) {
            lo, hi, err := ParseRange(tc.in)
            if tc.wantErr != nil {
                if !errors.Is(err, tc.wantErr) {
                    t.Fatalf("ParseRange(%q) err = %v, want errors.Is(..., %v)", tc.in, err, tc.wantErr)
                }
                return
            }
            ...
        })
    }
}
```

四个约定：

- **子测试名用中文/描述性文字**都行，空格会被替换成 `_`，`-run` 时用 `TestParseRange/正常区间` 定位；
- **错误信息写成 `got, want` 格式**并带上输入：`ParseRange(%q) = (%d, %d), want (%d, %d)`；
- **`Fatalf` 停止当前子测试，`Errorf` 继续**——前置条件不满足用 Fatal，结果不符用 Error（这样一次能看到多个失败点）；
- **错误用 `errors.Is` 比较**，不比 `err.Error()` 字符串（见 error.md 3.2）。

### 1.2 `t.Parallel`

```go
for _, in := range []string{"1", "2-3", "4-9"} {
    t.Run(in, func(t *testing.T) {
        t.Parallel()
        ...
    })
}
```

机制：调用 `t.Parallel()` 的子测试会**立即暂停**，等父测试的函数体跑完之后，所有并行子测试一起放行。

三个后果：

1. **父测试里写在 `t.Run` 之后的代码，会在并行子测试开始之前执行**——想在之后做事，得放进另一个 `t.Run` 或 `t.Cleanup`；
2. 并行度受 `-parallel`（默认 `GOMAXPROCS`）限制；
3. **`t.Setenv`/`t.Chdir` 和 `t.Parallel` 互斥**（前者改的是进程全局状态），同时用会 panic。

包之间默认就是并行的（`-p`，默认 `GOMAXPROCS`），所以**同一个包里的测试串行、不同包并行**是默认行为。

### 1.3 `t.Cleanup` 比 `defer` 好在哪

```go
func newTestStore(t *testing.T) *Store {
    t.Helper()
    s := NewStore()
    t.Cleanup(func() { s.Close() })    // ← 关键
    return s
}

func TestCleanup(t *testing.T) {
    s := newTestStore(t)     // 辅助函数里注册的清理也生效
    ...
}
```

`defer` 只能在**当前函数**返回时执行，所以辅助函数（`newTestStore`）里的 `defer` 会在辅助函数返回时就跑掉。`t.Cleanup` 注册到 `t` 上，**在这个测试（含子测试）结束时**才执行，LIFO 顺序。

配套的 `t.Helper()` 让失败行号指向**调用者**而不是辅助函数内部——写 assert 辅助函数必加。

### 1.4 环境隔离

```go
t.Setenv("KEY", "v")   // 1.17+，结束自动还原（与 Parallel 互斥）
dir := t.TempDir()      // 每个测试独立目录，结束自动删除
t.Chdir(dir)            // 1.24+，结束自动还原（与 Parallel 互斥）
ctx := t.Context()      // 1.24+，测试结束/超时时自动 cancel
```

`t.Context()` 的价值：以前要写 `ctx, cancel := context.WithCancel(...); defer cancel()`，现在一行，而且它在 `Cleanup` **之前**被取消，正好符合"先停活儿再清理"的顺序。

`t.Deadline()` 能拿到 `-timeout` 换算出的截止时间，长测试可以据此决定跑多少轮。

### 1.5 1.26 新增：`t.Attr` 与 `t.ArtifactDir`

```go
t.Attr("owner", "platform-team")     // 结构化标签，出现在 test2json 输出里
t.Attr("ticket", "GO-1234")

dir := t.ArtifactDir()               // 测试产物目录（截图、pprof、失败输入）
os.WriteFile(filepath.Join(dir, "report.txt"), data, 0o600)
```

输出形式：

```text
=== ATTR  TestAttrAndArtifacts owner platform-team
=== ATTR  TestAttrAndArtifacts ticket GO-1234
    calc_test.go:184: ArtifactDir = /var/.../TestAttrAndArtifacts3423913957/001
```

`ArtifactDir` 在**不带 `-artifacts`** 时是临时目录（结束删除），带上就落到输出目录留存。这解决了以前"测试失败时想留下现场文件，只能自己 `os.MkdirTemp` 并打日志"的问题。

## 二、Benchmark

### 2.1 `b.Loop`（1.24+）取代 `for range b.N`

```go
// ✓ 新写法
func BenchmarkSumLoop(b *testing.B) {
    lo, hi := 1, 1000            // setup 不计时
    for b.Loop() {
        intSink = Sum(lo, hi)
    }
}

// 老写法：等价但有两个包袱
func BenchmarkSumOldStyle(b *testing.B) {
    lo, hi := 1, 1000
    b.ResetTimer()               // ① 要手动重置计时器
    for range b.N {
        intSink = Sum(lo, hi)    // ② 要自己防止被优化掉
    }
}
```

`b.Loop` 的三个改进（文档明确写了）：

1. **第一次调用时自动重置计时器**，返回 false 时自动停表——`ResetTimer`/`StopTimer` 大部分场合不需要了；
2. **循环体内的参数、返回值、被赋值的变量会被 `runtime.KeepAlive` 保活**，编译器无法把整段消除——这是老写法最容易出的假数据来源；
3. 语义更清晰："setup 一次、循环体测量、cleanup 一次"。

### 2.2 子 benchmark 与自定义指标

```go
func BenchmarkSumSizes(b *testing.B) {
    for _, n := range []int{10, 1000, 100000} {
        b.Run(strconv.Itoa(n), func(b *testing.B) {
            b.ReportAllocs()          // 等价于 -benchmem，写在代码里更保险
            for b.Loop() { intSink = Sum(1, n) }
        })
    }
}

func BenchmarkCustomMetric(b *testing.B) {
    ...
    b.ReportMetric(float64(count)/b.Elapsed().Seconds()/1e6, "Mitem/s")
    b.ReportMetric(float64(n), "items/op")
}
```

```text
BenchmarkSumSizes/10-8         403948881    3.136 ns/op
BenchmarkSumSizes/1000-8         2773737  429.9   ns/op
BenchmarkSumSizes/100000-8         29517 40571    ns/op
BenchmarkCustomMetric-8          2848422  419.0   ns/op   2386 Mitem/s   1000 items/op
```

**规模扫描**（10/1000/100000）比单个数字有用得多——它能看出算法是不是线性的。

### 2.3 常见的 benchmark 写错

**① setup 写在循环里**：

```text
BenchmarkWrongSetupInLoop-8      1364 ns/op   10240 B/op   1 allocs/op
BenchmarkRightSetupOutside-8       26.37 ns/op    0 B/op   0 allocs/op
```

差 50 倍——测出来的全是 setup。

**② 结果没被使用**：老写法下编译器可能把整个循环删掉，得到荒谬的 `0.3 ns/op`。`b.Loop` 已经保护了这种情况，但仍建议赋值给包级变量（sink 模式）。

**③ 只跑一次就下结论**：benchmark 波动很大。正确姿势：

```bash
go test -bench . -count 10 ./tst > old.txt
# 改代码
go test -bench . -count 10 ./tst > new.txt
benchstat old.txt new.txt          # golang.org/x/perf/cmd/benchstat
```

`benchstat` 会给出均值、变异系数和 p 值——**没有它的 benchmark 对比基本没有说服力**。

**④ 忘了机器状态**：关掉 turbo/降频、别在编译的同时跑、`-cpu 1,2,4,8` 看扩展性。

## 三、模糊测试（fuzzing）

### 3.1 形态

```go
func FuzzParseRange(f *testing.F) {
    // 种子语料
    for _, seed := range []string{"", "5", "3-7", "9-2", "a", "-", "1-2-3", "０-１"} {
        f.Add(seed)
    }

    f.Fuzz(func(t *testing.T, s string) {
        lo, hi, err := ParseRange(s)
        if err != nil {
            // 不变量：所有错误都能被 errors.Is 分类
            if !errors.Is(err, ErrBadRange) {
                t.Fatalf("ParseRange(%q) 返回了未分类的错误: %v", s, err)
            }
            return
        }
        // 不变量：成功时 lo <= hi
        if lo > hi {
            t.Fatalf("ParseRange(%q) = (%d, %d)，违反 lo <= hi", s, lo, hi)
        }
    })
}
```

```bash
go test -run FuzzParseRange ./tst           # 只跑种子语料（相当于普通测试）
go test -fuzz FuzzParseRange ./tst          # 真正 fuzz，Ctrl+C 停
go test -fuzz FuzzParseRange -fuzztime 30s ./tst
```

### 3.2 关键认识

- **fuzz 断言的是"不变量"，不是具体值**。典型不变量：不 panic、错误可分类、往返一致（`Parse(Format(x)) == x`）、与朴素实现结果一致（differential testing）。
- **崩溃输入自动落盘**到 `testdata/fuzz/FuzzXxx/`，之后每次 `go test` 都作为种子跑——**自动变成回归测试**，而且这个文件应该提交进版本库。
- 支持的参数类型有限：`[]byte`、`string`、各种整数/浮点、`bool`、`rune`。要 fuzz 复杂结构就自己从 `[]byte` 解码（或者用 `github.com/google/gofuzz`）。
- 语料库放在 `$GOCACHE/fuzz`，`go clean -fuzzcache` 清理。
- CI 里通常跑固定时长（`-fuzztime 1m`），真正长跑放夜间任务（OSS-Fuzz 就是这个模式）。

## 四、`testing/synctest`（1.25 转正）

### 4.1 它解决什么问题

测"带超时的并发代码"历来只有两条烂路：真的 `Sleep`（测试变慢、还 flaky），或者把时钟抽象成接口注入（污染生产代码）。

`synctest` 给出第三条：**在一个"气泡"里跑，气泡内 `time` 包用假时钟**，当气泡里所有 goroutine 都 "durably blocked" 时，时间自动跳到下一个唤醒点。

```go
func TestFetchTimeout(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        ch := make(chan string)
        start := time.Now()                       // 气泡内恒为 2000-01-01 00:00:00 UTC

        if _, ok := fetchWithTimeout(ch, 30*time.Second); ok {
            t.Error("应该超时")
        }
        if elapsed := time.Since(start); elapsed != 30*time.Second {
            t.Errorf("气泡内经过时间 = %v, want 30s", elapsed)
        }
    })
}
```

**真实耗时接近 0，但气泡内确实"过了 30 秒"**：

```text
--- PASS: TestFetchTimeout (0.00s)
```

`synctest.Wait()` 则是"等所有其他 goroutine 都 durably blocked"，可以替代很多 `WaitGroup`/`channel` 样板：

```go
done := false
go func() { done = true }()
synctest.Wait()        // 阻塞直到上面那个 goroutine 跑完
t.Log(done)            // 一定是 true
```

### 4.2 什么算 "durably blocked"

**算**（气泡认可，能推进时间）：

- 收发**气泡内创建**的 channel；
- 所有 case 都是气泡内 channel 的 `select`；
- `sync.Cond.Wait`；
- `sync.WaitGroup.Wait`（`Add` 也在气泡内调用）；
- `time.Sleep`。

**不算**（会导致时间不推进，最终 `Test` panic 报死锁）：

- 加锁 `sync.Mutex`/`RWMutex`；
- 网络 I/O、文件 I/O；
- 系统调用。

所以使用前提是：**被测代码的并发完全靠 channel/time/WaitGroup 表达，外部依赖用 fake 替换**。这也是为什么它对"测重试/退避/超时/心跳逻辑"特别合适，对"测数据库客户端"基本用不上。

## 五、TestMain 与依赖打桩

### 5.1 TestMain

```go
func TestMain(m *testing.M) {
    if err := setupPackage(); err != nil {          // 起容器、跑迁移、准备 fixture
        fmt.Fprintln(os.Stderr, "setup failed:", err)
        os.Exit(1)
    }
    code := m.Run()                                  // 跑本包所有 Test/Benchmark/Example/Fuzz
    teardownPackage()                                // 注意：不能用 defer
    os.Exit(code)
}
```

四条规则：

1. 有 `TestMain` 时测试**不会自动跑**，必须显式 `m.Run()`；
2. `m.Run()` 的返回值要用 `os.Exit` 传出去，否则 CI 看不到失败；
3. **`os.Exit` 不执行 defer**，清理逻辑写在 Exit 之前（或包一层内层函数返回 code）；
4. 一个包只能有一个 `TestMain`。

常见用途：测试容器（testcontainers）、DB 迁移、`goleak.VerifyTestMain(m)` 检查 goroutine 泄漏、设置全局 flag。

### 5.2 HTTP 测试的三个层次

```go
// ① httptest.NewRecorder：最快，不走网络栈，直接调 handler
req := httptest.NewRequest(http.MethodGet, "/range?r=3-5", nil)
rec := httptest.NewRecorder()
rangeHandler(rec, req)
rec.Code; rec.Body.Bytes(); rec.Header()

// ② httptest.NewServer：真的起一个 listener（随机端口），测中间件/客户端/超时
srv := httptest.NewServer(http.HandlerFunc(rangeHandler))
t.Cleanup(srv.Close)
resp, err := srv.Client().Get(srv.URL + "/range?r=1-10")

// ③ 打桩 RoundTripper：测"调用外部服务"的代码，最容易造错误分支
type stubTransport struct{ status int; body string }
func (s *stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
    return &http.Response{StatusCode: s.status, Body: io.NopCloser(strings.NewReader(s.body)), Header: http.Header{}}, nil
}
client := &http.Client{Transport: &stubTransport{200, `{"sum":42}`}}
```

选择：**测 handler 用 ①，测客户端逻辑用 ③，只有真的需要网络行为（TLS、超时、连接复用）才用 ②**。

`srv.Client()` 比自己 `http.Get` 好——它带了正确的 TLS 配置（`NewTLSServer` 时尤其重要）。

### 5.3 依赖注入 vs mock 框架

Go 社区的主流做法是**小接口 + 手写 stub**，而不是 mock 框架：

```go
// 生产代码只依赖它需要的那 1-2 个方法
type Storer interface { Put(k, v string) error }

func Handle(s Storer, ...) error { ... }

// 测试里手写一个
type fakeStore struct{ putErr error; calls []string }
func (f *fakeStore) Put(k, v string) error { f.calls = append(f.calls, k); return f.putErr }
```

理由：接口小（1-3 个方法）时手写 stub 比生成代码更短、更好读、能精确控制并发和错误时序。需要 mock 框架（`gomock`/`mockery`）的信号通常是**接口太大**，那本身就该拆。

## 六、命令与工具

### 6.1 常用命令

```bash
go test ./...                        # 跑全部
go test -v -run 'TestParse/正常' ./tst   # 正则匹配，斜杠分隔子测试层级
go test -race ./...                  # 竞态检测（CI 必须）
go test -cover ./tst                 # 覆盖率
go test -coverprofile=c.out ./... && go tool cover -html=c.out
go test -count 1 ./...               # 禁用测试缓存（-count 1 是唯一可靠方式）
go test -short ./...                 # 配合 testing.Short() 跳过慢测试
go test -timeout 30s ./...           # 默认 10m；死锁时靠它拿到全部 goroutine 栈
go test -failfast ./...              # 第一个失败就停
go test -shuffle=on ./...            # 打乱顺序，暴露测试间的隐式依赖
go test -json ./... | tparse         # 结构化输出给 CI
```

### 6.2 覆盖率的正确认识

```text
ok  github.com/.../tst   1.671s   coverage: 96.7% of statements
```

- 默认是**语句覆盖**，不是分支/条件覆盖——`if a && b` 只走一条路径也算覆盖；
- `-coverpkg=./...` 才能统计**被测包之外**的覆盖（否则只算当前包）；
- 1.20 起 `go build -cover` 可以给**集成测试/端到端测试**收集覆盖率（`GOCOVERDIR`）；
- **覆盖率是下限指标，不是目标**：90% 覆盖率的烂测试（只调用不断言）比 60% 的好测试差远了。

### 6.3 缓存与 flaky

- `go test` **会缓存成功结果**（`ok ... (cached)`），只在包内容/环境变量/文件依赖变化时失效。要强制重跑用 `-count 1`。
- 命中缓存的条件之一是"测试没有做未声明的外部访问"——用 `t.Setenv`/`t.TempDir` 的测试仍可缓存，读 `os.Getenv` 却不声明的可能被错误缓存。
- **flaky 测试的三大来源**：真实时间（用 `synctest`）、并发顺序（用 `-race` + `-shuffle=on` 暴露）、共享全局状态（用 `t.Setenv`/独立 TempDir 隔离）。

## 七、常见面试题

**1. 表驱动测试为什么是 Go 的主流写法？**
用数据描述用例、用 `t.Run` 生成子测试树：加用例只加一行、失败信息自带用例名、能用 `-run Test/子用例` 精确重跑、天然支持并行。Go 标准库自己全是这个形态（见 1.1）。

**2. `t.Error` 和 `t.Fatal` 的区别？什么时候用哪个？**
`Fatal` = `Error` + `FailNow`，会终止当前测试函数（内部是 `runtime.Goexit`，**只终止当前 goroutine**）。前置条件不满足用 `Fatal`（继续跑没意义），结果断言用 `Error`（一次能看到多个失败点）。**注意在子 goroutine 里调 `Fatal` 是错的**——它只结束那个 goroutine，测试会继续。

**3. `t.Cleanup` 和 `defer` 有什么区别？**
`defer` 在当前函数返回时执行，所以辅助函数里的 `defer` 会提前跑掉。`t.Cleanup` 注册在 `t` 上，在整个测试（含子测试）结束时按 LIFO 执行——**辅助函数里注册清理**是它的杀手场景（见 1.3）。

**4. `t.Parallel()` 的执行顺序是怎样的？**
调用后子测试立即暂停，等父测试函数体执行完毕才一起放行。所以"父测试里 `t.Run` 之后的代码"会先于并行子测试执行。并行度受 `-parallel` 限制。`t.Setenv`/`t.Chdir` 与它互斥（见 1.2）。

**5. `b.Loop` 相比 `for range b.N` 好在哪？（1.24+）**
① 自动重置/停止计时器，`ResetTimer` 基本不用写了；② 循环体内的值被 `KeepAlive` 保活，编译器不能把循环整段删掉（这是老写法最常见的假数据源）；③ setup/cleanup 语义清晰（见 2.1）。

**6. 怎么让 benchmark 的结论可信？**
`-count 10` 多次采样 + `benchstat` 做统计比较；setup 挪到循环外；结果赋值给包级 sink；`-cpu 1,2,4,8` 看扩展性；机器状态干净。单次运行的 benchmark 数字基本没有说服力（见 2.3）。

**7. 模糊测试怎么写？断言什么？**
`f.Add` 给种子，`f.Fuzz(func(t, input){...})` 写**不变量**（不 panic、错误可分类、往返一致、与朴素实现一致），而不是具体期望值。崩溃输入自动写到 `testdata/fuzz/`，之后变成回归用例，应该提交进库（见 3.1、3.2）。

**8. `testing/synctest` 解决什么问题？限制是什么？**
在气泡内用假时钟，让"带 30 秒超时的测试"瞬间跑完，且不需要把时钟抽象成接口注入生产代码。限制：只有 channel/`time.Sleep`/`Cond.Wait`/`WaitGroup.Wait` 算 durably blocked，**加锁和 I/O 不算**——外部依赖必须用 fake 替换（见 4.1、4.2）。

**9. `TestMain` 有哪些坑？**
必须显式调 `m.Run()` 否则测试不跑；返回码要 `os.Exit` 出去；**`os.Exit` 不执行 defer**，清理要写在前面；一个包只能有一个（见 5.1）。

**10. HTTP handler 怎么测？要不要起真服务器？**
测 handler 用 `httptest.NewRecorder`（最快、不走网络）；测客户端逻辑打桩 `RoundTripper`；只有真的需要网络行为（TLS、超时、连接复用）才用 `httptest.NewServer`（见 5.2）。

**11. Go 里要用 mock 框架吗？**
通常不需要。小接口（1-3 方法）+ 手写 stub 更短更可控。需要 gomock 的信号往往是接口太大，那本身就该拆（"accept interfaces, return structs"）（见 5.3）。

**12. 为什么 `go test` 第二次跑显示 `(cached)`？怎么禁用？**
测试结果会被缓存，包内容/环境/文件依赖不变就直接复用。`-count 1` 是唯一可靠的禁用方式（`-count 2` 只是跑两遍，仍可能命中缓存逻辑）（见 6.3）。

**13. 覆盖率多少才够？**
覆盖率是**下限指标**，默认还只是语句覆盖（`if a && b` 走一条路径就算覆盖）。90% 的"只调用不断言"测试不如 60% 的好测试。关键路径、错误分支、边界条件的覆盖比总百分比重要。集成测试覆盖率要用 `go build -cover` + `GOCOVERDIR`（1.20+）（见 6.2）。

**14. 怎么定位 flaky 测试？**
`-shuffle=on` 暴露测试间的隐式依赖；`-race` 抓并发竞态；`-count 100 -run TestFlaky` 复现；把真实时间换成 `synctest`；共享状态用 `t.Setenv`/`t.TempDir` 隔离（见 6.3）。

**15. `Example` 函数的作用？什么时候会被执行？**
既是文档（出现在 `go doc` 和 pkg.go.dev）又是测试。**有 `// Output:` 注释才会被执行并比对输出**，没有就只编译。命名规则 `ExampleF`/`ExampleT_M`/`ExampleF_suffix`（suffix 小写开头）（见 `notes/tst/bench_test.go`）。
