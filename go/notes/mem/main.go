// mem 示例：对应 notes/mem.md
// 运行：go run ./mem
// 看逃逸分析：go build -gcflags='-m -l' ./mem 2>&1 | grep -E 'escapes|moved to heap|does not escape'
// 看栈增长：GODEBUG=... 见 mem.md 3.x；压测：go test -bench . -benchmem ./mem
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"strings"
	"unsafe"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	if mode := os.Getenv("MEM_DEMO"); mode != "" {
		runChild(mode)
		return
	}

	basicLayers()
	basicSizeClass()
	basicRounding()
	basicTiny()
	basicLargeObject()
	basicNoscan()

	escapeBasics()
	escapeInterface()
	escapeClosure()
	escapeSliceMap()

	stackGrowth()
	stackMove()
	stackOverflow()

	trapAllocCount()
	trapFragmentation()
}

// ---------------------------------------------------------------------------
// 1.1 三层分配器
// ---------------------------------------------------------------------------

func basicLayers() {
	section("1.1 分配器的三层结构")

	for _, row := range [][3]string{
		{"mcache", "每个 P 一个，无锁", "按 spanClass 存一批可分配的 mspan；小对象分配的快路径全在这里"},
		{"mcentral", "每个 spanClass 一个，全局", "partial/full 两组 spanSet，mcache 用完了来这里换一个 span"},
		{"mheap", "全局唯一，加锁", "管理所有 page；mcentral 没货了来这里申请，不够就找 OS 要"},
	} {
		fmt.Printf("  %-9s %-14s %s\n", row[0], row[1], row[2])
	}
	fmt.Println()
	fmt.Println("  分配路径: mallocgc -> mcache.alloc[spanClass] -> (空) mcentral.cacheSpan")
	fmt.Println("                                              -> (空) mheap.alloc -> sysAlloc(OS)")
	fmt.Printf("  常量: pageSize=%dKB heapArena=%dMB(amd64) NumSizeClasses=68 MaxSmallSize=32768\n",
		8, 64)
	fmt.Println("  mcache 在非 GC 内存里（sys.NotInHeap），所以它自己不会被 GC 扫描")
	fmt.Println("  1.26 新增 mcache.reusableNoscan：把回收的 noscan 小对象串成自由链表就地复用")
}

// ---------------------------------------------------------------------------
// 1.2 size class
// ---------------------------------------------------------------------------

var sizeClasses = []int{8, 16, 24, 32, 48, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224,
	240, 256, 288, 320, 352, 384, 416, 448, 480, 512, 576, 640, 704, 768, 896, 1024,
	1152, 1280, 1408, 1536, 1792, 2048, 2304, 2688, 3072, 3200, 3456, 4096, 4864, 5376,
	6144, 6528, 6784, 6912, 8192, 9472, 9728, 10240, 10880, 12288, 13568, 14336, 16384,
	18432, 19072, 20480, 21760, 24576, 27264, 28672, 32768}

func basicSizeClass() {
	section("1.2 size class（68 档）")

	fmt.Printf("前 20 档: %v\n", sizeClasses[:20])
	fmt.Printf("后 10 档: %v\n", sizeClasses[len(sizeClasses)-10:])
	fmt.Println("→ 8B 起步，小对象档位密（差 8/16 字节），大对象档位疏（最多浪费 ~12.5%）")
	fmt.Println("→ 一个 span 只放同一 size class 的对象，所以分配 = 从位图里找一个空位，O(1)")
	fmt.Println("→ 元数据: SizeClassToSize / SizeClassToNPages / SizeClassToDivMagic")
	fmt.Println("  (DivMagic 是为了用乘法+移位代替除法来算'对象在 span 内的下标')")
}

// ---------------------------------------------------------------------------
// 1.3 向上取整的真实代价
// ---------------------------------------------------------------------------

func roundUp(size int) int {
	for _, sc := range sizeClasses {
		if size <= sc {
			return sc
		}
	}
	return -1 // >32KB：按页取整
}

