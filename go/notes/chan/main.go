// channel 示例：对应 notes/chan.md
// 运行：go run ./chan
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
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	// 子进程模式：演示 fatal error（死锁），见 3.13
	if os.Getenv("CHAN_DEMO_DEADLOCK") == "1" {
		deadlock()
		return
	}

	basicDeclare()
	basicSendRecvClose()
	basicCommaOK()
	basicRange()
	basicDirectional()
	basicLenCap()
	basicSelect()

	patternBroadcast()
	patternWorkerPool()
	patternFanIn()
	patternPipeline()
	patternSemaphore()
	patternFuture()

	selectRandomness()
	unbufferedVsBuffered()
	happensBefore()

	trapSendOnClosed()
	trapDoubleClose()
	trapNilChannel()
	trapGoroutineLeak()
	trapRangeNeverEnds()
	trapTimeAfter()
	trapBusyLoop()
	trapLenRace()
	trapAckNotDone()
	trapValueCopy()
	trapDeadlock()
	trapCloseOrder()
}

// ---------------------------------------------------------------------------
// 1.1 声明与初始化
// ---------------------------------------------------------------------------

func basicDeclare() {
	section("1.1 声明与初始化")

	var c1 chan int           // nil channel，收发都永久阻塞
	c2 := make(chan int)      // 无缓冲（同步）
	c3 := make(chan int, 5)   // 有缓冲（异步，容量 5）
	c4 := make(chan struct{}) // 纯信号

	fmt.Printf("nil:%t  无缓冲 cap=%d  有缓冲 cap=%d  信号 chan %T\n",
		c1 == nil, cap(c2), cap(c3), c4)
}

// ---------------------------------------------------------------------------
// 1.2 发送、接收、关闭
// ---------------------------------------------------------------------------

func basicSendRecvClose() {
	section("1.2 发送/接收/关闭")

	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	close(ch) // 关闭后不能再发，但能把缓冲里剩下的读完

	fmt.Println("读:", <-ch, <-ch)
	fmt.Println("读空的已关闭 channel 立即返回零值:", <-ch)
}

// ---------------------------------------------------------------------------
// 1.3 comma-ok 接收与关闭语义
// ---------------------------------------------------------------------------

func basicCommaOK() {
	section("1.3 comma-ok 区分零值与关闭")

	ch := make(chan int, 1)
	ch <- 0 // 真实发送的零值
	close(ch)

	for range 2 {
		v, ok := <-ch
		fmt.Printf("v=%d ok=%t (%s)\n", v, ok,
			map[bool]string{true: "真实数据", false: "channel 已关闭且读空"}[ok])
	}
}

// ---------------------------------------------------------------------------
// 1.4 range 遍历 channel
// ---------------------------------------------------------------------------

func basicRange() {
	section("1.4 range 遍历")

	ch := make(chan int)
	go func() {
		defer close(ch) // 生产者负责关闭，range 才能正常结束
		for i := range 3 {
			ch <- i
		}
	}()

	for v := range ch {
		fmt.Print(v, " ")
	}
	fmt.Println("<- close 后 range 自然退出")
}

// ---------------------------------------------------------------------------
// 1.5 单向 channel：把"谁能关"写进签名
// ---------------------------------------------------------------------------

func produce(out chan<- int) { // 只能发送 + close
	defer close(out)
	for i := range 3 {
		out <- i * i
	}
}

func consume(in <-chan int) int { // 只能接收；close(in) 编译不过
	sum := 0
	for v := range in {
		sum += v
	}
	return sum
}

func basicDirectional() {
	section("1.5 单向 channel")

	ch := make(chan int, 3) // 双向 channel 可隐式转成单向，反之不行
	go produce(ch)
	fmt.Println("consume 求和:", consume(ch))
}

// ---------------------------------------------------------------------------
// 1.6 len 与 cap
// ---------------------------------------------------------------------------

