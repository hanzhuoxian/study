// gc 示例：对应 notes/gc.md
// 运行：go run ./gc
// 看 GC 日志：GODEBUG=gctrace=1 go run ./gc
// 看 pacer 决策：GODEBUG=gcpacertrace=1 go run ./gc
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"strings"
	"sync"
	"time"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	if mode := os.Getenv("GC_DEMO"); mode != "" {
		runChild(mode)
		return
	}

	basicWhenDoesGCRun()
	basicHeapGoal()
	basicMemStats()
	basicMetrics()

	phasesAndSTW()
	markAssist()
	scavenger()

	tuneGOGC()
	tuneMemoryLimit()
	tuneBallast()

	trapMemoryNotReturned()
	trapFinalizer()
	trapCleanup()
	trapGCPercentOff()
}

// ---------------------------------------------------------------------------
// 1.1 GC 什么时候触发
// ---------------------------------------------------------------------------

func basicWhenDoesGCRun() {
	section("1.1 GC 的三个触发源")

	fmt.Println("① gcTriggerHeap  —— 堆分配量达到 heap goal（最常见）")
	fmt.Println("② gcTriggerTime  —— 距上次 GC 超过 forcegcperiod = 2 分钟（sysmon 触发）")
	fmt.Println("③ gcTriggerCycle —— runtime.GC() / debug.FreeOSMemory() 手动触发")

	before := gcCount()
	// 制造足够多的垃圾，逼出一次 heap 触发的 GC
	for range 40 {
		_ = make([]byte, 1<<20)
	}
	fmt.Printf("分配 40MB 垃圾后 GC 次数: %d -> %d\n", before, gcCount())

	before = gcCount()
	runtime.GC()
	fmt.Printf("runtime.GC() 之后:        %d -> %d\n", before, gcCount())
}

func gcCount() uint32 {
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	return s.NumGC
}

// ---------------------------------------------------------------------------
// 1.2 heap goal：GOGC 到底控制什么
// ---------------------------------------------------------------------------

var keep [][]byte // 故意保持存活

func basicHeapGoal() {
	section("1.2 heap goal = live × (1 + GOGC/100)")

	gogc := debug.SetGCPercent(100) // 读回旧值并设成 100
	debug.SetGCPercent(gogc)
	fmt.Printf("当前 GOGC = %d\n", gogc)

	// 造 20MB 存活对象
	keep = nil
	for range 20 {
		keep = append(keep, make([]byte, 1<<20))
	}
	runtime.GC() // 让 live 稳定下来

	live := readMetric("/gc/heap/live:bytes")
	goal := readMetric("/gc/heap/goal:bytes")
	fmt.Printf("live = %.1f MB, goal = %.1f MB, goal/live = %.2f\n",
		mb(live), mb(goal), float64(goal)/float64(live))
	fmt.Println("→ GOGC=100 意味着'新分配量达到存活量的 100% 就开一轮 GC'")
	fmt.Println("→ 还有个下限 defaultHeapMinimum = 4MB（GOGC=100 时），小堆不会疯狂 GC")
}

// ---------------------------------------------------------------------------
// 1.3 MemStats 里最该看的几个字段
// ---------------------------------------------------------------------------

