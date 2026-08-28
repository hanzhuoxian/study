// struct 示例：对应 notes/struct.md
// 运行：go run ./struct
package main

import (
	"fmt"
	"time"
	"unsafe"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

type Employee struct {
	ID        int
	Name      string
	Position  string
	Salary    int
	DoB       time.Time
	ManagerID int
}

func main() {
	basicLiteral()
	basicFieldPointer()
	basicAddressable()
	basicAnonymous()
	basicEmbed()
	basicMethod()

	layoutAlign()
	valueSemantics()
	comparable()
	emptyStruct()

	trapRangeCopy()
	trapValueReceiver()
	trapShallowCopy()
	trapMapValue()
	trapShadowAmbiguous()
}

// ---------------------------------------------------------------------------
// 1.1 定义与初始化
// ---------------------------------------------------------------------------

func basicLiteral() {
	section("1.1 定义与初始化")

	var e1 Employee  // 零值：每个字段取各自类型零值
	e2 := Employee{} // 同上

	// 顺序字面量：必须写全所有字段且顺序一致 —— 字段增删会静默错位，脆弱
	e3 := Employee{1, "Bob", "Dev", 100, time.Time{}, 0}
	// 具名字段：只写关心的字段，与顺序无关 —— 工程默认写法
	e4 := Employee{ID: 2, Name: "Alice", Position: "SRE"}

	fmt.Printf("e1 零值: %+v\n", e1)
	fmt.Println("e1 == e2:", e1 == e2)
	fmt.Printf("e3: ID=%d Name=%s Salary=%d\n", e3.ID, e3.Name, e3.Salary)
	fmt.Printf("e4: ID=%d Name=%s Salary=%d(零值)\n", e4.ID, e4.Name, e4.Salary)
}

// ---------------------------------------------------------------------------
// 1.2 字段访问与指针
// ---------------------------------------------------------------------------

func basicFieldPointer() {
	section("1.2 字段访问与指针")

	e := Employee{Name: "Bob", Position: "Dev"}
	e.Salary = 1000

	position := &e.Position // 可以取字段地址
	*position = "CEO"

	var pe *Employee = &e
	pe.Salary += 2000    // 语法糖：等价于 (*pe).Salary
	(*pe).Salary += 3000 // 显式写法

	fmt.Printf("%s: position=%s salary=%d\n", e.Name, e.Position, e.Salary)
}

// ---------------------------------------------------------------------------
// 1.3 可寻址性：返回指针才能原地改
// ---------------------------------------------------------------------------

var employees = []Employee{{ID: 1, Name: "Bob"}, {ID: 2, Name: "Alice"}}

func byIDPtr(id int) *Employee {
	for i := range employees {
		if employees[i].ID == id {
			return &employees[i] // slice 元素可寻址，返回地址供调用方修改
		}
	}
	return nil
}

func byIDValue(id int) Employee {
	for i := range employees {
		if employees[i].ID == id {
			return employees[i] // 返回拷贝
		}
	}
	return Employee{}
}

func basicAddressable() {
	section("1.3 可寻址性")

	byIDPtr(1).Position = "CTO" // 合法：返回值是指针
	fmt.Printf("改后 employees[0] = %s/%s\n", employees[0].Name, employees[0].Position)

	e := byIDValue(1)
	e.Position = "COO" // 改的是拷贝
	fmt.Printf("改拷贝后 employees[0].Position = %s\n", employees[0].Position)

	// byIDValue(1).Position = "COO" // 编译错误：函数返回的临时值不可寻址
}

// ---------------------------------------------------------------------------
// 1.4 匿名结构体
// ---------------------------------------------------------------------------

func basicAnonymous() {
	section("1.4 匿名结构体")

	config := struct {
		Host string
		Port int
	}{Host: "localhost", Port: 8080}
	fmt.Printf("config = %+v\n", config)

	// 表驱动测试里最常见的用法
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"double 2", 2, 4},
		{"double 3", 3, 6},
	}
	for _, c := range cases {
		fmt.Printf("case %-9s in=%d want=%d ok=%t\n", c.name, c.in, c.want, c.in*2 == c.want)
	}

	// 集合：value 用空结构体不占空间
	set := map[string]struct{}{"a": {}, "b": {}}
	_, ok := set["a"]
	fmt.Println("set 命中 a:", ok)
}

// ---------------------------------------------------------------------------
// 1.5 结构体嵌入（组合）
// ---------------------------------------------------------------------------

type Point struct{ X, Y int }

type Circle struct {
	Point  // 嵌入字段：只写类型名
	Radius int
}

type Wheel struct {
	Circle
	Spokes int
}

