// method 示例：对应 notes/method.md
// 运行：go run ./method
package main

import (
	"fmt"
	"math"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	basicDeclare()
	basicReceiverKind()
	basicMethodValue()
	basicMethodExpression()
	basicEmbedPromotion()
	basicNilReceiver()

	desugar()
	methodSet()
	ifaceDispatch()

	trapValueReceiver()
	trapPointerReceiverInterface()
	trapBindTiming()
	trapNotAddressable()
	trapNilDeref()
}

// ---------------------------------------------------------------------------
// 1.1 方法声明
// ---------------------------------------------------------------------------

type Point struct{ X, Y float64 }

func (p Point) Distance(q Point) float64 { return math.Hypot(q.X-p.X, q.Y-p.Y) }

func basicDeclare() {
	section("1.1 方法声明")

	p, q := Point{1, 2}, Point{4, 6}
	fmt.Printf("p.Distance(q) = %.1f\n", p.Distance(q))

	// 接收者必须是当前包内定义的具名类型：可以给内置类型起别名后再定义方法
	fmt.Println("给具名类型定义方法:", Celsius(100).String())
}

type Celsius float64

func (c Celsius) String() string { return fmt.Sprintf("%.1f°C", float64(c)) }

// ---------------------------------------------------------------------------
// 1.2 值接收者与指针接收者
// ---------------------------------------------------------------------------

type Counter struct{ n int }

func (c Counter) IncByValue()    { c.n++ } // 改副本
func (c *Counter) IncByPointer() { c.n++ } // 改原对象

func basicReceiverKind() {
	section("1.2 值接收者 vs 指针接收者")

	var c Counter
	c.IncByValue()
	fmt.Println("值接收者调用后 n =", c.n)

	c.IncByPointer() // 语法糖：编译器自动改写成 (&c).IncByPointer()
	(&c).IncByPointer()
	fmt.Println("指针接收者调用后 n =", c.n)
}

// ---------------------------------------------------------------------------
// 1.3 方法值（method value）
// ---------------------------------------------------------------------------

func basicMethodValue() {
	section("1.3 方法值")

	p, q := Point{1, 2}, Point{4, 6}
	distance := p.Distance // 绑定了接收者 p，类型 func(Point) float64
	fmt.Printf("%T -> %.1f\n", distance, distance(q))

	// 常见用法：把绑定好接收者的方法当回调传出去
	apply := func(f func(Point) float64) float64 { return f(q) }
	fmt.Printf("作为回调: %.1f\n", apply(p.Distance))
}

// ---------------------------------------------------------------------------
// 1.4 方法表达式（method expression）
// ---------------------------------------------------------------------------

func basicMethodExpression() {
	section("1.4 方法表达式")

	p, q := Point{1, 2}, Point{4, 6}

	distanceExpr := Point.Distance // 不绑定接收者，类型 func(Point, Point) float64
	fmt.Printf("%T -> %.1f\n", distanceExpr, distanceExpr(p, q))

	incExpr := (*Counter).IncByPointer // 指针接收者的方法表达式
	c := &Counter{}
	incExpr(c)
	fmt.Printf("%T 调用后 n = %d\n", incExpr, c.n)
}

// ---------------------------------------------------------------------------
// 1.5 通过嵌入获得方法
// ---------------------------------------------------------------------------

type ColoredPoint struct {
	Point
	Color string
}

func basicEmbedPromotion() {
	section("1.5 嵌入获得方法")

	cp := ColoredPoint{Point{1, 2}, "red"}
	fmt.Printf("提升的方法 cp.Distance = %.1f\n", cp.Distance(Point{4, 6}))

	// 提升进方法集后，外层类型也能满足接口
	var s fmt.Stringer = Temp{Celsius(36.5)}
	fmt.Println("嵌入让外层满足接口:", s)
}

type Temp struct{ Celsius } // 嵌入 Celsius，String() 被提升

// ---------------------------------------------------------------------------
// 1.6 nil 接收者
// ---------------------------------------------------------------------------

type IntList struct {
	Value int
	Tail  *IntList
}

func (list *IntList) Sum() int {
	if list == nil { // 显式处理 nil 是安全调用的前提
		return 0
	}
	return list.Value + list.Tail.Sum()
}

func basicNilReceiver() {
	section("1.6 nil 接收者")

	var list *IntList
	fmt.Println("nil 链表 Sum =", list.Sum())

	list = &IntList{1, &IntList{2, &IntList{3, nil}}}
	fmt.Println("三节点链表 Sum =", list.Sum())
}

// ---------------------------------------------------------------------------
// 2.1 方法只是语法糖：接收者是隐式的第一个参数
// ---------------------------------------------------------------------------

func desugar() {
	section("2.1 方法是语法糖")

	p, q := Point{1, 2}, Point{4, 6}

	fmt.Printf("p.Distance(q)        = %.1f\n", p.Distance(q))
	fmt.Printf("Point.Distance(p, q) = %.1f （降级后的普通函数）\n", Point.Distance(p, q))

	var f1 func(Point) float64 = p.Distance            // 方法值 = 局部应用后的闭包
	var f2 func(Point, Point) float64 = Point.Distance // 方法表达式 = 原始函数
	fmt.Printf("方法值 %T / 方法表达式 %T\n", f1, f2)
}