func basicLenCap() {
	section("1.6 len / cap")

	ch := make(chan int, 5)
	ch <- 1
	fmt.Printf("有缓冲: len=%d cap=%d\n", len(ch), cap(ch))

	u := make(chan int)
	fmt.Printf("无缓冲: len=%d cap=%d\n", len(u), cap(u))

	var n chan int
	fmt.Printf("nil:    len=%d cap=%d（不 panic）\n", len(n), cap(n))
	fmt.Println("len/cap 只反映缓冲区，不含 sendq/recvq 上的等待者，只能用于监控")
}

// ---------------------------------------------------------------------------
// 1.7 select 多路复用
// ---------------------------------------------------------------------------

func basicSelect() {
	section("1.7 select")

	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	ch1 <- 42

	select {
	case v := <-ch1:
		fmt.Println("from ch1:", v)
	case ch2 <- 7:
		fmt.Println("sent to ch2")
	case <-time.After(time.Second):
		fmt.Println("timeout")
	}

	// ① 非阻塞尝试
	empty := make(chan int)
	select {
	case v := <-empty:
		fmt.Println("got", v)
	default:
		fmt.Println("非阻塞：没有数据就不等")
	}

	// ② 超时控制
	select {
	case v := <-empty:
		fmt.Println("got", v)
	case <-time.After(10 * time.Millisecond):
		fmt.Println("超时返回")
	}

	// ③ 可取消的循环
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	jobs := make(chan int)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("ctx 取消:", ctx.Err())
			return
		case j := <-jobs:
			_ = j
		}
	}
}

// ---------------------------------------------------------------------------
// 1.8 ① 信号通知 / 广播退出
// ---------------------------------------------------------------------------

func patternBroadcast() {
	section("1.8① 广播退出")

	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := range 3 {
		wg.Go(func() {
			<-done // 三个 goroutine 都阻塞在这里
			fmt.Println("  goroutine", i, "exit")
		})
	}
	close(done) // 一次 close 唤醒全部
	wg.Wait()
}

// ---------------------------------------------------------------------------
// 1.8 ② worker pool
// ---------------------------------------------------------------------------

func patternWorkerPool() {
	section("1.8② worker pool")

	jobs := make(chan int, 100)
	results := make(chan int, 100)

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for j := range jobs { // close(jobs) 后自然退出
				results <- j * 2
			}
		})
	}

	for i := 1; i <= 10; i++ {
		jobs <- i
	}
	close(jobs)                               // ① 关输入
	go func() { wg.Wait(); close(results) }() // ② worker 全退出再关输出

	sum := 0
	for r := range results { // ③ 读到结束
		sum += r
	}
	fmt.Println("结果之和:", sum)
}

// ---------------------------------------------------------------------------
// 1.8 ③ fan-in
// ---------------------------------------------------------------------------

func merge(chans ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, c := range chans {
		wg.Go(func() {
			for v := range c {
				out <- v
			}
		})
	}
	go func() { wg.Wait(); close(out) }() // 全部上游结束才关
	return out
}

func patternFanIn() {
	section("1.8③ fan-in")

	sum := 0
	for v := range merge(gen(1, 2), gen(3, 4), gen(5)) {
		sum += v
	}
	fmt.Println("合并三路之和:", sum)
}

// ---------------------------------------------------------------------------
// 1.8 ④ pipeline
// ---------------------------------------------------------------------------

func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

func sq(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

func patternPipeline() {
	section("1.8④ pipeline")

	for v := range sq(gen(1, 2, 3)) {
		fmt.Print(v, " ")
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// 1.8 ⑤ 信号量限流
// ---------------------------------------------------------------------------

func patternSemaphore() {
	section("1.8⑤ 带缓冲 channel 做限流")

	const limit = 3
	sem := make(chan struct{}, limit)

	var mu sync.Mutex
	cur, peak := 0, 0

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			sem <- struct{}{} // 获取令牌
			defer func() { <-sem }()

			mu.Lock()
			cur++
			peak = max(peak, cur)
			mu.Unlock()

			time.Sleep(time.Millisecond)

			mu.Lock()
			cur--
			mu.Unlock()
		})
	}
	wg.Wait()
	fmt.Printf("限流 %d，实测并发峰值 %d\n", limit, peak)
}

// ---------------------------------------------------------------------------
// 1.8 ⑥ future：容量 1 避免生产者泄漏
// ---------------------------------------------------------------------------

