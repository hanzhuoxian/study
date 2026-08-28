// 调度器示例：对应 notes/sched.md（结构见 gmp.md）
//
//	go run ./sched                跑全部演示
//	GODEBUG=schedtrace=200 go run ./sched   自己看调度器状态
//	GODEBUG=asyncpreemptoff=1 go run ./sched  关掉异步抢占（对比 3.1）
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	if mode := os.Getenv("SCHED_DEMO"); mode != "" {
		runChild(mode)
		return
	}

	basicGOMAXPROCS()
	basicGoroutineStates()
	basicYield()

	findRunnableOrder()
	workStealing()
	globalQueueFairness()

	preemption()
	syscallHandoff()
	lockOSThread()
	schedTrace()
}

// ---------------------------------------------------------------------------
// 1.1 GOMAXPROCS
// ---------------------------------------------------------------------------

func basicGOMAXPROCS() {
	section("1.1 GOMAXPROCS")

	fmt.Printf("  runtime.NumCPU()      = %d（机器/cgroup 可见的 CPU 数）\n", runtime.NumCPU())
	fmt.Printf("  runtime.GOMAXPROCS(0) = %d（P 的数量）\n", runtime.GOMAXPROCS(0))
	fmt.Printf("  runtime.NumGoroutine()= %d\n", runtime.NumGoroutine())

	fmt.Println()
	fmt.Println("  Go 1.25 起 runtime 会读 cgroup 的 CPU limit 自动设置 GOMAXPROCS，")
	fmt.Println("  并且 sysmon 每秒检查一次 limit 变化（sysmonUpdateGOMAXPROCS）。")
	fmt.Println("  1.25 之前：GOMAXPROCS = 宿主机核数，容器里会导致 GC 和调度抢走全部配额")
	fmt.Println("  （gc.md 3.9），当年只能靠 uber-go/automaxprocs 兜。")
	fmt.Println()
	fmt.Println("  三个 P 数量相关的事实：")
	fmt.Println("    · P 的数量决定**并行度**（同时能跑多少 G），不限制 goroutine 总数")
	fmt.Println("    · M（线程）数量上限是 sched.maxmcount = 10000，和 P 无关")
	fmt.Println("    · GOMAXPROCS 可以运行时改，但会触发 STW（stopTheWorld + procresize）")
}

// ---------------------------------------------------------------------------
// 1.2 goroutine 的状态
// ---------------------------------------------------------------------------

func basicGoroutineStates() {
	section("1.2 goroutine 的状态机")

	for _, row := range [][2]string{
		{"_Gidle", "刚分配，还没初始化"},
		{"_Grunnable", "在运行队列里等 P"},
		{"_Grunning", "正在某个 M 上跑（占着 P）"},
		{"_Gsyscall", "在系统调用里（可能已交出 P）"},
		{"_Gwaiting", "被阻塞（chan/锁/timer/netpoll），**不在**运行队列里"},
		{"_Gdead", "刚退出或刚从 gfree 里取出来"},
		{"_Gcopystack", "栈正在被拷贝（morestack/shrinkstack）"},
		{"_Gpreempted", "被抢占停下，等待恢复（1.14 异步抢占引入）"},
	} {
		fmt.Printf("  %-14s %s\n", row[0], row[1])
	}
	fmt.Println()
	fmt.Println("  两组关键转换：")
	fmt.Println("    gopark:  _Grunning -> _Gwaiting  （主动让出，比如 chan 阻塞）")
	fmt.Println("    goready: _Gwaiting -> _Grunnable （被唤醒，塞回运行队列）")
	fmt.Println("  → 阻塞的是 G，不是 M：M 会去跑别的 G（chan.md 2.7、sync.md 2.4）")

	// 用 goroutine dump 看真实状态
	out, _ := selfRun("states")
	fmt.Println("\n  子进程里制造各种状态，从 goroutine dump 里抓到的：")
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		fmt.Println("    " + l)
	}
}

// ---------------------------------------------------------------------------
// 1.3 Gosched / runtime.Goexit
// ---------------------------------------------------------------------------

func basicYield() {
	section("1.3 主动让出")

	fmt.Println("  runtime.Gosched()  让出 P，把当前 G 放回**全局队列**，然后重新调度")
	fmt.Println("  time.Sleep(0)      直接返回，不让出（time.md 4.4）")
	fmt.Println("  runtime.Goexit()   结束当前 goroutine，但**会执行完所有 defer**")
	fmt.Println("     · 在 main goroutine 里调用会 fatal error: no goroutines")
	fmt.Println("     · testing 的 t.Fatal 就是靠它实现的（test.md 面试题 2）")

	var order []string
	var mu sync.Mutex
	done := make(chan struct{})
	go func() {
		defer func() {
			mu.Lock()
			order = append(order, "defer 执行了")
			mu.Unlock()
			close(done)
		}()
		mu.Lock()
		order = append(order, "Goexit 之前")
		mu.Unlock()
		runtime.Goexit()
		mu.Lock()
		order = append(order, "这行不会执行")
		mu.Unlock()
	}()
	<-done
	fmt.Println("  Goexit 实测顺序:", order)
}

