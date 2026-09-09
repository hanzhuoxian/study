# Go 学习笔记

这是一组面向 Go 1.26 的专题笔记。每篇 Markdown 讲解一个主题；多数主题还配有同名目录，包含可运行示例或基准测试。

## 阅读路线

| 阶段 | 主题 |
| --- | --- |
| 数据与函数 | [字符串](string.md) · [切片](slice.md) · [Map](map.md) · [函数](func.md) |
| 类型与抽象 | [结构体](struct.md) · [方法](method.md) · [接口](interface.md) · [错误处理](error.md) · [泛型](generic.md) · [迭代器](iter.md) |
| 并发与生命周期 | [Channel](chan.md) · [sync](sync.md) · [atomic 与内存模型](atomic.md) · [时间](time.md) · [Context](context.md) |
| 标准库与测试 | [JSON](json.md) · [测试](test.md) |
| 运行时 | [GMP](gmp.md) · [程序启动](exec.md) · [调度器](sched.md) · [网络轮询器](netpoll.md) · [内存分配](mem.md) · [GC](gc.md) |
| 性能与底层 | [性能分析](profile.md) · [sync.Pool](pool.md) · [反射](reflect.md) · [unsafe](unsafe.md) |

这些是专题笔记，默认读者已了解变量、控制流、包和基本 Go 语法。首次阅读按表格从上到下，先走每篇的基础路线，再回头读源码；JSON 的使用不要求先掌握反射，但源码部分建议读完反射篇再看。

每篇开头提供定位、前置知识、示例入口和阅读导航。导航分开基础使用与源码深入，篇末问题用于自测。运行时结构和基准数字仅适用于标注的版本与环境，不能把不同机器上的数字直接横向比较。

## 内容归属

同一个知识点由一篇文档负责完整解释，其他篇通过链接补充应用场景：

| 知识点 | 主文档 | 其他篇的范围 |
| --- | --- | --- |
| 方法集 / 接口布局 | 方法 / 接口 | 结构体讲字段与嵌入，错误处理讲错误接口的使用 |
| 逃逸与栈堆分配 | 内存分配 | 函数、接口、Map 只说明本主题的表现 |
| Timer/Ticker 版本语义 | 时间 | Channel 讲 select 配合方式，Context 讲超时生命周期 |
| G/M/P / 启动 / 调度 | GMP / 程序启动 / 调度器 | 分别解释角色、初始化时间线、运行期行为 |
| 基准方法 / 性能定位 | 测试 / 性能分析 | 各主题保留本地实验，注明复现命令与环境 |

维护时优先引用稳定的篇内锚点；源码用文件路径和符号定位，避免依赖个人安装路径或易漂移的行号。示例中的实验编号与文档章节编号独立，按函数名、主题名定位。

## 运行示例

仓库要求 Go 1.26.3（见 [`go.mod`](go.mod)）。在本目录执行：

```bash
# 编译并运行全部测试
go test ./...

# 运行单个主题的示例
go run ./context

# 运行某个包的全部 benchmark
go test -run '^$' -bench . -benchmem ./sync

# 静态检查与格式检查
go vet ./...
test -z "$(gofmt -l .)"
```

`tst` 包中的 `TestWithRealServer` 会监听本地随机端口；受限容器或沙箱若禁止网络监听，可只运行不依赖真实 socket 的测试：

```bash
go test ./tst -run 'TestRangeHandler|TestFetchSumWithStub'
```

## 目录约定

- 根目录的 `*.md`：原理、源码分析、常见陷阱和实践建议。
- 同名子目录：可运行示例；通常用 `go run ./<目录>` 执行。
- `*_test.go`：单元测试、模糊测试或 benchmark；用 `go test` 执行。
- `errs/`、`refl/`、`str/`、`tm/`、`tst/`、`prof/`、`uns/` 分别对应 `error.md`、`reflect.md`、`string.md`、`time.md`、`test.md`、`profile.md`、`unsafe.md`；启动篇复用 `gmp/` 的示例。

部分示例会故意展示 panic、数据竞争或错误写法。运行前请先阅读所在函数的注释，不要直接复制标有“错误示范”的代码到生产环境。
