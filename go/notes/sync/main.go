// sync 示例：对应 notes/sync.md
// 运行：go run ./sync
// 竞态检查：go run -race ./sync
// 压测：go test -bench . -benchmem ./sync
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	// 子进程模式：用来演示不可 recover 的 fatal error
	if mode := os.Getenv("SYNC_DEMO_FATAL"); mode != "" {
		runFatalDemo(mode)
		return
	}

	basicMutex()
	basicRWMutex()
	basicWaitGroup()
	basicOnce()
	basicCond()
	basicMap()

	mutexStateBits()
	mutexStarvation()

	trapCopyLock()
	trapUnlockUnlocked()
	trapRecursiveRLock()
	trapLockGranularity()
	trapOnceDeadlock()
	trapWaitGroupMisuse()
	trapCondWithoutLoop()
	trapMapMisuse()
}

// ---------------------------------------------------------------------------
// 1.1 Mutex：Lock / Unlock / TryLock
// ---------------------------------------------------------------------------

type counter struct {
	mu sync.Mutex
	n  int
}

func (c *counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c *counter) Load() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

func basicMutex() {
	section("1.1 Mutex")

	var c counter
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(c.Inc) // Go 1.25+：wg.Add(1) + go func(){ defer wg.Done(); ... }()
	}
	wg.Wait()
	fmt.Println("100 个 goroutine 各 +1 =", c.Load())

	// 零值可用，不需要 NewMutex
	var mu sync.Mutex
	fmt.Println("零值 Mutex 直接 TryLock:", mu.TryLock())
	fmt.Println("已持锁时再 TryLock:     ", mu.TryLock())
	mu.Unlock()

	// 锁不属于任何 goroutine：A 加锁 B 解锁是合法的（但不推荐）
	mu.Lock()
	done := make(chan struct{})
	go func() { mu.Unlock(); close(done) }()
	<-done
	fmt.Println("在另一个 goroutine 里 Unlock：合法（锁不绑定 goroutine）")
}

// ---------------------------------------------------------------------------
// 1.2 RWMutex：读并发、写独占、写优先
// ---------------------------------------------------------------------------

func basicRWMutex() {
	section("1.2 RWMutex")

	var rw sync.RWMutex
	var wg sync.WaitGroup

	rw.RLock()
	fmt.Println("持有读锁时，再 RLock:  ", tryRLock(&rw), "（读可以并发）")
	fmt.Println("持有读锁时，TryLock:   ", rw.TryLock(), "（写要独占）")

	// 写者 pending 之后，新的读者必须排队 —— 这就是"写优先"
	writerReady := make(chan struct{})
	wg.Go(func() {
		close(writerReady)
		rw.Lock() // 阻塞，等已有读者退出
		rw.Unlock()
	})
	<-writerReady
	time.Sleep(20 * time.Millisecond) // 让写者真正进入等待
	fmt.Println("有写者在等时，TryRLock:", tryRLock(&rw), "（新读者被挡住，防写者饿死）")

	rw.RUnlock()
	wg.Wait()
	fmt.Println("→ 因此 RLock 不可递归：外层持读锁 + 中间来了写者 + 内层再 RLock = 死锁")
}

