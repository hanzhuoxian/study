# Go 程序的启动流程

以 [gmp/main.go](gmp/main.go) 这个最简单的程序为例，拆解从操作系统加载可执行文件到执行用户 `main.main` 的完整链路。

分析基于 **Go 1.26.3 / darwin-arm64**，源码路径 `$GOROOT/src/runtime/`。

```go
// gmp/main.go
func main() {
	f, err := os.Create("./tmp/trace.out")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()

	go func() {
		fmt.Println("go runtime")
	}()
}
```

这段代码里我们能直接看到的只有 `main.main`（第 9 行）和一个 `go func()`（第 18 行）。但在它被调用之前，runtime 已经完成了一整套 bootstrap。下面按执行顺序展开。

---

## 全景图

```
OS 内核 exec()
   │
   ▼
_rt0_arm64_darwin        (汇编入口, rt0_*.s)
   │
   ▼
runtime·rt0_go           (asm_arm64.s:105)   ← 真正的 runtime 引导
   │   ├─ 初始化 TLS / g0 栈
   │   ├─ 绑定 m0 <-> g0
   │   ├─ runtime·args      保存 argc/argv
   │   ├─ runtime·osinit    探测 CPU 核数等
   │   ├─ runtime·schedinit 调度器 / 内存 / GC 初始化   (proc.go:831)
   │   ├─ runtime·newproc   创建「主 goroutine」→ runtime.main  (proc.go:5295)
   │   └─ runtime·mstart    启动 m0，进入调度循环
   │
   ▼
runtime.main             (proc.go:149)  ← 运行在主 goroutine 上
   │   ├─ 新建 M 运行 sysmon（系统监控线程）
   │   ├─ lockOSThread     绑定主线程
   │   ├─ doInit(runtime_inittasks)   runtime 包自身的 init
   │   ├─ gcenable         正式开启 GC
   │   ├─ doInit(各模块 inittasks)    用户/依赖包的 init（含 var 初始化）
   │   └─ fn := main_main; fn()       ← 调用我们写的 main.main
   │
   ▼
main.main                (gmp/main.go:9)  ← 用户代码从这里开始
   │   └─ go func(){...}   →  runtime·newproc 创建新 goroutine
   │
   ▼
return → runtime.main 继续 → exit(0)  进程结束
```

一句话总结：**用户的 `main` 不是程序入口，它只是 `runtime.main` 里的一次普通函数调用。**

---

## 第一阶段：汇编引导 `rt0_go`

程序真正的入口点是平台相关的 `_rt0_arm64_darwin`，它整理好 `argc`/`argv` 后跳转到与平台无关的 `runtime·rt0_go`（[asm_arm64.s:105](../../../../.asdf/installs/golang/1.26.3/go/src/runtime/asm_arm64.s)）。这段汇编做的都是「在能运行 Go 代码之前必须用汇编手工完成」的事：

   1. **初始化 TLS（线程本地存储）**：`tlsinit`，为后续 `g` 寄存器的存取做准备。
1. **构造 g0 的栈**：把操作系统给的初始栈划出一段作为 `g0`（调度用的系统栈）的栈空间，设置 `stackguard`。
   ```asm
   MOVD  $runtime·g0(SB), g          // g 寄存器指向 g0
   ...
   MOVD  R7, (g_stack+stack_hi)(g)   // 栈高地址
   ```
2. **绑定 m0 与 g0**：`m0` 是主线程对应的 M，`g0` 是它的调度栈。
   ```asm
   MOVD  $runtime·m0(SB), R0
   MOVD  g, m_g0(R0)   // m0.g0 = g0
   MOVD  R0, g_m(g)    // g0.m  = m0
   ```
   > `m0` 和 `g0` 都是全局变量（不是堆分配），进程唯一。
3. **依次调用四个初始化函数**，这是整个引导的核心：
   ```asm
   BL  runtime·args(SB)       // 保存 argc/argv
   BL  runtime·osinit(SB)     // 操作系统相关初始化（如 numCPUStartup = CPU 核数）
   BL  runtime·schedinit(SB)  // 调度器、内存分配器、GC 的初始化
   ```
4. **创建主 goroutine**：把 `runtime.main` 的地址作为入口，交给 `newproc` 创建第一个「真正的」goroutine（即主 goroutine，`goid=1`）。
   ```asm
   MOVD  $runtime·mainPC(SB), R0   // mainPC 数据段里存的是 runtime·main 的地址
   ...
   BL    runtime·newproc(SB)
   ```
   注意：**此时只是把主 goroutine 放进运行队列，还没开始跑。**
