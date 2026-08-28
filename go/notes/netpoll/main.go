// 网络轮询器示例：对应 notes/netpoll.md（调度见 sched.md，结构见 gmp.md）
//
//	go run ./netpoll               跑全部演示
//	NETPOLL_N=2000 go run ./netpoll  加大空闲连接数（注意 ulimit -n）
//
// 平台说明：本文件在 darwin(kqueue) 与 linux(epoll) 上都能跑，
// 但 1.2 的 pollable 判定结果在两个平台上不同（见 netpoll.md 1.2）。
package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"runtime/metrics"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

// pad 按终端列宽补空格：CJK 字符占 2 列，fmt 的 %-Ns 按 rune 数算会对不齐。
func pad(s string, width int) string {
	w := 0
	for _, r := range s {
		if r >= 0x1100 && (r <= 0x115F || (r >= 0x2E80 && r <= 0xA4CF) ||
			(r >= 0xAC00 && r <= 0xD7A3) || (r >= 0xF900 && r <= 0xFAFF) ||
			(r >= 0xFE30 && r <= 0xFE6F) || (r >= 0xFF00 && r <= 0xFF60) ||
			(r >= 0xFFE0 && r <= 0xFFE6)) {
			w += 2
		} else {
			w++
		}
	}
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func main() {
	layers()
	whoIsPollable()
	fdSideEffect()

	blockedReadLooksLike()

	idleConnsDoNotEatThreads()
	blockingFDsDoEatThreads()

	deadlineBasics()
	deadlineIsAbsolute()
	deadlineAsCancel()

	leakedConnections()
	observability()
}

// ---------------------------------------------------------------------------
// 1.1 从 conn.Read() 到 epoll_wait
// ---------------------------------------------------------------------------

func layers() {
	section("1.1 从 net.Conn 到 epoll/kqueue 的六层")

	for _, row := range [][3]string{
		{"net.TCPConn", "net/tcpsock.go", "用户看到的同步阻塞 API"},
		{"net.netFD", "net/fd_posix.go", "网络语义（地址、类型）"},
		{"internal/poll.FD", "internal/poll/fd_unix.go", "重试循环：EAGAIN -> waitRead -> 重试"},
		{"internal/poll.pollDesc", "internal/poll/fd_poll_runtime.go", "只有一个 runtimeCtx uintptr"},
		{"runtime.pollDesc", "runtime/netpoll.go:75", "rg/wg 两个信号量 + 两个 deadline timer"},
		{"netpollopen/netpoll", "runtime/netpoll_{epoll,kqueue}.go", "平台实现：epoll_ctl / kevent"},
	} {
		fmt.Printf("  %-24s %-38s %s\n", row[0], row[1], row[2])
	}

	fmt.Println()
	fmt.Println("  runtime 要求每个平台实现 5 个函数（netpoll.go 顶部注释）：")
	for _, s := range []string{
		"netpollinit()                     初始化，只调用一次",
		"netpollopen(fd, pd) errno         把 fd 以**边缘触发**注册进来",
		"netpollclose(fd) errno            注销",
		"netpoll(delta) (gList, int32)     delta<0 阻塞 / ==0 非阻塞 / >0 超时，返回就绪的 G 列表",
		"netpollBreak()                    唤醒阻塞在 netpoll 里的那个 M",
	} {
		fmt.Println("    ·", s)
	}
}

// ---------------------------------------------------------------------------
// 1.2 谁进得了 netpoll：看 O_NONBLOCK
// ---------------------------------------------------------------------------

// nonblock 用 fcntl(F_GETFL) 读 fd 的状态标志。
// 走 SyscallConn().Control 而不是 Fd()，因为 Fd() 有副作用（见 1.3）。
func nonblock(c syscall.Conn) (bool, error) {
	rc, err := c.SyscallConn()
	if err != nil {
		return false, err
	}
	var fl uintptr
	var errno syscall.Errno
	if err := rc.Control(func(fd uintptr) {
		fl, _, errno = syscall.Syscall(syscall.SYS_FCNTL, fd, uintptr(syscall.F_GETFL), 0)
	}); err != nil {
		return false, err
	}
	if errno != 0 {
		return false, errno
	}
	return fl&syscall.O_NONBLOCK != 0, nil
}

func whoIsPollable() {
	section("1.2 谁进得了 netpoll（O_NONBLOCK == 被 runtime 接管）")

	fmt.Println("  Go 只给「能被 poll 的 fd」设置 O_NONBLOCK，所以这个标志就是 pollable 的指示灯。")
	fmt.Println()

	report := func(name string, c syscall.Conn, closer func()) {
		if closer != nil {
			defer closer()
		}
		nb, err := nonblock(c)
		switch {
		case err != nil:
			fmt.Printf("  %s ERROR %v\n", pad(name, 22), err)
		case nb:
			fmt.Printf("  %s O_NONBLOCK=true   -> 注册进 netpoll，阻塞时只挂起 G\n", pad(name, 22))
		default:
			fmt.Printf("  %s O_NONBLOCK=false  -> 没进 netpoll，阻塞时占住整个线程\n", pad(name, 22))
		}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("  listen 失败：", err)
		return
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { io.Copy(c, c); c.Close() }()
		}
	}()

	if c, err := net.Dial("tcp", ln.Addr().String()); err == nil {
		report("TCP conn", c.(syscall.Conn), func() { c.Close() })
	}
	if pc, err := net.ListenPacket("udp", "127.0.0.1:0"); err == nil {
		report("UDP conn", pc.(syscall.Conn), func() { pc.Close() })
	}
	if pr, pw, err := os.Pipe(); err == nil {
		report("os.Pipe（管道）", pr, func() { pr.Close(); pw.Close() })
	}
	if f, err := os.CreateTemp("", "netpoll"); err == nil {
		name := f.Name()
		report("os.File（普通文件）", f, func() { f.Close(); os.Remove(name) })
	}
	report("os.Stdin", os.Stdin, nil)

	fmt.Println()
	fmt.Printf("  当前平台 GOOS=%s：\n", runtime.GOOS)
	fmt.Println("    · linux   —— 普通文件会被尝试 epoll_ctl(ADD)，内核返回 EPERM，")
	fmt.Println("                  Init 里 fallback 成 isBlocking=1（fd_unix.go FD.Init）")
	fmt.Println("    · darwin/BSD —— os 包**主动跳过**普通文件和目录，不往 kqueue 里塞")
	fmt.Println("                  （os/file_unix.go newFile，issue #19093/#24164/#66239）")
	fmt.Println("    · darwin 上 FIFO 也被跳过：关掉最后一个写端不会产生 kqueue 事件")
	fmt.Println()
	fmt.Println("  结论：网络 fd 一定 pollable；普通文件一定不 pollable —— 这就是")
	fmt.Println("  「网络 IO 不涨线程、文件 IO 涨线程」的根因（见 3.1/3.2 和 sched.md 3.2）。")
}