// ---------------------------------------------------------------------------
// 2.1 findRunnable 的查找顺序
// ---------------------------------------------------------------------------

func findRunnableOrder() {
	section("2.1 schedule() -> findRunnable() 的查找顺序")

	for i, step := range []string{
		"trace reader / GC worker（有的话优先）",
		"**每 61 次调度**检查一次全局队列（防止两个 G 互相唤醒霸占本地队列）",
		"唤醒 finalizer G / cleanup G",
		"本地运行队列 runqget（含 runnext 槽）",
		"全局运行队列 globrunqgetbatch（一次搬 len(local)/2 个过来）",
		"netpoll(0) 非阻塞轮询（有等待者时的优化）",
		"**work stealing**：随机顺序遍历其他 P，偷一半（最多 4 轮）",
		"再检查一遍全局队列 / GC 工作 / timer",
		"netpoll(阻塞) 或者 stopm() 把 M 挂起",
	} {
		fmt.Printf("  %d. %s\n", i+1, step)
	}
	fmt.Println()
	fmt.Println("  源码：runtime/proc.go findRunnable()，第 2 步是 pp.schedtick%61 == 0")
	fmt.Println("  → runnext 是一个单槽的\"下一个要跑的 G\"：newproc 创建的 G 优先放这里，")
	fmt.Println("    目的是让\"刚 go 出来的 G\"尽快跑（局部性好），代价是可能推迟队列里的 G")
}

// ---------------------------------------------------------------------------
// 2.2 work stealing
// ---------------------------------------------------------------------------

func workStealing() {
	section("2.2 work stealing 实测")

	// 让一个 goroutine 在一个 P 上疯狂创建 G，观察它们是否被其他 P 偷走执行
	var perThread [64]atomic.Int64 // 用"哪个 M 执行了"近似看分布
	var wg sync.WaitGroup
	const nTask = 20000

	start := time.Now()
	// 全部从**同一个 goroutine** 创建 -> 大部分先进同一个 P 的本地队列
	for range nTask {
		wg.Go(func() {
			// 做一点纯 CPU 的活
			s := 0
			for i := range 200 {
				s += i
			}
			perThread[runtime.GOMAXPROCS(0)%64].Add(int64(s & 1))
		})
	}
	wg.Wait()

	fmt.Printf("  一个 goroutine 创建 %d 个 G，全部完成用了 %v\n", nTask, time.Since(start).Round(time.Millisecond))
	fmt.Println("  本地队列容量只有 256（runq [256]guintptr），放不下就批量搬到全局队列，")
	fmt.Println("  其他 P 空闲时会从全局队列取、或者直接从这个 P 偷走一半 —— 所以能跑满多核。")
	fmt.Println()
	fmt.Println("  偷取规则（runqsteal / stealWork）：")
	fmt.Println("    · 随机起点 + 随机步长遍历所有 P（避免所有 M 都从 P0 开始偷）")
	fmt.Println("    · 一次偷对方本地队列的**一半**")
	fmt.Println("    · 最多 4 轮；最后一轮才允许偷 runnext（且要等对方 3µs，给它机会自己跑）")
	fmt.Println("    · 偷不到就顺手检查对方的 timer 堆（time.md 3.1）")
}

// ---------------------------------------------------------------------------
// 2.3 全局队列的公平性
// ---------------------------------------------------------------------------

func globalQueueFairness() {
	section("2.3 为什么要每 61 次看一次全局队列")

	fmt.Println("  假设两个 G 不断互相唤醒（ping-pong），它们会一直待在同一个 P 的本地队列里：")
	fmt.Println("    G1 跑完 -> 唤醒 G2 放进 runnext -> G2 跑完 -> 唤醒 G1 ...")
	fmt.Println("  全局队列里的 G 就永远等不到。所以 findRunnable 里有：")
	fmt.Println("      if pp.schedtick%61 == 0 && !sched.runq.empty() { globrunqget() }")
	fmt.Println("  → 61 是个质数，避免和其他周期性行为共振；这是 Go 调度器唯一的\"强制公平\"点")
	fmt.Println("  → 注意：Go 的调度器**不是**完全公平的，它优先吞吐（局部性）而不是延迟")
}

