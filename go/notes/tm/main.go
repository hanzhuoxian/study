// time 示例：对应 notes/time.md
// 运行：go run ./tm
// 对比 1.23 之前的 timer 行为：GODEBUG=asynctimerchan=1 go run ./tm
// 压测：go test -bench . -benchmem ./tm
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	if mode := os.Getenv("TM_DEMO"); mode != "" {
		runChild(mode)
		return
	}

	basicTimeStruct()
	basicMonotonic()
	basicDuration()
	basicFormat()
	basicLocation()

	timerBasics()
	timerNew123()
	tickerBasics()
	afterFuncDemo()

	runtimeTimers()

	trapAfterLeak()
	trapTickerLeak()
	trapResetRace()
	trapEqualCompare()
	trapSleepPrecision()
	trapTimeInStruct()
}

// ---------------------------------------------------------------------------
// 1.1 time.Time 的内部结构
// ---------------------------------------------------------------------------

func basicTimeStruct() {
	section("1.1 time.Time 结构")

	fmt.Println("  type Time struct {")
	fmt.Println("      wall uint64   // 1 bit hasMonotonic | 33 bit 秒 | 30 bit 纳秒")
	fmt.Println("      ext  int64    // hasMonotonic=1 时存单调时钟读数，否则存完整秒数")
	fmt.Println("      loc  *Location")
	fmt.Println("  }")
	var t0 time.Time
	fmt.Printf("  Sizeof(time.Time) = %d 字节\n", unsafe.Sizeof(t0))

	fmt.Println("→ 一个 Time 同时可能带两个时钟读数：wall clock（墙上时间）+ monotonic（单调时钟）")
	fmt.Println("→ loc == nil 表示 UTC（所有 UTC 时间都用 nil，不用 &utcLoc）")
	fmt.Println("→ 33 位秒字段的基准是 1885 年，能表示到 2157 年；超出范围就退化成只有 wall")
}

// ---------------------------------------------------------------------------
// 1.2 单调时钟：为什么测耗时必须用 time.Since
// ---------------------------------------------------------------------------

func basicMonotonic() {
	section("1.2 单调时钟")

	t1 := time.Now()
	fmt.Printf("  time.Now() 带 monotonic: %v\n", t1)
	fmt.Println("    （打印里 m=+0.00xxxx 那一段就是单调时钟读数）")

	t2 := t1.Round(0) // Round(0) 专门用来剥掉单调时钟
	fmt.Printf("  t.Round(0) 之后:        %v ← m= 不见了\n", t2)
	fmt.Println("  t.UTC() / t.Local() / t.Truncate() / t.AddDate() 也都会剥掉单调时钟")

	fmt.Println()
	fmt.Println("  测耗时的正确/错误写法：")
	fmt.Println("    ✓ start := time.Now(); ...; time.Since(start)   // 用单调时钟，不受 NTP/改表影响")
	fmt.Println("    ✗ t2.Sub(t1) 其中任一被 Round(0)/UTC() 处理过    // 退化成墙上时间相减")
	fmt.Println("    ✗ 两个时间来自不同进程/机器                        // 单调时钟只在本进程内有意义")

	start := time.Now()
	time.Sleep(5 * time.Millisecond)
	fmt.Printf("  time.Since(start) = %v（单调）\n", time.Since(start).Round(time.Microsecond))
	fmt.Printf("  wall 相减        = %v（受系统时间影响）\n",
		time.Now().Round(0).Sub(start.Round(0)).Round(time.Microsecond))
}

// ---------------------------------------------------------------------------
// 1.3 Duration
// ---------------------------------------------------------------------------