func tryRLock(rw *sync.RWMutex) bool {
	if ok := rw.TryRLock(); ok {
		rw.RUnlock()
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// 1.3 WaitGroup
// ---------------------------------------------------------------------------

func basicWaitGroup() {
	section("1.3 WaitGroup")

	var wg sync.WaitGroup
	results := make([]int, 5)
	for i := range results {
		wg.Go(func() { results[i] = i * i }) // 1.22+ 循环变量每轮新建，不用再传参
	}
	wg.Wait()
	fmt.Println("wg.Go（1.25+）:", results)

	// 老写法：Add 必须在 go 之前
	wg.Add(1)
	go func() {
		defer wg.Done()
	}()
	wg.Wait()
	fmt.Println("Add/Done/Wait 三件套仍然有效")

	// state 是一个 uint64：高 32 位 counter，低 31 位 waiter 数，中间 1 位给 synctest
	fmt.Println("state 布局: [63:32]=counter [32]=synctest bubble [31:0]=waiter count")
}

// ---------------------------------------------------------------------------
// 1.4 Once / OnceFunc / OnceValue / OnceValues
// ---------------------------------------------------------------------------

var (
	once     sync.Once
	initCall int

	onceFn  = sync.OnceFunc(func() { fmt.Println("  OnceFunc 只打印一次") })
	onceVal = sync.OnceValue(func() int { fmt.Println("  OnceValue 只算一次"); return 42 })
	onceTwo = sync.OnceValues(func() (string, error) { return "v", nil })

	// panic 会被记住，之后每次调用都 panic 同一个值
	oncePanic = sync.OnceValue(func() int { panic("boom") })
)

func basicOnce() {
	section("1.4 Once 家族")

	for range 3 {
		once.Do(func() { initCall++ })
	}
	fmt.Println("once.Do 执行次数:", initCall)

	onceFn()
	onceFn()
	fmt.Println("OnceValue:", onceVal(), onceVal())
	v, err := onceTwo()
	fmt.Println("OnceValues:", v, err)

	for i := range 2 {
		func() {
			defer func() { fmt.Printf("  第 %d 次调用 panic: %v\n", i+1, recover()) }()
			_ = oncePanic()
		}()
	}
	fmt.Println("→ Once 的正确性关键：done 必须在 f() 返回之后才置位（否则并发下第二个调用会提前返回）")
}

// ---------------------------------------------------------------------------
// 1.5 Cond：等待条件变化（Wait 必须写在 for 里）
// ---------------------------------------------------------------------------

type queue struct {
	mu    sync.Mutex
	cond  *sync.Cond
	items []int
	done  bool
}

func newQueue() *queue {
	q := &queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *queue) Push(v int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, v)
	q.cond.Signal() // 唤醒一个等待者
}

func (q *queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.done = true
	q.cond.Broadcast() // 唤醒所有等待者
}

func (q *queue) Pop() (int, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.done { // 必须 for，不能 if：被唤醒不代表条件成立
		q.cond.Wait() // Wait 内部：Unlock -> 挂起 -> 被唤醒 -> Lock
	}
	if len(q.items) == 0 {
		return 0, false
	}
	v := q.items[0]
	q.items = q.items[1:]
	return v, true
}

func basicCond() {
	section("1.5 Cond")

	q := newQueue()
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			v, ok := q.Pop()
			if !ok {
				fmt.Println("  消费者退出")
				return
			}
			fmt.Println("  消费:", v)
		}
	})

	for i := 1; i <= 3; i++ {
		q.Push(i)
		time.Sleep(5 * time.Millisecond)
	}
	q.Close()
	wg.Wait()
	fmt.Println("→ Cond 的三条铁律：持锁调用 Wait / Wait 写在 for 里 / 优先考虑用 channel 代替")
}

// ---------------------------------------------------------------------------
// 1.6 sync.Map（Go 1.24+ 底层换成了 HashTrieMap）
// ---------------------------------------------------------------------------

func basicMap() {
	section("1.6 sync.Map")

	var m sync.Map

	m.Store("a", 1)
	actual, loaded := m.LoadOrStore("a", 999)
	fmt.Println("LoadOrStore 已存在:", actual, loaded)
	actual, loaded = m.LoadOrStore("b", 2)
	fmt.Println("LoadOrStore 新建:  ", actual, loaded)

	prev, loaded := m.Swap("a", 10)
	fmt.Println("Swap:              ", prev, loaded)
	fmt.Println("CompareAndSwap:    ", m.CompareAndSwap("a", 10, 11))
	fmt.Println("CompareAndDelete:  ", m.CompareAndDelete("b", 2))

	v, loaded := m.LoadAndDelete("a")
	fmt.Println("LoadAndDelete:     ", v, loaded)

	m.Store("x", 1)
	m.Store("y", 2)
	n := 0
	m.Range(func(k, v any) bool { n++; return true })
	fmt.Println("Range 遍历到", n, "个（不是快照：遍历期间的修改可能被看到，也可能看不到）")

	m.Clear()
	m.Range(func(k, v any) bool { n++; return true })
	fmt.Println("Clear 之后 Range 什么都没有")

	fmt.Println("→ 1.24 起底层是 concurrent hash-trie：16 路分支、读路径全 atomic 无锁、")
	fmt.Println("  写只锁住那一个 indirect 节点；老的 read/dirty + misses 提升机制已经不存在了")
}