// ---------------------------------------------------------------------------
// 3.1 抢占：1.14 之前的死循环问题
// ---------------------------------------------------------------------------

func preemption() {
	section("3.1 抢占（1.14 的异步抢占）")

	fmt.Println("  协作式抢占（1.14 之前）：只在**函数调用**处检查 stackguard0 == stackPreempt，")
	fmt.Println("  所以没有函数调用的紧密循环无法被抢占：")
	fmt.Println("      for {}   // 1.14 之前：GOMAXPROCS=1 时会让整个程序卡死（GC 也进不来）")
	fmt.Println()
	fmt.Println("  异步抢占（1.14+）：sysmon 发现 G 跑超过 forcePreemptNS = 10ms 就")
	fmt.Println("  给对应线程发 SIGURG，信号处理函数把 G 的 PC 改到 asyncPreempt，")
	fmt.Println("  保存寄存器后进入调度器。这要求编译器为每个函数生成精确的栈图（安全点）。")
	fmt.Println()

	out, _ := selfRun("preempt")
	fmt.Println("  子进程 GOMAXPROCS=1 跑一个没有函数调用的死循环：")
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		fmt.Println("    " + l)
	}
	fmt.Println()
	fmt.Println("  → GODEBUG=asyncpreemptoff=1 可以关掉异步抢占（排查抢占相关问题时用）")
	fmt.Println("  → 抢占点变多的代价：SIGURG 信号在某些系统调用密集的场景会带来 EINTR 重试")
}

// ---------------------------------------------------------------------------
// 3.2 系统调用与 handoff
// ---------------------------------------------------------------------------

func syscallHandoff() {
	section("3.2 系统调用：P 的移交")

	fmt.Println("  进入系统调用（entersyscall）：")
	fmt.Println("    · G 状态 -> _Gsyscall，P 状态 -> _Psyscall")
	fmt.Println("    · **P 不立刻交出**（乐观：大部分 syscall 很快返回）")
	fmt.Println("  如果 syscall 很慢：")
	fmt.Println("    · sysmon 的 retake() 发现 P 在 _Psyscall 且超过 **20µs**（一次 sysmon 周期）")
	fmt.Println("    · 调 handoffp()：把 P 交给别的 M（新建或从 midle 取），业务继续跑")
	fmt.Println("    · syscall 返回后（exitsyscall）原 M 尝试抢回一个 P，抢不到就把 G 放全局队列、自己休眠")
	fmt.Println()
	fmt.Println("  两类系统调用的区别：")
	fmt.Println("    · **网络 IO** 不走这条路——它被 netpoll 接管，G 直接 _Gwaiting，P 立刻去跑别的（netpoll.md）")
	fmt.Println("    · **文件 IO / DNS / cgo 调用**是真的阻塞线程，走 handoff，所以会创建更多 M")
	fmt.Println()

	threads := func() int { return pprof.Lookup("threadcreate").Count() }
	before := threads()

	var wg sync.WaitGroup
	for range 200 {
		wg.Go(func() {
			// 文件 IO + sleep：真的阻塞线程，逼 runtime 创建更多 M
			f, err := os.Open(os.DevNull)
			if err == nil {
				buf := make([]byte, 1)
				_, _ = f.Read(buf)
				f.Close()
			}
			time.Sleep(time.Millisecond)
		})
	}
	wg.Wait()
	fmt.Printf("  200 个并发阻塞 IO 前后的线程数（threadcreate profile）: %d -> %d\n",
		before, threads())
	fmt.Println("  → 这就是\"大量阻塞式 IO 会让 Go 程序线程数暴涨\"的原因；上限 maxmcount=10000")
}

// ---------------------------------------------------------------------------
// 3.3 LockOSThread
// ---------------------------------------------------------------------------

func lockOSThread() {
	section("3.3 LockOSThread")

	done := make(chan string, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		// 这个 goroutine 从此绑定在当前 OS 线程上
		done <- "绑定期间：这个 G 只会在这个线程上跑，别的 G 也不会用这个线程"
	}()
	fmt.Println(" ", <-done)

	fmt.Println()
	fmt.Println("  用途：")
	fmt.Println("    · 调用要求线程本地状态的 C 库（OpenGL、某些 GUI 框架）")
	fmt.Println("    · 需要固定 OS 线程身份的系统调用（Linux 的 namespace、setns、gettid）")
	fmt.Println("    · runtime.main 自己就锁在 m0 上（signal 处理需要）")
	fmt.Println("  代价：")
	fmt.Println("    · 该 M 被独占，P 会被 handoff 出去，等价于多占一个线程")
	fmt.Println("    · 忘记 Unlock 且 goroutine 退出 -> 那个线程会被销毁（不能复用）")
	fmt.Println("    · 锁定期间该 G 无法被迁移，长时间 CPU 密集会影响这个线程上的其他工作")
}