func basicMemStats() {
	section("1.3 MemStats 关键字段")

	var s runtime.MemStats
	runtime.ReadMemStats(&s) // ⚠️ 会 STW

	rows := [][2]any{
		{"HeapAlloc", "当前存活对象字节数（= 分配 - 释放）"},
		{"HeapSys", "从 OS 拿到的堆虚拟内存"},
		{"HeapIdle", "空闲 span 的字节数（其中 HeapReleased 已还给 OS）"},
		{"HeapReleased", "已经还给 OS 的字节数"},
		{"HeapObjects", "存活对象个数"},
		{"NextGC", "下一轮 GC 的 heap goal"},
		{"NumGC", "已完成的 GC 轮数"},
		{"NumForcedGC", "被 runtime.GC() 强制触发的轮数"},
		{"PauseTotalNs", "所有 STW 暂停累计（两段：sweep term + mark term）"},
		{"GCCPUFraction", "程序启动至今 GC 占用的 CPU 比例"},
	}
	vals := map[string]string{
		"HeapAlloc":     fmt.Sprintf("%.1f MB", mb(s.HeapAlloc)),
		"HeapSys":       fmt.Sprintf("%.1f MB", mb(s.HeapSys)),
		"HeapIdle":      fmt.Sprintf("%.1f MB", mb(s.HeapIdle)),
		"HeapReleased":  fmt.Sprintf("%.1f MB", mb(s.HeapReleased)),
		"HeapObjects":   fmt.Sprint(s.HeapObjects),
		"NextGC":        fmt.Sprintf("%.1f MB", mb(s.NextGC)),
		"NumGC":         fmt.Sprint(s.NumGC),
		"NumForcedGC":   fmt.Sprint(s.NumForcedGC),
		"PauseTotalNs":  time.Duration(s.PauseTotalNs).String(),
		"GCCPUFraction": fmt.Sprintf("%.4f%%", s.GCCPUFraction*100),
	}
	for _, r := range rows {
		name := r[0].(string)
		fmt.Printf("  %-14s %-12s %s\n", name, vals[name], r[1])
	}
	fmt.Println("→ ReadMemStats 会 STW，生产环境高频调用本身就是性能问题；用 runtime/metrics")
}

// ---------------------------------------------------------------------------
// 1.4 runtime/metrics：现代的正确姿势
// ---------------------------------------------------------------------------

var gcMetrics = []string{
	"/gc/heap/live:bytes",
	"/gc/heap/goal:bytes",
	"/gc/heap/objects:objects",
	"/gc/heap/allocs:bytes",
	"/gc/heap/frees:bytes",
	"/gc/cycles/total:gc-cycles",
	"/gc/pauses:seconds",
	"/cpu/classes/gc/total:cpu-seconds",
	"/memory/classes/heap/objects:bytes",
	"/memory/classes/heap/released:bytes",
	"/memory/classes/total:bytes",
	"/sched/gomaxprocs:threads",
}

func basicMetrics() {
	section("1.4 runtime/metrics")

	samples := make([]metrics.Sample, len(gcMetrics))
	for i, name := range gcMetrics {
		samples[i].Name = name
	}
	metrics.Read(samples)

	for _, s := range samples {
		switch s.Value.Kind() {
		case metrics.KindUint64:
			v := s.Value.Uint64()
			if strings.HasSuffix(s.Name, ":bytes") {
				fmt.Printf("  %-42s %.2f MB\n", s.Name, mb(v))
			} else {
				fmt.Printf("  %-42s %d\n", s.Name, v)
			}
		case metrics.KindFloat64:
			fmt.Printf("  %-42s %.4f\n", s.Name, s.Value.Float64())
		case metrics.KindFloat64Histogram:
			h := s.Value.Float64Histogram()
			fmt.Printf("  %-42s p50=%s p99=%s（%d 个桶）\n", s.Name,
				histQuantile(h, 0.5), histQuantile(h, 0.99), len(h.Counts))
		}
	}
	fmt.Printf("→ metrics.All() 共导出 %d 个指标，且 Read 不 STW\n", len(metrics.All()))
}

func histQuantile(h *metrics.Float64Histogram, q float64) string {
	var total uint64
	for _, c := range h.Counts {
		total += c
	}
	if total == 0 {
		return "n/a"
	}
	target, cum := float64(total)*q, uint64(0)
	for i, c := range h.Counts {
		cum += c
		if float64(cum) >= target {
			return time.Duration(h.Buckets[i+1] * float64(time.Second)).Round(time.Microsecond).String()
		}
	}
	return "n/a"
}

// ---------------------------------------------------------------------------
// 2.1 GC 的阶段与两次 STW
// ---------------------------------------------------------------------------