func basicRounding() {
	section("1.3 取整浪费")

	fmt.Printf("  %-8s %-10s %-10s %s\n", "请求", "实际占用", "浪费", "浪费率")
	for _, n := range []int{1, 8, 9, 17, 33, 65, 100, 129, 1000, 2000, 5000, 17000, 32769} {
		got := roundUp(n)
		if got < 0 {
			fmt.Printf("  %-8d %-10s %-10s %s\n", n, ">32KB", "按 8KB 页取整", "-")
			continue
		}
		fmt.Printf("  %-8d %-10d %-10d %.1f%%\n", n, got, got-n, float64(got-n)/float64(got)*100)
	}

	// 实测：分配 10 万个 33 字节的对象，看真实堆增长
	const n, want = 100000, 33
	runtime.GC()
	before := heapAlloc()
	keep := make([][]byte, 0, n)
	for range n {
		keep = append(keep, make([]byte, want))
	}
	after := heapAlloc()
	runtime.KeepAlive(keep)
	perObj := float64(after-before) / n
	fmt.Printf("\n实测 %d 个 %d 字节对象：平均每个占 %.1f 字节（size class = %d，另有 slice header 开销）\n",
		n, want, perObj, roundUp(want))
}

// ---------------------------------------------------------------------------
// 1.4 tiny allocator
// ---------------------------------------------------------------------------

func basicTiny() {
	section("1.4 tiny allocator（<16B 且无指针）")

	const n = 200000
	runtime.GC()

	// noscan 小对象：会被 tiny allocator 合并进同一个 16 字节块
	before := heapAlloc()
	tiny := make([]*int8, 0, n)
	for range n {
		v := int8(1)
		tiny = append(tiny, &v)
	}
	tinyCost := float64(heapAlloc()-before) / n
	runtime.KeepAlive(tiny)

	// 含指针的小对象：不能走 tiny，各自独立
	runtime.GC()
	before = heapAlloc()
	type withPtr struct{ p *int8 }
	ptrs := make([]*withPtr, 0, n)
	for range n {
		ptrs = append(ptrs, &withPtr{})
	}
	ptrCost := float64(heapAlloc()-before) / n
	runtime.KeepAlive(ptrs)

	fmt.Printf("  %d 个 *int8（1 字节 noscan）: 平均 %.1f B/个\n", n, tinyCost)
	fmt.Printf("  %d 个 struct{p *int8}（8 字节含指针）: 平均 %.1f B/个\n", n, ptrCost)
	fmt.Println("→ tiny allocator：把多个 <16B 的 noscan 对象塞进同一个 16 字节块（mcache.tiny/tinyoffset）")
	fmt.Println("→ 含指针的对象不能合并，否则一个对象存活就会拖住整块，且指针位图无法表达")
	fmt.Println("→ 副作用：tiny 块里只要一个对象活着，整块就不能回收")
}

// ---------------------------------------------------------------------------
// 1.5 大对象
// ---------------------------------------------------------------------------

var bigSink []byte // 用全局变量强制逃逸，否则常量长度的 make 可能留在栈上

func basicLargeObject() {
	section("1.5 大对象（>32KB）")

	measure := func(n int) uint64 {
		runtime.GC()
		before := heapAlloc()
		bigSink = make([]byte, n)
		return heapAlloc() - before
	}

	for _, n := range []int{32 << 10, 32<<10 + 1, 40 << 10, 64 << 10, 65<<10 + 1} {
		cost := measure(n)
		fmt.Printf("  make([]byte, %6d) -> 堆增长 %6d 字节（%.2f 页）\n", n, cost, float64(cost)/8192)
	}
	bigSink = nil

	// 对比：同样 40KB，但不逃逸的常量长度 make 会留在栈上
	runtime.GC()
	before := heapAlloc()
	stackBig := stackyBig()
	fmt.Printf("  不逃逸的 make([]byte, 40<<10)：堆增长 %d 字节（在栈上！）sum=%d\n",
		heapAlloc()-before, stackBig)

	fmt.Println("→ >32KB 绕过 mcache/mcentral，直接 mheap.alloc 按 8KB 页取整（spanClass 记为 large）")
	fmt.Println("→ 页的管理是 pageAlloc 基数树（1.14 引入，替代了老的 treap）")
	fmt.Println("→ 大对象每次分配都要加 mheap 的锁，高频分配大对象是典型的锁竞争点")
	fmt.Println("→ 注意：'大对象'说的是堆分配路径；只要长度是编译期常量且不逃逸，")
	fmt.Println("  即使 40KB 也可能被放在栈上（单个隐式栈对象上限约 64KB）")
}

