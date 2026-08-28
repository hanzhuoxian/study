package main

import (
	"iter"
	"slices"
	"strings"
	"testing"
)

// go test -bench . -benchmem ./iter
//
// 对应 notes/iter.md 2.7（内联与逃逸）、2.8（三种遍历方式实测）、3.9（不要为了好看套迭代器）。

var (
	data    = makeData(1000)
	intSink int
	strSink []string
)

func makeData(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

// ① 直接 range slice：最快，无闭包调用
func BenchmarkRangeSlice(b *testing.B) {
	for b.Loop() {
		sum := 0
		for _, v := range data {
			sum += v
		}
		intSink = sum
	}
}

// ② range over func（slices.Values）：多一层闭包调用
func BenchmarkRangeSeq(b *testing.B) {
	for b.Loop() {
		sum := 0
		for v := range slices.Values(data) {
			sum += v
		}
		intSink = sum
	}
}

// ③ iter.Pull：背后是一个 coro，每次 next 都要切换协程栈，最慢
func BenchmarkPull(b *testing.B) {
	for b.Loop() {
		next, stop := iter.Pull(slices.Values(data))
		sum := 0
		for {
			v, ok := next()
			if !ok {
				break
			}
			sum += v
		}
		stop()
		intSink = sum
	}
}

// 组合子链路的开销：每层都是一次闭包调用
func BenchmarkMapFilterChain(b *testing.B) {
	seq := Map(Filter(slices.Values(data), func(v int) bool { return v%2 == 0 }),
		func(v int) int { return v * 2 })
	for b.Loop() {
		sum := 0
		for v := range seq {
			sum += v
		}
		intSink = sum
	}
}

// strings.Split 一次性分配整个 []string，SplitSeq 惰性产出、零分配
var line = strings.Repeat("field,", 100)

func BenchmarkStringsSplit(b *testing.B) {
	for b.Loop() {
		strSink = strings.Split(line, ",")
	}
}

func BenchmarkStringsSplitSeq(b *testing.B) {
	for b.Loop() {
		n := 0
		for range strings.SplitSeq(line, ",") {
			n++
		}
		intSink = n
	}
}
