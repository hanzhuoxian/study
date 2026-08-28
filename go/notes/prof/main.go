// pprof / trace 示例：对应 notes/profile.md
//
//	go run ./prof                 跑全部演示，把 profile 写到 notes/prof/out/
//	go run ./prof serve           起 HTTP pprof 端点（:6060），配合 go tool pprof 在线抓
//	go run ./prof cpu             只生成 CPU profile
//
// 抓完之后：
//
//	go tool pprof -top -nodecount=10 prof/out/cpu.pprof
//	go tool pprof -http=:8080 prof/out/cpu.pprof
//	go tool trace prof/out/trace.out
package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	_ "net/http/pprof" // 注册 /debug/pprof/* 到 DefaultServeMux
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	mode := "all"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "serve":
		serveHTTP()
	case "cpu":
		must(writeCPUProfile(2 * time.Second))
	case "all":
		runAll()
	default:
		fmt.Println("usage: go run ./prof [all|cpu|serve]")
	}
}

// ---------------------------------------------------------------------------
// 输出目录
// ---------------------------------------------------------------------------

func outDir() string {
	_, file, _, _ := runtimeCaller()
	dir := filepath.Join(filepath.Dir(file), "out")
	must(os.MkdirAll(dir, 0o755))
	return dir
}

func runtimeCaller() (uintptr, string, int, bool) { return runtime.Caller(1) }

func outPath(name string) string { return filepath.Join(outDir(), name) }

// ---------------------------------------------------------------------------
// 1. CPU profile
// ---------------------------------------------------------------------------

// 故意写慢的三个函数，各自代表一类典型热点。
// 拆成独立函数是为了在 profile 里能一眼分清谁贵——
// 这也是实践建议：**别把不同性质的工作塞进一个大函数**，否则 profile 只能告诉你"这个函数很慢"。

// 热点 A：分配密集（O(n²) 字符串拼接）—— 在 profile 里表现为 GC + memmove
func concatSlow(n int) string {
	s := ""
	for i := range n {
		s += strconv.Itoa(i % 10)
	}
	return s
}

// 热点 B：排序 —— 典型的 O(n log n) + 大量比较回调
func sortSlow(s string) []string {
	words := strings.Split(s, "")
	sort.Strings(words)
	return words
}

// 热点 C：纯 CPU（哈希）—— 在 profile 里 flat 很高、几乎不分配
func hashSlow(data []byte, rounds int) string {
	h := sha256.New()
	for range rounds {
		h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

var hashBuf = make([]byte, 8<<10)

func slowWork(n int) string {
	s := concatSlow(n)
	words := sortSlow(s)
	_ = words
	return hashSlow(hashBuf, 400)
}

func writeCPUProfile(d time.Duration) error {
	f, err := os.Create(outPath("cpu.pprof"))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		return err
	}
	defer pprof.StopCPUProfile()

	deadline := time.Now().Add(d)
	var sink string
	for time.Now().Before(deadline) {
		sink = slowWork(500)
	}
	_ = sink
	fmt.Println("  写出", outPath("cpu.pprof"))
	return nil
}

// ---------------------------------------------------------------------------
// 2. heap profile：制造一个"看起来像泄漏"的场景
// ---------------------------------------------------------------------------

// 全局 cache 只增不删 —— 最常见的内存泄漏形态
var cache = struct {
	sync.Mutex
	m map[string][]byte
}{m: map[string][]byte{}}

func leakMemory(n int) {
	for i := range n {
		key := "key-" + strconv.Itoa(i)
		cache.Lock()
		cache.m[key] = make([]byte, 4096) // 每条 4KB
		cache.Unlock()
	}
}

func writeHeapProfile() error {
	leakMemory(20000) // 约 80MB

	f, err := os.Create(outPath("heap.pprof"))
	if err != nil {
		return err
	}
	defer f.Close()

	runtime.GC() // 官方建议：抓 heap profile 前先 GC，去掉待回收对象的噪声
	if err := pprof.WriteHeapProfile(f); err != nil {
		return err
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	fmt.Printf("  写出 %s（此时 HeapAlloc=%.0fMB，cache 里 %d 条）\n",
		outPath("heap.pprof"), float64(ms.HeapAlloc)/(1<<20), len(cache.m))
	return nil
}

// ---------------------------------------------------------------------------
// 3. goroutine profile：制造泄漏的 goroutine
// ---------------------------------------------------------------------------

func leakGoroutines(n int) {
	for range n {
		// 经典泄漏：往无人接收的 channel 发送
		ch := make(chan int)
		go func() { ch <- 1 }()

		// 另一种：等一个永远不会关闭的 channel
		never := make(chan struct{})
		go func() { <-never }()
	}
}

func writeGoroutineProfile() error {
	before := runtime.NumGoroutine()
	leakGoroutines(500)
	time.Sleep(50 * time.Millisecond)

	f, err := os.Create(outPath("goroutine.pprof"))
	if err != nil {
		return err
	}
	defer f.Close()

	// debug=1 输出人可读的文本（带栈聚合），debug=0 是 protobuf 二进制
	if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
		return err
	}

	txt, err := os.Create(outPath("goroutine.txt"))
	if err != nil {
		return err
	}
	defer txt.Close()
	if err := pprof.Lookup("goroutine").WriteTo(txt, 1); err != nil {
		return err
	}

	fmt.Printf("  写出 %s / goroutine.txt（goroutine 数 %d -> %d）\n",
		outPath("goroutine.pprof"), before, runtime.NumGoroutine())
	return nil
}

// ---------------------------------------------------------------------------
// 4. mutex / block profile：默认关闭，必须显式打开
// ---------------------------------------------------------------------------

func writeContentionProfiles() error {
	runtime.SetMutexProfileFraction(1) // 采样比例，1 = 全采
	runtime.SetBlockProfileRate(1)     // 纳秒阈值，1 = 全采（生产上要调大）
	defer func() {
		runtime.SetMutexProfileFraction(0)
		runtime.SetBlockProfileRate(0)
	}()

	// 制造锁竞争
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) * 4 {
		wg.Go(func() {
			for range 2000 {
				mu.Lock()
				for range 200 { // 临界区里做点活，放大竞争
					_ = 1
				}
				mu.Unlock()
			}
		})
	}

	// 制造 channel 阻塞
	ch := make(chan int)
	for range 8 {
		wg.Go(func() {
			for range 500 {
				ch <- 1
			}
		})
	}
	wg.Go(func() {
		for range 4000 {
			<-ch
			time.Sleep(time.Microsecond)
		}
	})
	wg.Wait()

	for _, name := range []string{"mutex", "block"} {
		f, err := os.Create(outPath(name + ".pprof"))
		if err != nil {
			return err
		}
		if err := pprof.Lookup(name).WriteTo(f, 0); err != nil {
			f.Close()
			return err
		}
		f.Close()
		fmt.Println("  写出", outPath(name+".pprof"))
	}
	return nil
}

