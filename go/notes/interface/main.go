// interface 示例：对应 notes/interface.md
// 运行：go run ./interface
package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"unsafe"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	basicImplicit()
	basicAssign()
	basicEmptyInterface()
	basicAssertion()
	basicTypeSwitch()
	basicEmbedding()
	basicProgramToInterface()
	basicCapabilityProbe()

	efaceIface()
	boxing()
	ifaceCompare()
	nilInterface()

	trapTypedNil()
	trapUncomparable()
	trapUnsafeAssert()
	trapStrictType()
	trapReceiverKind()
	trapShadow()
	trapWrappedError()
}

// ---------------------------------------------------------------------------
// 1.1 接口声明与隐式实现
// ---------------------------------------------------------------------------

type Animal interface{ Sound() string }

type Dog struct{}
type Cat struct{}

func (Dog) Sound() string { return "Woof" }
func (Cat) Sound() string { return "Meow" }

func basicImplicit() {
	section("1.1 隐式实现")

	// 没有 implements 关键字：方法集匹配即满足
	animals := []Animal{Dog{}, Cat{}}
	for _, a := range animals {
		fmt.Printf("%-9T -> %s\n", a, a.Sound())
	}

	// 编译期断言某类型实现了接口的惯用写法
	var _ Animal = Dog{}
}

// ---------------------------------------------------------------------------
// 1.2 接口变量的赋值与调用
// ---------------------------------------------------------------------------

func basicAssign() {
	section("1.2 接口变量赋值与调用")

	var w io.Writer = os.Stdout // 具体类型 *os.File 赋给接口
	n, err := w.Write([]byte("  写到 Stdout\n"))
	fmt.Printf("  n=%d err=%v 动态类型=%T\n", n, err, w)

	w = &strings.Builder{} // 同一个接口变量可以换装不同动态类型
	fmt.Fprint(w, "写到 Builder")
	fmt.Printf("  换成 %T: %q\n", w, w.(*strings.Builder).String())
}

// ---------------------------------------------------------------------------
// 1.3 空接口 interface{} / any
// ---------------------------------------------------------------------------

func basicEmptyInterface() {
	section("1.3 空接口 any")

	var x any = 42
	fmt.Printf("%T %v\n", x, x)
	x = "hello"
	fmt.Printf("%T %v\n", x, x)
	x = struct{ N int }{1}
	fmt.Printf("%T %v\n", x, x)

	// any 就是 interface{} 的别名
	var y interface{} = x
	fmt.Println("any 与 interface{} 是别名:", y == x)
}

// ---------------------------------------------------------------------------
// 1.4 类型断言
// ---------------------------------------------------------------------------

func basicAssertion() {
	section("1.4 类型断言")

	var x any = "hello"

	s := x.(string)  // 不安全形式
	n, ok := x.(int) // 安全形式：失败返回零值 + false
	fmt.Printf("s=%q  n=%d ok=%t\n", s, n, ok)

	// 断言成具体类型 = 拆箱；断言成接口 = 换壳
	var w io.Writer = os.Stdout
	f := w.(*os.File)       // 拆箱：静态类型变成 *os.File
	rw := w.(io.ReadWriter) // 换壳：动态类型仍是 *os.File，方法集变大
	fmt.Printf("拆箱 %T / 换壳 %T（动态类型 %T）\n", f, rw, rw)

	// nil 接口的断言恒失败
	var nilW io.Writer
	_, ok1 := nilW.(io.Writer)
	_, ok2 := nilW.(*os.File)
	fmt.Printf("nil 接口断言: io.Writer=%t *os.File=%t\n", ok1, ok2)

	// 断言成功 ≠ 拿到非 nil 值
	var err error = (*MyError)(nil)
	e, ok := err.(*MyError)
	fmt.Printf("类型化 nil: ok=%t e==nil=%t\n", ok, e == nil)

	// 泛型类型参数要先转成接口才能断言
	fmt.Println("泛型里断言:", assertString(any("x")), assertString(any(1)))
}