func phasesAndSTW() {
	section("2.1 一轮 GC 的阶段")

	fmt.Println("  gcphase: _GCoff --(STW sweep term)--> _GCmark --(STW mark term)--> _GCmarktermination --> _GCoff")
	fmt.Println()
	for _, row := range [][2]string{
		{"① sweep termination", "STW：所有 P 到安全点，清扫上一轮遗留的 span"},
		{"② mark 准备", "仍在 STW 内：置 _GCmark、开写屏障、开 assist、入队 root 扫描任务"},
		{"③ 并发标记", "Start the world；后台 worker（25% CPU）+ mutator assist 一起标记"},
		{"④ mark termination", "STW：关 worker/assist、flush mcache；这一段要尽量短"},
		{"⑤ 并发清扫", "置 _GCoff、关写屏障、Start the world；后台 + 按需惰性清扫"},
	} {
		fmt.Printf("  %-20s %s\n", row[0], row[1])
	}
	fmt.Println()
	fmt.Println("→ 两次 STW 都是'全局同步点'，目标都在百微秒级；停顿时间与堆大小基本无关")
	fmt.Println("→ 这就是 1.8 之后 GC 停顿从'和堆成正比'变成'亚毫秒'的原因：")
	fmt.Println("  混合写屏障（Dijkstra 插入 + Yuasa 删除）消掉了 mark termination 里的栈重扫")

	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	if s.NumGC > 0 {
		n := int(s.NumGC)
		if n > len(s.PauseNs) {
			n = len(s.PauseNs)
		}
		var max, sum uint64
		for i := range n {
			p := s.PauseNs[i]
			sum += p
			if p > max {
				max = p
			}
		}
		fmt.Printf("最近 %d 段 STW：平均 %v，最大 %v\n", n,
			time.Duration(sum/uint64(n)).Round(time.Microsecond),
			time.Duration(max).Round(time.Microsecond))
	}
}

// ---------------------------------------------------------------------------
// 2.2 mark assist：分配太快会被罚去干 GC 的活
// ---------------------------------------------------------------------------

func markAssist() {
	section("2.2 mark assist")

	assistBefore := readFloatMetric("/gc/cycles/total:gc-cycles")
	cpuBefore := readFloatMetric("/cpu/classes/gc/total:cpu-seconds")
	start := time.Now()

	// 高速分配：故意让 mutator 跑在 GC 前面，触发 assist
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			var sink [][]byte
			for range 2000 {
				sink = append(sink, make([]byte, 4<<10))
				if len(sink) > 100 {
					sink = sink[50:]
				}
			}
			_ = sink
		})
	}
	wg.Wait()

	fmt.Printf("高速分配 %v，期间 GC 轮数 +%.0f，GC CPU 时间 +%.3fs\n",
		time.Since(start).Round(time.Millisecond),
		readFloatMetric("/gc/cycles/total:gc-cycles")-assistBefore,
		readFloatMetric("/cpu/classes/gc/total:cpu-seconds")-cpuBefore)

	fmt.Println("→ 后台标记固定占 gcBackgroundUtilization = 25% 的 GOMAXPROCS")
	fmt.Println("→ 不够的部分由 mark assist 补：谁分配得快，谁在 mallocgc 里被扣'扫描信用'，")
	fmt.Println("  信用不足就地干标记活（gcAssistAlloc），这就是'分配越猛越慢'的直接原因")
	fmt.Println("→ gcOverAssistWork = 64KB：每次 assist 多干一点，预付未来的分配，摊薄开销")
	fmt.Println("→ 看 assist 占比：GODEBUG=gctrace=1 里 '#+#/#/#+# ms cpu' 的第一个数就是 assist")
}

// ---------------------------------------------------------------------------
// 2.3 scavenger：内存怎么还给 OS
// ---------------------------------------------------------------------------