// ---------------------------------------------------------------------------
// 5. execution trace
// ---------------------------------------------------------------------------

func writeTrace(d time.Duration) error {
	f, err := os.Create(outPath("trace.out"))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := trace.Start(f); err != nil {
		return err
	}
	defer trace.Stop()

	// 用 region/task 给 trace 加业务语义标注
	ctx, task := trace.NewTask(context.Background(), "demo-request")
	defer task.End()

	var wg sync.WaitGroup
	deadline := time.Now().Add(d)
	for i := range 4 {
		wg.Go(func() {
			for time.Now().Before(deadline) {
				trace.WithRegion(ctx, "compute-"+strconv.Itoa(i), func() {
					_ = slowWork(300)
				})
				time.Sleep(time.Millisecond) // 制造 goroutine 阻塞/唤醒
			}
		})
	}
	wg.Wait()
	fmt.Println("  写出", outPath("trace.out"), "（go tool trace 打开）")
	return nil
}

// ---------------------------------------------------------------------------
// 6. HTTP pprof 端点
// ---------------------------------------------------------------------------

func serveHTTP() {
	runtime.SetMutexProfileFraction(5)    // 生产建议值：1/5 采样
	runtime.SetBlockProfileRate(int(1e6)) // 1ms 以上的阻塞才记

	go func() {
		var sink string
		for {
			sink = slowWork(500)
			_ = sink
		}
	}()

	fmt.Println("pprof 端点已启动: http://localhost:6060/debug/pprof/")
	fmt.Println("常用命令：")
	fmt.Println("  go tool pprof -http=:8080 'http://localhost:6060/debug/pprof/profile?seconds=10'")
	fmt.Println("  go tool pprof 'http://localhost:6060/debug/pprof/heap'")
	fmt.Println("  curl -o trace.out 'http://localhost:6060/debug/pprof/trace?seconds=5'")
	fmt.Println("  curl 'http://localhost:6060/debug/pprof/goroutine?debug=2' | head -50")
	fmt.Println("Ctrl+C 退出")
	//nolint:gosec // 演示代码
	if err := http.ListenAndServe("localhost:6060", nil); err != nil {
		fmt.Println("ListenAndServe:", err)
	}
}

// ---------------------------------------------------------------------------

func runAll() {
	fmt.Println("=== 1. CPU profile（2s） ===")
	must(writeCPUProfile(2 * time.Second))

	fmt.Println("=== 2. heap profile ===")
	must(writeHeapProfile())

	fmt.Println("=== 3. goroutine profile ===")
	must(writeGoroutineProfile())

	fmt.Println("=== 4. mutex / block profile ===")
	must(writeContentionProfiles())

	fmt.Println("=== 5. execution trace（1s） ===")
	must(writeTrace(time.Second))

	fmt.Println()
	fmt.Println("接下来试试：")
	fmt.Println("  go tool pprof -top -nodecount=10 prof/out/cpu.pprof")
	fmt.Println("  go tool pprof -top -sample_index=alloc_space prof/out/heap.pprof")
	fmt.Println("  head -30 prof/out/goroutine.txt")
	fmt.Println("  go tool pprof -top prof/out/mutex.pprof")
	fmt.Println("  go tool trace prof/out/trace.out")
	fmt.Println("  go run ./prof serve   # 然后用 -http=:8080 看火焰图")
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