//go:noinline
func stackyBig() int {
	s := make([]byte, 40<<10) // 常量长度、不逃逸
	s[0], s[len(s)-1] = 1, 2
	return int(s[0]) + int(s[len(s)-1])
}

// ---------------------------------------------------------------------------
// 1.6 noscan：有没有指针决定 GC 要不要扫
// ---------------------------------------------------------------------------

type noPtr struct {
	a, b, c, d int64
}

type hasPtr struct {
	a, b, c *int64
	d       int64
}

func basicNoscan() {
	section("1.6 noscan span")

	fmt.Printf("  noPtr  大小 %d，含指针 false -> 落在 noscan span，GC 根本不扫\n", unsafe.Sizeof(noPtr{}))
	fmt.Printf("  hasPtr 大小 %d，含指针 true  -> 落在 scan span，每次标记都要遍历它的指针字段\n", unsafe.Sizeof(hasPtr{}))
	fmt.Println("→ spanClass = sizeclass<<1 | noscan，所以 68 个 size class 对应 136 个 spanClass")
	fmt.Println("→ 1.22 起，对象 ≥ MinSizeForMallocHeader(512B) 的 scan 对象会在头部存一个类型指针")
	fmt.Println("  （malloc header），小于它的仍用 span 级别的位图，这是为了减小位图体积")
	fmt.Println("→ 工程含义：把 []*T 换成 []T 或 []int32 索引，能直接砍掉一大块 GC 标记时间")
}

// ---------------------------------------------------------------------------
// 2.1 逃逸分析基础
// ---------------------------------------------------------------------------

type point struct{ x, y int }

//go:noinline
func stackOnly() int { // 不逃逸：p 的地址没离开函数
	p := point{1, 2}
	return p.x + p.y
}

//go:noinline
func escapeReturn() *point { // 逃逸：返回局部变量地址
	p := point{1, 2}
	return &p
}

//go:noinline
func noEscapeParam(p *point) int { // 不逃逸：只读参数，没存起来
	return p.x
}

var global *point

//go:noinline
func escapeToGlobal(p *point) { // 逃逸：存进了全局变量
	global = p
}

func escapeBasics() {
	section("2.1 逃逸分析：四种基本情形")

	p := point{3, 4}
	fmt.Println("  stackOnly()      =", stackOnly(), "  // 栈上，0 alloc")
	fmt.Println("  escapeReturn()   =", *escapeReturn(), " // moved to heap: p")
	fmt.Println("  noEscapeParam(&p)=", noEscapeParam(&p), "  // p does not escape")
	escapeToGlobal(&p)
	fmt.Println("  escapeToGlobal(&p)     // &p escapes to heap")

	fmt.Println("→ 判定原则只有一条：编译器能否证明这个值的生命周期不超过函数栈帧")
	fmt.Println("→ 查看命令: go build -gcflags='-m -l' ./mem  （-l 关内联，结论更直白）")
	fmt.Println("→ 注意 -m 输出的两类信息：'escapes to heap'（值逃逸）/'moved to heap'（变量整体搬走）")
}

// ---------------------------------------------------------------------------
// 2.2 接口装箱导致的逃逸
// ---------------------------------------------------------------------------

//go:noinline
func viaInterface(v any) { _ = v }