func basicDuration() {
	section("1.3 Duration")

	fmt.Println("  type Duration int64  // 纳秒数，所以最大约 292 年")
	fmt.Printf("  max Duration = %v\n", time.Duration(1<<63-1))

	d := 90 * time.Minute
	fmt.Printf("  90*time.Minute      = %v\n", d)
	fmt.Printf("  .Hours()=%.2f .Minutes()=%.0f .Seconds()=%.0f\n", d.Hours(), d.Minutes(), d.Seconds())
	fmt.Printf("  Round(time.Hour)    = %v   Truncate(time.Hour) = %v\n",
		d.Round(time.Hour), d.Truncate(time.Hour))

	pd, _ := time.ParseDuration("1h30m10.5s")
	fmt.Printf("  ParseDuration       = %v\n", pd)

	fmt.Println("  ⚠️ 陷阱：Duration 是 int64，和数字相乘时要注意类型")
	n := 5
	fmt.Printf("    time.Duration(n) * time.Second = %v  ✓\n", time.Duration(n)*time.Second)
	fmt.Println("    n * time.Second                 ✗ 编译错误（int 和 Duration 不能直接乘）")
	fmt.Println("    5 * time.Second                 ✓ 常量可以（无类型常量自动转换）")

	fmt.Println("  ⚠️ 陷阱：秒数转 Duration 别写成 time.Duration(seconds)（那是纳秒）")
	sec := 3
	fmt.Printf("    错: time.Duration(%d)        = %v\n", sec, time.Duration(sec))
	fmt.Printf("    对: time.Duration(%d)*time.Second = %v\n", sec, time.Duration(sec)*time.Second)
}

// ---------------------------------------------------------------------------
// 1.4 格式化：Go 用参考时间而不是 %Y-%m-%d
// ---------------------------------------------------------------------------

func basicFormat() {
	section("1.4 格式化")

	t := time.Date(2026, 8, 28, 15, 4, 5, 123456789, time.UTC)

	fmt.Println("  参考时间: Mon Jan 2 15:04:05 MST 2006  (= 01/02 03:04:05PM '06 -0700)")
	fmt.Println("  记忆法: 1月2日3点4分5秒2006年，时区-7")
	fmt.Println()
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.DateOnly, // 1.20+
		time.TimeOnly, // 1.20+
		time.DateTime, // 1.20+
		"2006-01-02 15:04:05.000",
		"15:04:05.000000",
	} {
		fmt.Printf("    %-28q -> %s\n", layout, t.Format(layout))
	}

	parsed, err := time.Parse(time.RFC3339, "2026-08-28T15:04:05Z")
	fmt.Printf("  Parse: %v err=%v\n", parsed.Format(time.DateTime), err)

	_, err = time.Parse("2006-01-02", "2026-8-28")
	fmt.Printf("  Parse 严格匹配: %v\n", err != nil)
	fmt.Println("  → 补零敏感：01 要求两位，1 才接受一位（\"2006-1-2\"）")
}

// ---------------------------------------------------------------------------
// 1.5 时区
// ---------------------------------------------------------------------------

func basicLocation() {
	section("1.5 时区")

	utc := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	sh, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println("  LoadLocation 失败（容器里可能没有 tzdata）:", err)
		fmt.Println("  → 解决办法：import _ \"time/tzdata\"（1.15+，把时区库编进二进制，约 450KB）")
		return
	}

	fmt.Printf("  UTC:       %v\n", utc.Format(time.RFC3339))
	fmt.Printf("  Shanghai:  %v ← 同一时刻，不同表示\n", utc.In(sh).Format(time.RFC3339))
	fmt.Printf("  Unix 时间戳相同: %d == %d\n", utc.Unix(), utc.In(sh).Unix())

	fmt.Println("  time.Local 取决于 TZ 环境变量 / /etc/localtime")
	fmt.Printf("  当前 time.Local = %v, TZ=%q\n", time.Local, os.Getenv("TZ"))

	fmt.Println("→ 三条实践规则：")
	fmt.Println("  ① 存储和传输一律用 UTC 或 Unix 时间戳")
	fmt.Println("  ② 只在展示层转成用户时区")
	fmt.Println("  ③ 容器镜像（scratch/alpine）记得 import _ \"time/tzdata\" 或装 tzdata 包")
}

// ---------------------------------------------------------------------------
// 2.1 Timer 基础
// ---------------------------------------------------------------------------