// ---------------------------------------------------------------------------
// 1.3 f.Fd() 的副作用
// ---------------------------------------------------------------------------

func fdSideEffect() {
	section("1.3 坑：f.Fd() 会把 fd 退回阻塞模式")

	pr, pw, err := os.Pipe()
	if err != nil {
		fmt.Println("  pipe 失败：", err)
		return
	}
	defer pr.Close()
	defer pw.Close()

	before, _ := nonblock(pr)
	raw := pr.Fd() // 副作用就在这一行
	after, _ := nonblock(pr)

	fmt.Printf("  os.Pipe 读端：Fd() 之前 O_NONBLOCK=%v，Fd() 之后 O_NONBLOCK=%v（fd=%d）\n",
		before, after, raw)
	fmt.Println()
	fmt.Println("  os/file_unix.go (*File).fd()：")
	fmt.Println("    if f.nonblock { f.pfd.SetBlocking() }")
	fmt.Println("  注释写得很直白——历史上 Fd() 一直返回阻塞 fd，为了兼容只能保持。")
	fmt.Println("  代价：这个 File 从此脱离 netpoll，之后每次读写都占住一个 OS 线程。")
	fmt.Println()
	fmt.Println("  正确做法：需要 fd 做 setsockopt 之类的事情时用 SyscallConn().Control(fn)，")
	fmt.Println("  它在 Control 期间只加引用计数，不改变阻塞模式。")
}