func assertString[T any](v T) bool {
	// _, ok := v.(string) // 编译错误
	_, ok := any(v).(string)
	return ok
}

// ---------------------------------------------------------------------------
// 1.5 类型选择 type switch
// ---------------------------------------------------------------------------

func describe(x any) string {
	switch v := x.(type) {
	case nil: // 只匹配接口值本身为 nil
		return "nil 接口"
	case int:
		return fmt.Sprintf("int: %d", v)
	case string:
		return fmt.Sprintf("string: %q", v)
	case *os.File: // 单类型 case：v 被收窄，可直接调具体方法
		return "file: " + v.Name()
	case io.Reader, io.Writer: // 多类型 case：v 不收窄，仍是 any
		return fmt.Sprintf("reader/writer: %T", v)
	default:
		return fmt.Sprintf("unknown: %T", v)
	}
}

func basicTypeSwitch() {
	section("1.5 type switch")

	inputs := []any{nil, 1, "s", os.Stdout, strings.NewReader("r"), 3.14, (*MyError)(nil)}
	for _, in := range inputs {
		fmt.Println(" ", describe(in))
	}
	fmt.Println("注意：接口里装 nil 指针走的是具体类型分支，不是 case nil")
}

// ---------------------------------------------------------------------------
// 1.6 接口的组合
// ---------------------------------------------------------------------------

type Reader interface{ Read(p []byte) (int, error) }
type Writer interface{ Write(p []byte) (int, error) }

type ReadWriter interface { // 嵌入 = 方法集并集
	Reader
	Writer
}

func basicEmbedding() {
	section("1.6 接口组合")

	var rw ReadWriter = &struct { // 临时组合一个满足 ReadWriter 的类型
		io.Reader
		io.Writer
	}{strings.NewReader("hello"), os.Stdout}

	buf := make([]byte, 5)
	n, _ := rw.Read(buf)
	fmt.Printf("Read %d 字节: %q\n", n, buf[:n])
	fmt.Println("标准库里的组合接口: io.ReadWriteCloser 等")
}

// ---------------------------------------------------------------------------
// 1.7 面向接口编程
// ---------------------------------------------------------------------------

type ByLen []string

func (s ByLen) Len() int           { return len(s) }
func (s ByLen) Less(i, j int) bool { return len(s[i]) < len(s[j]) }
func (s ByLen) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func basicProgramToInterface() {
	section("1.7 面向接口编程")

	data := ByLen{"banana", "kiwi", "apple"}
	sort.Sort(data) // sort 只依赖三个方法
	fmt.Println("sort.Interface:", data)

	// 依赖接口便于测试打桩
	fmt.Println("写到 Stdout:", report(os.Stdout))
	var sb strings.Builder
	report(&sb)
	fmt.Printf("写到 Builder: %q\n", sb.String())
}

func report(w io.Writer) int {
	n, _ := fmt.Fprint(w, "report done\n")
	return n
}

// ---------------------------------------------------------------------------
// 1.8 断言的典型用法：能力探测
// ---------------------------------------------------------------------------

type stringWriter interface{ WriteString(string) (int, error) }

// 模仿 io.Copy / io.WriteString 的可选能力探测
func writeString(w io.Writer, s string) (int, error) {
	if sw, ok := w.(stringWriter); ok { // 有更快的路径就走更快的
		return sw.WriteString(s)
	}
	return w.Write([]byte(s)) // 否则退化
}

func basicCapabilityProbe() {
	section("1.8 能力探测")

	var sb strings.Builder
	n, _ := writeString(&sb, "fast path")
	fmt.Printf("*strings.Builder 有 WriteString: n=%d\n", n)

	if _, ok := any(os.Stdout).(io.ReaderFrom); ok {
		fmt.Println("*os.File 实现了 io.ReaderFrom -> io.Copy 会走 ReadFrom")
	}

	// net.Error 探测是否可重试
	var err error = &net.DNSError{Err: "timeout", IsTimeout: true}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		fmt.Println("net.Error 超时可重试:", ne)
	}
}

