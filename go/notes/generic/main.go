// generic 示例：对应 notes/generic.md
// 运行：go run ./generic
// 看 shape / 字典符号：
//
//	go build -o /tmp/generic ./generic && go tool nm /tmp/generic | grep -E 'go\.shape|\.\.dict' | sort
//
// 看内联决策与字典传参：
//
//	go build -gcflags='-m -S' ./generic 2>&1 | grep -E 'inline|dict'
package main

import (
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	basicInstantiate()
	basicTypeSet()
	basicTilde()
	basicAnyComparable()
	basicGenericType()
	basicInference()
	basicConstraintInference()
	basicStdlib()
	basicVsInterface()

	shapeSharing()
	dictAndReflect()
	opLegality()

	trapTypeSetAsType()
	trapMissingTilde()
	trapMethodTypeParam()
	trapTypeSwitch()
	trapNilAndZero()
	trapComparablePanic()
	trapLoseNamedType()
	trapInferFromReturn()
	trapUninstantiatedValue()
	trapFakeOverload()
}

// ---------------------------------------------------------------------------
// 1.1 类型参数与实例化
// ---------------------------------------------------------------------------

// T 是类型参数，int | float64 | string 是它的约束
func Max[T int | float64 | string](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func basicInstantiate() {
	section("1.1 类型参数与实例化")

	fmt.Println("显式实例化 Max[int](1, 2) =", Max[int](1, 2))
	fmt.Println("类型推断   Max(1, 2)      =", Max(1, 2))
	fmt.Println("类型推断   Max(\"a\", \"b\")  =", Max("a", "b"))

	// 实例化之后就是一个普通函数值
	f := Max[float64]
	fmt.Printf("f := Max[float64] -> %T, f(1.5, 2.5) = %v\n", f, f(1.5, 2.5))

	// g := Max // ✗ cannot use generic function Max without instantiation（见 3.9）
}

// ---------------------------------------------------------------------------
// 1.2 约束就是接口：从"方法集"到"类型集"
// ---------------------------------------------------------------------------

// 类型项（type term）的并集（union）
type Number interface{ ~int | ~int64 | ~float64 }

// 类型项和方法同时出现 = 交集：底层类型是 string 且实现了 Stringer
type Stringish interface {
	~string
	fmt.Stringer
}

type Word string

func (w Word) String() string { return "<" + string(w) + ">" }

// 约束是双向的：既限制调用方，也决定函数体内能做什么
// Number 的类型集里所有类型都支持 +，所以 s += x 合法
func SumNum[T Number](xs []T) T {
	var s T
	for _, x := range xs {
		s += x
	}
	return s
}

// Stringish 里有方法项，所以可以调 x.String()
func JoinStringish[T Stringish](xs []T) string {
	parts := make([]string, 0, len(xs))
	for _, x := range xs {
		parts = append(parts, x.String())
	}
	return strings.Join(parts, ",")
}

func basicTypeSet() {
	section("1.2 约束就是接口（类型集）")

	fmt.Println("SumNum([]int)     =", SumNum([]int{1, 2, 3}))
	fmt.Println("SumNum([]float64) =", SumNum([]float64{1.5, 2.5}))
	fmt.Println("JoinStringish     =", JoinStringish([]Word{"a", "b"}))

	// 约束只有一个类型项时可以省略 interface{} 外壳
	fmt.Println("单类型项约束 head([]byte) =", string(head[[]byte]([]byte("hello"))))
}

func head[T ~[]byte](s T) T { return s[:1] }

// ---------------------------------------------------------------------------
// 1.3 ~T：底层类型近似
// ---------------------------------------------------------------------------

type MyInt int

func AddStrict[T int](a, b T) T { return a + b } // 只接受 int 本身
func AddLoose[T ~int](a, b T) T { return a + b } // 接受所有底层类型是 int 的类型

func basicTilde() {
	section("1.3 ~T 底层类型近似")

	fmt.Println("AddStrict(1, 2)             =", AddStrict(1, 2))
	fmt.Println("AddLoose(MyInt(1), MyInt(2)) =", AddLoose(MyInt(1), MyInt(2)))

	// AddStrict(MyInt(1), MyInt(2))
	// ✗ MyInt does not satisfy int (possibly missing ~ for int in int)

	// type C1 interface{ ~MyInt } // ✗ invalid use of ~ (underlying type of MyInt is int)
	// type C2 interface{ ~error } // ✗ invalid use of ~ (error is an interface)
	fmt.Println("结论：写约束默认加 ~，除非确实要排除具名类型")
}

// ---------------------------------------------------------------------------
// 1.4 预声明约束：any 与 comparable
// ---------------------------------------------------------------------------

func Index[T comparable](s []T, v T) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

type Set[K comparable] map[K]struct{}

// 有序比较用标准库的 cmp.Ordered，不要自己造
func MinOf[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func basicAnyComparable() {
	section("1.4 any 与 comparable")

	fmt.Println("Index([]string{a,b,c}, b) =", Index([]string{"a", "b", "c"}, "b"))
	fmt.Println("MinOf(3, 1)               =", MinOf(3, 1))

	// Go 1.20+：any 可比较，"满足"但并不"实现" comparable
	s := Set[any]{}
	s[1] = struct{}{}
	s["k"] = struct{}{}
	fmt.Println("Set[any] 合法（1.20+），len =", len(s))
	fmt.Println("但 s[[]int{1}] = ... 会 panic：见 3.6")
}

// ---------------------------------------------------------------------------
// 1.5 泛型类型及其方法
// ---------------------------------------------------------------------------

type Stack[T any] struct {
	items []T
}

// 接收者上要重复写出类型参数名，这里的 T 是"声明"而不是"使用"
func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
	var zero T // 类型参数的零值只能这么拿
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}

func (s *Stack[T]) Len() int { return len(s.items) }

type Pair[K comparable, V any] struct {
	Key K
	Val V
}

// 泛型类型别名：Go 1.24+
type StrPair[V any] = Pair[string, V]

func basicGenericType() {
	section("1.5 泛型类型及其方法")

	s := &Stack[int]{} // 泛型类型必须实例化后才能使用
	s.Push(1)
	s.Push(2)
	v, ok := s.Pop()
	fmt.Printf("Stack[int]: pop=%v ok=%v len=%d\n", v, ok, s.Len())

	empty := &Stack[string]{}
	v2, ok2 := empty.Pop()
	fmt.Printf("空栈 pop: %q %v（零值靠 var zero T）\n", v2, ok2)

	fmt.Println("泛型类型别名 StrPair[int] =", StrPair[int]{"a", 1})

	// 方法不能引入自己的类型参数，见 3.3
}

// ---------------------------------------------------------------------------
// 1.6 类型推断
// ---------------------------------------------------------------------------

func Zero[T any]() T { var z T; return z }

func basicInference() {
	section("1.6 类型推断")

	fmt.Println("Max(1, 2)   -> T=int    =", Max(1, 2))
	fmt.Println("Max(1, 2.5) -> T=float64 =", Max(1, 2.5)) // 两个无类型常量，取默认类型

	var i int = 3
	_ = i
	// Max(i, 2.5) // ✗ cannot use 2.5 (untyped float constant) as int value（i 已定型）

	// var x int = Zero() // ✗ in call to Zero, cannot infer T
	var x int = Zero[int]()
	fmt.Println("不能从赋值目标反推，只能显式实例化 Zero[int]() =", x)
}

// ---------------------------------------------------------------------------
// 1.7 约束类型推断：S ~[]E 惯用法
// ---------------------------------------------------------------------------

type Nums []int

func (n Nums) String() string { return fmt.Sprint([]int(n)) }

// 坏写法：返回值退化成 []E
func ScaleBad[E ~int](s []E, c E) []E {
	out := make([]E, len(s))
	for i, v := range s {
		out[i] = v * c
	}
	return out
}

// 好写法：S 被推断为调用方的具名类型，原样返回
func ScaleGood[S ~[]E, E ~int](s S, c E) S {
	out := make(S, len(s))
	for i, v := range s {
		out[i] = v * c
	}
	return out
}

func basicConstraintInference() {
	section("1.7 约束类型推断 S ~[]E")

	n := Nums{1, 2, 3}

	good := ScaleGood(n, 2)
	bad := ScaleBad(n, 2)
	fmt.Printf("ScaleGood -> %T = %v\n", good, good)
	fmt.Printf("ScaleBad  -> %T = %v\n", bad, bad)

	var a fmt.Stringer = ScaleGood(n, 2) // OK：还是 Nums，方法没丢
	fmt.Println("ScaleGood 结果仍实现 Stringer:", a.String())

	// var b fmt.Stringer = ScaleBad(n, 2)
	// ✗ []int does not implement fmt.Stringer (missing method String)

	fmt.Println("推断过程：实参 n -> S=Nums；再由约束 S ~[]E 反推 E=int")
}

// ---------------------------------------------------------------------------
// 1.8 标准库里的泛型
// ---------------------------------------------------------------------------

var loadConfig = sync.OnceValue(func() string {
	fmt.Println("  （OnceValue：只会执行一次）")
	return "config-loaded"
})

func basicStdlib() {
	section("1.8 标准库里的泛型")

	xs := []int{3, 1, 2}
	slices.Sort(xs)
	fmt.Println("slices.Sort            =", xs)
	fmt.Println("slices.Contains        =", slices.Contains(xs, 2))
	i, found := slices.BinarySearch(xs, 2)
	fmt.Println("slices.BinarySearch    =", i, found)
	fmt.Println("cmp.Or(\"\", \"fallback\") =", cmp.Or("", "fallback"))

	m := map[string]int{"b": 2, "a": 1}
	// maps.Keys 返回的是迭代器 iter.Seq[K]，要切片得 slices.Collect
	keys := slices.Sorted(maps.Keys(m))
	fmt.Println("slices.Sorted(maps.Keys) =", keys)

	fmt.Println("sync.OnceValue:", loadConfig(), loadConfig())

	// 类型安全的原子指针，替代 atomic.Value 的运行时类型检查
	var p atomic.Pointer[Pair[string, int]]
	p.Store(&Pair[string, int]{"a", 1})
	fmt.Println("atomic.Pointer[T]      =", *p.Load())
}

// ---------------------------------------------------------------------------
// 1.9 什么时候用泛型，什么时候用接口
// ---------------------------------------------------------------------------

// 用接口：表达行为契约，实现方运行时才确定
type Storage interface {
	Save(k string, v []byte) error
}

type memStorage map[string][]byte

func (m memStorage) Save(k string, v []byte) error { m[k] = v; return nil }

// 用泛型：同一段算法作用在多个编译期已知的具体类型上
func MapSlice[S ~[]E, E, R any](s S, f func(E) R) []R {
	out := make([]R, 0, len(s))
	for _, v := range s {
		out = append(out, f(v))
	}
	return out
}

func basicVsInterface() {
	section("1.9 泛型 vs 接口")

	var st Storage = memStorage{}
	_ = st.Save("k", []byte("v"))
	fmt.Println("接口：行为契约 / 依赖倒置 / 可 mock")

	fmt.Println("泛型：同一算法多类型 MapSlice =",
		MapSlice([]int{1, 2, 3}, func(v int) string { return fmt.Sprint(v * 2) }))

	fmt.Println("判据：类型参数只用来调方法 -> 一个接口就够了（多一层字典开销）")
}

// ---------------------------------------------------------------------------
// 2.2 shape 类型：哪些实例共享同一份代码
// ---------------------------------------------------------------------------

// 场景一：约束含类型项，按底层类型分组
//
//go:noinline
func Sum[T ~int | ~int64 | ~float64](xs []T) T {
	var s T
	for _, v := range xs {
		s += v
	}
	return s
}

type A struct{ n int }

func (a A) String() string { return fmt.Sprintf("A(%d)", a.n) }

type B struct{ s string }

func (b B) String() string { return "B(" + b.s + ")" }

// 场景二：约束是基本接口（只有方法），指针实参全部塌缩成 go.shape.*uint8
//
//go:noinline
func Show[T fmt.Stringer](v T) string {
	var s any = v // 这一句会让字典里多一个 R_USEIFACE type:interface {}
	_ = s
	return v.String()
}

// 场景三：约束 any，非指针实参各自成形
//
//go:noinline
func Id[T any](v T) T { return v }

func shapeSharing() {
	section("2.2 shape：哪些实例共享代码")

	fmt.Println("Sum[int]     =", Sum([]int{1, 2, 3}))
	fmt.Println("Sum[MyInt]   =", Sum([]MyInt{1, 2, 3})) // 和 Sum[int] 共用一份机器码
	fmt.Println("Sum[int64]   =", Sum([]int64{1, 2, 3})) // 底层类型不同，另一份
	fmt.Println("Sum[float64] =", Sum([]float64{1, 2, 3}))

	fmt.Println("Show[*A]     =", Show(&A{1})) // Show[*A] / Show[*B] 共用 go.shape.*uint8
	fmt.Println("Show[*B]     =", Show(&B{"x"}))
	fmt.Println("Show[A]      =", Show(A{2})) // 非指针：按底层结构成形

	fmt.Println("Id[*int]     =", *Id(new(int)))
	fmt.Println("Id[[]int]    =", Id([]int{1}))
	fmt.Println("Id[map]      =", Id(map[int]int{1: 1}))

	fmt.Println(`验证：go build -o /tmp/generic ./generic && go tool nm /tmp/generic | grep -E 'go\.shape|\.\.dict'`)
	fmt.Println("  机器码按 shape 一份，字典每个实例一份（..dict.Sum[int] / ..dict.Sum[main.MyInt]）")
}

// ---------------------------------------------------------------------------
// 2.3 字典里装了什么 —— reflect 看到的是真类型，不是 shape 类型
// ---------------------------------------------------------------------------

func TypeOf[T any](v T) string {
	return fmt.Sprintf("%v / %v", reflect.TypeOf(v), reflect.TypeFor[T]())
}

func dictAndReflect() {
	section("2.3 字典与 reflect")

	fmt.Println("TypeOf(int(1))   =", TypeOf(int(1)))
	fmt.Println("TypeOf(MyInt(1)) =", TypeOf(MyInt(1)))
	fmt.Println("两者共用一份代码，但 rtype 从各自字典里取，所以 reflect 看到真类型")
	fmt.Println("字典是 SRODATA dupok：只读、可去重、不参与 GC 扫描")
}

// ---------------------------------------------------------------------------
// 2.5 操作合法性：类型集里所有类型的底层类型必须相同
// ---------------------------------------------------------------------------

type IntSlice []int

// ✓ 底层类型相同的多个具名类型
func SumEither[S IntSlice | Nums](s S) int {
	total := 0
	for _, v := range s {
		total += v
	}
	return total
}

// ✓ 通道的方向例外：元素类型相同即可
func Recv[T ~chan int | ~<-chan int](c T) int { return <-c }

// len 对 ~[]byte | ~string 恰好合法（range 不行，见 3.14）
func Len[T ~[]byte | ~string](v T) int { return len(v) }

func opLegality() {
	section("2.5 操作合法性（1.25 起删掉了'核心类型'概念）")

	fmt.Println("SumEither(IntSlice) =", SumEither(IntSlice{1, 2}))
	fmt.Println("SumEither(Nums)     =", SumEither(Nums{3, 4}))

	ch := make(chan int, 1)
	ch <- 42
	fmt.Println("Recv(chan int)      =", Recv(ch))

	fmt.Println("Len([]byte)         =", Len([]byte("abc")))
	fmt.Println("Len(string)         =", Len("abcd"))

	// func Count[T ~[]byte | ~string](v T) int { for range v {}; return 0 }
	// ✗ cannot range over v: []byte and string have different underlying types
	// func Make[T ~[]int | ~map[string]int](n int) T { return make(T, n) }
	// ✗ cannot make T: []int and map[string]int have different underlying types
}

// ---------------------------------------------------------------------------
// 3.1 含类型项的接口当普通类型用
// ---------------------------------------------------------------------------

type Num interface{ ~int | ~float64 }

func trapTypeSetAsType() {
	section("3.1 含类型项的接口只能当约束")

	// var x Num          // ✗ cannot use type Num outside a type constraint
	// func f(n Num) {}   // ✗ 同上
	fmt.Println("Num 只能当约束：Double(2.5) =", Double(2.5))
	fmt.Println("原因：类型集里的类型没有共同方法表，运行时无法构造 itab")
}

func Double[T Num](v T) T { return v * 2 }

// ---------------------------------------------------------------------------
// 3.2 忘了 ~，具名类型不满足约束
// ---------------------------------------------------------------------------

type Celsius float64

func SumStrict[T int | float64](xs []T) T {
	var s T
	for _, v := range xs {
		s += v
	}
	return s
}
func SumTilde[T ~int | ~float64](xs []T) T {
	var s T
	for _, v := range xs {
		s += v
	}
	return s
}

func trapMissingTilde() {
	section("3.2 忘了 ~")

	// SumStrict([]Celsius{1, 2})
	// ✗ Celsius does not satisfy float64 (possibly missing ~ for float64 in float64)
	fmt.Println("SumStrict([]float64) =", SumStrict([]float64{1, 2}))
	fmt.Println("SumTilde([]Celsius)  =", SumTilde([]Celsius{1, 2}))
	fmt.Println("工程上：约束一律写 ~，或直接用 cmp.Ordered（内部就是 ~ 形式）")
}

// ---------------------------------------------------------------------------
// 3.3 方法不能有类型参数
// ---------------------------------------------------------------------------

type List[T any] struct{ items []T }

func (l *List[T]) Add(v T) { l.items = append(l.items, v) }

// func (l *List[T]) Map[U any](f func(T) U) []U { ... }
// ✗ syntax error: method must have no type parameters

// 正确写法：提为顶层函数（这也是标准库 slices.Map 风格的原因）
func ListMap[T, U any](l *List[T], f func(T) U) []U {
	out := make([]U, 0, len(l.items))
	for _, v := range l.items {
		out = append(out, f(v))
	}
	return out
}

func trapMethodTypeParam() {
	section("3.3 方法不能有类型参数")

	l := &List[int]{}
	l.Add(1)
	l.Add(2)
	fmt.Println("ListMap(l, itoa) =", ListMap(l, func(v int) string { return fmt.Sprint(v) }))
	fmt.Println("原因：参数化方法等价于无穷多签名，无法建 itab / 判定接口实现")
}

// ---------------------------------------------------------------------------
// 3.4 不能直接对类型参数做类型断言 / type switch
// ---------------------------------------------------------------------------

func Describe[T any](v T) string {
	// switch v.(type) { case int: } // ✗ cannot use type switch on type parameter value v
	switch x := any(v).(type) { // 先转成接口
	case int:
		return fmt.Sprintf("int %d", x)
	case string:
		return fmt.Sprintf("string %q", x)
	default:
		return fmt.Sprintf("other %T", x)
	}
}

func trapTypeSwitch() {
	section("3.4 类型参数不能直接 type switch")

	fmt.Println(Describe(1))
	fmt.Println(Describe("a"))
	fmt.Println(Describe(1.5))
	fmt.Println("警惕：写出 any(v).(type) 说明泛型往往是错的抽象")
}

// ---------------------------------------------------------------------------
// 3.5 v == nil 非法；零值判断需要 comparable
// ---------------------------------------------------------------------------

// func IsNil[T any](v T) bool  { return v == nil }        // ✗ mismatched types T and untyped nil
// func IsZeroBad[T any](v T) bool { var z T; return v == z } // ✗ incomparable types in type set

func IsZero[T comparable](v T) bool { var z T; return v == z } // 约束收紧

func IsZeroAny[T any](v T) bool { // 兜底方案
	return reflect.ValueOf(&v).Elem().IsZero()
}

func trapNilAndZero() {
	section("3.5 v == nil 非法")

	fmt.Println("IsZero(0)             =", IsZero(0))
	fmt.Println("IsZero(\"a\")           =", IsZero("a"))
	fmt.Println("IsZeroAny([]int(nil)) =", IsZeroAny([]int(nil))) // 切片不可比较，只能走反射
	fmt.Println("想表达'可能为 nil'就别用 T any，用 *T 或 T ~*E | ~[]E")
}

// ---------------------------------------------------------------------------
// 3.6 comparable 不保证运行时不 panic
// ---------------------------------------------------------------------------

func trapComparablePanic() {
	section("3.6 comparable 不保证运行时不 panic")

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recover:", r) // hash of unhashable type []int
		}
	}()

	s := Set[any]{}
	s["ok"] = struct{}{}
	fmt.Println("Set[any] 放可哈希的 key 没问题，len =", len(s))
	s[[]int{1}] = struct{}{} // panic
	fmt.Println("这行不会执行")
}

