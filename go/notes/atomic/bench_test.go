package main

import (
	"sync"
	"sync/atomic"
	"testing"
)

// go test -bench . -benchmem ./atomic
// 看竞争随核数的变化：go test -bench 'Contended' -cpu 1,2,4,8 ./atomic

var (
	plainInt int64
	atomInt  atomic.Int64
	mu       sync.Mutex
	sink     int64
)

// ---------------------------------------------------------------------------
// ① 单线程：atomic 相对普通变量的固定开销
// ---------------------------------------------------------------------------

func BenchmarkPlainAdd(b *testing.B) {
	for b.Loop() {
		plainInt++
	}
}

func BenchmarkAtomicAdd(b *testing.B) {
	for b.Loop() {
		atomInt.Add(1)
	}
}

func BenchmarkAtomicLoad(b *testing.B) {
	for b.Loop() {
		sink = atomInt.Load()
	}
}

func BenchmarkMutexAdd(b *testing.B) {
	for b.Loop() {
		mu.Lock()
		plainInt++
		mu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// ② 多线程竞争：atomic vs mutex vs 分片
// ---------------------------------------------------------------------------

func BenchmarkAtomicContended(b *testing.B) {
	var n atomic.Int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n.Add(1)
		}
	})
}

func BenchmarkMutexContended(b *testing.B) {
	var n int64
	var mu sync.Mutex
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			n++
			mu.Unlock()
		}
	})
}

func BenchmarkCASLoopContended(b *testing.B) {
	var n atomic.Int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for { // 手写 CAS 循环：高竞争下重试次数暴涨
				old := n.Load()
				if n.CompareAndSwap(old, old+1) {
					break
				}
			}
		}
	})
}

// 读多写少：atomic.Load 完全不受写者影响
func BenchmarkAtomicLoadContended(b *testing.B) {
	var n atomic.Int64
	n.Store(1)
	b.RunParallel(func(pb *testing.PB) {
		var local int64
		for pb.Next() {
			local += n.Load()
		}
		sink = local
	})
}

func BenchmarkRWMutexReadContended(b *testing.B) {
	var n int64 = 1
	var rw sync.RWMutex
	b.RunParallel(func(pb *testing.PB) {
		var local int64
		for pb.Next() {
			rw.RLock()
			local += n
			rw.RUnlock()
		}
		sink = local
	})
}

// ---------------------------------------------------------------------------
// ③ 配置热更新：atomic.Pointer vs atomic.Value vs RWMutex
// ---------------------------------------------------------------------------

type conf struct {
	a, b, c int64
}

var (
	confPtr   atomic.Pointer[conf]
	confValue atomic.Value
	confMu    sync.RWMutex
	confPlain = &conf{1, 2, 3}
)

func init() {
	confPtr.Store(&conf{1, 2, 3})
	confValue.Store(&conf{1, 2, 3})
}

func BenchmarkConfigAtomicPointer(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := confPtr.Load()
			sink = c.a + c.b + c.c
		}
	})
}

func BenchmarkConfigAtomicValue(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := confValue.Load().(*conf)
			sink = c.a + c.b + c.c
		}
	})
}

func BenchmarkConfigRWMutex(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			confMu.RLock()
			c := confPlain
			sink = c.a + c.b + c.c
			confMu.RUnlock()
		}
	})
}
