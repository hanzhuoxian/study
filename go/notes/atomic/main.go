// atomic 与内存模型示例：对应 notes/atomic.md
// 运行：go run ./atomic
// 竞态检测：go run -race ./atomic   （会报出 3.1 里故意留的 data race）
// 压测：go test -bench . -benchmem ./atomic
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	if mode := os.Getenv("ATOMIC_DEMO"); mode != "" {
		runChild(mode)
		return
	}

	basicTypes()
	basicOps()
	basicPointer()
	basicValue()
	basicAndOr()

	memoryModelBasics()
	seqCst()
	casLoop()
	aba()

	trapNonAtomicMix()
	trapAlign()
	trapMultiVar()
	trapValueRules()
	trapRaceDetector()
	trapBusyWait()
}

// ---------------------------------------------------------------------------
// 1.1 类型化 atomic（1.19+）：优先用这些，别用函数版
// ---------------------------------------------------------------------------

type counters struct {
	requests atomic.Int64
	errs     atomic.Int64
	ok       atomic.Bool
	name     atomic.Pointer[string]
}

func basicTypes() {
	section("1.1 类型化 atomic（1.19+）")

	var c counters
	c.requests.Add(1)
	c.errs.Store(2)
	c.ok.Store(true)
	s := "svc-a"
	c.name.Store(&s)

	fmt.Printf("  requests=%d errs=%d ok=%v name=%s\n",
		c.requests.Load(), c.errs.Load(), c.ok.Load(), *c.name.Load())

	fmt.Println("  可用类型: Int32 Int64 Uint32 Uint64 Uintptr Bool Pointer[T] Value")
	fmt.Println("→ 相比函数版 atomic.AddInt64(&x, 1) 的三个好处：")
	fmt.Println("  ① 不可能忘记用原子操作去读写它（字段是未导出的）")
	fmt.Println("  ② 自动保证 64 位对齐（32 位平台上函数版要自己保证，见 3.2）")
	fmt.Println("  ③ 带 noCopy 语义：go vet 能发现拷贝")
}

// ---------------------------------------------------------------------------
// 1.2 五种操作
// ---------------------------------------------------------------------------

func basicOps() {
	section("1.2 五种基本操作")

	var v atomic.Int64

	v.Store(10)
	fmt.Println("  Store(10) -> Load() =", v.Load())
	fmt.Println("  Add(5)              =", v.Add(5), "（返回新值）")
	fmt.Println("  Swap(100)           =", v.Swap(100), "（返回旧值）")
	fmt.Println("  CompareAndSwap(100, 7) =", v.CompareAndSwap(100, 7), "-> Load() =", v.Load())
	fmt.Println("  CompareAndSwap(100, 8) =", v.CompareAndSwap(100, 8), "（旧值不匹配，失败）")
	fmt.Println("  Add(-3) 就是减法      =", v.Add(-3))
	fmt.Println("→ 只有这五种（Load/Store/Add/Swap/CompareAndSwap）+ 1.23 的 And/Or")
	fmt.Println("→ 没有原子的乘法/除法/浮点加，需要的话自己写 CAS 循环（见 2.3）")
}

// ---------------------------------------------------------------------------
// 1.3 atomic.Pointer[T]：类型安全的原子指针
// ---------------------------------------------------------------------------

type config struct {
	Timeout time.Duration
	Retries int
}

var cfg atomic.Pointer[config]

func basicPointer() {
	section("1.3 atomic.Pointer[T]")

	cfg.Store(&config{Timeout: time.Second, Retries: 3})

	// 热更新配置：整体替换指针，读者永远看到自洽的一份
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for range 4 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					c := cfg.Load() // 一次原子读，拿到完整快照
					_ = c.Timeout + time.Duration(c.Retries)
				}
			}
		})
	}
	for i := range 3 {
		cfg.Store(&config{Timeout: time.Duration(i) * time.Second, Retries: i})
		time.Sleep(time.Millisecond)
	}
	close(stop)
	wg.Wait()

	c := cfg.Load()
	fmt.Printf("  最终配置: %+v\n", *c)
	fmt.Println("→ 这是'多字段一致读'的标准解法：把它们打包成 struct，用一个原子指针整体换")
	fmt.Println("→ 千万别在原地改 *c 的字段（那就又变成数据竞争了）：每次都换一个新对象")
	fmt.Println("→ 相比 atomic.Value：编译期类型安全、没有 any 装箱、也不会出现类型不一致 panic")
}