func scavenger() {
	section("2.3 内存归还 OS")

	keep = nil
	for range 100 {
		keep = append(keep, make([]byte, 1<<20)) // 100MB
	}
	runtime.GC()
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	fmt.Printf("持有 100MB:      HeapAlloc=%.0fMB HeapIdle=%.0fMB HeapReleased=%.0fMB RSS≈HeapSys=%.0fMB\n",
		mb(s.HeapAlloc), mb(s.HeapIdle), mb(s.HeapReleased), mb(s.HeapSys))

	keep = nil
	runtime.GC()
	runtime.ReadMemStats(&s)
	fmt.Printf("丢弃后立即:      HeapAlloc=%.0fMB HeapIdle=%.0fMB HeapReleased=%.0fMB\n",
		mb(s.HeapAlloc), mb(s.HeapIdle), mb(s.HeapReleased))

	debug.FreeOSMemory() // 强制归还
	runtime.ReadMemStats(&s)
	fmt.Printf("FreeOSMemory 后: HeapAlloc=%.0fMB HeapIdle=%.0fMB HeapReleased=%.0fMB\n",
		mb(s.HeapAlloc), mb(s.HeapIdle), mb(s.HeapReleased))

	fmt.Println("→ GC 只把对象标成可回收，span 先进 HeapIdle 由 runtime 自己留着复用")
	fmt.Println("→ 后台 scavenger 按 pacing 慢慢 madvise 还给 OS（GODEBUG=scavtrace=1 可观测）")
	fmt.Println("→ 所以'GC 跑完了 RSS 没降'是正常现象，不是内存泄漏")
	fmt.Println("→ darwin/linux 默认 MADV_DONTNEED（GODEBUG=madvdontneed=0 改 MADV_FREE，RSS 降得更慢）")
}

// ---------------------------------------------------------------------------
// 3.1 GOGC
// ---------------------------------------------------------------------------

func tuneGOGC() {
	section("3.1 GOGC 调优")

	keep = nil
	for range 20 {
		keep = append(keep, make([]byte, 1<<20))
	}
	runtime.GC()

	for _, pct := range []int{50, 100, 400} {
		old := debug.SetGCPercent(pct)
		runtime.GC() // 让新的 goal 生效
		live, goal := readMetric("/gc/heap/live:bytes"), readMetric("/gc/heap/goal:bytes")
		fmt.Printf("  GOGC=%-4d live=%.0fMB goal=%.0fMB（%.1fx）\n", pct, mb(live), mb(goal),
			float64(goal)/float64(live))
		debug.SetGCPercent(old)
	}
	debug.SetGCPercent(100)
	fmt.Println("→ GOGC 越大：GC 次数越少、CPU 省、内存涨；越小反之。这是一条纯粹的空间换时间曲线")
	fmt.Println("→ 典型服务器实践：GOGC=200~400 + GOMEMLIMIT 兜底，用内存换 CPU")
}

// ---------------------------------------------------------------------------
// 3.2 GOMEMLIMIT（1.19+）：软内存上限
// ---------------------------------------------------------------------------

func tuneMemoryLimit() {
	section("3.2 GOMEMLIMIT")

	old := debug.SetMemoryLimit(-1) // -1 = 只读不改
	fmt.Printf("当前 memory limit = %v\n", limitStr(old))

	keep = nil
	for range 20 {
		keep = append(keep, make([]byte, 1<<20))
	}
	runtime.GC()

	debug.SetMemoryLimit(40 << 20) // 40MB
	runtime.GC()
	live, goal := readMetric("/gc/heap/live:bytes"), readMetric("/gc/heap/goal:bytes")
	fmt.Printf("limit=40MB 时: live=%.0fMB goal=%.0fMB（被 limit 压下来了）\n", mb(live), mb(goal))

	debug.SetMemoryLimit(old)
	fmt.Println("→ 它是'软'上限：runtime 会加大 GC 频率去逼近，但不会为它 OOM-kill 自己")
	fmt.Println("→ 统计范围是 Go runtime 管的所有内存（堆+栈+元数据），不含 cgo/mmap 的部分")
	fmt.Println("→ 死亡螺旋（death spiral）防护：GC CPU 占比被限制在 50%，宁可超限也不把 CPU 烧光")
}

