// func 示例：对应 notes/func.md
// 运行：go run ./func
// 逃逸分析：go build -gcflags='-m' ./func
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	basicDeclare()
	basicMultiReturn()
	basicNamedResult()
	basicVariadic()
	basicFuncValue()
	basicClosure()
	basicDefer()
	basicPanicRecover()

	closureShared()
	deferCost()
	escapeAnalysis()

	trapDeferNamedResult()
	trapLoopVar()
	trapDeferArgEval()
	trapRecoverIndirect()
	trapValueSemantics()
	trapDeferInLoop()
}

// ---------------------------------------------------------------------------
// 1.1 函数声明与调用
// ---------------------------------------------------------------------------

func add(x, y int) int                  { return x + y } // 同类型参数可合并
func swap(x, y string) (string, string) { return y, x }
func noReturn()                         { fmt.Println("  无返回值函数") }

func basicDeclare() {
	section("1.1 函数声明与调用")

	fmt.Println("add(3,4) =", add(3, 4))
	a, b := swap("x", "y")
	fmt.Println("swap:", a, b)
	noReturn()
	// Go 没有默认参数、没有重载：签名必须唯一
}

// ---------------------------------------------------------------------------
// 1.2 多返回值
// ---------------------------------------------------------------------------

func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

func basicMultiReturn() {
	section("1.2 多返回值")

	if r, err := divide(10, 2); err == nil {
		fmt.Println("10/2 =", r)
	}
	if _, err := divide(1, 0); err != nil {
		fmt.Println("错误:", err)
	}
	r, _ := divide(9, 3) // 不关心的返回值用 _ 忽略
	fmt.Println("9/3 =", r)
}

// ---------------------------------------------------------------------------
// 1.3 命名返回值
// ---------------------------------------------------------------------------