// ---------------------------------------------------------------------------
// 3.4 schedtrace 实测
// ---------------------------------------------------------------------------

func schedTrace() {
	section("3.4 GODEBUG=schedtrace 实测")

	out, _ := selfRun("schedtrace")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i, l := range lines {
		if i >= 5 {
			fmt.Printf("    ...（共 %d 行）\n", len(lines))
			break
		}
		fmt.Println("    " + l)
	}
	fmt.Println()
	fmt.Println("  字段含义：")
	fmt.Println("    gomaxprocs  P 的数量")
	fmt.Println("    idleprocs   空闲 P 数量（持续等于 gomaxprocs 说明没活干）")
	fmt.Println("    threads     线程总数（含 sysmon、GC worker）")
	fmt.Println("    spinningthreads  正在自旋找活的 M（多说明任务分布不均或频繁唤醒）")
	fmt.Println("    needspinning     需要有 M 去自旋")
	fmt.Println("    idlethreads 空闲线程数")
	fmt.Println("    runqueue    **全局**运行队列长度")
	fmt.Println("    [n n n n]   每个 P 的**本地**队列长度")
	fmt.Println()
	fmt.Println("  怎么读：")
	fmt.Println("    · runqueue 持续很大 + 本地队列也满 -> CPU 不够，或者有 G 长时间不让出")
	fmt.Println("    · 本地队列长度极不均匀 -> 任务粒度差异大（work stealing 也救不了）")
	fmt.Println("    · threads 远大于 gomaxprocs -> 大量阻塞式 syscall/cgo（见 3.2）")
	fmt.Println("    · GODEBUG=scheddetail=1 会额外打印每个 P/M/G 的明细")
}

// ---------------------------------------------------------------------------
// 子进程
// ---------------------------------------------------------------------------

func runChild(mode string) {
	switch mode {
	case "states":
		// 制造几种典型阻塞状态，然后把 goroutine dump 里的状态行打出来
		go func() { ch := make(chan int); <-ch }()              // chan receive
		go func() { ch := make(chan int); ch <- 1 }()           // chan send
		go func() { var mu sync.Mutex; mu.Lock(); mu.Lock() }() // sync.Mutex.Lock
		go func() { time.Sleep(time.Hour) }()                   // sleep
		go func() { select {} }()                               // select (no cases)
		time.Sleep(100 * time.Millisecond)

		buf := make([]byte, 1<<16)
		n := runtime.Stack(buf, true)
		for _, l := range strings.Split(string(buf[:n]), "\n") {
			if strings.HasPrefix(l, "goroutine ") {
				// goroutine 7 gp=0x... m=nil [chan receive]:
				if i := strings.Index(l, "["); i >= 0 {
					fmt.Println(strings.TrimSuffix(l[i:], ":"))
				}
			}
		}

	case "preempt":
		runtime.GOMAXPROCS(1)
		var spinning atomic.Bool
		spinning.Store(true)
		go func() {
			// 没有函数调用、没有分配的紧密循环
			for spinning.Load() {
			}
		}()
		time.Sleep(50 * time.Millisecond)
		// 如果抢占不生效，这个 goroutine 永远拿不到 P，下面这行打不出来
		fmt.Println("GOMAXPROCS=1，死循环 goroutine 存在的情况下，main 仍然被调度到了 ✓")
		fmt.Println("（1.14 之前这里会卡死；异步抢占靠 SIGURG + asyncPreempt 实现）")
		spinning.Store(false)

	case "schedtrace":
		// 跑约 300ms，制造持续的队列积压，好让 schedtrace 输出多行
		deadline := time.Now().Add(300 * time.Millisecond)
		var wg sync.WaitGroup
		for range runtime.GOMAXPROCS(0) {
			wg.Go(func() {
				for time.Now().Before(deadline) {
					var inner sync.WaitGroup
					for range 500 {
						inner.Go(func() {
							s := 0
							for i := range 5000 {
								s += i
							}
							_ = s
						})
					}
					inner.Wait()
				}
			})
		}
		wg.Wait()
	}
}

func selfRun(mode string) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	_ = filepath.Dir(file)
	cmd := exec.Command(exe)
	env := append(os.Environ(), "SCHED_DEMO="+mode)
	if mode == "schedtrace" {
		env = append(env, "GODEBUG=schedtrace=50")
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}