func doAsync() <-chan int {
	ch := make(chan int, 1) // 缓冲 1：调用方放弃接收也不会泄漏
	go func() { ch <- 42 }()
	return ch
}

func patternFuture() {
	section("1.8⑥ future")

	f := doAsync()
	fmt.Println("异步结果:", <-f)

	_ = doAsync() // 即使不读，生产者也能发完退出
	fmt.Println("放弃接收也不泄漏（缓冲区容量 = 发送者数量）")
}

// ---------------------------------------------------------------------------
// 2.8 select 的随机性
// ---------------------------------------------------------------------------

func selectRandomness() {
	section("2.8 select 多路就绪时随机选")

	a, b := make(chan int, 1), make(chan int, 1)
	countA, countB := 0, 0
	for range 100000 {
		a <- 1
		b <- 1
		select {
		case <-a:
			countA++
			<-b
		case <-b:
			countB++
			<-a
		}
	}
	fmt.Printf("两路都就绪 10 万次: A=%d B=%d（case 顺序不构成优先级）\n", countA, countB)
}

// ---------------------------------------------------------------------------
// 2.10 无缓冲 vs 有缓冲
// ---------------------------------------------------------------------------

func unbufferedVsBuffered() {
	section("2.10 无缓冲 vs 有缓冲")

	unbuf := make(chan int)
	start := time.Now()
	go func() {
		time.Sleep(30 * time.Millisecond)
		<-unbuf // 发送方要等到这里才返回
	}()
	unbuf <- 1
	fmt.Printf("无缓冲发送耗时 %v（必须等接收方就位，天然同步点）\n",
		time.Since(start).Round(10*time.Millisecond))

	buf := make(chan int, 1)
	start = time.Now()
	buf <- 1 // 缓冲区没满，立即返回
	fmt.Printf("有缓冲发送耗时 %v（只要没满就立即返回）\n",
		time.Since(start).Round(time.Millisecond))
}

// ---------------------------------------------------------------------------
// 2.9 内存模型：channel 提供的 happens-before
// ---------------------------------------------------------------------------

func happensBefore() {
	section("2.9 happens-before")

	var data string
	done := make(chan struct{})

	go func() {
		data = "写入完成" // 发生在 close 之前
		close(done)
	}()

	<-done // close happens-before 接收返回，因此这里读 data 无竞态
	fmt.Println("close/接收建立同步关系:", data)
}

// ---------------------------------------------------------------------------
// 3.1 向已关闭的 channel 发送
// ---------------------------------------------------------------------------

func trapSendOnClosed() {
	section("3.1 向已关闭 channel 发送")

	defer func() { fmt.Println("recover:", recover()) }()

	ch := make(chan int, 1)
	close(ch)
	fmt.Println("Go 没有 isClosed API：check-then-act 必然有竞态")
	ch <- 1
}

// ---------------------------------------------------------------------------
// 3.2 重复 close / close(nil)
// ---------------------------------------------------------------------------

type Stopper struct {
	done chan struct{}
	once sync.Once
}

func (s *Stopper) Stop() { s.once.Do(func() { close(s.done) }) }

func trapDoubleClose() {
	section("3.2 重复 close")

	func() {
		defer func() { fmt.Println("recover:", recover()) }()
		ch := make(chan int)
		close(ch)
		close(ch)
	}()

	func() {
		defer func() { fmt.Println("recover:", recover()) }()
		var n chan int
		close(n)
	}()

	s := &Stopper{done: make(chan struct{})}
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(s.Stop) // 调多少次都安全
	}
	wg.Wait()
	<-s.done
	fmt.Println("sync.Once 收口后多次 Stop 安全")
}

// ---------------------------------------------------------------------------
// 3.3 nil channel：坑与特性
// ---------------------------------------------------------------------------