// ---------------------------------------------------------------------------
// 2.1 阻塞在网络读上的 goroutine 长什么样
// ---------------------------------------------------------------------------

func blockedReadLooksLike() {
	section("2.1 阻塞在网络读上的 goroutine")

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("  listen 失败：", err)
		return
	}
	defer ln.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		io.Copy(io.Discard, c) // 卡在这里：对端不发数据
	}()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		fmt.Println("  dial 失败：", err)
		return
	}
	time.Sleep(50 * time.Millisecond)

	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, true)
	for _, line := range strings.Split(string(buf[:n]), "\n") {
		if strings.HasPrefix(line, "goroutine ") && strings.Contains(line, "IO wait") {
			fmt.Println("  dump:", line)
		}
		if strings.Contains(line, "internal/poll.runtime_pollWait") ||
			strings.Contains(line, "internal/poll.(*pollDesc).wait") {
			fmt.Println("  栈帧:", strings.TrimSpace(line))
		}
	}

	fmt.Println()
	fmt.Println("  [IO wait] 就是 waitReasonIOWait（runtime2.go:1225），只有 netpollblock")
	fmt.Println("  里的那一处 gopark 会产生它（netpoll.go:575）：")
	fmt.Println("    gopark(netpollblockcommit, ..., waitReasonIOWait, traceBlockNet, 5)")
	fmt.Println()
	fmt.Println("  完整的读路径（internal/poll/fd_unix.go (*FD).Read）：")
	fmt.Println("    for {")
	fmt.Println("      n, err := syscall.Read(fd.Sysfd, p)")
	fmt.Println("      if err == EAGAIN && fd.pd.pollable() {")
	fmt.Println("        if err = fd.pd.waitRead(fd.isFile); err == nil { continue }  // <- 挂起在这")
	fmt.Println("      }")
	fmt.Println("      return n, err")
	fmt.Println("    }")
	fmt.Println("  唤醒时不是「拿到数据」，只是「fd 可读了」，所以必须 continue 重新读。")

	c.Close()
	wg.Wait()
}

// ---------------------------------------------------------------------------
// 3.1 一万个空闲连接不会变成一万个线程
// ---------------------------------------------------------------------------

func threads() int { return pprof.Lookup("threadcreate").Count() }

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}

func idleConnsDoNotEatThreads() {
	section("3.1 N 个空闲连接 -> N 个 goroutine，但线程数不动")

	n := envInt("NETPOLL_N", 500)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("  listen 失败：", err)
		return
	}
	defer ln.Close()

	g0, t0 := runtime.NumGoroutine(), threads()
	fmt.Printf("  开始：goroutines=%d threads=%d GOMAXPROCS=%d\n", g0, t0, runtime.GOMAXPROCS(0))

	var mu sync.Mutex
	var served []net.Conn
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			served = append(served, c)
			mu.Unlock()
			go io.Copy(io.Discard, c) // 每个连接一个 goroutine，全部卡在 IO wait
		}
	}()

	clients := make([]net.Conn, 0, n)
	for i := 0; i < n; i++ {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			fmt.Printf("  第 %d 个连接失败（多半是 ulimit -n）：%v\n", i, err)
			break
		}
		clients = append(clients, c)
	}
	time.Sleep(200 * time.Millisecond)

	g1, t1 := runtime.NumGoroutine(), threads()
	fmt.Printf("  %d 个空闲连接后：goroutines=%d (+%d) threads=%d (+%d)\n",
		len(clients), g1, g1-g0, t1, t1-t0)
	fmt.Println()
	fmt.Println("  goroutine 数量线性增长（每个连接一个），线程数几乎不动：")
	fmt.Println("  所有 G 都 _Gwaiting 在 pollDesc.rg 上，没有任何一个占着 M。")
	fmt.Println("  它们的唤醒全靠 findRunnable/sysmon 调用 netpoll() 一次性捞回一批。")

	for _, c := range clients {
		c.Close()
	}
	ln.Close()
	<-done
	mu.Lock()
	for _, c := range served {
		c.Close()
	}
	mu.Unlock()
}