// ---------------------------------------------------------------------------
// 1.4 atomic.Value：1.19 之前的通用方案
// ---------------------------------------------------------------------------

func basicValue() {
	section("1.4 atomic.Value")

	var v atomic.Value
	fmt.Println("  零值 Load() =", v.Load(), "（返回 nil）")

	v.Store(config{Timeout: time.Second})
	fmt.Printf("  Store 值类型也行: %+v\n", v.Load().(config))

	func() {
		defer func() { fmt.Println("  存不同类型 -> panic:", recover()) }()
		v.Store("string") // ✗ 类型必须前后一致
	}()

	func() {
		defer func() { fmt.Println("  存 nil -> panic:", recover()) }()
		v.Store(nil) // ✗ 不能存 nil
	}()

	fmt.Println("→ 三条限制：类型必须一致、不能存 nil、Store 之后不能拷贝")
	fmt.Println("→ 1.19 之后基本可以退休了：用 atomic.Pointer[T]")
}

// ---------------------------------------------------------------------------
// 1.5 And / Or（1.23+）：原子位操作
// ---------------------------------------------------------------------------

const (
	flagReady uint32 = 1 << 0
	flagBusy  uint32 = 1 << 1
	flagDone  uint32 = 1 << 2
)

func basicAndOr() {
	section("1.5 And / Or（1.23+）")

	var flags atomic.Uint32

	old := flags.Or(flagReady) // 置位，返回旧值
	fmt.Printf("  Or(ready):  old=%03b new=%03b\n", old, flags.Load())
	old = flags.Or(flagBusy)
	fmt.Printf("  Or(busy):   old=%03b new=%03b\n", old, flags.Load())
	old = flags.And(^flagBusy) // 清位
	fmt.Printf("  And(^busy): old=%03b new=%03b\n", old, flags.Load())

	fmt.Println("→ 1.23 之前这类需求只能写 CAS 循环，现在一条指令（amd64 的 LOCK AND/OR）")
	fmt.Println("→ 注意返回的是旧值，和 Add 返回新值不一致，很容易记错")
}

// ---------------------------------------------------------------------------
// 2.1 内存模型：happens-before / synchronizes before
// ---------------------------------------------------------------------------

func memoryModelBasics() {
	section("2.1 Go 内存模型（1.19 正式版）")

	fmt.Println("  官方定义（go.dev/ref/mem）：")
	fmt.Println("    如果原子操作 A 的效果被原子操作 B 观察到，则 A \"synchronizes before\" B；")
	fmt.Println("    且程序中所有原子操作表现得像按某个顺序一致（sequentially consistent）的顺序执行。")
	fmt.Println("    → 等价于 C++ 的 seq_cst 原子和 Java 的 volatile。")
	fmt.Println()
	fmt.Println("  建立 happens-before 的手段一共就这几种：")
	for _, row := range [][2]string{
		{"go 语句", "go f() 之前的写，f 里能看到"},
		{"channel 发送/接收", "发送 happens-before 对应的接收完成（chan.md 2.9）"},
		{"channel close", "close happens-before 从已关闭通道收到零值"},
		{"Mutex Unlock/Lock", "第 n 次 Unlock happens-before 第 m 次 Lock（n<m）"},
		{"RWMutex", "Unlock -> RLock；RUnlock -> 下一次 Lock"},
		{"WaitGroup", "Done 全部完成 happens-before Wait 返回"},
		{"Once.Do", "f() 返回 happens-before 任何 Do 的返回"},
		{"atomic 操作", "被观察到的原子写 synchronizes before 观察到它的原子读"},
	} {
		fmt.Printf("    %-22s %s\n", row[0], row[1])
	}
	fmt.Println()
	fmt.Println("  ⚠️ 没有 happens-before 关系的并发读写 = data race = 未定义行为")
	fmt.Println("     Go 不像 Java 那样给 race 定义'至少读到某个旧值'，它是真·UB")
}