//go:noinline
func viaConcrete(v int) { _ = v }

func escapeInterface() {
	section("2.2 接口与 fmt 导致的逃逸")

	fmt.Println("  func viaInterface(v any)  -> 实参装箱，v escapes to heap")
	fmt.Println("  func viaConcrete(v int)   -> 值传递，不逃逸")
	viaInterface(42)
	viaConcrete(42)

	fmt.Println("→ fmt.Println(x) 的参数是 ...any，所以任何传给它的东西都逃逸；")
	fmt.Println("  这也是为什么在热路径上加一行日志会让 benchmark 的 allocs/op 突然涨")
	fmt.Println("→ 小整数（0-255）和小字符串有 staticuint64s / 静态只读区兜底，不会真分配")
	fmt.Println("→ 逃逸的本质原因：接口的 data 字段是指针，值必须有一个可取地址的家（interface.md 2.3）")
}

// ---------------------------------------------------------------------------
// 2.3 闭包与 defer
// ---------------------------------------------------------------------------

//go:noinline
func closureNoEscape() int {
	n := 0
	f := func() { n++ } // 闭包不逃出函数 -> n 仍在栈上
	f()
	f()
	return n
}

//go:noinline
func closureEscape() func() int {
	n := 0
	return func() int { n++; return n } // 闭包逃出 -> n moved to heap
}

func escapeClosure() {
	section("2.3 闭包捕获")

	fmt.Println("  closureNoEscape() =", closureNoEscape(), "// 闭包没逃出去，捕获变量留在栈上")
	f := closureEscape()
	fmt.Println("  closureEscape()   =", f(), f(), "// n moved to heap")
	fmt.Println("→ 判据是'闭包本身会不会逃逸'，不是'有没有闭包'")
	fmt.Println("→ go func(){...}() 里捕获的变量一定逃逸（新 goroutine 的生命周期不受本栈帧约束）")
}

// ---------------------------------------------------------------------------
// 2.4 slice / map 的逃逸规则
// ---------------------------------------------------------------------------

//go:noinline
func fixedSizeSlice() int {
	s := make([]int, 64) // 常量长度且不逃逸 -> 栈上数组
	s[0] = 1
	return s[0]
}

//go:noinline
func dynamicSizeSlice(n int) int {
	s := make([]int, n) // 长度不是编译期常量 -> 必须堆分配
	s[0] = 1
	return s[0]
}

//go:noinline
func hugeSlice() int {
	s := make([]int, 10000) // 太大（>64KB 的隐式栈对象上限）-> 堆
	s[0] = 1
	return s[0]
}

func escapeSliceMap() {
	section("2.4 slice/map 的逃逸")

	fmt.Println("  make([]int, 64)     不逃逸 -> 栈上（编译期常量长度 + 不超上限）")
	fmt.Println("  make([]int, n)      逃逸   -> 长度未知，编译器不敢在栈上开")
	fmt.Println("  make([]int, 10000)  逃逸   -> 单个栈对象上限约 64KB（implicit variable too large）")
	_ = fixedSizeSlice()
	_ = dynamicSizeSlice(64)
	_ = hugeSlice()

	fmt.Println("→ append 扩容出来的新底层数组永远在堆上")
	fmt.Println("→ map 无论多小都在堆上（hmap 里有指针，且大小不定）")
	fmt.Println("→ 所以'预分配 + 复用'比'指望逃逸分析'靠谱得多")
}

// ---------------------------------------------------------------------------
// 3.1 栈增长
// ---------------------------------------------------------------------------

func recurse(depth int) uintptr {
	var pad [64]byte // 每层吃 64 字节，加速栈增长
	pad[0] = byte(depth)
	if depth == 0 {
		return uintptr(unsafe.Pointer(&pad[0]))
	}
	return recurse(depth-1) + uintptr(pad[0])&0
}