// 具名字段（对照组）：没有提升，只能写全路径
type NamedCircle struct {
	Point  Point
	Radius int
}

func basicEmbed() {
	section("1.5 结构体嵌入")

	w := Wheel{Circle{Point{1, 2}, 3}, 4}
	w.X = 10       // 字段提升：等价于 w.Circle.Point.X
	w.Point.Y = 20 // 也可以写中间层级
	fmt.Printf("w = %+v  w.X=%d w.Radius=%d w.Spokes=%d\n", w, w.X, w.Radius, w.Spokes)

	nc := NamedCircle{Point: Point{1, 2}, Radius: 3}
	// nc.X // 编译错误：具名字段没有提升
	fmt.Println("具名字段必须写全路径:", nc.Point.X)
}

// ---------------------------------------------------------------------------
// 1.6 方法与接收者（含方法提升）
// ---------------------------------------------------------------------------

func (p Point) Distance() int { return p.X*p.X + p.Y*p.Y } // 值接收者
func (p *Point) Scale(f int)  { p.X *= f; p.Y *= f }       // 指针接收者

func basicMethod() {
	section("1.6 方法与接收者")

	w := Wheel{Circle{Point{1, 2}, 3}, 4}
	fmt.Println("方法提升 w.Distance() =", w.Distance()) // 来自嵌入的 Point

	w.Scale(10) // w 可寻址，自动取地址 (&w.Circle.Point).Scale(10)
	fmt.Printf("Scale 后 w.Point = %+v\n", w.Point)
}

// ---------------------------------------------------------------------------
// 2.1 内存布局与字段对齐
// ---------------------------------------------------------------------------

type Bad struct {
	a bool  // 1 + 7 padding
	b int64 // 8
	c bool  // 1 + 7 padding
}

type Good struct {
	b int64 // 8
	a bool  // 1
	c bool  // 1 + 6 padding
}

func layoutAlign() {
	section("2.1 内存布局与对齐")

	var b Bad
	fmt.Printf("Bad  Sizeof=%d Alignof=%d  offsets: a=%d b=%d c=%d\n",
		unsafe.Sizeof(b), unsafe.Alignof(b),
		unsafe.Offsetof(b.a), unsafe.Offsetof(b.b), unsafe.Offsetof(b.c))

	var g Good
	fmt.Printf("Good Sizeof=%d Alignof=%d  offsets: b=%d a=%d c=%d\n",
		unsafe.Sizeof(g), unsafe.Alignof(g),
		unsafe.Offsetof(g.b), unsafe.Offsetof(g.a), unsafe.Offsetof(g.c))

	fmt.Printf("string=%d slice=%d map=%d chan=%d interface{}=%d\n",
		unsafe.Sizeof(""), unsafe.Sizeof([]int(nil)), unsafe.Sizeof(map[int]int(nil)),
		unsafe.Sizeof(make(chan int)), unsafe.Sizeof(any(nil)))
}

// ---------------------------------------------------------------------------
// 2.2 结构体是值类型
// ---------------------------------------------------------------------------

func valueSemantics() {
	section("2.2 结构体是值类型")

	p := Point{1, 2}
	q := p // 整体拷贝
	q.X = 100
	fmt.Println("p =", p, " q =", q)

	byValue(p)
	fmt.Println("值传参后 p =", p)
	byPointer(&p)
	fmt.Println("指针传参后 p =", p)

	// 大结构体按值传递的拷贝开销
	type Big struct{ Buf [1024]byte }
	fmt.Printf("Big 一次传参要拷贝 %d 字节\n", unsafe.Sizeof(Big{}))
}

func byValue(p Point)    { p.X = 999 }
func byPointer(p *Point) { p.X = 999 }

// ---------------------------------------------------------------------------
// 2.4 结构体的可比较性
// ---------------------------------------------------------------------------

type WithSlice struct {
	Name  string
	Items []int // 不可比较字段 -> 整个结构体不可比较
}

func comparable() {
	section("2.4 可比较性")

	w := Wheel{Circle{Point{1, 2}, 3}, 4}
	w1 := Wheel{Circle{Point{1, 2}, 3}, 4}
	fmt.Println("w == w1:", w == w1) // 逐字段递归比较

	// 可比较的结构体可以做 map key
	dist := map[Point]int{{1, 2}: 5}
	fmt.Println("map[Point]int:", dist[Point{1, 2}])

	// var a, b WithSlice; a == b // 编译错误：invalid operation, Items 不可比较
	// map[WithSlice]int{}        // 编译错误：invalid map key type
	fmt.Println("含 slice 字段的结构体不可比较、不能做 map key")
}