// ---------------------------------------------------------------------------
// 2.1 eface / iface 的内存布局
// ---------------------------------------------------------------------------

type eface struct {
	Type unsafe.Pointer
	Data unsafe.Pointer
}

func efaceIface() {
	section("2.1 eface / iface 布局")

	var x any = 42
	e := (*eface)(unsafe.Pointer(&x))
	fmt.Printf("any{42}:  Type=%p Data=%p sizeof=%d\n", e.Type, e.Data, unsafe.Sizeof(x))

	var w io.Writer = os.Stdout
	i := (*eface)(unsafe.Pointer(&w)) // 非空接口第一个字是 *itab
	fmt.Printf("io.Writer: ITab=%p Data=%p sizeof=%d\n", i.Type, i.Data, unsafe.Sizeof(w))

	var w2 io.Writer = os.Stdout
	i2 := (*eface)(unsafe.Pointer(&w2))
	fmt.Printf("同一 (接口,类型) 组合共享 itab: %t\n", i.Type == i2.Type)
}

// ---------------------------------------------------------------------------
// 2.3 装箱：值类型赋给接口需要一份拷贝
// ---------------------------------------------------------------------------

type big struct{ Buf [256]byte }

func boxing() {
	section("2.3 装箱开销")

	fmt.Println("小整数 0~255 装箱复用 runtime.staticuint64s，不分配")
	fmt.Println("大 struct 值装箱要堆分配一份拷贝；传指针几乎零开销")
	fmt.Println("详见 interface/bench_test.go 的 BenchmarkBox* 对比")

	var x any = big{}
	fmt.Printf("装箱后接口本身仍是 %d 字节（数据在 Data 指向的堆上）\n", unsafe.Sizeof(x))

	// 装进去的是拷贝：改原值不影响接口里的副本
	b := big{}
	b.Buf[0] = 1
	var y any = b
	b.Buf[0] = 2
	fmt.Println("接口里是拷贝:", y.(big).Buf[0], " 原值:", b.Buf[0])
}

// ---------------------------------------------------------------------------
// 2.4 接口值的比较
// ---------------------------------------------------------------------------

func ifaceCompare() {
	section("2.4 接口值比较")

	var a, b any = 1, 1
	var c, d any = 1, "1"
	fmt.Println("1 == 1:", a == b, "  1 == \"1\":", c == d, "（动态类型不同直接 false）")

	type MyInt int
	var e, f any = MyInt(1), 1
	fmt.Println("MyInt(1) == 1:", e == f, "（具名类型与底层类型不同）")
}

// ---------------------------------------------------------------------------
// 2.5 nil 接口 vs 接口里装了 nil
// ---------------------------------------------------------------------------

func nilInterface() {
	section("2.5 nil 接口 vs 装了 nil")

	var i1 any // Type=nil Data=nil
	var p *int
	var i2 any = p // Type=*int Data=nil

	e1 := (*eface)(unsafe.Pointer(&i1))
	e2 := (*eface)(unsafe.Pointer(&i2))
	fmt.Printf("i1: Type=%p Data=%p  i1==nil: %t\n", e1.Type, e1.Data, i1 == nil)
	fmt.Printf("i2: Type=%p Data=%p  i2==nil: %t\n", e2.Type, e2.Data, i2 == nil)
}

// ---------------------------------------------------------------------------
// 3.1 类型化 nil 陷阱
// ---------------------------------------------------------------------------

type MyError struct{}

func (*MyError) Error() string { return "boom" }

func doSomething() *MyError { return nil }

func runBad() error {
	var err *MyError = doSomething()
	return err // 危险：(*MyError)(nil) 装进 error 接口后不等于 nil
}

func runGood() error {
	if err := doSomething(); err != nil {
		return err
	}
	return nil // 真正的接口 nil
}