func timerBasics() {
	section("2.1 Timer 与 Stop 的返回值")

	t := time.NewTimer(10 * time.Millisecond)
	start := time.Now()
	fired := <-t.C
	fmt.Printf("  NewTimer(10ms) 触发，实际等了 %v，通道里的时间 %v\n",
		time.Since(start).Round(time.Millisecond), fired.Format("15:04:05.000"))

	// 情形 A：还没到期就 Stop
	t2 := time.NewTimer(time.Hour)
	fmt.Println("  未到期 Stop():        ", t2.Stop(), "（true = 成功阻止）")
	fmt.Println("  再 Stop() 一次:       ", t2.Stop(), "（false = 已经停了）")

	// 情形 B：已经收过值之后再 Stop
	t3 := time.NewTimer(time.Millisecond)
	<-t3.C
	fmt.Println("  已收过值再 Stop():    ", t3.Stop(), "（false）")

	// 情形 C：时间已过但没人接收 —— 1.23 前后行为完全不同！
	t4 := time.NewTimer(time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	fmt.Println("  时间已过但没收过 Stop():", t4.Stop(), "← 1.23+ 是 true，1.22 是 false")
	fmt.Printf("  此时 len(t4.C) = %d ← 1.23+ 通道是同步的，值根本没进去\n", len(t4.C))
	fmt.Println("  ⚠️ 所以 1.23+ 这里再写 <-t4.C 会**永久阻塞**（老代码的 drain 写法反而成了 bug）")

	// 用子进程对比 1.22 的行为
	out, _ := selfRun("oldtimer")
	fmt.Print("  GODEBUG=asynctimerchan=1（模拟 1.22）:\n")
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		fmt.Println("    " + l)
	}
}