// ---------------------------------------------------------------------------
// 2.5 空结构体
// ---------------------------------------------------------------------------

func emptyStruct() {
	section("2.5 空结构体")

	var e struct{}
	fmt.Println("Sizeof(struct{}{}) =", unsafe.Sizeof(e))

	a, b := struct{}{}, struct{}{}
	fmt.Printf("零大小值共享 runtime.zerobase: &a=%p &b=%p\n", &a, &b)

	done := make(chan struct{}) // 纯信号
	go func() { close(done) }()
	<-done
	fmt.Println("chan struct{} 信号已收到")
}

// ---------------------------------------------------------------------------
// 3.1 range 循环变量是拷贝
// ---------------------------------------------------------------------------

type T struct{ V int }

func trapRangeCopy() {
	section("3.1 range 循环变量是拷贝")

	s := []T{{1}, {2}, {3}}

	var bad []*T
	for _, v := range s {
		bad = append(bad, &v) // Go 1.22+ 每轮是新变量，但仍是"拷贝的地址"
	}
	bad[0].V = 100
	fmt.Println("改 &v 后 s =", s, "（原 slice 未变）")

	var good []*T
	for i := range s {
		good = append(good, &s[i]) // 拿原元素地址
	}
	good[0].V = 100
	fmt.Println("改 &s[i] 后 s =", s)
}

// ---------------------------------------------------------------------------
// 3.2 值接收者方法改不动原对象
// ---------------------------------------------------------------------------

type Emp struct{ Salary int }

func (e Emp) RaiseByValue() { e.Salary += 1000 } // 改的是拷贝
func (e *Emp) RaiseByPtr()  { e.Salary += 1000 }

func trapValueReceiver() {
	section("3.2 值接收者改不动原对象")

	e := Emp{Salary: 100}
	e.RaiseByValue()
	fmt.Println("值接收者调用后:", e.Salary)
	e.RaiseByPtr()
	fmt.Println("指针接收者调用后:", e.Salary)
}

// ---------------------------------------------------------------------------
// 3.3 含引用字段的浅拷贝
// ---------------------------------------------------------------------------

type Box struct {
	Items []int
	Meta  map[string]string
}

func trapShallowCopy() {
	section("3.3 浅拷贝")

	a := Box{Items: []int{1, 2, 3}, Meta: map[string]string{"k": "v"}}
	b := a // 拷贝 header，底层数组/哈希表仍共享
	b.Items[0] = 99
	b.Meta["k"] = "changed"
	fmt.Printf("a.Items=%v a.Meta=%v （被 b 改到了）\n", a.Items, a.Meta)

	// 需要独立副本要显式深拷贝
	c := Box{Items: append([]int(nil), a.Items...), Meta: map[string]string{}}
	for k, v := range a.Meta {
		c.Meta[k] = v
	}
	c.Items[0] = -1
	c.Meta["k"] = "deep"
	fmt.Printf("深拷贝后 a.Items=%v a.Meta=%v\n", a.Items, a.Meta)
}

// ---------------------------------------------------------------------------
// 3.4 map 中的结构体不可寻址
// ---------------------------------------------------------------------------

func trapMapValue() {
	section("3.4 map value 不可寻址")

	m := map[string]Emp{"a": {Salary: 100}}
	// m["a"].Salary = 200 // 编译错误：cannot assign to struct field m["a"].Salary

	e := m["a"] // 取出
	e.Salary = 200
	m["a"] = e // 放回
	fmt.Println("取出-修改-放回:", m["a"].Salary)

	mp := map[string]*Emp{"a": {Salary: 100}}
	mp["a"].Salary = 200 // value 是指针就可以直接改
	fmt.Println("map[K]*V:", mp["a"].Salary)
}

// ---------------------------------------------------------------------------
// 3.5 嵌入同名字段：遮蔽与歧义
// ---------------------------------------------------------------------------

type A struct{ Name string }
type B struct{ Name string }

type C struct { // 同深度冲突
	A
	B
}

type D struct { // 外层遮蔽内层
	A
	Name string
}

func trapShadowAmbiguous() {
	section("3.5 遮蔽与歧义")

	var c C
	// c.Name = "x" // 编译错误：ambiguous selector c.Name
	c.A.Name = "fromA"
	c.B.Name = "fromB"
	fmt.Printf("同深度冲突必须写全路径: A=%s B=%s\n", c.A.Name, c.B.Name)

	var d D
	d.Name = "outer" // 命中外层
	d.A.Name = "inner"
	fmt.Printf("浅层遮蔽深层: d.Name=%s d.A.Name=%s\n", d.Name, d.A.Name)
}