// ---------------------------------------------------------------------------
// 3.3 ballast：GOMEMLIMIT 出现之前的土办法
// ---------------------------------------------------------------------------

func tuneBallast() {
	section("3.3 ballast（历史技巧，现在别用了）")

	fmt.Println(`  var ballast = make([]byte, 10<<30)   // 10GB 虚拟内存，永不触碰
  runtime.KeepAlive(ballast)`)
	fmt.Println("  原理：把 live 撑大，heap goal = live×2 也跟着变大，GC 次数骤降；")
	fmt.Println("       因为从没写过，OS 不会真的分配物理页，RSS 不涨。")
	fmt.Println("→ 1.19 之后请改用 GOGC + GOMEMLIMIT，语义清楚、可观测、不依赖 OS 的懒分配行为")
}

// ---------------------------------------------------------------------------
// 4.1 常见误解：GC 完 RSS 不降 = 泄漏？
// ---------------------------------------------------------------------------

func trapMemoryNotReturned() {
	section("4.1 陷阱：RSS 不降不等于泄漏")

	fmt.Println("排查顺序：")
	fmt.Println("  ① /memory/classes/heap/objects:bytes 涨 -> 真有对象活着（业务泄漏）")
	fmt.Println("  ② heap/objects 平 + heap/released 低 -> 只是没还给 OS（scavenger 节奏）")
	fmt.Println("  ③ /memory/classes/heap/free:bytes 大 -> 碎片或者峰值过后的空 span")
	fmt.Println("  ④ 都不是 -> 看 /memory/classes/total:bytes 减去 heap，可能是栈/元数据/cgo")
	fmt.Println("常见的真泄漏源：goroutine 泄漏（连带它的栈和闭包）、全局 map 只增不删、")
	fmt.Println("  大 slice 截小后仍持有底层数组（slice.md 3.3）、time.Ticker 忘了 Stop")
}

// ---------------------------------------------------------------------------
// 4.2 finalizer
// ---------------------------------------------------------------------------

type resource struct{ name string }

func trapFinalizer() {
	section("4.2 SetFinalizer 的坑")

	done := make(chan string, 1)
	func() {
		r := &resource{name: "fd-1"}
		runtime.SetFinalizer(r, func(r *resource) { done <- r.name })
	}()

	runtime.GC() // 第一轮：发现不可达，把 finalizer 排队，对象"复活"
	select {
	case name := <-done:
		fmt.Println("finalizer 执行了:", name)
	case <-time.After(time.Second):
		fmt.Println("finalizer 一秒内没执行（不保证时机）")
	}
	runtime.GC() // 第二轮：finalizer 已清除，对象才真正被回收

	fmt.Println("→ 带 finalizer 的对象至少需要两轮 GC 才能真正释放")
	fmt.Println("→ 不保证一定执行（进程退出前没跑到就没了），不保证顺序，循环引用时可能永不执行")
	fmt.Println("→ finalizer 里让对象重新可达会导致对象'永生'")
	fmt.Println("→ 1.24 起首选 runtime.AddCleanup，见 4.3")
}

// ---------------------------------------------------------------------------
// 4.3 AddCleanup（1.24+）
// ---------------------------------------------------------------------------

type fileHandle struct{ fd int }

func trapCleanup() {
	section("4.3 runtime.AddCleanup（1.24+）")

	done := make(chan int, 1)
	func() {
		h := &fileHandle{fd: 42}
		// 注意：cleanup 的参数不能是 h 本身（会让 h 永远可达），只能传"底层资源"
		runtime.AddCleanup(h, func(fd int) { done <- fd }, h.fd)
	}()

	// cleanup 由专门的 goroutine 在 GC 之后执行，可能要等一两轮
	var fd int
	for range 3 {
		runtime.GC()
		select {
		case fd = <-done:
		case <-time.After(200 * time.Millisecond):
			continue
		}
		break
	}
	if fd != 0 {
		fmt.Println("cleanup 执行了, fd =", fd)
	} else {
		fmt.Println("cleanup 在 3 轮 GC 内没执行到（时机本身就不保证）")
	}

	fmt.Println("→ 相比 SetFinalizer：不复活对象（一轮 GC 就能回收）、可以给同一对象挂多个、")
	fmt.Println("  循环引用也能正常执行、返回的 Cleanup 可以 Stop()")
	fmt.Println("→ 但依然不保证执行时机；关资源请老老实实用 defer Close()，cleanup 只当兜底告警")
}

