package memo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"testing"
	"time"
)

// fakeFunc 是一个可控的 Func：进入时通过 started 上报，然后一直阻塞到
// release 被关闭（返回结果）或 done 被关闭（返回取消错误）。
type fakeFunc struct {
	mu      sync.Mutex
	calls   int
	started chan string
	release chan struct{}
}

func newFakeFunc() *fakeFunc {
	return &fakeFunc{
		started: make(chan string, 16),
		release: make(chan struct{}),
	}
}

func (s *fakeFunc) call(key string, done <-chan struct{}) (any, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	s.started <- key
	select {
	case <-s.release:
		return "value:" + key, nil
	case <-done:
		return nil, ErrCancelled
	}
}

func (s *fakeFunc) numCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// 取消唯一的等待者：Get 返回 ErrCancelled，且结果不进缓存，
// 后续的 Get 必须重新调用 f。
func TestCancelledResultIsNotCached(t *testing.T) {
	s := newFakeFunc()
	m := New(s.call)
	defer m.Close()

	cancel := make(chan struct{})
	errc := make(chan error, 1)
	go func() {
		_, err := m.Get("k", cancel)
		errc <- err
	}()

	<-s.started // f 已经开始执行，说明请求已被 server 受理
	close(cancel)

	if err := <-errc; !errors.Is(err, ErrCancelled) {
		t.Fatalf("Get 返回 %v, 期望 %v", err, ErrCancelled)
	}

	// 放行 f，若被取消的结果被错误地缓存了，下面的 Get 就不会重新调用 f。
	close(s.release)

	value, err := m.Get("k", nil)
	if err != nil {
		t.Fatalf("重新 Get 失败: %v", err)
	}
	if value != "value:k" {
		t.Errorf("value = %v, 期望 value:k", value)
	}
	if got := s.numCalls(); got != 2 {
		t.Errorf("f 被调用 %d 次, 期望 2 次（取消的结果不应被缓存）", got)
	}
}

// 一个调用方的取消不应该影响其他等待同一个 key 的调用方，
// 也不应该阻止有效结果进入缓存。
func TestCancelOneWaiterKeepsSharedResult(t *testing.T) {
	s := newFakeFunc()
	m := New(s.call)
	defer m.Close()

	// 先发起一个不会取消的请求，确保 entry 建立且一直有人在等。
	keep := make(chan result, 1)
	go func() {
		v, err := m.Get("k", nil)
		keep <- result{v, err}
	}()
	<-s.started

	// 再发起一个会被取消的请求。
	cancel := make(chan struct{})
	errc := make(chan error, 1)
	go func() {
		_, err := m.Get("k", cancel)
		errc <- err
	}()
	close(cancel)

	if err := <-errc; !errors.Is(err, ErrCancelled) {
		t.Fatalf("被取消的 Get 返回 %v, 期望 %v", err, ErrCancelled)
	}

	// f 不应该被取消，因为第一个调用方还在等。
	close(s.release)
	if res := <-keep; res.err != nil || res.value != "value:k" {
		t.Fatalf("未取消的 Get 得到 (%v, %v), 期望 (value:k, nil)", res.value, res.err)
	}

	// 有效结果应当已缓存。
	if _, err := m.Get("k", nil); err != nil {
		t.Fatalf("缓存命中的 Get 失败: %v", err)
	}
	if got := s.numCalls(); got != 1 {
		t.Errorf("f 被调用 %d 次, 期望 1 次（结果应被缓存并复用）", got)
	}
}

// 并发请求同一个 key 时 f 只应被调用一次。
func TestConcurrentDuplicateSuppression(t *testing.T) {
	s := newFakeFunc()
	close(s.release) // f 立即返回
	m := New(s.call)
	defer m.Close()

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := m.Get("k", nil)
			if err != nil {
				t.Errorf("Get 失败: %v", err)
				return
			}
			if value != "value:k" {
				t.Errorf("value = %v, 期望 value:k", value)
			}
		}()
	}
	wg.Wait()

	if got := s.numCalls(); got != 1 {
		t.Errorf("f 被调用 %d 次, 期望 1 次", got)
	}
}

// httpGetBody 把 done channel 桥接到 http 请求的 context 上，
// done 关闭时正在进行的请求会立即中断。
func httpGetBody(url string, done <-chan struct{}) (any, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-done:
			cancel()
		case <-ctx.Done():
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func TestMemo(t *testing.T) {
	if testing.Short() {
		t.Skip("需要网络访问")
	}
	urls := []string{
		"https://golang.org",
		"https://godoc.org",
		"https://play.golang.org",
		"http://gopl.io",
		"https://golang.org",
		"https://godoc.org",
		"https://play.golang.org",
		"http://gopl.io",
	}
	m := New(httpGetBody)
	defer m.Close()

	var n sync.WaitGroup
	for _, url := range urls {
		n.Add(1)
		go func(url string) {
			defer n.Done()
			start := time.Now()
			value, err := m.Get(url, nil)
			if err != nil {
				log.Print(err)
				return
			}
			fmt.Printf("%s, %s, %d bytes\n",
				url, time.Since(start), len(value.([]byte)))
		}(url)
	}
	n.Wait()
}

// 用超时 channel 取消一次真实的 HTTP 请求。
func TestMemoHTTPCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("需要网络访问")
	}
	m := New(httpGetBody)
	defer m.Close()

	done := make(chan struct{})
	time.AfterFunc(10*time.Millisecond, func() { close(done) })

	start := time.Now()
	if _, err := m.Get("https://golang.org", done); err == nil {
		t.Skip("请求在取消生效前就完成了")
	} else {
		t.Logf("%s 后被取消: %v", time.Since(start), err)
	}
}
