package memo

import (
	"io"
	"net/http"
	"sync"
)

type Memo struct {
	f     Func
	cache map[string]*entry
	mu    sync.Mutex
}

type Func func(key string) (any, error)

type result struct {
	value any
	err   error
}

type entry struct {
	res   result
	ready chan struct{}
}

func New(f Func) *Memo {
	return &Memo{
		f:     f,
		cache: make(map[string]*entry),
		mu:    sync.Mutex{},
	}
}
func (m *Memo) Get(key string) (any, error) {
	m.mu.Lock()
	e := m.cache[key]
	if e == nil {
		// This is the first request for this key.
		// This goroutine becomes responsible for computing
		// the value and broadcasting the ready condition.
		e = &entry{ready: make(chan struct{})}
		m.cache[key] = e
		m.mu.Unlock()

		e.res.value, e.res.err = m.f(key)

		close(e.ready) // broadcast ready condition
	} else {
		// This is a repeat request for this key.
		m.mu.Unlock()

		<-e.ready // wait for ready condition
	}
	return e.res.value, e.res.err
}

func httpGetBody(url string) (any, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