// ---------------------------------------------------------------------------
// 4.4 GOGC=off 与 gctrace 实测（子进程）
// ---------------------------------------------------------------------------

func trapGCPercentOff() {
	section("4.4 GODEBUG=gctrace=1 实测（子进程）")

	for _, mode := range []string{"gctrace", "gcoff", "limit"} {
		out, _ := selfRun(mode)
		fmt.Printf("--- %s ---\n%s", mode, indent(strings.TrimSpace(out)))
	}
	fmt.Println("gctrace 行的含义：")
	fmt.Println("  gc 1 @0.01s 0%: 0.02+0.3+0.01 ms clock, 0.1+0.1/0.3/0+0.1 ms cpu, 4->4->1 MB, 5 MB goal, ...")
	fmt.Println("     │    │    │   └ STW sweepterm + 并发 mark + STW marktermination")
	fmt.Println("     │    │    └ 启动至今 GC 占的 CPU 百分比")
	fmt.Println("     │    └ 距程序启动的时间")
	fmt.Println("     └ 第几轮 GC")
	fmt.Println("  cpu 段 = assist + background/idle + 终止；MB 段 = GC 开始 -> 结束 -> 存活")
}

func runChild(mode string) {
	switch mode {
	case "gctrace", "gcoff":
		var live [][]byte
		for range 30 {
			live = append(live, make([]byte, 1<<20))
			_ = make([]byte, 1<<20) // 垃圾
		}
		runtime.KeepAlive(live)
		var s runtime.MemStats
		runtime.ReadMemStats(&s)
		fmt.Printf("NumGC=%d HeapAlloc=%.0fMB HeapSys=%.0fMB\n",
			s.NumGC, mb(s.HeapAlloc), mb(s.HeapSys))
	case "limit":
		debug.SetMemoryLimit(32 << 20)
		var live [][]byte
		for range 30 {
			live = append(live, make([]byte, 1<<20))
			if len(live) > 10 {
				live = live[5:]
			}
		}
		var s runtime.MemStats
		runtime.ReadMemStats(&s)
		fmt.Printf("limit=32MB: NumGC=%d HeapAlloc=%.1fMB HeapSys=%.1fMB\n",
			s.NumGC, mb(s.HeapAlloc), mb(s.HeapSys))
	}
}

func selfRun(mode string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(exe)
	env := append(os.Environ(), "GC_DEMO="+mode)
	switch mode {
	case "gctrace":
		env = append(env, "GODEBUG=gctrace=1")
	case "gcoff":
		env = append(env, "GOGC=off", "GODEBUG=gctrace=1")
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func mb[T uint64 | int64](v T) float64 { return float64(v) / (1 << 20) }

func readMetric(name string) uint64 {
	s := []metrics.Sample{{Name: name}}
	metrics.Read(s)
	return s[0].Value.Uint64()
}

func readFloatMetric(name string) float64 {
	s := []metrics.Sample{{Name: name}}
	metrics.Read(s)
	switch s[0].Value.Kind() {
	case metrics.KindUint64:
		return float64(s[0].Value.Uint64())
	case metrics.KindFloat64:
		return s[0].Value.Float64()
	}
	return 0
}

func limitStr(v int64) string {
	if v == int64(^uint64(0)>>1) {
		return "math.MaxInt64（未设置）"
	}
	return fmt.Sprintf("%.1f MB", mb(v))
}

func indent(s string) string {
	if s == "" {
		return "  (无输出)\n"
	}
	return "  " + strings.ReplaceAll(s, "\n", "\n  ") + "\n"
}