5. **启动 M0**：
   ```asm
   BL  runtime·mstart(SB)
   UNDEF                       // mstart 永不返回，返回了就是 bug
   ```

### `schedinit` 做了什么（proc.go:831）

`schedinit` 运行在 `g0` 上，是「让 runtime 变得可用」的总开关，关键步骤：

| 调用                    | 作用                                                             |
| ----------------------- | ---------------------------------------------------------------- |
| `lockInit(...)` 一堆    | 初始化各类全局锁                                                 |
| `worldStopped()`        | 世界起步时是「停止」状态，P 还不能运行                           |
| `stackinit()`           | 栈分配器（栈缓存池）                                             |
| `mallocinit()`          | 堆内存分配器（mheap/mcentral/mcache 等）                         |
| `mcommoninit(gp.m, -1)` | 初始化 m0 的公共字段                                             |
| `gcinit()`              | GC 相关数据结构初始化（但还没启动 GC）                           |
| `goargs()` / `goenvs()` | 把命令行参数、环境变量转成 Go 的 slice                           |
| 读取 `GOMAXPROCS`       | 决定 P 的数量：环境变量优先，否则用 CPU 核数                     |
| `procresize(procs)`     | **按 GOMAXPROCS 创建 P（processor）数组**，并把当前 M 关联一个 P |
| `worldStarted()`        | 世界启动，P 可以运行了                                           |

到此为止，GMP 里的 **G（g0）、M（m0）、P（procresize 创建）** 三者都已就位。

---

## 第二阶段：`mstart` → 调度循环

`mstart` → `mstart0`（[proc.go:1862](../../../../.asdf/installs/golang/1.26.3/go/src/runtime/proc.go)）→ `mstart1` → `schedule()`：

- `mstart0` 校正 g0 栈边界，设置 `stackguard`。
- `mstart1` 保存 `m.g0.sched`（一个「返回标签」，供 `goexit0`/`mcall` 之后跳回来退出线程用），执行 `minit()`（线程级初始化，安装信号处理等），m0 还会额外跑 `mstartm0()`。
- 最后调用 `schedule()` 进入**调度循环**：找一个可运行的 G 来跑。

因为第一阶段的 `newproc` 已经把「主 goroutine（`runtime.main`）」放进了运行队列，`schedule()` 找到的第一个 G 就是它，于是 **`runtime.main` 开始在 m0 上执行**。

---

## 第三阶段：`runtime.main`（proc.go:149）

这是运行在主 goroutine 上的函数，也是用户代码执行前的最后准备：

```go
func main() {
	mp := getg().m
	maxstacksize = 1000000000        // 64 位下单 goroutine 栈上限 1GB

	mainStarted = true               // 允许 newproc 唤醒/新建 M

	if haveSysmon {
		systemstack(func() {
			newm(sysmon, nil, -1)    // 启动 sysmon 系统监控线程（不绑定 P，独立运行）
		})
	}

	lockOSThread()                   // init 期间把主 goroutine 锁在主 OS 线程上

	doInit(runtime_inittasks)        // 1) 先跑 runtime 包自己的 init

	gcenable()                       // 2) 正式开启垃圾回收（启动后台 GC goroutine）

	main_init_done = make(chan bool)

	// 3) 按依赖顺序执行所有模块（用户包 + 依赖包）的 init 任务
	for m := &firstmoduledata; true; m = m.next {
		doInit(m.inittasks)
		if m == last { break }
	}
	close(main_init_done)
	unlockOSThread()

	fn := main_main                  // 4) 间接调用用户的 main.main
	fn()

	...
	exit(0)                          // 5) main.main 返回后，进程退出
}
```

几个关键点：

- **sysmon**：不依赖 P 的后台监控线程，负责抢占长时间运行的 G、回收空闲 P、触发网络轮询、强制 GC 等。它在这里被拉起。
- **init 顺序**：`runtime.init` → 各依赖包 `init`（含包级 `var` 初始化）→ 用户包 `init`。对于我们的 `main.go`，`fmt`、`os`、`runtime/trace` 等包的 `init` 都在这一步完成——所以进入 `main.main` 时，这些包已经可用。
- **`main_main` 是间接调用**：链接器在编译 runtime 时并不知道用户 `main` 的地址，用 `go:linkname` 把 `main.main` 关联到 `runtime.main_main`，运行时再间接调用。
- **`main.main` 返回即退出**：`fn()` 返回后直接 `exit(0)`。这解释了为什么 `main` 结束时不会等待其它 goroutine（见下）。