func minMax(s []int) (min, max int) {
	min, max = s[0], s[0]
	for _, v := range s[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return // 裸 return：返回已命名的 min, max
}

func basicNamedResult() {
	section("1.3 命名返回值")

	lo, hi := minMax([]int{3, 1, 4, 1, 5, 9, 2, 6})
	fmt.Printf("min=%d max=%d\n", lo, hi)
}

// ---------------------------------------------------------------------------
// 1.4 可变参数
// ---------------------------------------------------------------------------

func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// 展开传入不拷贝底层数组：函数内改元素调用方可见
func zeroFirst(nums ...int) {
	if len(nums) > 0 {
		nums[0] = 0
	}
}

func basicVariadic() {
	section("1.4 可变参数")

	fmt.Println("sum() =", sum(), " sum(1,2,3) =", sum(1, 2, 3))

	s := []int{1, 2, 3}
	fmt.Println("sum(s...) =", sum(s...))

	zeroFirst(s...) // 共享底层数组
	fmt.Println("展开传入后原 slice 被改:", s)
}

// ---------------------------------------------------------------------------
// 1.5 函数作为值与函数类型
// ---------------------------------------------------------------------------

type BinaryOp func(int, int) int

func apply(op BinaryOp, x, y int) int { return op(x, y) }

func basicFuncValue() {
	section("1.5 函数值与函数类型")

	plus := func(x, y int) int { return x + y }
	fmt.Println("apply(plus, 3, 4) =", apply(plus, 3, 4))
	fmt.Println("apply(匿名乘法) =", apply(func(x, y int) int { return x * y }, 3, 4))

	// 函数值不可比较，只能和 nil 比
	var f BinaryOp
	fmt.Println("零值函数 == nil:", f == nil)

	// 高阶函数：返回函数
	double := multiplier(2)
	fmt.Println("multiplier(2)(21) =", double(21))
}

func multiplier(k int) func(int) int { return func(v int) int { return k * v } }

// ---------------------------------------------------------------------------
// 1.6 闭包
// ---------------------------------------------------------------------------

func counter(start int) func() int {
	n := start // 逃逸到堆，被闭包持有
	return func() int {
		n++
		return n
	}
}

func basicClosure() {
	section("1.6 闭包")

	c1 := counter(0)
	fmt.Println(c1(), c1(), c1()) // 1 2 3

	c2 := counter(10)
	fmt.Println("独立实例:", c2(), " c1 继续:", c1())

	// 捕获的是变量本身，不是快照
	x := 1
	show := func() { fmt.Println("闭包看到的 x =", x) }
	x = 100
	show()
}

// ---------------------------------------------------------------------------
// 1.7 defer
// ---------------------------------------------------------------------------

func basicDefer() {
	section("1.7 defer")

	deferOrder()

	// 典型用法：成对的资源获取/释放
	if err := writeTemp(); err != nil {
		fmt.Println("写文件失败:", err)
	}
}

func deferOrder() {
	defer fmt.Println("  defer 1 (最后执行)")
	defer fmt.Println("  defer 2")
	defer fmt.Println("  defer 3 (最先执行)")
	fmt.Println("  函数体先执行完")
}

func writeTemp() error {
	f, err := os.CreateTemp("", "func-demo-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name()) // LIFO：后注册先执行，先删再关顺序要想清楚
	defer f.Close()

	_, err = f.WriteString("hello defer")
	fmt.Println("  临时文件:", f.Name())
	return err
}

// ---------------------------------------------------------------------------
// 1.8 panic 与 recover
// ---------------------------------------------------------------------------

func safeDiv(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()
	return a / b, nil // b == 0 触发 panic
}

func basicPanicRecover() {
	section("1.8 panic / recover")

	fmt.Println(safeDiv(10, 2))
	fmt.Println(safeDiv(10, 0))

	// panic 会沿调用栈向上，沿途执行每层 defer
	func() {
		defer fmt.Println("  内层 defer 也会执行")
		defer func() { fmt.Println("  外层 recover:", recover()) }()
		panic("boom")
	}()
}

// ---------------------------------------------------------------------------
// 2.2 闭包的内部实现：多个闭包共享同一个捕获变量
// ---------------------------------------------------------------------------

func makeAdders() (func(int) int, func(int) int) {
	n := 0 // 逃逸到堆，inc/dec 持有同一个指针
	inc := func(x int) int { n += x; return n }
	dec := func(x int) int { n -= x; return n }
	return inc, dec
}

func closureShared() {
	section("2.2 闭包共享捕获变量")

	inc, dec := makeAdders()
	fmt.Println("inc(5) =", inc(5))
	fmt.Println("dec(2) =", dec(2), "（共享同一个 n）")
}

// ---------------------------------------------------------------------------
// 2.3 defer 的实现：open-coded vs 链表
// ---------------------------------------------------------------------------

func deferCost() {
	section("2.3 defer 的两种实现")

	fmt.Println("固定数量的 defer -> open-coded，编译期展开，几乎零开销")
	fmt.Println("循环里的 defer   -> 退化为运行时 _defer 链表，有分配开销")
	fmt.Println("用 go test -bench 对比可见差异，见 func/bench_test.go")

	openCoded()
	loopDefer(3)
}

func openCoded() {
	defer fmt.Println("  open-coded defer")
}

func loopDefer(n int) {
	for i := range n { // 数量运行时才知道 -> 走 _defer 链表
		defer fmt.Println("  heap defer", i)
	}
}

// ---------------------------------------------------------------------------
// 2.4 逃逸分析
// ---------------------------------------------------------------------------

func escapes() *int  { x := 42; return &x } // 逃逸：地址被带出函数
func stackOnly() int { x := 42; return x }  // 不逃逸

func escapeAnalysis() {
	section("2.4 逃逸分析")

	p := escapes()
	fmt.Println("返回指针（堆分配）:", *p)
	fmt.Println("返回值（栈分配）:", stackOnly())
	fmt.Println("用 go build -gcflags='-m' ./func 查看：")
	fmt.Println("  ./main.go: moved to heap: x  <- escapes")
}

// ---------------------------------------------------------------------------
// 3.1 defer 修改命名返回值
// ---------------------------------------------------------------------------

func namedResult() (result int) {
	defer func() { result = -1 }() // 命名返回值：改动生效
	return 42
}

func anonymousResult() int {
	result := 42
	defer func() { result = -1 }() // 改的是局部变量，返回值已经拷贝走了
	return result
}

// 正确利用这个特性：出错时收尾并把错误传出去
func openChecked(path string) (f *os.File, err error) {
	f, err = os.Open(path)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			f.Close()
			f = nil
		}
	}()
	if fi, statErr := f.Stat(); statErr != nil {
		err = statErr
	} else if fi.IsDir() {
		err = errors.New("是目录，不是文件")
	}
	return
}

func trapDeferNamedResult() {
	section("3.1 defer 修改命名返回值")

	fmt.Println("命名返回值:", namedResult(), "（return 42 被 defer 覆盖）")
	fmt.Println("匿名返回值:", anonymousResult(), "（defer 改不到）")

	if _, err := openChecked(os.TempDir()); err != nil {
		fmt.Println("openChecked 自动收尾:", err)
	}
}

// ---------------------------------------------------------------------------
// 3.2 闭包捕获循环变量
// ---------------------------------------------------------------------------

