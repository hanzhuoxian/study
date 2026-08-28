package tst

import (
	"testing"
	"testing/synctest"
	"time"
)

// testing/synctest（1.24 实验性，1.25 转正）
//
// 解决的问题：测并发/超时代码时，要么真的 sleep（测试慢），要么注入假时钟（侵入代码）。
// synctest 给出第三条路：在一个"气泡"里跑，气泡内 time 包用**假时钟**，
// 所有 goroutine 都 durably blocked 时时间自动跳到下一个唤醒点。

// ① 假时钟：这个测试瞬间跑完，不是真等 2 秒
func TestSynctestFakeClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now() // 气泡内初始时间恒为 2000-01-01 00:00:00 UTC
		t.Logf("气泡内起始时间: %v", start.UTC().Format(time.RFC3339))

		go func() {
			time.Sleep(time.Second)
			if d := time.Since(start); d != time.Second {
				t.Errorf("Since = %v, want 1s", d)
			}
		}()

		time.Sleep(2 * time.Second) // 真实耗时接近 0
		if d := time.Since(start); d != 2*time.Second {
			t.Errorf("Since = %v, want 2s", d)
		}
	})
}

// ② synctest.Wait：等所有其他 goroutine 都 durably blocked
func TestSynctestWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		done := false
		go func() { done = true }()

		synctest.Wait() // 阻塞直到上面的 goroutine 跑完（不需要 channel/WaitGroup）

		if !done {
			t.Error("Wait 之后 goroutine 应该已经完成")
		}
	})
}

// ③ 实战：测一个带超时的函数，不用真等
func fetchWithTimeout(ch <-chan string, timeout time.Duration) (string, bool) {
	select {
	case v := <-ch:
		return v, true
	case <-time.After(timeout):
		return "", false
	}
}

func TestFetchTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ch := make(chan string)

		// 超时分支：假时钟直接跳过 30 秒
		start := time.Now()
		if _, ok := fetchWithTimeout(ch, 30*time.Second); ok {
			t.Error("应该超时")
		}
		if elapsed := time.Since(start); elapsed != 30*time.Second {
			t.Errorf("气泡内经过时间 = %v, want 30s", elapsed)
		}
	})
}

func TestFetchSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ch := make(chan string, 1)
		ch <- "value"

		v, ok := fetchWithTimeout(ch, 30*time.Second)
		if !ok || v != "value" {
			t.Errorf("= %q, %v", v, ok)
		}
	})
}

// ④ 注意事项（文档里明确列出的）
//
// 会让 goroutine "durably blocked"（气泡认可、能推进时间）：
//   - 收发气泡内创建的 channel
//   - 所有 case 都是气泡内 channel 的 select
//   - sync.Cond.Wait
//   - sync.WaitGroup.Wait（Add 也在气泡内调用）
//   - time.Sleep
//
// **不算** durably blocked（会导致死锁 panic 或时间不推进）：
//   - 加锁 sync.Mutex / RWMutex
//   - 网络 I/O、文件 I/O
//   - 系统调用
//
// 所以用 synctest 的前提是：被测代码的并发完全靠 channel/time/WaitGroup 表达，
// 外部依赖要用 fake 替换掉。