// ---------------------------------------------------------------------------
// 2.1 Mutex 的 state 位布局
// ---------------------------------------------------------------------------

func mutexStateBits() {
	section("2.1 Mutex state 位布局")

	fmt.Println("internal/sync.Mutex{ state int32; sema uint32 }  // 一共 8 字节")
	for _, row := range [][2]string{
		{"bit 0  mutexLocked", "已加锁"},
		{"bit 1  mutexWoken", "已有 goroutine 被唤醒，Unlock 不用再唤人"},
		{"bit 2  mutexStarving", "饥饿模式：所有权直接交棒，新来的不许抢"},
		{"bit 3+ waiterShift", "等待者数量（state >> 3）"},
	} {
		fmt.Printf("  %-22s %s\n", row[0], row[1])
	}
	fmt.Printf("  starvationThresholdNs = %v（等待超过它就切饥饿模式）\n", time.Millisecond)
	fmt.Println("  active_spin = 4：自旋最多 4 轮，每轮 30 次 PAUSE（active_spin_cnt）")
	fmt.Println("  能自旋的前提：多核 && GOMAXPROCS>1 && 有其他运行中的 P && 本地队列为空")
	fmt.Printf("  当前 GOMAXPROCS=%d NumCPU=%d\n", runtime.GOMAXPROCS(0), runtime.NumCPU())
}

// ---------------------------------------------------------------------------
// 2.2 正常模式 vs 饥饿模式
// ---------------------------------------------------------------------------