// ---------------------------------------------------------------------------
// 3.2 反例：把 fd 踢出 netpoll，线程立刻涨
// ---------------------------------------------------------------------------

func blockingFDsDoEatThreads() {
	section("3.2 反例：同样的管道，脱离 netpoll 之后线程就涨了")

	const n = 100

	run := func(label string, escape bool) (dg, dt int) {
		type pipe struct{ r, w *os.File }
		pipes := make([]pipe, 0, n)
		defer func() {
			for _, p := range pipes {
				p.r.Close()
				p.w.Close()
			}
		}()

		g0, t0 := runtime.NumGoroutine(), threads()
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			r, w, err := os.Pipe()
			if err != nil {
				break
			}
			if escape {
				_ = r.Fd() // 退回阻塞模式（见 1.3）
			}
			pipes = append(pipes, pipe{r, w})
			wg.Add(1)
			go func(r *os.File) {
				defer wg.Done()
				io.Copy(io.Discard, r) // 卡住，直到写端关闭
			}(r)
		}
		time.Sleep(300 * time.Millisecond)
		dg, dt = runtime.NumGoroutine()-g0, threads()-t0

		for _, p := range pipes {
			p.w.Close()
		}
		wg.Wait()
		fmt.Printf("  %s goroutines +%-4d threads +%d\n", pad(label, 34), dg, dt)
		return
	}

	_, tPoll := run(fmt.Sprintf("%d 个管道（netpoll 接管）", n), false)
	_, tBlock := run(fmt.Sprintf("%d 个管道（Fd() 后阻塞）", n), true)

	fmt.Println()
	fmt.Printf("  线程增量：%d vs %d\n", tPoll, tBlock)
	fmt.Println("  同样的 fd、同样的读法，唯一区别是有没有被 netpoll 接管。")
	fmt.Println("  阻塞式读会走 entersyscall，sysmon 的 retake() 把 P 交给新 M（sched.md 3.2），")
	fmt.Println("  于是「并发数」直接翻译成「线程数」，上限 sched.maxmcount = 10000。")
	fmt.Println()
	fmt.Println("  普通文件 IO 也是这条路（1.2），所以高并发读文件要自己限流，")
	fmt.Println("  别指望 goroutine 便宜就无脑起。")
}

// ---------------------------------------------------------------------------
// 4.1 deadline 的基本行为
// ---------------------------------------------------------------------------

func dialEcho() (server net.Listener, conn net.Conn, err error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		io.Copy(c, c)
		c.Close()
	}()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		ln.Close()
		return nil, nil, err
	}
	return ln, c, nil
}

func deadlineBasics() {
	section("4.1 SetReadDeadline：唯一能打断阻塞 IO 的正规手段")

	ln, c, err := dialEcho()
	if err != nil {
		fmt.Println("  建连失败：", err)
		return
	}
	defer ln.Close()
	defer c.Close()

	c.SetReadDeadline(time.Now().Add(80 * time.Millisecond))
	start := time.Now()
	_, err = c.Read(make([]byte, 16))
	cost := time.Since(start)

	fmt.Printf("  Read 返回耗时 %v\n", cost.Round(time.Millisecond))
	fmt.Printf("  err                                  = %v\n", err)
	var ne net.Error
	fmt.Printf("  errors.As(err, &net.Error)           = %v\n", errors.As(err, &ne))
	if ne != nil {
		fmt.Printf("  ne.Timeout()                         = %v\n", ne.Timeout())
	}
	fmt.Printf("  errors.Is(err, os.ErrDeadlineExceeded) = %v\n", errors.Is(err, os.ErrDeadlineExceeded))
	fmt.Printf("  errors.Is(err, context.DeadlineExceeded) = false（两套体系，别混）\n")

	fmt.Println()
	fmt.Println("  实现：SetReadDeadline -> poll_runtime_pollSetDeadline（netpoll.go:372）")
	fmt.Println("  在 pollDesc 上挂一个 runtime timer（rt/rd/rseq）。timer 到期时")
	fmt.Println("  netpolldeadlineimpl 把 rd 置为 -1、publishInfo、再 netpollunblock 唤醒 G，")
	fmt.Println("  G 醒来后 netpollcheckerr 返回 pollErrTimeout -> ErrDeadlineExceeded。")
	fmt.Println("  读写共用一个 deadline 时（rd==wd）只挂一个 timer，省一半开销。")
}