func trapTypedNil() {
	section("3.1 类型化 nil 陷阱")

	if err := runBad(); err != nil {
		fmt.Printf("runBad:  err != nil 恒成立（动态类型 %T，动态值 nil），打印出来是 %v\n", err, err)
	}
	if err := runGood(); err == nil {
		fmt.Println("runGood: err == nil ✓")
	}
}

// ---------------------------------------------------------------------------
// 3.2 不可比较的动态类型
// ---------------------------------------------------------------------------

func trapUncomparable() {
	section("3.2 比较不可比较的动态类型")

	defer func() { fmt.Println("recover:", recover()) }()

	var a, b any = []int{1}, []int{2}
	fmt.Println("编译期放行，运行时才检查 ->", a == b)
}

// ---------------------------------------------------------------------------
// 3.3 不安全断言
// ---------------------------------------------------------------------------

func trapUnsafeAssert() {
	section("3.3 不安全断言直接 panic")

	defer func() { fmt.Println("recover:", recover()) }()

	var x any = "hello"
	if n, ok := x.(int); !ok {
		fmt.Printf("comma-ok 安全: n=%d ok=%t\n", n, ok)
	}
	_ = x.(int) // panic: interface conversion
}

// ---------------------------------------------------------------------------
// 3.6 断言要求类型严格相等
// ---------------------------------------------------------------------------

type MyInt int
type AliasInt = int // 别名，不是新类型

func trapStrictType() {
	section("3.6 断言不看底层类型")

	var a any = MyInt(5)
	_, ok1 := a.(int)
	_, ok2 := a.(MyInt)
	fmt.Printf("MyInt(5).(int)=%t  .(MyInt)=%t\n", ok1, ok2)

	var b any = 5
	_, ok3 := b.(AliasInt)
	fmt.Printf("int(5).(AliasInt)=%t（别名是同一类型）\n", ok3)

	// 需要按底层类型处理时用 reflect.Kind 或泛型的 ~T 约束（见 generic.md）
}

// ---------------------------------------------------------------------------
// 3.7 值接收者 vs 指针接收者导致断言失败
// ---------------------------------------------------------------------------

type Foo struct{}

func (*Foo) Foo() {} // 指针接收者

func trapReceiverKind() {
	section("3.7 装值还是装指针决定方法集")

	var a any = Foo{}
	_, ok1 := a.(interface{ Foo() })

	var b any = &Foo{}
	_, ok2 := b.(interface{ Foo() })

	fmt.Printf("装 T{}: %t   装 &T{}: %t\n", ok1, ok2)
}

// ---------------------------------------------------------------------------
// 3.8 comma-ok 里的变量遮蔽
// ---------------------------------------------------------------------------

func trapShadow() {
	section("3.8 comma-ok 变量遮蔽")

	var w io.Writer = os.Stdout
	if w, ok := w.(*os.File); ok { // 这里的 w 是新的 *os.File 局部变量
		fmt.Printf("  内层 w 的静态类型: %T name=%s\n", w, w.Name())
	}
	fmt.Printf("  出了 if，w 仍是 %T（静态类型 io.Writer）\n", w)
}

// ---------------------------------------------------------------------------
// 3.9 被 %w 包装后的错误
// ---------------------------------------------------------------------------

func trapWrappedError() {
	section("3.9 包装后的错误要用 errors.As/Is")

	base := &os.PathError{Op: "open", Path: "/no/such", Err: os.ErrNotExist}
	err := fmt.Errorf("open config: %w", base)

	_, ok := err.(*os.PathError)
	fmt.Printf("裸断言: %t（动态类型是 %T）\n", ok, err)

	var pe *os.PathError
	fmt.Println("errors.As:", errors.As(err, &pe), "->", pe.Path)
	fmt.Println("errors.Is(err, os.ErrNotExist):", errors.Is(err, os.ErrNotExist))
}