// ---------------------------------------------------------------------------
// 2.2 顺序一致性：atomic 不只是"不撕裂"
// ---------------------------------------------------------------------------

func seqCst() {
	section("2.2 顺序一致性（不只是防撕裂）")

	// 经典的 store buffer 测试（Dekker）：如果没有顺序一致性，两个 r 都可能是 0
	var x, y atomic.Int32
	both0 := 0
	const rounds = 20000

	for range rounds {
		x.Store(0)
		y.Store(0)
		var r1, r2 int32
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); x.Store(1); r1 = y.Load() }()
		go func() { defer wg.Done(); y.Store(1); r2 = x.Load() }()
		wg.Wait()
		if r1 == 0 && r2 == 0 {
			both0++
		}
	}
	fmt.Printf("  atomic 版本跑 %d 轮，两边同时读到 0 的次数: %d（顺序一致性保证为 0）\n", rounds, both0)
	fmt.Println("→ 如果用普通变量 + 编译器/CPU 重排，理论上可以两边都读到 0（x86 的 store buffer）")
	fmt.Println("→ atomic 除了'不撕裂'，还插入了必要的内存屏障（amd64 上 Store 是 XCHG，自带 full barrier）")
	fmt.Println("→ Go 只提供 seq_cst 一种强度，没有 C++ 的 relaxed/acquire/release 可选")
	fmt.Println("  （提案 golang/go#68578 讨论过 weak atomics，至今未进；理由是太容易写错）")
}

// ---------------------------------------------------------------------------
// 2.3 CAS 循环：实现 atomic 没提供的操作
// ---------------------------------------------------------------------------

// 原子地把 v 更新为 max(v, n)
func atomicMax(v *atomic.Int64, n int64) {
	for {
		old := v.Load()
		if old >= n {
			return
		}
		if v.CompareAndSwap(old, n) {
			return
		}
		// CAS 失败说明别人改了，重新读再试
	}
}

// 原子浮点加：借 Uint64 存 float64 的 bit pattern
func atomicAddFloat(v *atomic.Uint64, delta float64) float64 {
	for {
		oldBits := v.Load()
		old := float64FromBits(oldBits)
		newVal := old + delta
		if v.CompareAndSwap(oldBits, float64ToBits(newVal)) {
			return newVal
		}
	}
}

func float64ToBits(f float64) uint64   { return *(*uint64)(unsafe.Pointer(&f)) }
func float64FromBits(b uint64) float64 { return *(*float64)(unsafe.Pointer(&b)) }

func casLoop() {
	section("2.3 CAS 循环")

	var maxV atomic.Int64
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() { atomicMax(&maxV, int64(i*7%100)) })
	}
	wg.Wait()
	fmt.Println("  100 个 goroutine 并发求 max:", maxV.Load())

	var f atomic.Uint64
	wg = sync.WaitGroup{}
	for range 100 {
		wg.Go(func() { atomicAddFloat(&f, 0.5) })
	}
	wg.Wait()
	fmt.Println("  100 次并发 +0.5:", float64FromBits(f.Load()))

	fmt.Println("→ CAS 循环的三段式：读旧值 -> 算新值 -> CAS，失败就重来")
	fmt.Println("→ 高竞争下 CAS 循环会疯狂重试（活锁风险），这时候锁反而更快（见 bench）")
	fmt.Println("→ 标准库自己也这么干：math/rand、runtime 的很多计数器")
}

// ---------------------------------------------------------------------------
// 2.4 ABA 问题
// ---------------------------------------------------------------------------

func aba() {
	section("2.4 ABA 问题")

	fmt.Println("  场景：CAS 只比较值，不知道'中间被改过又改回来了'")
	fmt.Println("    T1: 读到 head=A，准备 CAS(A -> B)")
	fmt.Println("    T2: pop A、pop B、又 push A（A 已经是另一块内存/另一种含义）")
	fmt.Println("    T1: CAS(A -> B) 成功——但语义已经错了")
	fmt.Println()
	fmt.Println("  Go 里为什么很少中招：")
	fmt.Println("    ① 有 GC：只要还有指针引用，节点内存就不会被复用成别的对象")
	fmt.Println("    ② 无锁数据结构在业务代码里本来就少见")
	fmt.Println("  但仍会出现在'带版本语义'的场合，解法是加版本号：")
	fmt.Println("    把 (指针, 计数) 打包成一个 struct 用 atomic.Pointer 换，或者用 64 位里的高位存版本")

	// 带版本号的状态机
	type state struct {
		version uint64
		value   string
	}
	var st atomic.Pointer[state]
	st.Store(&state{version: 1, value: "A"})

	cur := st.Load()
	next := &state{version: cur.version + 1, value: "B"}
	ok := st.CompareAndSwap(cur, next)
	fmt.Printf("\n  带版本的 CAS: ok=%v -> version=%d value=%s\n",
		ok, st.Load().version, st.Load().value)
}