// ---------------------------------------------------------------------------
// 4.2 deadline 是绝对时间点，不是「每次操作的超时」
// ---------------------------------------------------------------------------

func deadlineIsAbsolute() {
	section("4.2 deadline 是绝对时间点，不会自动续期")

	ln, c, err := dialEcho()
	if err != nil {
		fmt.Println("  建连失败：", err)
		return
	}
	defer ln.Close()
	defer c.Close()

	const budget = 120 * time.Millisecond
	c.SetReadDeadline(time.Now().Add(budget))
	buf := make([]byte, 4)

	for i := 1; i <= 3; i++ {
		if _, err := c.Write([]byte("ping")); err != nil {
			fmt.Println("  write 失败：", err)
			break
		}
		start := time.Now()
		n, err := io.ReadFull(c, buf)
		fmt.Printf("  第 %d 次读：n=%d cost=%-6v err=%v\n",
			i, n, time.Since(start).Round(time.Millisecond), err)
		if err != nil {
			break
		}
		time.Sleep(80 * time.Millisecond)
	}

	fmt.Println()
	fmt.Printf("  第三次读撞上了 %v 前设好的那个绝对时间点——deadline 不随读操作重置。\n", budget)
	fmt.Println("  正确用法：每次读之前重新 SetReadDeadline(time.Now().Add(d))；")
	fmt.Println("  bufio.Reader 包一层时尤其容易忘（一次 Read 可能对应多次底层 syscall）。")
	fmt.Println()
	fmt.Println("  清除：SetReadDeadline(time.Time{})，d==0 表示永不超时。")
	fmt.Println("  一旦 deadline 已过期，后续所有读都立刻返回超时，连接并**没有**被关闭，")
	fmt.Println("  但半截的协议状态通常已经不可恢复了，实践中直接 Close 更安全。")
}

// ---------------------------------------------------------------------------
// 4.3 用 deadline 实现取消
// ---------------------------------------------------------------------------

func deadlineAsCancel() {
	section("4.3 在另一个 goroutine 里 SetDeadline = 取消一次阻塞的 IO")

	ln, c, err := dialEcho()
	if err != nil {
		fmt.Println("  建连失败：", err)
		return
	}
	defer ln.Close()
	defer c.Close()

	done := make(chan error, 1)
	go func() {
		_, err := c.Read(make([]byte, 16))
		done <- err
	}()

	time.Sleep(60 * time.Millisecond)
	// 把 deadline 设到过去 -> 立刻唤醒卡在 Read 里的那个 goroutine
	c.SetReadDeadline(time.Now().Add(-time.Second))

	select {
	case err := <-done:
		fmt.Printf("  被唤醒的 Read 返回：%v\n", err)
	case <-time.After(time.Second):
		fmt.Println("  没被唤醒（不应该发生）")
	}

	fmt.Println()
	fmt.Println("  SetDeadline 是并发安全的，可以在任意 goroutine 调用。")
	fmt.Println("  这是把 context 接到网络 IO 上的标准写法：")
	fmt.Println("    stop := context.AfterFunc(ctx, func() { conn.SetDeadline(时间点在过去) })")
	fmt.Println("    defer stop()")
	fmt.Println("  Close() 也能唤醒（poll_runtime_pollUnblock -> pollErrClosing），")
	fmt.Println("  但那是一次性的：连接没法再用了。")
}

