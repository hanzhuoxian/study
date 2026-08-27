package memo

import (
	"io"
	"net/http"
	"sync"
)

type Memo struct {
	f     Func
	cache map[string]result
	mu    sync.Mutex
}

type Func func(key string) (any, error)

type result struct {
	value any
	err   error
}

func New(f Func) *Memo {
	return &Memo{
		f:     f,
		cache: make(map[string]result),
		mu:    sync.Mutex{},
	}
}
func (m *Memo) Get(key string) (any, error) {
	m.mu.Lock()
	res, ok := m.cache[key]
	m.mu.Unlock()
	if !ok {
		res.value, res.err = m.f(key)
		m.mu.Lock()
		m.cache[key] = res
		m.mu.Unlock()
	}
	return res.value, res.err
}

func httpGetBody(url string) (any, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