func stackGrowth() {
	section("3.1 栈增长")

	fmt.Println("  stackMin = 2048（Go 代码最小栈），实际首次分配 fixedStack = 2048（amd64，向上取整到 2 的幂）")
	fmt.Println("  maxstacksize = 1GB（64 位）/ 250MB（32 位），可用 debug.SetMaxStack 改")
	fmt.Println("  增长方式：morestack -> newstack -> newsize = oldsize*2（不够就继续翻倍）-> copystack")

	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	fmt.Printf("\n  当前 StackInuse=%.0fKB StackSys=%.0fKB（%d 个 goroutine）\n",
		float64(s.StackInuse)/1024, float64(s.StackSys)/1024, runtime.NumGoroutine())

	// 起一批深递归的 goroutine，观察栈总量
	done := make(chan struct{})
	for range 100 {
		go func() { _ = recurse(2000); <-done }()
	}
	runtime.Gosched()
	runtime.ReadMemStats(&s)
	fmt.Printf("  100 个深递归 goroutine 后: StackInuse=%.0fKB（%d 个 goroutine）\n",
		float64(s.StackInuse)/1024, runtime.NumGoroutine())
	close(done)

	fmt.Println("  → 栈内存也是从 mheap 拿的（stackpool/stackcache 分级缓存），会计入 StackSys")
	fmt.Println("  → GC 时会 shrinkstack：使用量 < 1/4 就把栈缩一半（也是靠 copystack）")
}

// ---------------------------------------------------------------------------
// 3.2 栈是会被搬走的：局部变量地址会变
// ---------------------------------------------------------------------------

func stackMove() {
	section("3.2 copystack：局部变量地址会变")

	var addrs []uintptr
	var probe func(depth int) uintptr
	probe = func(depth int) uintptr {
		var pad [128]byte
		pad[0] = byte(depth)
		addr := uintptr(unsafe.Pointer(&pad[0]))
		if depth%40 == 0 {
			addrs = append(addrs, addr)
		}
		if depth == 0 {
			return addr
		}
		return probe(depth-1) + uintptr(pad[0])&0
	}
	go func() { probe(400) }()
	runtime.Gosched()
	for range 10 {
		runtime.Gosched()
	}

	jumps := 0
	for i := 1; i < len(addrs); i++ {
		// 正常递归每层地址差是固定的（约 128+ 字节），栈被拷走时会出现巨大跳变
		d := int64(addrs[i]) - int64(addrs[i-1])
		if d < -1<<16 || d > 1<<16 {
			jumps++
		}
	}
	fmt.Printf("  递归 400 层，采样 %d 个局部变量地址，检测到 %d 次大跳变（= 栈被整体拷贝搬家）\n",
		len(addrs), jumps)
	fmt.Println("  → 这就是 Go 不允许把 Go 指针长期交给 C 保存的原因之一：栈上对象地址不稳定")
	fmt.Println("  → copystack 会遍历栈帧，按 funcdata 里的指针位图逐个调整指针（所以栈必须精确可扫描）")
	fmt.Println("  → 也是 unsafe.Pointer 转 uintptr 后不能存起来的原因（uintptr 不会被 copystack 修正）")
}

// ---------------------------------------------------------------------------
// 3.3 栈溢出
// ---------------------------------------------------------------------------

func stackOverflow() {
	section("3.3 栈溢出（子进程）")

	out, _ := selfRun("overflow")
	// 栈溢出的 panic 输出有几十万帧，只看前 3 行
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i, l := range lines {
		if i >= 3 {
			fmt.Printf("  ...（后面还有 %d 行栈帧，其中一句是 \"...349379 frames elided...\"）\n", len(lines)-3)
			break
		}
		fmt.Println("  " + l)
	}
	fmt.Println("  → 超过 maxstacksize 直接 throw(\"stack overflow\")，是 fatal error，无法 recover")
	fmt.Println("  → debug.SetMaxStack 可以调小它来快速失败（这里子进程设成了 8MB）")
}