// ---------------------------------------------------------------------------
// 3.7 返回 []E 丢失具名类型
// ---------------------------------------------------------------------------

func FilterBad[E any](s []E, f func(E) bool) []E {
	out := make([]E, 0, len(s))
	for _, v := range s {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

func FilterGood[S ~[]E, E any](s S, f func(E) bool) S {
	out := make(S, 0, len(s))
	for _, v := range s {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

func trapLoseNamedType() {
	section("3.7 返回 []E 丢失具名类型")

	n := Nums{1, 2, 3, 4}
	odd := func(v int) bool { return v%2 == 1 }
	fmt.Printf("FilterBad  -> %T\n", FilterBad(n, odd))
	fmt.Printf("FilterGood -> %T\n", FilterGood(n, odd))
	fmt.Println("凡是'入什么切片类型、出就该是什么类型'的函数，都用 S ~[]E 双参数形式")
}

// ---------------------------------------------------------------------------
// 3.8 不能从返回值推断类型参数
// ---------------------------------------------------------------------------

func Make[T any]() T { var z T; return z }

// 工厂函数的改良写法：让类型参数出现在参数里
type Box[T any] struct{ V T }

func NewBox[T any](proto T) *Box[T] { return &Box[T]{V: proto} }

func trapInferFromReturn() {
	section("3.8 不能从返回值推断类型参数")

	// var m map[string]int = Make() // ✗ cannot infer T
	m := Make[map[string]int]()
	fmt.Printf("Make[map[string]int]() -> %T %v\n", m, m)

	b := NewBox(42) // 类型参数出现在参数里，调用方不用手写实例化
	fmt.Printf("NewBox(42) -> %T %v\n", b, *b)
}

// ---------------------------------------------------------------------------
// 3.9 未实例化的泛型函数不能当值用
// ---------------------------------------------------------------------------

func apply(f func(int, int) int, a, b int) int { return f(a, b) }

func trapUninstantiatedValue() {
	section("3.9 未实例化的泛型函数不能当值用")

	// g := Max                          // ✗ cannot use generic function Max without instantiation
	// apply(Max, 1, 2)                  // ✗ 同上
	var h func(int, int) int = Max[int]
	fmt.Println("Max[int] 可以当函数值:", h(1, 2), apply(Max[int], 3, 4))

	// 3.10 实例化循环：
	// func F[T any](x T) { F([]T{x}) } // ✗ instantiation cycle: T instantiated as []T
	fmt.Println("3.10 递归泛型函数的类型参数必须保持不变，否则 instantiation cycle")
}

// ---------------------------------------------------------------------------
// 3.15 用泛型模拟函数重载（反模式）
// ---------------------------------------------------------------------------

// ✗ 反模式：签名统一、函数体立刻分叉
func ProcessBad[T int | string](v T) string {
	switch x := any(v).(type) {
	case int:
		return fmt.Sprintf("加倍 %d", x*2)
	case string:
		return "大写 " + strings.ToUpper(x)
	}
	return ""
}

// ✓ 逻辑不同就写不同的函数
func ProcessInt(v int) string       { return fmt.Sprintf("加倍 %d", v*2) }
func ProcessString(v string) string { return "大写 " + strings.ToUpper(v) }

func trapFakeOverload() {
	section("3.15 用泛型模拟函数重载（反模式）")

	fmt.Println("反模式 ProcessBad:", ProcessBad(2), "/", ProcessBad("ab"))
	fmt.Println("正确做法:", ProcessInt(2), "/", ProcessString("ab"))
	fmt.Println("3.11 泛型的性能收益来自消除装箱，不来自消除动态派发（见 bench_test.go）")
	fmt.Println("3.12 每种底层类型一份机器码 + 每个实例一份字典 -> 二进制膨胀、编译变慢")
	fmt.Println("3.13 union 里不能放带方法的接口，方法和类型项要用交集（分行并列）")
}
