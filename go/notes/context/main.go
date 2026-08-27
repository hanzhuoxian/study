package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// 0. WithValue 的标准姿势：自定义空结构体做 key，避免跨包碰撞
// ---------------------------------------------------------------------------

type key struct{}

func New(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key{}, id)
}

func From(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(key{}).(string)
	return id, ok
}

// 另一个包里同样写 type key struct{} 也不会冲突：类型不同 ⇒ key 不相等。
type otherKey struct{}

func exampleWithValue() {
	root := context.Background()

	// 没有值时断言失败，ok == false，不会 panic（nil.(string) 的第二返回值兜底）
	if id, ok := From(root); !ok {
		fmt.Println("root 没有 requestID:", id == "")
	}

	ctx := New(root, "req-1001")
	id, ok := From(ctx)
	fmt.Println("From:", id, ok)

	// 同名不同类型的 key 互不影响
	ctx = context.WithValue(ctx, otherKey{}, "other")
	fmt.Println("otherKey:", ctx.Value(otherKey{}))

	// 派生 context 可以"遮蔽"父层同 key 的值：Value 沿链向上找，先命中最近的
	child := New(ctx, "req-2002")
	cid, _ := From(child)
	pid, _ := From(ctx)
	fmt.Printf("child=%s parent=%s（子层遮蔽，父层不受影响）\n", cid, pid)
}

// ---------------------------------------------------------------------------
// 1. WithCancel：手动取消 + worker 侧的标准 select 写法
// ---------------------------------------------------------------------------

// worker 是最常见的骨架：干活循环里始终留一路 <-ctx.Done()。
func worker(ctx context.Context, name string, out chan<- string) error {
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()

	for i := 1; ; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err() // Canceled 或 DeadlineExceeded
		case <-tick.C:
			select { // 发送也要可被取消，否则 out 无人接收时会永久阻塞
			case out <- fmt.Sprintf("%s#%d", name, i):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func exampleWithCancel() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 必须调用：即使下面已经手动 cancel 过，重复调用也是安全的（幂等）

	out := make(chan string)
	done := make(chan error, 1)
	go func() { done <- worker(ctx, "w", out) }()

	for range 3 {
		fmt.Println("收到:", <-out)
	}
	cancel() // 关闭 ctx.Done()，并级联取消所有派生 context

	fmt.Println("worker 退出:", <-done)
	fmt.Println("ctx.Err():", ctx.Err(), "| errors.Is(Canceled):", errors.Is(ctx.Err(), context.Canceled))
}

// ---------------------------------------------------------------------------
// 2. WithTimeout / WithDeadline：超时控制与 deadline 收敛
// ---------------------------------------------------------------------------

func exampleWithTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel() // 提前完成时释放定时器，别等它自然到期

	select {
	case <-time.After(500 * time.Millisecond): // 模拟一个慢活
		fmt.Println("活干完了")
	case <-ctx.Done():
		// 超时得到 DeadlineExceeded；它实现了 Timeout() bool，能被 net.Error 那套识别
		fmt.Println("超时:", ctx.Err(), "| errors.Is(DeadlineExceeded):", errors.Is(ctx.Err(), context.DeadlineExceeded))
	}

	// deadline 只能收紧不能放宽：子的 deadline 取 min(父, 子)
	parent, cancelP := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelP()
	child, cancelC := context.WithTimeout(parent, time.Hour) // 想要 1 小时，实际仍是 100ms
	defer cancelC()

	pd, _ := parent.Deadline()
	cd, _ := child.Deadline()
	fmt.Println("child deadline == parent deadline:", cd.Equal(pd))

	// 已经过期的 deadline：context 立刻就是取消态
	expired, cancelE := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelE()
	fmt.Println("过期 deadline 立即取消:", expired.Err())
}

// ---------------------------------------------------------------------------
// 3. WithCancelCause / Cause / WithTimeoutCause：区分"为什么取消"
// ---------------------------------------------------------------------------

var errBadInput = errors.New("bad input")

