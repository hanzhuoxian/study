package main

import "testing"

// go test -bench . -benchmem ./func
//
// 对应 notes/func.md 2.3（defer 的两种实现）与 2.4（逃逸分析）。

var sink int

func noDefer() {
	sink++
}

func withOpenCodedDefer() {
	defer func() { sink++ }() // 数量固定 -> open-coded，编译期展开
}

func withLoopDefer(n int) {
	for range n { // 数量运行时才知道 -> 退化为 _defer 链表
		defer func() { sink++ }()
	}
}

func BenchmarkNoDefer(b *testing.B) {
	for b.Loop() {
		noDefer()
	}
}

func BenchmarkOpenCodedDefer(b *testing.B) {
	for b.Loop() {
		withOpenCodedDefer()
	}
}

func BenchmarkLoopDefer(b *testing.B) {
	for b.Loop() {
		withLoopDefer(4)
	}
}

// 逃逸：闭包捕获的变量、被带出函数的地址必须堆分配。
// 加 //go:noinline 是为了阻止内联把分配优化掉，否则观察不到 allocs/op。
var ptrSink *int
var funcSink func() int

//go:noinline
func makeValue() int { x := 42; return x } // 不逃逸，栈分配

//go:noinline
func makePointer() *int { x := 42; return &x } // 逃逸，堆分配

//go:noinline
func makeClosure(start int) func() int { // 捕获变量逃逸到堆
	n := start
	return func() int { n++; return n }
}

func BenchmarkNoEscape(b *testing.B) {
	for b.Loop() {
		sink = makeValue()
	}
}

func BenchmarkEscape(b *testing.B) {
	for b.Loop() {
		ptrSink = makePointer()
	}
}

func BenchmarkClosureAlloc(b *testing.B) {
	for b.Loop() {
		funcSink = makeClosure(0)
	}
}