func mutexStarvation() {
	section("2.2 正常模式 vs 饥饿模式")

	var mu sync.Mutex
	var wg sync.WaitGroup
	const nWorker = 4

	counts := make([]int, nWorker)
	maxWait := make([]time.Duration, nWorker)
	deadline := time.Now().Add(200 * time.Millisecond)

	for i := range nWorker {
		wg.Go(func() {
			for time.Now().Before(deadline) {
				t0 := time.Now()
				mu.Lock()
				if w := time.Since(t0); w > maxWait[i] {
					maxWait[i] = w
				}
				counts[i]++
				for range 200 { // 临界区里做一点点活，制造真实竞争
					_ = i
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	total := 0
	for _, c := range counts {
		total += c
	}
	fmt.Printf("200ms 内各 worker 抢到锁的次数: %v（合计 %d）\n", counts, total)
	fmt.Printf("各自遇到的最长一次等待:         %v\n", roundAll(maxWait))
	fmt.Println("→ 正常模式：被唤醒的等待者不直接拿锁，要和新来的抢；新来的已经在 CPU 上，")
	fmt.Println("  所以吞吐更高，但等待者可能被反复插队（尾延迟差）")
	fmt.Println("→ 某个等待者饿满 1ms 就置上 mutexStarving：此后 Unlock 把所有权直接交棒给队首")
	fmt.Println("  （Semrelease handoff=true），新来的一律排队尾、也不许自旋，尾延迟被压住")
	fmt.Println("→ 退出饥饿模式：拿到锁的是最后一个等待者，或它这次等待不足 1ms")
}

func roundAll(ds []time.Duration) []time.Duration {
	out := make([]time.Duration, len(ds))
	for i, d := range ds {
		out[i] = d.Round(time.Microsecond)
	}
	return out
}

// ---------------------------------------------------------------------------
// 3.1 拷贝锁
// ---------------------------------------------------------------------------

type safeConfig struct {
	mu   sync.Mutex
	data map[string]string
}

// ✓ 指针接收者：整个进程只有一份锁
func (c *safeConfig) Get(k string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.data[k]
}

// ✗ 值接收者会拷贝锁，等于没锁。这里只能写成注释，因为 go vet 直接拦下来：
//
//	func (c safeConfig) GetBad(k string) string { c.mu.Lock(); ... }
//	→ GetBad passes lock by value: safeConfig contains sync.Mutex
//
// 同样被 vet 的 copylocks 拦住的还有：
//
//	copied := *c            // assignment copies lock value
//	func f(c safeConfig) {} // f passes lock by value
//	for _, c := range cfgs  // range var c copies lock value

func trapCopyLock() {
	section("3.1 拷贝锁")

	c := &safeConfig{data: map[string]string{"k": "v"}}
	fmt.Println("指针接收者:", c.Get("k"))

	fmt.Println("go vet 的 copylocks 检查会拦住所有拷贝路径：")
	fmt.Println("  · 值接收者的方法        passes lock by value")
	fmt.Println("  · 按值传参 / 按值返回    passes lock by value")
	fmt.Println("  · v := *p 这类赋值      assignment copies lock value")
	fmt.Println("  · range 出来的循环变量   range var copies lock value")
	fmt.Println("→ 靠的是 sync.Mutex 里那个空的 noCopy 字段：它实现了 Lock/Unlock 方法，")
	fmt.Println("  除了给 vet 当标记之外没有任何运行时作用（运行时拷贝锁是不会报错的，只是静默失效）")
}

// ---------------------------------------------------------------------------
// 3.2 Unlock 未加锁的 mutex：fatal error，recover 救不了
// ---------------------------------------------------------------------------

func runFatalDemo(mode string) {
	switch mode {
	case "unlock":
		defer func() { fmt.Println("这行不会执行:", recover()) }()
		var mu sync.Mutex
		mu.Unlock()
	case "negative":
		defer func() { fmt.Println("Add 负数是 panic，可以 recover:", recover()) }()
		var wg sync.WaitGroup
		wg.Add(-1)
	}
}

func trapUnlockUnlocked() {
	section("3.2 fatal error vs panic")

	for _, mode := range []string{"unlock", "negative"} {
		out, _ := selfRun(mode)
		fmt.Printf("[%s] %s\n", mode, firstLine(out))
	}
	fmt.Println("→ sync 内部用 fatal() 而不是 panic()：unlock of unlocked mutex 无法 recover，进程必死")
	fmt.Println("→ 而 WaitGroup 计数变负用的是 panic()，可以被 recover（但你不该依赖这个）")
}

func selfRun(mode string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "SYNC_DEMO_FATAL="+mode)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}

// ---------------------------------------------------------------------------
// 3.3 RWMutex 递归读锁
// ---------------------------------------------------------------------------

func trapRecursiveRLock() {
	section("3.3 RWMutex 递归读锁 = 潜在死锁")

	fmt.Println(`  rw.RLock()
    ... 中间某个时刻另一个 goroutine 调了 rw.Lock()（readerCount 变成负数）
    rw.RLock()   // ✗ 阻塞：新读者要等写者，写者要等第一个 RLock 释放 -> 死锁
  rw.RUnlock()`)
	fmt.Println("→ 文档原文：this prohibits recursive read-locking")
	fmt.Println("→ 也不能把 RLock 升级成 Lock，或把 Lock 降级成 RLock")
	fmt.Println("→ 实现细节：Lock 里 readerCount.Add(-1<<30)，读者看到负数就去 readerSem 排队")
}

// ---------------------------------------------------------------------------
// 3.4 锁粒度：defer Unlock 会把锁持有到函数结束
// ---------------------------------------------------------------------------

type cache struct {
	mu sync.Mutex
	m  map[string]string
}

// ✗ 慢操作被圈在锁里
func (c *cache) GetBad(k string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := c.m[k]
	time.Sleep(time.Millisecond) // 假装是一次 RPC / 磁盘 IO
	return v
}

// ✓ 只在访问共享状态时持锁
func (c *cache) GetGood(k string) string {
	c.mu.Lock()
	v := c.m[k]
	c.mu.Unlock()

	time.Sleep(time.Millisecond)
	return v
}

func trapLockGranularity() {
	section("3.4 锁粒度")

	c := &cache{m: map[string]string{"k": "v"}}
	const n = 8

	for _, tc := range []struct {
		name string
		fn   func(string) string
	}{{"GetBad", c.GetBad}, {"GetGood", c.GetGood}} {
		start := time.Now()
		var wg sync.WaitGroup
		for range n {
			wg.Go(func() { _ = tc.fn("k") })
		}
		wg.Wait()
		fmt.Printf("  %-8s %d 个并发耗时 %v\n", tc.name, n, time.Since(start).Round(time.Millisecond))
	}
	fmt.Println("→ defer Unlock 很安全但会放大临界区；慢 IO 一定要挪出锁外")
}

// ---------------------------------------------------------------------------
// 3.5 Once.Do 里再调同一个 Once = 死锁
// ---------------------------------------------------------------------------

func trapOnceDeadlock() {
	section("3.5 Once 嵌套调用")

	fmt.Println(`  var once sync.Once
  once.Do(func() { once.Do(func(){}) })  // ✗ doSlow 里已持 o.m，内层再 Lock -> 自死锁`)
	fmt.Println("→ Once 内部是 atomic.Bool + Mutex：快路径读 done，慢路径进锁再判一次（双检查）")
	fmt.Println("→ 派生的坑：Do 里 panic 之后 done 仍会被置位（defer o.done.Store(true)），")
	fmt.Println("  也就是说初始化失败不会重试；要重试就别用 Once，或者用 OnceValue 拿到那个 panic")
}

// ---------------------------------------------------------------------------
// 3.6 WaitGroup 误用
// ---------------------------------------------------------------------------

func trapWaitGroupMisuse() {
	section("3.6 WaitGroup 误用")

	fmt.Println(`  ✗ Add 写在 goroutine 内部：Wait 可能在 Add 之前跑完，直接返回
     go func() { wg.Add(1); defer wg.Done(); ... }()
  ✓ Add 在 go 之前，或者直接用 wg.Go(f)`)

	fmt.Println(`  ✗ 复用未清空的 WaitGroup：Wait 返回前又 Add，报 WaitGroup misuse
  ✗ 按值传 WaitGroup 进函数（拷贝锁），必须传 *sync.WaitGroup`)

	// wg.Go 里 panic 不会调 Done，而是直接把整个进程带走（1.25 的设计）
	fmt.Println("  ⚠️ wg.Go(f) 的 f 不允许 panic：源码里 recover 到之后故意不调 Done 再重新 panic，")
	fmt.Println("     避免 Wait 被解除阻塞后 main 抢先 os.Exit(0) 把 panic 现场吞掉")
}

// ---------------------------------------------------------------------------
// 3.7 Cond 忘记循环检查
// ---------------------------------------------------------------------------

func trapCondWithoutLoop() {
	section("3.7 Cond 的两个必错写法")

	fmt.Println(`  ✗ if 而不是 for：Broadcast 唤醒后条件可能已被别人消费掉（虚假唤醒）
     if len(q.items) == 0 { c.Wait() }
  ✗ 不持锁调 Wait：Wait 内部会 Unlock，未加锁就 Unlock -> fatal error`)
	fmt.Println("→ Cond 没有 WaitWithTimeout；要超时就用 channel + select，或者 context")
	fmt.Println("→ Cond 里的 copyChecker 会在拷贝后 panic: sync.Cond is copied")
}

// ---------------------------------------------------------------------------
// 3.8 sync.Map 误用
// ---------------------------------------------------------------------------

var expensiveCalls atomic.Int64

func expensive() string {
	expensiveCalls.Add(1)
	return "value"
}

func trapMapMisuse() {
	section("3.8 sync.Map 误用")

	var m sync.Map
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { m.LoadOrStore("k", expensive()) }) // 实参先求值！
	}
	wg.Wait()
	fmt.Printf("LoadOrStore 只存一次，但 expensive() 被调了 %d 次\n", expensiveCalls.Load())
	fmt.Println("→ 想要'只构造一次'得用 Load + 失败再 LoadOrStore，或者 sync.OnceValue / singleflight")

	fmt.Println("→ 其他要点：")
	fmt.Println("  · key/value 都是 any：存取有装箱开销，也丢了类型安全（自己包一层泛型）")
	fmt.Println("  · 没有 Len()：只能 Range 数，而且数出来的值天生是过时的")
	fmt.Println("  · Range 不是快照、也不保证顺序；回调里做重活会拖长遍历")
	fmt.Println("  · 只在'读多写少'或'各 goroutine 操作互不相交的 key'时才比 RWMutex+map 划算")
}