// ---------------------------------------------------------------------------
// 3.1 原子写 + 普通读 = 竞态
// ---------------------------------------------------------------------------

func trapNonAtomicMix() {
	section("3.1 混用原子和普通访问")

	fmt.Println("  ✗ 一半原子一半不原子，等于没有原子：")
	fmt.Println("      atomic.AddInt64(&n, 1)   // 一个 goroutine")
	fmt.Println("      fmt.Println(n)           // 另一个 goroutine 直接读 -> data race")
	fmt.Println("  ✓ 用类型化 atomic 从根上避免：atomic.Int64 的字段是未导出的，")
	fmt.Println("    你**没有办法**绕过 Load/Store 直接访问它")

	out, _ := selfRun("race")
	if strings.Contains(out, "DATA RACE") {
		fmt.Println("  子进程（-race 编译）确认检测到 DATA RACE ✓")
	} else {
		fmt.Println("  子进程输出:", firstLine(out))
	}
}

// ---------------------------------------------------------------------------
// 3.2 32 位平台的 64 位对齐
// ---------------------------------------------------------------------------

type misaligned struct {
	flag  bool  // 1 字节
	count int64 // 32 位平台上偏移 8？不一定
	_     uint32
}

func trapAlign() {
	section("3.2 64 位对齐（32 位平台）")

	fmt.Printf("  本机是 %s，指针 %d 字节，所以不会踩到这个坑\n", runtime.GOARCH, unsafe.Sizeof(uintptr(0)))
	fmt.Printf("  misaligned.count 的偏移 = %d\n", unsafe.Offsetof(misaligned{}.count))
	fmt.Println("  在 386 / arm / 32 位 mips 上：")
	fmt.Println("    · 函数版 atomic.AddInt64(&s.count, 1) 要求地址 8 字节对齐，否则运行时 panic")
	fmt.Println("    · 文档承诺'分配对象/全局变量/切片的第一个字'是 8 字节对齐的")
	fmt.Println("    · 老代码的经典 hack：把 int64 字段放结构体第一位，或者加 padding")
	fmt.Println("  → 用 atomic.Int64 类型就完全不用操心：它内部自带 align64 字段")
}

// ---------------------------------------------------------------------------
// 3.3 多个原子变量之间没有一致性
// ---------------------------------------------------------------------------

func trapMultiVar() {
	section("3.3 多个原子变量 ≠ 一致快照")

	var total, count atomic.Int64
	total.Store(100)
	count.Store(10)

	fmt.Println("  ✗ avg := total.Load() / count.Load()")
	fmt.Println("    两次 Load 之间别人可能改了任意一个，算出来的不是任何时刻的真实平均值")
	fmt.Println("  ✓ 三种解法：")
	fmt.Println("    ① 用锁保护这一组变量")
	fmt.Println("    ② 打包进 struct，用 atomic.Pointer[T] 整体替换（见 1.3）")
	fmt.Println("    ③ 塞进同一个 64 位里（比如高 32 位存 count、低 32 位存 total，")
	fmt.Println("       WaitGroup 就是这么干的，见 sync.md 2.6）")

	// 演示 ③
	var packed atomic.Uint64
	pack := func(hi, lo uint32) uint64 { return uint64(hi)<<32 | uint64(lo) }
	unpack := func(v uint64) (uint32, uint32) { return uint32(v >> 32), uint32(v) }
	packed.Store(pack(10, 100))
	packed.Add(pack(1, 5)) // 一次原子操作同时更新两个计数
	c, t := unpack(packed.Load())
	fmt.Printf("    ③ 实测: count=%d total=%d（一次 Add 同时改两个）\n", c, t)
}