func runChild(mode string) {
	switch mode {
	case "overflow":
		debug.SetMaxStack(8 << 20)
		var f func(int) int
		f = func(n int) int {
			var pad [1024]byte
			pad[0] = byte(n)
			return f(n+1) + int(pad[0])&0
		}
		fmt.Println(f(0))
	}
}

func selfRun(mode string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "MEM_DEMO="+mode)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ---------------------------------------------------------------------------
// 4.1 分配次数 vs 分配字节数
// ---------------------------------------------------------------------------

func trapAllocCount() {
	section("4.1 分配次数才是关键")

	const total = 1 << 20 // 都是 1MB

	runtime.GC()
	c0 := allocCount()
	var many [][]byte
	for range 1 << 17 { // 13 万个 8 字节
		many = append(many, make([]byte, 8))
	}
	manyCost := allocCount() - c0
	runtime.KeepAlive(many)

	runtime.GC()
	c0 = allocCount()
	one := make([]byte, total) // 1 个 1MB
	oneCost := allocCount() - c0
	runtime.KeepAlive(one)

	fmt.Printf("  131072 个 8 字节 []byte: %d 次分配（比对象数还少：tiny allocator 合并了）\n", manyCost)
	fmt.Printf("  1 个 1MB []byte:        %d 次分配\n", oneCost)
	fmt.Println("→ 标记成本 ∝ 对象个数 + 指针个数，不是字节数（gc.md 3.10）")
	fmt.Println("→ pprof 要看 -alloc_objects（次数），不是只看 -alloc_space（字节）")
}

// ---------------------------------------------------------------------------
// 4.2 碎片
// ---------------------------------------------------------------------------

func trapFragmentation() {
	section("4.2 碎片与'内存不降'")

	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	fmt.Printf("  HeapInuse=%.1fMB HeapIdle=%.1fMB HeapReleased=%.1fMB HeapSys=%.1fMB\n",
		mbF(s.HeapInuse), mbF(s.HeapIdle), mbF(s.HeapReleased), mbF(s.HeapSys))
	fmt.Printf("  内部碎片指标: HeapInuse - HeapAlloc = %.1fMB（span 里已分配但对象未用满的部分）\n",
		mbF(s.HeapInuse-s.HeapAlloc))

	fmt.Println("→ 内部碎片：size class 取整的浪费（1.3 实测最坏 ~12.5%）")
	fmt.Println("→ 外部碎片：span 只服务一个 size class，某个 class 的空 span 不能直接给别的 class 用")
	fmt.Println("   （span 被释放回 mheap 之后才能重新切成别的 class）")
	fmt.Println("→ Go 是非移动 GC，不做内存整理，所以碎片只能靠 size class 设计和复用来控制")

	fmt.Println("\n有用的 GODEBUG（都在 runtime/extern.go 里有官方说明）：")
	for _, row := range [][2]string{
		{"gctrace=1", "每轮 GC 一行日志"},
		{"scavtrace=1", "归还 OS 的节奏"},
		{"madvdontneed=0", "改用 MADV_FREE（RSS 降得更慢）"},
		{"harddecommit=1", "归还时真正 decommit，用来暴露 use-after-free"},
		{"clobberfree=1", "释放对象时用垃圾值填充，抓悬垂指针"},
		{"efence=1", "每个对象独占一页，抓越界（极慢）"},
		{"invalidptr=1", "默认开：发现非法指针值就崩，别关"},
		{"inittrace=1", "每个包 init 的耗时和分配量"},
	} {
		fmt.Printf("  %-16s %s\n", row[0], row[1])
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func heapAlloc() uint64 {
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	return s.HeapAlloc
}

func allocCount() uint64 {
	sample := []metrics.Sample{{Name: "/gc/heap/allocs:objects"}}
	metrics.Read(sample)
	return sample[0].Value.Uint64()
}

func mbF[T uint64 | int64](v T) float64 { return float64(v) / (1 << 20) }

// stringBuilder 是 strings.Builder 的别名，仅为 bench_test.go 少一个 import
type stringBuilder = strings.Builder
