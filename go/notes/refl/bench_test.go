package main

import (
	"reflect"
	"testing"
)

// go test -bench . -benchmem ./refl

var (
	u        = User{Name: "bob", Age: 30}
	pu       = &u
	strSink  string
	intSink  int
	anySink  any
	boolSink bool
	typeSink reflect.Type
)

// ---------------------------------------------------------------------------
// ① 取字段：三种方式
// ---------------------------------------------------------------------------

func BenchmarkFieldDirect(b *testing.B) {
	for b.Loop() {
		strSink = u.Name
	}
}

func BenchmarkFieldReflectIndex(b *testing.B) {
	v := reflect.ValueOf(u) // ValueOf 放在循环外，只测取字段
	for b.Loop() {
		strSink = v.Field(0).String()
	}
}

func BenchmarkFieldReflectByName(b *testing.B) {
	v := reflect.ValueOf(u)
	for b.Loop() {
		strSink = v.FieldByName("Name").String()
	}
}

// 包含 ValueOf 的完整开销（真实使用形态）
func BenchmarkFieldReflectFull(b *testing.B) {
	for b.Loop() {
		strSink = reflect.ValueOf(u).Field(0).String()
	}
}

// ---------------------------------------------------------------------------
// ② 设字段
// ---------------------------------------------------------------------------

func BenchmarkSetDirect(b *testing.B) {
	for b.Loop() {
		u.Age = 1
	}
}

func BenchmarkSetReflect(b *testing.B) {
	v := reflect.ValueOf(pu).Elem()
	f := v.Field(1)
	for b.Loop() {
		f.SetInt(1)
	}
}

// ---------------------------------------------------------------------------
// ③ 调方法
// ---------------------------------------------------------------------------

func BenchmarkCallDirect(b *testing.B) {
	for b.Loop() {
		strSink = u.Greet()
	}
}

func BenchmarkCallReflect(b *testing.B) {
	m := reflect.ValueOf(u).MethodByName("Greet")
	for b.Loop() {
		strSink = m.Call(nil)[0].String()
	}
}

// MakeFunc 包装后的调用
var wrapped = func() func(int, int) int {
	f := func(a, b int) int { return a + b }
	fv := reflect.ValueOf(f)
	return reflect.MakeFunc(fv.Type(), func(in []reflect.Value) []reflect.Value {
		return fv.Call(in)
	}).Interface().(func(int, int) int)
}()

func BenchmarkCallMakeFunc(b *testing.B) {
	for b.Loop() {
		intSink = wrapped(1, 2)
	}
}

func BenchmarkCallPlainFunc(b *testing.B) {
	f := func(a, c int) int { return a + c }
	for b.Loop() {
		intSink = f(1, 2)
	}
}

// ---------------------------------------------------------------------------
// ④ TypeOf / TypeFor / 装箱
// ---------------------------------------------------------------------------

func BenchmarkTypeOf(b *testing.B) {
	for b.Loop() {
		typeSink = reflect.TypeOf(u)
	}
}

func BenchmarkTypeFor(b *testing.B) {
	for b.Loop() {
		typeSink = reflect.TypeFor[User]()
	}
}

func BenchmarkValueInterface(b *testing.B) {
	v := reflect.ValueOf(u)
	for b.Loop() {
		anySink = v.Interface()
	}
}

func BenchmarkTypeAssertGeneric(b *testing.B) {
	v := reflect.ValueOf(u)
	for b.Loop() {
		_, boolSink = reflect.TypeAssert[User](v)
	}
}

// ---------------------------------------------------------------------------
// ⑤ DeepEqual vs 直接比较
// ---------------------------------------------------------------------------

var (
	a1 = User{Name: "bob", Age: 30}
	a2 = User{Name: "bob", Age: 30}
	s1 = []int{1, 2, 3, 4, 5}
	s2 = []int{1, 2, 3, 4, 5}
)

func BenchmarkStructEqualDirect(b *testing.B) {
	for b.Loop() {
		boolSink = a1 == a2
	}
}

func BenchmarkStructDeepEqual(b *testing.B) {
	for b.Loop() {
		boolSink = reflect.DeepEqual(a1, a2)
	}
}

func BenchmarkSliceDeepEqual(b *testing.B) {
	for b.Loop() {
		boolSink = reflect.DeepEqual(s1, s2)
	}
}

// ---------------------------------------------------------------------------
// ⑥ 典型优化：缓存字段索引路径
// ---------------------------------------------------------------------------

var fieldIndexCache = func() map[string][]int {
	m := map[string][]int{}
	t := reflect.TypeFor[User]()
	for i := range t.NumField() {
		m[t.Field(i).Name] = []int{i}
	}
	return m
}()

func BenchmarkFieldByNameCached(b *testing.B) {
	v := reflect.ValueOf(u)
	for b.Loop() {
		strSink = v.FieldByIndex(fieldIndexCache["Name"]).String()
	}
}

// TypeAssert 的公平对照：完整的 Interface().(T)
func BenchmarkInterfaceThenAssert(b *testing.B) {
	v := reflect.ValueOf(u)
	for b.Loop() {
		_, boolSink = v.Interface().(User)
	}
}