// ---------------------------------------------------------------------------
// 3.4 atomic.Value 的三条规则
// ---------------------------------------------------------------------------

func trapValueRules() {
	section("3.4 atomic.Value 的坑")

	fmt.Println("  ① 类型必须前后一致 -> 否则 panic: sync/atomic: store of inconsistently typed value")
	fmt.Println("  ② 不能 Store(nil)  -> panic: sync/atomic: store of nil value into Value")
	fmt.Println("     想表达'空'就存一个 typed nil：v.Store((*T)(nil))")
	fmt.Println("  ③ Store 之后不能拷贝（内部存的是 (typ, data) 两个字，拷贝会破坏一致性）")
	fmt.Println("  ④ CompareAndSwap 要求值可比较，存 slice/map/func 会 panic")

	var v atomic.Value
	v.Store((*config)(nil))
	fmt.Printf("  typed nil 可以存: %v (%T)\n", v.Load(), v.Load())
}

// ---------------------------------------------------------------------------
// 3.5 race detector 能做什么、不能做什么
// ---------------------------------------------------------------------------

func trapRaceDetector() {
	section("3.5 race detector")

	fmt.Println("  原理：ThreadSanitizer 的 vector clock。每个内存字（8 字节粒度）记录")
	fmt.Println("        最近 4 次访问的 (goroutine, 时钟, 读/写)，新访问来了就比较是否 happens-before。")
	fmt.Println()
	fmt.Println("  能做：只要竞态**真的发生了**（两个访问都执行到了），基本 100% 报出来，几乎无误报")
	fmt.Println("  不能做：")
	fmt.Println("    · 没走到的代码路径检测不到 —— 它是动态分析，不是静态分析")
	fmt.Println("    · 内存 ~5-10x、CPU ~2-20x 开销，不能在生产常开")
	fmt.Println("    · 默认只记录最近 4 个访问、8192 个 goroutine（GORACE 可调）")
	fmt.Println("    · 检测不到'逻辑竞态'（比如 3.3 的两次 Load，每次都是原子的，但组合起来是错的）")
	fmt.Println()
	fmt.Println("  实践：CI 里 go test -race ./... 是底线；GORACE=\"halt_on_error=1\" 让它第一时间失败")
	fmt.Printf("  当前 GORACE=%q\n", os.Getenv("GORACE"))
}

// ---------------------------------------------------------------------------
// 3.6 自旋等待
// ---------------------------------------------------------------------------

func trapBusyWait() {
	section("3.6 用 atomic 做自旋等待")

	fmt.Println("  ✗ for !done.Load() {}            // 死等，吃满一个核，还可能永远看不到更新")
	fmt.Println("  ✗ for !done.Load() { runtime.Gosched() }   // 稍好，但仍在烧 CPU")
	fmt.Println("  ✓ 用 channel / sync.WaitGroup / sync.Cond 让 goroutine 真正挂起")

	// 正确写法：channel 通知
	var done atomic.Bool
	ch := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Millisecond)
		done.Store(true)
		close(ch) // 通知
	}()
	<-ch
	fmt.Println("  channel 版本：goroutine 被挂起，0 CPU，done =", done.Load())
	fmt.Println("→ Go 没有 runtime.Pause()/spin 提示给用户代码；runtime 自己的自旋（procyield）不对外")
}

// ---------------------------------------------------------------------------
// 子进程：故意制造 data race
// ---------------------------------------------------------------------------

func runChild(mode string) {
	switch mode {
	case "race":
		var n int64
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 1000 {
				atomic.AddInt64(&n, 1)
			}
		}()
		go func() {
			defer wg.Done()
			for range 1000 {
				_ = n
			}
		}() // ✗ 普通读
		wg.Wait()
		fmt.Println("n =", atomic.LoadInt64(&n))
	}
}

// selfRun 用 go run -race 重新编译本包跑一个子进程。
// 包目录用 runtime.Caller 取，这样不依赖调用时的工作目录。
func selfRun(mode string) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	cmd := exec.Command("go", "run", "-race", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "ATOMIC_DEMO="+mode)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
