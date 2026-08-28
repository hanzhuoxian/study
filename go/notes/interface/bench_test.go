package main

import (
	"strconv"
	"testing"
)

// go test -bench . -benchmem ./interface
//
// 对应 notes/interface.md 2.3（装箱开销）、2.6（断言开销）、3.4/3.5（用泛型替代空接口）。

var (
	anySink  any
	intSink  int
	boolSink bool
)

//go:noinline
func boxValue(v big) any { return v } // 大 struct 值装箱：要堆分配一份拷贝

//go:noinline
func boxPointer(v *big) any { return v } // 指针装箱：Data 直接存指针

//go:noinline
func boxSmallInt(v int) any { return v } // 0~255 复用 runtime.staticuint64s

//go:noinline
func boxLargeInt(v int) any { return v }

func BenchmarkBoxBigValue(b *testing.B) {
	v := big{}
	for b.Loop() {
		anySink = boxValue(v)
	}
}

func BenchmarkBoxBigPointer(b *testing.B) {
	v := &big{}
	for b.Loop() {
		anySink = boxPointer(v)
	}
}

func BenchmarkBoxSmallInt(b *testing.B) {
	for b.Loop() {
		anySink = boxSmallInt(42) // 落在 [0,256) 缓存区间
	}
}

func BenchmarkBoxLargeInt(b *testing.B) {
	for b.Loop() {
		anySink = boxLargeInt(1 << 20)
	}
}

// 断言开销：具体类型是一次指针比较，接口类型要查 itab 表
func BenchmarkAssertConcrete(b *testing.B) {
	var x any = 42
	for b.Loop() {
		intSink, _ = x.(int)
	}
}

func BenchmarkAssertInterface(b *testing.B) {
	var x any = strconv.NumError{}
	for b.Loop() {
		_, boolSink = x.(error)
	}
}

func BenchmarkTypeSwitch(b *testing.B) {
	inputs := []any{1, "s", 3.14, true}
	for b.Loop() {
		for _, in := range inputs {
			switch in.(type) {
			case int:
				intSink++
			case string:
				intSink += 2
			case float64:
				intSink += 3
			default:
				intSink += 4
			}
		}
	}
}

// 空接口 vs 泛型：泛型在编译期单态化，没有装箱
//
//go:noinline
func maxAny(a, b any) any {
	if a.(int) > b.(int) {
		return a
	}
	return b
}

//go:noinline
func maxGeneric[T int | float64 | string](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func BenchmarkMaxAny(b *testing.B) {
	for b.Loop() {
		anySink = maxAny(1<<20, 1<<21)
	}
}

func BenchmarkMaxGeneric(b *testing.B) {
	for b.Loop() {
		intSink = maxGeneric(1<<20, 1<<21)
	}
}