func trapNilChannel() {
	section("3.3 nil channel")

	// 坑：select 里 nil channel 的 case 被跳过，永远不就绪
	var nilCh chan int
	select {
	case <-nilCh:
		fmt.Println("永远不会走到这里")
	case <-time.After(10 * time.Millisecond):
		fmt.Println("nil channel 的 case 被 selectgo 跳过")
	}

	// 特性：置 nil = 动态关掉一条分支
	ch1, ch2 := gen(1, 2, 3), gen(10, 20)
	sum := 0
	for ch1 != nil || ch2 != nil {
		select {
		case v, ok := <-ch1:
			if !ok {
				ch1 = nil // 关掉分支，否则已关闭的 channel 会一直就绪导致忙轮询
				continue
			}
			sum += v
		case v, ok := <-ch2:
			if !ok {
				ch2 = nil
				continue
			}
			sum += v
		}
	}
	fmt.Println("两路耗尽后退出，和 =", sum)
}

// ---------------------------------------------------------------------------
// 3.4 goroutine 泄漏
// ---------------------------------------------------------------------------

func queryLeak(n int) int {
	ch := make(chan int) // 无缓冲：只有第一个发送者能发出去
	for i := range n {
		go func() { ch <- i }()
	}
	return <-ch // 只收一个 -> 其余永久阻塞
}

func queryBuffered(n int) int {
	ch := make(chan int, n) // 修法①：缓冲区容量 = 发送者数量
	for i := range n {
		go func() { ch <- i }()
	}
	return <-ch
}

func queryCtx(n int) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 修法②③：发送方也能放弃

	ch := make(chan int)
	for i := range n {
		go func() {
			select {
			case ch <- i:
			case <-ctx.Done():
			}
		}()
	}
	return <-ch
}

func trapGoroutineLeak() {
	section("3.4 goroutine 泄漏")

	before := runtime.NumGoroutine()
	queryLeak(20)
	time.Sleep(50 * time.Millisecond)
	fmt.Printf("无缓冲版本: goroutine %d -> %d（泄漏 19 个）\n", before, runtime.NumGoroutine())

	before = runtime.NumGoroutine()
	queryBuffered(20)
	queryCtx(20)
	time.Sleep(50 * time.Millisecond)
	fmt.Printf("修复版本:   goroutine %d -> %d\n", before, runtime.NumGoroutine())
	fmt.Println("排查手段: runtime.NumGoroutine / goleak / pprof goroutine profile（-race 查不出泄漏）")
}

// ---------------------------------------------------------------------------
// 3.7 for range 不会自己结束
// ---------------------------------------------------------------------------

func trapRangeNeverEnds() {
	section("3.7 range 的退出条件是'关闭且读空'")

	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3

	// 不 close 直接 range 会永久阻塞；这里改成读固定次数
	for range 3 {
		fmt.Print(<-ch, " ")
	}
	fmt.Println("<- 读固定次数，或者由生产者 close")

	close(ch)
	for v := range ch { // close 之后 range 正常结束
		fmt.Print(v)
	}
	fmt.Println("close 后 range 正常退出")
}

// ---------------------------------------------------------------------------
// 3.8 select + time.After 的定时器开销
// ---------------------------------------------------------------------------

func trapTimeAfter() {
	section("3.8 循环里复用 Timer")

	ch := make(chan int)

	// 错误示范（这里只跑 3 轮）：每轮都新建一个 timer
	for range 3 {
		select {
		case <-ch:
		case <-time.After(time.Millisecond):
		}
	}
	fmt.Println("time.After: 每轮新建 timer，到期前不会被回收")

	// 正确写法：复用一个 Timer
	t := time.NewTimer(time.Millisecond)
	defer t.Stop()
	for range 3 {
		if !t.Stop() { // 复用前先停掉并排空
			select {
			case <-t.C:
			default:
			}
		}
		t.Reset(time.Millisecond)
		select {
		case <-ch:
		case <-t.C:
		}
	}
	fmt.Println("NewTimer + Reset: 全程只有一个 timer")
}

// ---------------------------------------------------------------------------
// 3.9 default 让 select 变成忙轮询
// ---------------------------------------------------------------------------

func trapBusyLoop() {
	section("3.9 default 与忙轮询")

	ch := make(chan int)
	go func() { time.Sleep(20 * time.Millisecond); ch <- 1 }()

	// 错误：for + default = 自旋 100% CPU（这里加计数来展示空转次数）
	spins := 0
	for {
		select {
		case v := <-ch:
			fmt.Printf("忙轮询拿到 %d，期间空转 %d 次\n", v, spins)
			goto correct
		default:
			spins++
		}
	}

correct:
	go func() { time.Sleep(20 * time.Millisecond); ch <- 2 }()
	select { // 去掉 default，让 goroutine 挂起等待
	case v := <-ch:
		fmt.Println("去掉 default 后阻塞等待，拿到", v, "，零空转")
	}
}