func exampleCause() {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errBadInput) // 只有第一次调用的原因生效，之后的调用被忽略
	cancel(errors.New("忽略"))

	<-ctx.Done()
	// Err() 永远是 Canceled（保持兼容），真正的原因要用 Cause 取
	fmt.Println("Err:", ctx.Err(), "| Cause:", context.Cause(ctx), "| Is(errBadInput):", errors.Is(context.Cause(ctx), errBadInput))

	// cancel(nil) 等价于普通 cancel：Cause 退化成 Canceled
	c2, cancel2 := context.WithCancelCause(context.Background())
	cancel2(nil)
	fmt.Println("cancel(nil) 的 Cause:", context.Cause(c2))

	// 超时也能带原因
	errSLA := errors.New("超出 SLA 30ms")
	c3, cancel3 := context.WithTimeoutCause(context.Background(), 30*time.Millisecond, errSLA)
	defer cancel3()
	<-c3.Done()
	fmt.Println("超时 Err:", c3.Err(), "| Cause:", context.Cause(c3))

	// 未取消时 Cause 返回 nil
	c4, cancel4 := context.WithCancelCause(context.Background())
	defer cancel4(nil)
	fmt.Println("未取消的 Cause:", context.Cause(c4))
}

// ---------------------------------------------------------------------------
// 4. WithoutCancel：切断取消传播，保留 Value 链
// ---------------------------------------------------------------------------

// 典型场景：请求已经返回/取消，但要用同一份请求元数据把审计日志异步写完。
func exampleWithoutCancel() {
	ctx, cancel := context.WithCancel(New(context.Background(), "req-3003"))

	detached := context.WithoutCancel(ctx)
	cancel() // 父被取消

	id, _ := From(detached)
	d, ok := detached.Deadline()
	fmt.Printf("父 Err=%v | detached Err=%v | 值仍在=%s | Done()==nil:%v | Deadline ok=%v(%v)\n",
		ctx.Err(), detached.Err(), id, detached.Done() == nil, ok, d.IsZero())

	// detached 上还能再挂自己的超时，与父完全独立
	bg, cancelBG := context.WithTimeout(detached, 20*time.Millisecond)
	defer cancelBG()
	<-bg.Done()
	fmt.Println("detached 的独立超时:", bg.Err())
}

// ---------------------------------------------------------------------------
// 5. AfterFunc：取消后回调（不用自己起 goroutine 守着 Done）
// ---------------------------------------------------------------------------

func exampleAfterFunc() {
	ctx, cancel := context.WithCancel(context.Background())

	fired := make(chan string, 1)
	stop := context.AfterFunc(ctx, func() {
		// 在自己的 goroutine 里执行；ctx 已取消，这里别再用它做阻塞等待
		fired <- "cleanup: " + context.Cause(ctx).Error()
	})
	defer stop()

	cancel()
	fmt.Println(<-fired)

	// stop() 返回是否"由本次调用阻止了执行"：已经跑过了就是 false
	fmt.Println("stop() 阻止成功:", stop())

	// 在已取消的 ctx 上注册会立刻异步执行
	fired2 := make(chan struct{})
	context.AfterFunc(ctx, func() { close(fired2) })
	<-fired2
	fmt.Println("已取消的 ctx 上注册也会执行")

	// stop() 及时调用可以避免回调执行
	c2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	stop2 := context.AfterFunc(c2, func() { fmt.Println("不该打印") })
	fmt.Println("提前 stop 成功:", stop2())
	cancel2()
	time.Sleep(10 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// 6. 并发模式一：fan-out worker 池，任一失败即整体取消
// ---------------------------------------------------------------------------

func exampleFanOut() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan int)
	var wg sync.WaitGroup
	errOnce := make(chan error, 1)

	for w := range 3 {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case j, ok := <-jobs:
					if !ok {
						return
					}
					if j == 7 { // 模拟一个致命错误
						select {
						case errOnce <- fmt.Errorf("worker%d: job %d 失败", w, j):
						default: // 只记第一个错误
						}
						cancel() // 通知其它 worker 别干了
						return
					}
					time.Sleep(2 * time.Millisecond)
				}
			}
		}(w)
	}

producer:
	for j := 1; j <= 100; j++ {
		select {
		case jobs <- j:
		case <-ctx.Done(): // 生产者也必须能退出，否则会卡在无人消费的 jobs 上
			break producer
		}
	}
	close(jobs)
	wg.Wait()

	select {
	case err := <-errOnce:
		fmt.Println("整体失败:", err, "| ctx:", ctx.Err())
	default:
		fmt.Println("全部成功")
	}
}

// ---------------------------------------------------------------------------
// 7. 并发模式二：并发请求多个后端，取最快返回的那个
// ---------------------------------------------------------------------------

type result struct {
	from string
	val  string
	err  error
}

func fetch(ctx context.Context, name string, cost time.Duration) result {
	select {
	case <-time.After(cost):
		return result{from: name, val: name + "-data"}
	case <-ctx.Done():
		return result{from: name, err: ctx.Err()}
	}
}