// ---------------------------------------------------------------------------
// 5.1 泄漏
// ---------------------------------------------------------------------------

func leakedConnections() {
	section("5.1 忘记 Close：goroutine 泄漏 + fd 泄漏 + CLOSE_WAIT")

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("  listen 失败：", err)
		return
	}

	g0 := runtime.NumGoroutine()
	stop := make(chan struct{})
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				<-stop // 故意不 Close，也不读：模拟「处理完了但连接没释放」
				c.Close()
			}()
		}
	}()

	for i := 0; i < 20; i++ {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			break
		}
		c.Close() // 客户端主动关：服务端进入 CLOSE_WAIT
	}
	time.Sleep(150 * time.Millisecond)

	fmt.Printf("  客户端全部 Close 之后，服务端 goroutine 仍然 +%d\n", runtime.NumGoroutine()-g0)
	fmt.Println("  此时用 `lsof -p <pid> | grep CLOSE_WAIT` 能看到一串半关闭连接。")
	fmt.Println()
	fmt.Println("  三条经验：")
	fmt.Println("    · CLOSE_WAIT 堆积 = **你的代码**没 Close（TIME_WAIT 堆积才是对端/内核的事）")
	fmt.Println("    · http.Response.Body 必须 Close，而且要读完（io.Copy(io.Discard, body)），")
	fmt.Println("      否则 Transport 判定连接脏，不放回池子，等于每次请求重新握手")
	fmt.Println("    · pollDesc 是从 pollCache 里分配的、NotInHeap 的，泄漏了 GC 也回收不了")

	close(stop)
	ln.Close()
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// 6.1 可观测性
// ---------------------------------------------------------------------------

func observability() {
	section("6.1 怎么看 netpoll 的状态")

	names := []string{
		"/sched/goroutines:goroutines",
		"/sched/goroutines/running:goroutines",
		"/sched/goroutines/runnable:goroutines",
		"/sched/goroutines/waiting:goroutines",
		"/sched/goroutines/not-in-go:goroutines",
	}
	samples := make([]metrics.Sample, len(names))
	for i, n := range names {
		samples[i].Name = n
	}
	metrics.Read(samples)
	for _, s := range samples {
		if s.Value.Kind() == metrics.KindUint64 {
			fmt.Printf("  %-42s = %d\n", s.Name, s.Value.Uint64())
		} else {
			fmt.Printf("  %-42s = (本版本不支持)\n", s.Name)
		}
	}

	fmt.Println()
	fmt.Println("  /sched/goroutines/{running,runnable,waiting,not-in-go} 是 Go 1.26 新增的，")
	fmt.Println("  无 STW，适合常驻采集：waiting 高而 runnable 低 = 大量 G 挂在 IO/锁上（正常的连接池），")
	fmt.Println("  not-in-go 高 = 阻塞式 syscall/cgo 多（线程要涨了）。")
	fmt.Println()
	for _, row := range [][2]string{
		{"/debug/pprof/goroutine?debug=2", "找 [IO wait] 和 internal/poll.(*pollDesc).wait 栈帧"},
		{"runtime/trace 的 Network blocking profile", "G 在网络上总共等了多久（profile.md 5.2）"},
		{"trace 里的 GoUnblock(reason=network)", "netpoll 唤醒事件，能看到 poll -> 运行的延迟"},
		{"pprof.Lookup(\"threadcreate\").Count()", "线程数；远超 GOMAXPROCS 说明有非 pollable 的阻塞"},
		{"GODEBUG=schedtrace=1000", "idleprocs 常年满 + runqueue 空 = 在等 IO，不是 CPU 瓶颈"},
		{"lsof / ss -s", "fd 总数、CLOSE_WAIT 数量"},
	} {
		fmt.Printf("  %s %s\n", pad(row[0], 44), row[1])
	}
}
