package main

import "testing"

// go test -bench . -benchmem ./generic
//
// 对应 notes/generic.md 2.4（字典间接调用）、2.6（性能实测）、3.11（泛型不是性能银弹）。
//
// 看内联决策（泛型版函数体一大就彻底失去优化机会）：
//	go test -gcflags=-m -bench=xxx ./generic 2>&1 | grep -E 'inline.*sum'

// ---------------------------------------------------------------------------
// 场景一：类型参数上的方法调用 —— 泛型 ≈ 接口，都比具体类型慢
// ---------------------------------------------------------------------------

type Cnt int

func (c Cnt) Val() int { return int(c) }

type Valuer interface{ Val() int }

const n = 1000

var (
	concreteData = makeConcrete(n)
	ifaceData    = makeIface(n)
	intSink      int
)

func makeConcrete(n int) []Cnt {
	s := make([]Cnt, n)
	for i := range s {
		s[i] = Cnt(i)
	}
	return s
}

func makeIface(n int) []Valuer {
	s := make([]Valuer, n)
	for i := range s {
		s[i] = Cnt(i)
	}
	return s
}

// 具体类型：Val() 可以被内联，最快
func sumConcrete(xs []Cnt) int {
	total := 0
	for _, v := range xs {
		total += v.Val()
	}
	return total
}

// 泛型：v.Val() 编译成"从字典槽取函数地址 + CALL CX"，无法内联、无法去虚化
func sumGeneric[T Valuer](xs []T) int {
	total := 0
	for _, v := range xs {
		total += v.Val()
	}
	return total
}

// 接口：itab 动态派发，和泛型同一个量级
func sumIface(xs []Valuer) int {
	total := 0
	for _, v := range xs {
		total += v.Val()
	}
	return total
}

func BenchmarkConcrete(b *testing.B) {
	for b.Loop() {
		intSink = sumConcrete(concreteData)
	}
}

func BenchmarkGeneric(b *testing.B) {
	for b.Loop() {
		intSink = sumGeneric(concreteData)
	}
}

func BenchmarkIface(b *testing.B) {
	for b.Loop() {
		intSink = sumIface(ifaceData)
	}
}

// ---------------------------------------------------------------------------
// 场景二：避免 any 装箱 —— 这才是泛型真正的性能收益来源
// ---------------------------------------------------------------------------

// 128 字节，装进接口必然堆分配（interface.md 2.3）
type Big struct{ a [16]int64 }

// 泛型 sink：把值原样存进去，不经过接口
type box[T any] struct{ v T }

var (
	big     = Big{}
	anySink any
	bigBox  box[Big]
)

//go:noinline
func takeAny(v any) { anySink = v }

//go:noinline
func takeGeneric[T any](v T, dst *box[T]) { dst.v = v }

func BenchmarkBoxAny(b *testing.B) {
	for b.Loop() {
		takeAny(big) // 128 B/op、1 allocs/op
	}
}

func BenchmarkNoBoxGeneric(b *testing.B) {
	for b.Loop() {
		takeGeneric(big, &bigBox) // 值以原本形状传递，0 allocs/op
	}
}
