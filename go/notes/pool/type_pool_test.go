package main

import (
	"sync"
	"testing"
)

// go test -bench . -benchmem ./pool
//
// 对应 notes/pool.md 1.4（泛型封装）、4.1（Put 值类型会额外分配）、4.7（小对象池化反而更慢）。
// TypedPool 的定义在 main.go 里。

// ---------------------------------------------------------------------------
// 1.4 泛型封装：只是消掉调用方的断言样板，性能和裸 sync.Pool 一致
// ---------------------------------------------------------------------------

var (
	rawBytePool = sync.Pool{New: func() any { s := make([]byte, 0, 64); return &s }}
	typedPool   = NewTypedPool(func() *[]byte { s := make([]byte, 0, 64); return &s })
)

func BenchmarkRawPool(b *testing.B) {
	for b.Loop() {
		buf := rawBytePool.Get().(*[]byte)
		*buf = append((*buf)[:0], 'a')
		rawBytePool.Put(buf)
	}
}

func BenchmarkTypedPool(b *testing.B) {
	for b.Loop() {
		buf := typedPool.Get()
		*buf = append((*buf)[:0], 'a')
		typedPool.Put(buf)
	}
}

// ---------------------------------------------------------------------------
// 4.1 Put 值类型 vs Put 指针
// ---------------------------------------------------------------------------

var (
	valuePool = sync.Pool{New: func() any { return make([]byte, 0, 1024) }}
	ptrPool   = sync.Pool{New: func() any { s := make([]byte, 0, 1024); return &s }}
)

func BenchmarkPutSliceValue(b *testing.B) {
	for b.Loop() {
		s := valuePool.Get().([]byte)
		valuePool.Put(s[:0]) //nolint:staticcheck // SA6002：故意演示装箱分配
	}
}

func BenchmarkPutSlicePtr(b *testing.B) {
	for b.Loop() {
		s := ptrPool.Get().(*[]byte)
		*s = (*s)[:0]
		ptrPool.Put(s)
	}
}

// ---------------------------------------------------------------------------
// 4.7 小对象池化划不划算：取决于它本来会不会逃逸到堆上
//
// 实测（i5-1038NG7）：
//	BenchmarkSmallStack-8    1.09 ns/op    0 B/op   0 allocs/op   栈分配
//	BenchmarkSmallNew-8     20.6  ns/op   16 B/op   1 allocs/op   逃逸 -> 堆分配
//	BenchmarkSmallPool-8    12.8  ns/op    0 B/op   0 allocs/op   Get/Put 固定开销 ~13ns
//
// 结论比 pool.md 的说法更精确：Get/Put 的固定成本（procPin + 原子操作 + 装箱断言）
// 在 10~15ns 量级，只有当对象**本来就会逃逸到堆**、构造成本高于这个数时才划算。
// 对象本来能栈分配的话，池化是纯亏——还顺带把栈访问变成了堆访问、丢了逃逸分析。
// ---------------------------------------------------------------------------

type small struct {
	a, b int64
} // 16 字节

var (
	smallPool = sync.Pool{New: func() any { return new(small) }}
	smallSink *small
	intSink   int64
)

// 不逃逸：编译器直接栈分配（go test -gcflags=-m 看不到 "escapes to heap"）
func BenchmarkSmallStack(b *testing.B) {
	for b.Loop() {
		s := small{a: 1}
		intSink = s.a
	}
}

// 逃逸到堆：每次一次 16 字节分配
func BenchmarkSmallNew(b *testing.B) {
	for b.Loop() {
		s := new(small)
		s.a = 1
		smallSink = s
	}
}

func BenchmarkSmallPool(b *testing.B) {
	for b.Loop() {
		s := smallPool.Get().(*small)
		s.a, s.b = 1, 0
		smallSink = s
		smallPool.Put(s)
	}
}