func runChild(mode string) {
	switch mode {
	case "oldtimer":
		t := time.NewTimer(time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		fmt.Printf("时间已过但没收过 Stop() = %v，len(t.C) = %d\n", t.Stop(), len(t.C))
		fmt.Println("→ 值已经在缓冲通道里躺着，Stop 已经拦不住了，必须 drain 才能复用")
	}
}

func selfRun(mode string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "TM_DEMO="+mode, "GODEBUG=asynctimerchan=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ---------------------------------------------------------------------------
// 2.2 Go 1.23 的 timer 大改
// ---------------------------------------------------------------------------

func timerNew123() {
	section("2.2 Go 1.23 的三个行为变化")

	fmt.Println("  ① 未触发的 timer 现在能被 GC 回收")
	fmt.Println("     1.23 之前：runtime 持有 timer，不 Stop 就泄漏，所以要 defer t.Stop()")
	fmt.Println("     1.23 之后：GC 能回收无人引用的 timer，Stop 只用于'我不想它响了'")
	fmt.Println()
	fmt.Println("  ② timer 通道从有缓冲(cap=1)变成同步(cap=0)")
	t := time.NewTimer(time.Hour)
	fmt.Printf("     len(t.C)=%d cap(t.C)=%d\n", len(t.C), cap(t.C))
	t.Stop()
	fmt.Println("     → 不会再收到 Stop/Reset 之前的'过期值'")
	fmt.Println()
	fmt.Println("  ③ 因此 Stop/Reset 之后不需要手工 drain 了")
	fmt.Println("     1.23 之前的标准姿势（现在不需要）：")
	fmt.Println("       if !t.Stop() { <-t.C }")
	fmt.Println("       t.Reset(d)")
	fmt.Println()
	fmt.Println("  回退开关: GODEBUG=asynctimerchan=1（1.27 可能移除）")
	fmt.Println("  文档原话: as of Go 1.23, any receive from t.C after Stop has returned")
	fmt.Println("            is guaranteed to block rather than receive a stale time value")

	// Reset 的正确用法（1.23+）
	t4 := time.NewTimer(time.Hour)
	fmt.Println("  Reset 到 5ms:", t4.Reset(5*time.Millisecond))
	<-t4.C
	fmt.Println("  触发 ✓")
}

// ---------------------------------------------------------------------------
// 2.3 Ticker
// ---------------------------------------------------------------------------

func tickerBasics() {
	section("2.3 Ticker")

	tk := time.NewTicker(5 * time.Millisecond)
	defer tk.Stop() // Ticker 必须 Stop！它会一直重新装填

	start := time.Now()
	for i := range 3 {
		<-tk.C
		fmt.Printf("  tick %d at %v\n", i+1, time.Since(start).Round(time.Millisecond))
	}

	fmt.Println("→ Ticker 通道容量 1，来不及消费的 tick 会被**丢弃**（不会堆积）")
	fmt.Println("→ 所以 Ticker 不保证'每个周期都收到一次'，只保证'不早于周期'")
	fmt.Println("→ Ticker 没有 GC 兜底（它自己在 runtime 里注册着），忘记 Stop 就是泄漏")
	fmt.Println("→ time.Tick(d) 返回的 ticker 永远无法 Stop，只能用于'活到进程结束'的场景")
}

// ---------------------------------------------------------------------------
// 2.4 AfterFunc
// ---------------------------------------------------------------------------

func afterFuncDemo() {
	section("2.4 AfterFunc")

	var wg sync.WaitGroup
	wg.Add(1)
	t := time.AfterFunc(5*time.Millisecond, func() {
		fmt.Println("  回调在**新 goroutine**里执行，goroutine id 不是调用者的")
		wg.Done()
	})
	wg.Wait()
	fmt.Println("  Stop 已执行完的 AfterFunc:", t.Stop())

	fmt.Println("→ AfterFunc 的 Timer 没有 C 字段可用（C 是 nil）")
	fmt.Println("→ Stop 返回 false 不代表回调已完成——它可能正在跑；要同步得自己配合")
	fmt.Println("→ 用途：超时后触发一个动作（比如取消 context），比起 select+timer 更省一个 goroutine")

	// 典型用法：超时取消
	ctx, cancel := context.WithCancel(context.Background())
	stop := time.AfterFunc(5*time.Millisecond, cancel)
	<-ctx.Done()
	stop.Stop()
	fmt.Println("  AfterFunc + cancel 实现超时:", ctx.Err())
	fmt.Println("  （实际业务直接用 context.WithTimeout，它内部就是 AfterFunc）")
}

// ---------------------------------------------------------------------------
// 3.1 runtime 侧的 timer 实现
// ---------------------------------------------------------------------------

func runtimeTimers() {
	section("3.1 runtime 里的 timer")

	fmt.Println("  数据结构（runtime/time.go:131）：")
	fmt.Println("    type timers struct {")
	fmt.Println("        mu          mutex          // per-P，但别的 P 可能来偷，所以还是要锁")
	fmt.Println("        heap        []timerWhen    // 四叉小顶堆，按 when 排序")
	fmt.Println("        len         atomic.Uint32")
	fmt.Println("        zombies     atomic.Int32   // 已标记删除但还在堆里的数量")
	fmt.Println("        minWhenHeap atomic.Int64   // heap[0].when，让 wakeTime 不用加锁")
	fmt.Println("    }")
	fmt.Println()
	fmt.Println("  演进：")
	fmt.Println("    1.9 之前:  一个全局 timer 堆 + 一个专门的 timerproc goroutine（锁竞争严重）")
	fmt.Println("    1.10-1.13: 分成 64 个全局桶，减少锁竞争")
	fmt.Println("    1.14+:     **每个 P 一个 timer 堆**，由调度器在 schedule() 里顺便检查")
	fmt.Println("               没有专门的 timer goroutine 了；netpoll 的超时也用它")
	fmt.Println("    1.23:      timer 变成能被 GC 回收；通道语义改为同步")
	fmt.Println()
	fmt.Println("  触发路径: schedule() -> checkTimers() -> runtimer() -> 执行 f（sendTime/goroutineReady）")
	fmt.Println("  sysmon 也会检查最近的 when，必要时唤醒空闲 P")
	fmt.Printf("  当前 GOMAXPROCS=%d，也就是有 %d 个 timer 堆\n",
		runtime.GOMAXPROCS(0), runtime.GOMAXPROCS(0))
	fmt.Println("  → 所以'大量 timer 会不会成为瓶颈'的答案是：分散在各 P 上，比全局堆好很多，")
	fmt.Println("    但单个 P 上百万 timer 的插入/删除仍是 O(log n) 且要抢那把 mu")
}

// ---------------------------------------------------------------------------
// 4.1 time.After 泄漏
// ---------------------------------------------------------------------------

func trapAfterLeak() {
	section("4.1 time.After 的泄漏（1.23 之后缓解了）")

	fmt.Println("  经典写法：")
	fmt.Println("    for {")
	fmt.Println("        select {")
	fmt.Println("        case v := <-ch:            // 每次循环都新建一个 timer")
	fmt.Println("        case <-time.After(time.Minute):")
	fmt.Println("        }")
	fmt.Println("    }")
	fmt.Println()
	fmt.Println("  1.23 之前：每次 After 都在 runtime 里挂一个 timer，直到到期才释放。")
	fmt.Println("             循环快的话瞬间挂几万个 timer + 几万个 Time 值，内存和 CPU 都吃紧。")
	fmt.Println("  1.23 之后：不再被引用的 timer 能被 GC 回收，泄漏问题基本消失，")
	fmt.Println("             但**每次循环仍然要新建 timer 并插堆**，热路径上依然浪费。")
	fmt.Println()
	fmt.Println("  ✓ 正确写法：复用一个 Timer")
	fmt.Println("    t := time.NewTimer(time.Minute)")
	fmt.Println("    defer t.Stop()")
	fmt.Println("    for {")
	fmt.Println("        t.Reset(time.Minute)   // 1.23+ 不需要先 drain")
	fmt.Println("        select { case v := <-ch: ...; case <-t.C: ... }")
	fmt.Println("    }")

	// 实测：对比两种写法的分配
	loop := func(useAfter bool, n int) {
		ch := make(chan int)
		t := time.NewTimer(time.Hour)
		defer t.Stop()
		for range n {
			if useAfter {
				select {
				case <-ch:
				case <-time.After(time.Hour):
				default:
				}
			} else {
				t.Reset(time.Hour)
				select {
				case <-ch:
				case <-t.C:
				default:
				}
			}
		}
	}
	loop(true, 100)
	loop(false, 100)
	fmt.Println("  （分配对比见 BenchmarkSelectAfter / BenchmarkSelectReuseTimer）")
}

// ---------------------------------------------------------------------------
// 4.2 Ticker 忘记 Stop
// ---------------------------------------------------------------------------

func trapTickerLeak() {
	section("4.2 Ticker 忘记 Stop = 真泄漏")

	before := runtime.NumGoroutine()
	for range 3 {
		tk := time.NewTicker(time.Hour)
		_ = tk // 忘记 Stop
	}
	runtime.GC()
	fmt.Printf("  goroutine 数没变（%d -> %d）：Ticker 不开 goroutine\n",
		before, runtime.NumGoroutine())
	fmt.Println("  但它在 runtime 的 timer 堆里注册着，且会不断重新装填 -> 永远不会被回收")
	fmt.Println("→ Ticker 和 Timer 不一样：**Ticker 必须 Stop**（1.23 的 GC 改进不覆盖它）")
	fmt.Println("→ time.Tick(d) 更糟：拿不到 Ticker 对象，压根没法 Stop")
	fmt.Println("   文档原话：the underlying Ticker is not recovered by the garbage collector")
}

// ---------------------------------------------------------------------------
// 4.3 Reset 的竞态（1.23 之前的老坑）
// ---------------------------------------------------------------------------

func trapResetRace() {
	section("4.3 Stop/Reset 的历史陷阱")

	fmt.Println("  1.23 之前的问题：通道有 1 个缓冲，Stop() 返回 false 时值可能已经进通道了")
	fmt.Println("    if !t.Stop() { <-t.C }   // 必须 drain，否则下次 select 立刻拿到过期值")
	fmt.Println("    但这个 drain 本身有竞态：如果值还没进通道，<-t.C 就永久阻塞")
	fmt.Println("    正确的老写法要配合 select + default，非常难写对")
	fmt.Println()
	fmt.Println("  1.23 之后：通道无缓冲 + Stop/Reset 与发送在 runtime 内同步，")
	fmt.Println("             文档保证 Stop 之后的接收一定阻塞、Reset 之后不会收到旧值。")
	fmt.Println("  → 也就是说：**新代码直接 t.Reset(d) 就行，不要再抄那段 drain 代码**")
	fmt.Println("  → 但如果你的代码要兼容 1.22 及更早，drain 逻辑还得留着")
}

// ---------------------------------------------------------------------------
// 4.4 时间比较
// ---------------------------------------------------------------------------

func trapEqualCompare() {
	section("4.4 时间比较")

	t1 := time.Now()
	t2 := t1.Round(0) // 只剥掉单调时钟，墙上时间不变

	fmt.Printf("  t1 == t2 ?        %v ← struct 比较，连 loc 指针和单调时钟都比\n", t1 == t2)
	fmt.Printf("  t1.Equal(t2) ?    %v ← 只比时刻，这才是对的\n", t1.Equal(t2))

	utc := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sh := utc.In(time.FixedZone("CST", 8*3600))
	fmt.Printf("  同一时刻不同时区: == %v, Equal %v\n", utc == sh, utc.Equal(sh))

	fmt.Println("→ 铁律：比较时间用 Equal / Before / After，**永远不要用 ==**")
	fmt.Println("→ 也不要把 time.Time 当 map key（同一时刻的不同表示会是不同的 key）")
	fmt.Println("→ 要当 key 就用 t.UnixNano() 或 t.UTC().Format(time.RFC3339Nano)")
}

// ---------------------------------------------------------------------------
// 4.5 Sleep 的精度
// ---------------------------------------------------------------------------

func trapSleepPrecision() {
	section("4.5 Sleep / Timer 的精度")

	for _, d := range []time.Duration{
		time.Nanosecond, time.Microsecond, 100 * time.Microsecond, time.Millisecond,
	} {
		start := time.Now()
		time.Sleep(d)
		actual := time.Since(start)
		fmt.Printf("  Sleep(%-8v) 实际 %v（%.0fx）\n", d, actual.Round(time.Microsecond),
			float64(actual)/float64(d))
	}
	fmt.Println("→ 保证'至少睡 d'，绝不保证'恰好睡 d'：受 OS 定时器精度 + 调度延迟影响")
	fmt.Println("→ 亚毫秒级定时在通用 OS 上不可靠；要精确节拍得靠忙等（烧 CPU）或实时内核")
	fmt.Println("→ time.Sleep(0) 不等于 runtime.Gosched()：Sleep(0) 直接返回，不让出")
}

// ---------------------------------------------------------------------------
// 4.6 结构体里的 time.Time
// ---------------------------------------------------------------------------

type record struct {
	ID        int
	CreatedAt time.Time
}

func trapTimeInStruct() {
	section("4.6 结构体里的 time.Time")

	r := record{ID: 1, CreatedAt: time.Now()}
	fmt.Printf("  Sizeof(record) = %d（int 8 + time.Time %d）\n",
		unsafe.Sizeof(r), unsafe.Sizeof(r.CreatedAt))

	fmt.Println("→ time.Time 里有 *Location 指针，所以：")
	fmt.Println("  · 含 time.Time 的 struct 一定落在 scan span，GC 要扫它（mem.md 1.5）")
	fmt.Println("  · 海量记录时用 int64 存 Unix 纳秒更省（8 字节 + noscan）")
	fmt.Println("→ JSON 序列化：time.Time 默认输出 RFC3339Nano，反序列化也只认这个格式")
	fmt.Println("  自定义格式要实现 MarshalJSON/UnmarshalJSON（json.md 2.1）")
	fmt.Println("→ 零值判断用 t.IsZero()，不是 t == time.Time{}")
	fmt.Printf("  零值: IsZero()=%v, 打印=%v\n", time.Time{}.IsZero(), time.Time{})
	_ = r
}
