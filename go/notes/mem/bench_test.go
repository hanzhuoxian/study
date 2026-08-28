package main

import (
	"strconv"
	"testing"
)

// go test -bench . -benchmem ./mem
//
// 对应 notes/mem.md 二（逃逸分析）、四（分配优化）。

type small struct{ a, b int }

var (
	ptrSink   *small
	intSink   int
	bytesSink []byte
	strSink   string
	anySink   any
)

// ---------------------------------------------------------------------------
// ① 栈 vs 堆：同一个 struct，差别只在有没有逃逸
// ---------------------------------------------------------------------------

//go:noinline
func newOnStack() int {
	s := small{1, 2}
	return s.a + s.b
}

//go:noinline
func newOnHeap() *small {
	return &small{1, 2}
}

func BenchmarkStackAlloc(b *testing.B) {
	for b.Loop() {
		intSink = newOnStack()
	}
}

func BenchmarkHeapAlloc(b *testing.B) {
	for b.Loop() {
		ptrSink = newOnHeap()
	}
}

// ---------------------------------------------------------------------------
// ② 预分配容量的价值
// ---------------------------------------------------------------------------

const n = 1000

func BenchmarkSliceNoPrealloc(b *testing.B) {
	for b.Loop() {
		s := []int{}
		for i := range n {
			s = append(s, i)
		}
		intSink = len(s)
	}
}

func BenchmarkSlicePrealloc(b *testing.B) {
	for b.Loop() {
		s := make([]int, 0, n)
		for i := range n {
			s = append(s, i)
		}
		intSink = len(s)
	}
}

func BenchmarkMapNoPrealloc(b *testing.B) {
	for b.Loop() {
		m := map[int]int{}
		for i := range n {
			m[i] = i
		}
		intSink = len(m)
	}
}

func BenchmarkMapPrealloc(b *testing.B) {
	for b.Loop() {
		m := make(map[int]int, n)
		for i := range n {
			m[i] = i
		}
		intSink = len(m)
	}
}

// ---------------------------------------------------------------------------
// ③ size class 取整：89 字节和 96 字节一样贵
// ---------------------------------------------------------------------------

func BenchmarkAlloc89(b *testing.B) {
	for b.Loop() {
		bytesSink = make([]byte, 89)
	}
}

func BenchmarkAlloc96(b *testing.B) {
	for b.Loop() {
		bytesSink = make([]byte, 96)
	}
}

func BenchmarkAlloc97(b *testing.B) { // 跨到下一档 112
	for b.Loop() {
		bytesSink = make([]byte, 97)
	}
}

// ---------------------------------------------------------------------------
// ④ 大对象：跨过 32KB 之后走 mheap，要加全局锁
// ---------------------------------------------------------------------------

func BenchmarkAlloc32KB(b *testing.B) {
	for b.Loop() {
		bytesSink = make([]byte, 32<<10)
	}
}

func BenchmarkAlloc33KB(b *testing.B) {
	for b.Loop() {
		bytesSink = make([]byte, 33<<10)
	}
}

func BenchmarkAlloc32KBParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bytesSink = make([]byte, 32<<10)
		}
	})
}

func BenchmarkAlloc33KBParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bytesSink = make([]byte, 33<<10)
		}
	})
}

// ---------------------------------------------------------------------------
// ⑤ 接口装箱的代价
// ---------------------------------------------------------------------------

//go:noinline
func takeAny(v any) { anySink = v }

//go:noinline
func takeInt(v int) { intSink = v }

func BenchmarkPassInt(b *testing.B) {
	for b.Loop() {
		takeInt(1 << 20)
	}
}

// 注意：常量装箱会被编译器优化成指向静态只读符号，看不出分配，所以这里用变量
var (
	dynBig   = 1 << 20
	dynSmall = 42
)

func BenchmarkPassIntAsAny(b *testing.B) {
	for b.Loop() {
		takeAny(dynBig) // 变量装箱 -> convT64 -> 真的分配 8 字节
	}
}

func BenchmarkPassSmallIntAsAny(b *testing.B) {
	for b.Loop() {
		takeAny(dynSmall) // 0-255 命中 runtime.staticuint64s，0 alloc
	}
}

func BenchmarkPassConstAsAny(b *testing.B) {
	for b.Loop() {
		takeAny(1 << 20) // 常量：编译期就放进只读段，0 alloc
	}
}

// ---------------------------------------------------------------------------
// ⑥ 字符串拼接：四种写法
// ---------------------------------------------------------------------------

func BenchmarkConcatPlus(b *testing.B) {
	for b.Loop() {
		s := ""
		for i := range 100 {
			s += strconv.Itoa(i)
		}
		strSink = s
	}
}

func BenchmarkConcatBuilder(b *testing.B) {
	for b.Loop() {
		var sb stringBuilder
		for i := range 100 {
			sb.WriteString(strconv.Itoa(i))
		}
		strSink = sb.String()
	}
}

func BenchmarkConcatBuilderGrow(b *testing.B) {
	for b.Loop() {
		var sb stringBuilder
		sb.Grow(300)
		for i := range 100 {
			sb.WriteString(strconv.Itoa(i))
		}
		strSink = sb.String()
	}
}

func BenchmarkConcatBytes(b *testing.B) {
	for b.Loop() {
		buf := make([]byte, 0, 300)
		for i := range 100 {
			buf = strconv.AppendInt(buf, int64(i), 10)
		}
		strSink = string(buf)
	}
}