---

## 第四阶段：回到我们的 `main.main`

现在终于执行到 [gmp/main.go:9](gmp/main.go#L9) 的代码：

```go
trace.Start(f)          // 开始采集执行链路 trace
defer trace.Stop()

go func() {             // ← 这一行触发 runtime·newproc
	fmt.Println("go runtime")
}()
```

### `go func(){...}` 背后：`newproc`（proc.go:5295）

`go` 关键字被编译成对 `runtime.newproc` 的调用：

```go
func newproc(fn *funcval) {
	gp := getg()
	pc := sys.GetCallerPC()
	systemstack(func() {
		newg := newproc1(fn, gp, pc, false, waitReasonZero)  // 创建 / 复用一个 g
		pp := getg().m.p.ptr()
		runqput(pp, newg, true)     // 放入当前 P 的本地运行队列（next 槽）
		if mainStarted {
			wakep()                 // 有需要时唤醒/新建一个 M 来并行执行
		}
	})
}
```

`newproc1` 的要点：

- `gfget(pp)` 先尝试从 P 的空闲 g 缓存里复用，没有再 `malg(stackMin)` 新建（默认 2KB 栈）。
- 设置新 g 的 `sched.pc = goexit`，再用 `gostartcallfn` 把真正的 `fn` 塞进去——这样 **goroutine 执行完 `fn` 会自动返回到 `goexit`**，由 `goexit` 完成清理并把 g 放回缓存池。
- 记录 `parentGoid`、`gopc`（`go` 语句地址）、`startpc`（函数地址）等，用于 trace / 栈回溯。
- 新 g 状态置为 `_Grunnable`，等待被调度。

### 一个重要现象：这个 goroutine 很可能来不及执行

`main.main` 里 `go func()` 之后没有任何同步（没有 `WaitGroup`、`channel`、`time.Sleep`）。新 goroutine 只是被放进运行队列，而主 goroutine 继续往下走、遇到两个 `defer`（`trace.Stop()`、`f.Close()`）后 `main.main` 返回，`runtime.main` 随即 `exit(0)`。

**进程退出不会等待其它 goroutine 完成**，所以 `fmt.Println("go runtime")` 很可能根本不会被打印出来（是否打印取决于调度时机，属于竞态）。这正是「`main` 返回 = 程序结束」的直接体现。如果想确保它执行，需要显式同步：

```go
done := make(chan struct{})
go func() {
	fmt.Println("go runtime")
	close(done)
}()
<-done
```

---

## 小结：GMP 三要素何时诞生

| 要素               | 诞生时机                         | 说明                                 |
| ------------------ | -------------------------------- | ------------------------------------ |
| **g0 / m0**        | `rt0_go` 汇编阶段                | 全局变量，进程唯一，负责引导         |
| **P**              | `schedinit → procresize`         | 数量 = `GOMAXPROCS`（默认 CPU 核数） |
| **主 goroutine**   | `rt0_go → newproc`               | 入口是 `runtime.main`，`goid=1`      |
| **sysmon 线程**    | `runtime.main` 内 `newm(sysmon)` | 后台监控，不绑定 P                   |
| **GC**             | `runtime.main` 内 `gcenable`     | init 之后、用户 main 之前开启        |
| **用户 goroutine** | `main.main` 里的 `go` 语句       | 由 `newproc` 创建，入队等待调度      |

整条链路的设计精髓：**运行时先把自己（内存、调度器、GC、监控）全部拉起来，再以「一个普通 goroutine」的形式去运行用户的 `main`。** 用户看到的 `main` 只是 runtime 大厦封顶后的一次函数调用。

---

## 附：用 trace 验证

本例已经用 `runtime/trace` 采集了执行链路，运行后可视化查看：

```bash
cd go/notes/gmp
mkdir -p tmp
go run main.go
go tool trace ./tmp/trace.out
```

在 trace 的 goroutine 视图里能看到主 goroutine（G1）、sysmon、以及那个 `fmt.Println` 的 goroutine 是否被调度执行，直观印证上面的分析。