func trapLoopVar() {
	section("3.2 闭包捕获循环变量")

	// Go 1.22+ 每次迭代都是新变量，这里输出 0 1 2 3 4
	funcs := make([]func(), 5)
	for i := 0; i < 5; i++ {
		funcs[i] = func() { fmt.Print(i, " ") }
	}
	fmt.Print("Go 1.22+ 直接捕获: ")
	for _, f := range funcs {
		f()
	}
	fmt.Println()

	// 显式创建局部变量（旧版本写法，语义永远正确）
	for i := 0; i < 5; i++ {
		i := i
		funcs[i] = func() { fmt.Print(i, " ") }
	}
	fmt.Print("显式遮蔽:       ")
	for _, f := range funcs {
		f()
	}
	fmt.Println()

	// 但"共享一个非循环变量"依旧会踩坑：这才是问题的本质
	shared := 0
	var fs []func()
	for i := 0; i < 3; i++ {
		shared = i
		fs = append(fs, func() { fmt.Print(shared, " ") })
	}
	fmt.Print("共享外层变量:   ")
	for _, f := range fs {
		f()
	}
	fmt.Println("（全是 2）")
}

// ---------------------------------------------------------------------------
// 3.3 defer 参数立即求值
// ---------------------------------------------------------------------------

func trapDeferArgEval() {
	section("3.3 defer 参数立即求值")

	func() {
		x := 10
		defer fmt.Println("  参数立即求值:", x)              // 打印 10
		defer func() { fmt.Println("  闭包延迟求值:", x) }() // 打印 20
		x = 20
	}()

	// 方法接收者同样立即求值
	func() {
		b := &strings.Builder{}
		b.WriteString("A")
		defer fmt.Println("  接收者立即求值:", b.String()) // "A"
		b.WriteString("B")
	}()
}

// ---------------------------------------------------------------------------
// 3.4 recover 只在直接 defer 的函数中有效
// ---------------------------------------------------------------------------

func helperRecover() {
	if r := recover(); r != nil { // 间接调用，永远拿不到
		fmt.Println("  helper 捕获到:", r)
	} else {
		fmt.Println("  helper 里的 recover 无效，r == nil")
	}
}

func trapRecoverIndirect() {
	section("3.4 recover 必须直接在 defer 函数里")

	func() {
		defer func() { fmt.Println("  最外层兜底 recover:", recover()) }()
		defer func() { helperRecover() }() // 间接调用无效
		panic("boom")
	}()
}

// ---------------------------------------------------------------------------
// 3.5 值传递陷阱
// ---------------------------------------------------------------------------

func doubleWrong(n int)  { n *= 2 }
func doubleRight(n *int) { *n *= 2 }

func fillWrong(s []int)       { s = append(s, 4) } // 扩容后新 header 调用方看不到
func fillRight(s []int) []int { return append(s, 4) }

func trapValueSemantics() {
	section("3.5 一切都是值传递")

	x := 10
	doubleWrong(x)
	fmt.Println("doubleWrong 后 x =", x)
	doubleRight(&x)
	fmt.Println("doubleRight 后 x =", x)

	s := make([]int, 3, 3)
	fillWrong(s)
	fmt.Println("fillWrong 后 s =", s, "len =", len(s))
	s = fillRight(s)
	fmt.Println("fillRight 后 s =", s, "len =", len(s))

	// 元素修改是可见的（共享底层数组）
	setFirst(s)
	fmt.Println("修改元素可见:", s)
}

func setFirst(s []int) {
	if len(s) > 0 {
		s[0] = 99
	}
}

// ---------------------------------------------------------------------------
// 3.6 循环里的 defer 会堆积到函数返回
// ---------------------------------------------------------------------------

func trapDeferInLoop() {
	section("3.6 循环里的 defer")

	fmt.Println("错误写法：资源直到函数返回才释放")
	badLoop()

	fmt.Println("正确写法：每次迭代封装成独立函数")
	goodLoop()
}

func badLoop() {
	for i := range 3 {
		f, err := os.CreateTemp("", "defer-bad-*")
		if err != nil {
			return
		}
		defer f.Close() // 3 个 fd 一直开到 badLoop 返回
		defer os.Remove(f.Name())
		fmt.Println("  打开", i)
	}
}

func goodLoop() {
	for i := range 3 {
		func() {
			f, err := os.CreateTemp("", "defer-good-*")
			if err != nil {
				return
			}
			defer f.Close() // 本轮迭代结束立刻释放
			defer os.Remove(f.Name())
			fmt.Println("  打开并立即关闭", i)
		}()
	}
}
