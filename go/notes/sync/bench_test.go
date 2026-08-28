package main

import (
	"sync"
	"sync/atomic"
	"testing"
)

// go test -bench . -benchmem ./sync
// 只看竞争场景：go test -bench 'Contended|Map' -benchmem -cpu 1,2,4,8 ./sync

// ---------------------------------------------------------------------------
// ① 四种"加一"的成本：无竞争
// ---------------------------------------------------------------------------

var (
	mu      sync.Mutex
	rw      sync.RWMutex
	cnt     int
	atomCnt atomic.Int64
	semCh   = make(chan struct{}, 1)
)

func BenchmarkMutexUncontended(b *testing.B) {
	for b.Loop() {
		mu.Lock()
		cnt++
		mu.Unlock()
	}
}

func BenchmarkRWMutexWriteUncontended(b *testing.B) {
	for b.Loop() {
		rw.Lock()
		cnt++
		rw.Unlock()
	}
}

func BenchmarkRWMutexReadUncontended(b *testing.B) {
	for b.Loop() {
		rw.RLock()
		_ = cnt
		rw.RUnlock()
	}
}

func BenchmarkAtomicUncontended(b *testing.B) {
	for b.Loop() {
		atomCnt.Add(1)
	}
}

// 用带缓冲 channel 当互斥锁：能用，但贵得多
func BenchmarkChanAsMutex(b *testing.B) {
	for b.Loop() {
		semCh <- struct{}{}
		cnt++
		<-semCh
	}
}

// ---------------------------------------------------------------------------
// ② 竞争场景：锁 vs 原子 vs 分片锁
// ---------------------------------------------------------------------------

func BenchmarkMutexContended(b *testing.B) {
	var mu sync.Mutex
	n := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			n++
			mu.Unlock()
		}
	})
}

func BenchmarkAtomicContended(b *testing.B) {
	var n atomic.Int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n.Add(1)
		}
	})
}

// 分片计数器：每个分片独占一条 cache line，消除伪共享
type shard struct {
	n atomic.Int64
	_ [64 - 8]byte // padding
}

func BenchmarkShardedCounter(b *testing.B) {
	shards := make([]shard, 64)
	var i atomic.Int64
	b.RunParallel(func(pb *testing.PB) {
		s := &shards[int(i.Add(1))%len(shards)]
		for pb.Next() {
			s.n.Add(1)
		}
	})
}

// 没有 padding 的版本：相邻计数器落在同一条 cache line 上，互相打脸
type unpaddedShard struct{ n atomic.Int64 }

func BenchmarkShardedCounterFalseSharing(b *testing.B) {
	shards := make([]unpaddedShard, 64)
	var i atomic.Int64
	b.RunParallel(func(pb *testing.PB) {
		s := &shards[int(i.Add(1))%len(shards)]
		for pb.Next() {
			s.n.Add(1)
		}
	})
}

// ---------------------------------------------------------------------------
// ③ 读多写少：RWMutex+map vs sync.Map
// ---------------------------------------------------------------------------

const nKeys = 1024

type rwMap struct {
	mu sync.RWMutex
	m  map[int]int
}

func newRWMap() *rwMap {
	m := &rwMap{m: make(map[int]int, nKeys)}
	for i := range nKeys {
		m.m[i] = i
	}
	return m
}

func (m *rwMap) Load(k int) (int, bool) {
	m.mu.RLock()
	v, ok := m.m[k]
	m.mu.RUnlock()
	return v, ok
}

func (m *rwMap) Store(k, v int) {
	m.mu.Lock()
	m.m[k] = v
	m.mu.Unlock()
}

func newSyncMap() *sync.Map {
	var m sync.Map
	for i := range nKeys {
		m.Store(i, i)
	}
	return &m
}

// 纯读
func BenchmarkRWMapReadOnly(b *testing.B) {
	m := newRWMap()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Load(i % nKeys)
			i++
		}
	})
}

func BenchmarkSyncMapReadOnly(b *testing.B) {
	m := newSyncMap()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Load(i % nKeys)
			i++
		}
	})
}

// 读多写少（10% 写）
func BenchmarkRWMapMostlyRead(b *testing.B) {
	m := newRWMap()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				m.Store(i%nKeys, i)
			} else {
				m.Load(i % nKeys)
			}
			i++
		}
	})
}

func BenchmarkSyncMapMostlyRead(b *testing.B) {
	m := newSyncMap()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				m.Store(i%nKeys, i)
			} else {
				m.Load(i % nKeys)
			}
			i++
		}
	})
}

// 写为主（50% 写）：sync.Map 的劣势区间
func BenchmarkRWMapHeavyWrite(b *testing.B) {
	m := newRWMap()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				m.Store(i%nKeys, i)
			} else {
				m.Load(i % nKeys)
			}
			i++
		}
	})
}

func BenchmarkSyncMapHeavyWrite(b *testing.B) {
	m := newSyncMap()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				m.Store(i%nKeys, i)
			} else {
				m.Load(i % nKeys)
			}
			i++
		}
	})
}

// ---------------------------------------------------------------------------
// ④ Once vs 每次都判一个 atomic.Bool
// ---------------------------------------------------------------------------

var (
	initOnce sync.Once
	inited   atomic.Bool
	valSink  int
)

func BenchmarkOnceDo(b *testing.B) {
	initOnce.Do(func() { valSink = 1 })
	for b.Loop() {
		initOnce.Do(func() { valSink = 1 })
	}
}

func BenchmarkAtomicBoolCheck(b *testing.B) {
	inited.Store(true)
	for b.Loop() {
		if !inited.Load() {
			valSink = 1
		}
	}
}

var onceValueFn = sync.OnceValue(func() int { return 42 })

func BenchmarkOnceValue(b *testing.B) {
	for b.Loop() {
		valSink = onceValueFn()
	}
}