// ---------------------------------------------------------------------------
// 2.2 方法集规则：T 只含值接收者方法，*T 含全部
// ---------------------------------------------------------------------------

type Both struct{ v int }

func (b Both) Val() int   { return b.v }
func (b *Both) Set(v int) { b.v = v }

type Valuer interface{ Val() int }
type Setter interface{ Set(int) }

func methodSet() {
	section("2.2 方法集")

	var _ Valuer = Both{}  // T 满足只含值接收者方法的接口
	var _ Valuer = &Both{} // *T 也满足
	// var _ Setter = Both{} // 编译错误：Set 是指针接收者
	var _ Setter = &Both{}

	fmt.Println("T  的方法集: Val")
	fmt.Println("*T 的方法集: Val, Set")
}

// ---------------------------------------------------------------------------
// 2.3 接口值与动态派发
// ---------------------------------------------------------------------------

type Shape interface{ Area() float64 }

type Rect struct{ W, H float64 }
type Circ struct{ R float64 }

func (r Rect) Area() float64 { return r.W * r.H }
func (c Circ) Area() float64 { return math.Pi * c.R * c.R }

func ifaceDispatch() {
	section("2.3 接口动态派发（itab）")

	shapes := []Shape{Rect{3, 4}, Circ{1}}
	for _, s := range shapes {
		// 编译期不知道具体类型，运行时查 itab.fun[0] 拿到函数指针
		fmt.Printf("%-12T Area=%.2f\n", s, s.Area())
	}
}

// ---------------------------------------------------------------------------
// 3.1 值接收者改不动原对象
// ---------------------------------------------------------------------------

func trapValueReceiver() {
	section("3.1 值接收者改不动原对象")

	c := Counter{}
	for range 3 {
		c.IncByValue()
	}
	fmt.Println("调用三次值接收者后 n =", c.n)

	// 常见踩坑：把值接收者方法当作"能改状态"的方法挂进 goroutine / 回调
	incs := []func(){c.IncByValue, c.IncByPointer}
	for _, f := range incs {
		f()
	}
	fmt.Println("混用之后 n =", c.n, "（只有指针接收者那次生效）")
}

// ---------------------------------------------------------------------------
// 3.2 值类型无法满足指针接收者方法的接口
// ---------------------------------------------------------------------------

type Named struct{ v string }

func (t *Named) Set(s string) { t.v = s }

type StrSetter interface{ Set(string) }

func trapPointerReceiverInterface() {
	section("3.2 值类型不满足指针接收者接口")

	t := Named{}
	t.Set("ok") // 直接调用没问题：t 可寻址，自动取址
	fmt.Println("直接调用:", t.v)

	// var s StrSetter = Named{}  // 编译错误：Set method has pointer receiver
	var s StrSetter = &Named{}
	s.Set("via interface")
	fmt.Println("赋值给接口必须用 *T:", s.(*Named).v)
}

// ---------------------------------------------------------------------------
// 3.3 方法值绑定接收者的时机
// ---------------------------------------------------------------------------

func (c Counter) Get() int   { return c.n }
func (c *Counter) GetP() int { return c.n }

func trapBindTiming() {
	section("3.3 方法值的绑定时机")

	c := Counter{n: 1}

	get := c.Get // 值接收者：此刻拷贝了 c（n=1）
	c.n = 100
	fmt.Println("值接收者方法值:", get(), "（不是 100）")

	getP := c.GetP // 指针接收者：绑定的是 &c
	c.n = 200
	fmt.Println("指针接收者方法值:", getP(), "（跟随最新状态）")
}

// ---------------------------------------------------------------------------
// 3.4 不可寻址的值无法调用指针接收者方法
// ---------------------------------------------------------------------------

type Item struct{ v int }

func (t *Item) Inc()     { t.v++ }
func (t Item) Show() int { return t.v }

func newItem() Item { return Item{} }

func trapNotAddressable() {
	section("3.4 不可寻址的值不能调用指针方法")

	m := map[string]Item{"a": {v: 1}}
	// m["a"].Inc()  // 编译错误：cannot call pointer method Inc on Item
	fmt.Println("map value 只能调用值接收者方法:", m["a"].Show())

	it := m["a"] // 取出到局部变量（可寻址）
	it.Inc()
	m["a"] = it
	fmt.Println("取出-改-写回:", m["a"].Show())

	// newItem().Inc() // 编译错误：函数返回的临时值不可寻址
	tmp := newItem()
	tmp.Inc()
	fmt.Println("先赋给变量再调用:", tmp.Show())

	mp := map[string]*Item{"a": {v: 1}} // value 用指针最省事
	mp["a"].Inc()
	fmt.Println("map[K]*V:", mp["a"].Show())
}

// ---------------------------------------------------------------------------
// 3.5 nil 指针调用方法未做判空
// ---------------------------------------------------------------------------

type Unsafe struct{ Name string }

func (t *Unsafe) String() string { return t.Name } // 没有处理 nil

func trapNilDeref() {
	section("3.5 nil 指针解引用")

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recover:", r)
		}
	}()

	var p *Unsafe
	fmt.Println("调用本身合法（nil 只是普通参数值），解引用字段才 panic")
	_ = p.String()
}
