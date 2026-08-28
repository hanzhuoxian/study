package main

import (
	"bytes"
	"testing"
)

// go test -bench . -benchmem ./pool
//
// 对应 notes/pool.md 1.5（收益有多大）。
// 并行版本更能体现收益（省掉的 GC 扫描开销随并发放大）：
//	go test -bench 'Parallel' -benchmem -benchtime=300000x ./pool

func BenchmarkWorkNoPool(b *testing.B) {
	for b.Loop() {
		workNoPool()
	}
}

func BenchmarkWorkWithPool(b *testing.B) {
	for b.Loop() {
		workWithPool()
	}
}

func BenchmarkWorkNoPoolParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			workNoPool()
		}
	})
}

func BenchmarkWorkWithPoolParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			workWithPool()
		}
	})
}

// ---------------------------------------------------------------------------
// 4.4 大对象回池：设上限的版本不会让 1MB buffer 常驻
// ---------------------------------------------------------------------------

func BenchmarkPutCapped(b *testing.B) {
	buf := new(bytes.Buffer)
	buf.Grow(1 << 20)
	b.ResetTimer()
	for b.Loop() {
		putCapped(buf) // 每次都因为超上限被丢弃，纯粹量一下判断开销
	}
}