// ---------------------------------------------------------------------------
// 3.10 用 len/cap 做控制流是竞态的
// ---------------------------------------------------------------------------

func trapLenRace() {
	section("3.10 len/cap 不能做控制流")

	ch := make(chan int, 1)
	ch <- 1

	go func() { <-ch }() // 另一个 goroutine 可能抢先取走
	time.Sleep(10 * time.Millisecond)

	// 错误写法：if len(ch) > 0 { <-ch }  —— 判断和接收之间有竞态窗口
	select { // 正确：运行时在锁内做原子判断
	case v := <-ch:
		fmt.Println("非阻塞取到", v)
	default:
		fmt.Println("非阻塞：确实没有数据（select+default 才是原子的）")
	}
}

// ---------------------------------------------------------------------------
// 3.11 发送成功 ≠ 处理完成
// ---------------------------------------------------------------------------

type Task struct {
	payload string
	reply   chan error // 回执
}

func trapAckNotDone() {
	section("3.11 发送成功 ≠ 处理完成")

	tasks := make(chan Task)
	go func() {
		for t := range tasks {
			time.Sleep(10 * time.Millisecond) // 真正的处理
			t.reply <- nil
		}
	}()

	t := Task{payload: "x", reply: make(chan error, 1)}
	start := time.Now()
	tasks <- t
	fmt.Printf("发送返回耗时 %v（对方只是取走了任务）\n", time.Since(start).Round(time.Millisecond))

	<-t.reply
	fmt.Printf("等到回执耗时 %v（这才是处理完成）\n", time.Since(start).Round(10*time.Millisecond))
}

// ---------------------------------------------------------------------------
// 3.12 channel 传的是值的拷贝
// ---------------------------------------------------------------------------

type Msg struct{ Data []byte }

func trapValueCopy() {
	section("3.12 传值是拷贝，但是浅拷贝")

	ch := make(chan Msg, 1)
	buf := []byte("hello")
	ch <- Msg{Data: buf}

	buf[0] = 'H' // 改的是同一块底层数组
	got := <-ch
	fmt.Printf("接收方看到 %q（struct 是拷贝，底层数组仍共享）\n", got.Data)

	fmt.Println("大对象应该传 *T，但要约定'发出后不再修改'的所有权，否则 -race 会报竞争")
}

// ---------------------------------------------------------------------------
// 3.13 死锁（fatal error，recover 抓不住）
// ---------------------------------------------------------------------------

func trapDeadlock() {
	section("3.13 死锁检测")

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "CHAN_DEMO_DEADLOCK=1")
	out, err := cmd.CombinedOutput()
	firstLine := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	fmt.Printf("子进程退出: %v\n输出首行: %s\n", err, firstLine)
	fmt.Println("运行时只能检测'所有 goroutine 都休眠'，部分泄漏检测不了")
}

func deadlock() {
	defer func() { recover() }() // 抓不住 fatal error

	ch := make(chan int)
	ch <- 1 // 同一个 goroutine 先发后收 -> 自锁
	fmt.Println(<-ch)
}

// ---------------------------------------------------------------------------
// 3.15 关闭顺序
// ---------------------------------------------------------------------------

func trapCloseOrder() {
	section("3.15 关闭顺序：关输入 -> 等 worker -> 关输出")

	jobs := make(chan int, 10)
	results := make(chan int, 10)

	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() {
			for j := range jobs {
				results <- j
			}
		})
	}
	for i := range 10 {
		jobs <- i
	}

	close(jobs) // ①
	go func() { // ② 必须放在另一个 goroutine：否则 worker 阻塞在 results<- 时会互等
		wg.Wait()
		close(results)
	}()

	n := 0
	for range results { // ③
		n++
	}
	fmt.Println("按正确顺序关闭，收到", n, "条结果")
}