func exampleFirstWin() {
	// 派生一个可取消的子 ctx，拿到答案后立刻 cancel，让落败的请求尽快释放资源
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	backends := map[string]time.Duration{"fast": 20 * time.Millisecond, "slow": 150 * time.Millisecond}
	// 缓冲 = 分支数，保证落败者写完就退出，不会因为没人读而泄漏
	ch := make(chan result, len(backends))
	for name, cost := range backends {
		go func(name string, cost time.Duration) { ch <- fetch(ctx, name, cost) }(name, cost)
	}

	first := <-ch
	cancel() // 关键：主动取消剩余请求
	fmt.Printf("最快返回: %s=%s err=%v\n", first.from, first.val, first.err)

	loser := <-ch
	fmt.Printf("落败者被取消: %s err=%v\n", loser.from, loser.err)
}

// ---------------------------------------------------------------------------
// 8. 并发模式三：pipeline 各级都携带同一个 ctx
// ---------------------------------------------------------------------------

func gen(ctx context.Context, n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out) // 上游负责关闭自己的 out，下游据此自然收尾
		for i := 1; i <= n; i++ {
			select {
			case out <- i:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- v * v:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func examplePipeline() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got := make([]int, 0, 3)
	for v := range square(ctx, gen(ctx, 1000)) {
		got = append(got, v)
		if len(got) == 3 {
			cancel() // 提前退出：cancel 让上游所有级都停下来，不泄漏 goroutine
			break
		}
	}
	fmt.Println("pipeline 提前退出，已取到:", got)
}

// ---------------------------------------------------------------------------
// 9. HTTP 场景：中间件注入值 + 请求取消的传播
// ---------------------------------------------------------------------------

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := New(r.Context(), "req-"+r.URL.Path[1:])
		next.ServeHTTP(w, r.WithContext(ctx)) // 用 WithContext 造新 *Request，不改原来的
	})
}

func exampleHTTP() {
	h := withRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := From(r.Context())
		fmt.Fprintf(w, "handler 看到 requestID=%s", id)
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/abc")
	if err != nil {
		fmt.Println("请求失败:", err)
		return
	}
	defer resp.Body.Close()
	buf := make([]byte, 128)
	n, _ := resp.Body.Read(buf)
	fmt.Println(string(buf[:n]))

	// 客户端侧：用 ctx 控制单次请求的超时
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(time.Second):
		case <-r.Context().Done(): // 客户端断开时服务端也能感知
			fmt.Println("服务端感知到客户端取消:", r.Context().Err())
		}
	}))
	defer slow.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, slow.URL, nil)
	if _, err := http.DefaultClient.Do(req); err != nil {
		// 包装后的错误要用 errors.Is 判断，不能用 ==
		fmt.Println("客户端超时:", errors.Is(err, context.DeadlineExceeded))
	}
	time.Sleep(20 * time.Millisecond) // 等服务端把它那行日志打出来
}

// ---------------------------------------------------------------------------
// 10. 两个易踩的坑
// ---------------------------------------------------------------------------

func examplePitfalls() {
	// 坑一：Background/TODO 的 Done() 是 nil channel，select 上它永远阻塞
	fmt.Println("Background().Done() == nil:", context.Background().Done() == nil)

	// 坑二：Done 与业务 case 同时就绪时 select 是随机选的。
	// 要求"取消优先"就得先单独探一次 Done。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	work := make(chan int, 1)
	work <- 42

	select { // 抢占式检查
	case <-ctx.Done():
		fmt.Println("取消优先，丢弃已就绪的业务数据:", ctx.Err())
	default:
		select {
		case v := <-work:
			fmt.Println("处理:", v)
		case <-ctx.Done():
			fmt.Println("取消:", ctx.Err())
		}
	}
}

func main() {
	run := []struct {
		name string
		fn   func()
	}{
		{"WithValue", exampleWithValue},
		{"WithCancel", exampleWithCancel},
		{"WithTimeout/WithDeadline", exampleWithTimeout},
		{"WithCancelCause/Cause", exampleCause},
		{"WithoutCancel", exampleWithoutCancel},
		{"AfterFunc", exampleAfterFunc},
		{"fan-out worker 池", exampleFanOut},
		{"最快返回胜出", exampleFirstWin},
		{"pipeline", examplePipeline},
		{"HTTP", exampleHTTP},
		{"常见坑", examplePitfalls},
	}
	for i, c := range run {
		fmt.Printf("\n===== %d. %s =====\n", i+1, c.name)
		c.fn()
	}
}
