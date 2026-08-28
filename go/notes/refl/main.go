// reflect 示例：对应 notes/reflect.md
// 运行：go run ./refl
// 压测：go test -bench . -benchmem ./refl
package main

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	basicThreeLaws()
	basicTypeValue()
	basicKind()
	basicSettable()
	basicStructTags()
	basicNewAPIs()

	principleEface()
	principleMakeFunc()
	principleDeepEqual()

	trapCanSet()
	trapNilAndZero()
	trapUnexported()
	trapPerf()
	trapInterfaceRoundTrip()
}

// ---------------------------------------------------------------------------
// 1.1 反射三定律
// ---------------------------------------------------------------------------

func basicThreeLaws() {
	section("1.1 反射三定律（Rob Pike 的 laws of reflection）")

	// 定律一：从接口值可以反射出 reflect.Value
	var x float64 = 3.4
	v := reflect.ValueOf(x)
	fmt.Printf("  ① interface -> reflect.Value: type=%v kind=%v value=%v\n", v.Type(), v.Kind(), v.Float())

	// 定律二：从 reflect.Value 可以还原成接口值
	back := v.Interface().(float64)
	fmt.Printf("  ② reflect.Value -> interface: %v\n", back)

	// 定律三：要修改一个 reflect.Value，它必须是"可设置的"（settable）
	// 可设置 = 可寻址 且 不是通过未导出字段拿到的
	fmt.Printf("  ③ reflect.ValueOf(x).CanSet() = %v ← 传的是副本，改不了\n", v.CanSet())
	p := reflect.ValueOf(&x).Elem()
	fmt.Printf("     reflect.ValueOf(&x).Elem().CanSet() = %v\n", p.CanSet())
	p.SetFloat(7.1)
	fmt.Printf("     SetFloat(7.1) 之后 x = %v\n", x)
}

// ---------------------------------------------------------------------------
// 1.2 Type 与 Value
// ---------------------------------------------------------------------------

type Address struct {
	City string `json:"city"`
}

type User struct {
	Name    string `json:"name" validate:"required,min=2"`
	Age     int    `json:"age" validate:"gte=0,lte=150"`
	Email   string `json:"email,omitempty"`
	Addr    Address
	private string
}

func (u User) Greet() string     { return "hi " + u.Name }
func (u *User) SetName(n string) { u.Name = n }
func (u User) String() string    { return "User(" + u.Name + ")" }

func basicTypeValue() {
	section("1.2 Type 与 Value")

	u := User{Name: "bob", Age: 30, Addr: Address{City: "SH"}}
	t := reflect.TypeOf(u)
	v := reflect.ValueOf(u)

	fmt.Printf("  Type:  Name=%q PkgPath=%q Kind=%v Size=%d NumField=%d NumMethod=%d\n",
		t.Name(), t.PkgPath(), t.Kind(), t.Size(), t.NumField(), t.NumMethod())

	fmt.Println("  遍历字段：")
	for i := range t.NumField() {
		f := t.Field(i)
		fmt.Printf("    [%d] %-8s %-8v offset=%-2d exported=%v tag=%q\n",
			i, f.Name, f.Type, f.Offset, f.IsExported(), f.Tag)
	}

	fmt.Println("  按名字取：", v.FieldByName("Name").String())
	fmt.Println("  嵌套取：  ", v.FieldByName("Addr").FieldByName("City").String())
	fmt.Println("  索引路径：", v.FieldByIndex([]int{3, 0}).String())

	// 值类型 vs 指针类型的方法集差异（method.md 2.2）
	fmt.Printf("  reflect.TypeOf(u).NumMethod()  = %d（值方法：Greet/String）\n", t.NumMethod())
	fmt.Printf("  reflect.TypeOf(&u).NumMethod() = %d（值方法 + 指针方法 SetName）\n",
		reflect.TypeOf(&u).NumMethod())

	// 调用方法
	out := v.MethodByName("Greet").Call(nil)
	fmt.Println("  Call Greet():", out[0].String())

	pv := reflect.ValueOf(&u)
	pv.MethodByName("SetName").Call([]reflect.Value{reflect.ValueOf("alice")})
	fmt.Println("  Call SetName(\"alice\") 之后:", u.Name)

	// TypeFor（1.22+）：不需要造一个零值就能拿到 Type
	fmt.Printf("  reflect.TypeFor[User]() = %v（不用写 reflect.TypeOf(User{})）\n", reflect.TypeFor[User]())
}

// ---------------------------------------------------------------------------
// 1.3 Kind 与 Type 的区别
// ---------------------------------------------------------------------------

type MyInt int
type MySlice []MyInt

func basicKind() {
	section("1.3 Kind vs Type")

	for _, v := range []any{int(1), MyInt(1), []int{1}, MySlice{1}, &User{}, map[string]int{}, make(chan int)} {
		t := reflect.TypeOf(v)
		fmt.Printf("  %-18v Type=%-18v Kind=%v\n", fmt.Sprintf("%T", v), t, t.Kind())
	}
	fmt.Println("→ Kind 是底层种类（就那 26 种），Type 是具体类型（无穷多）")
	fmt.Println("→ 写通用代码 switch 的是 Kind；判断具体类型才比 Type")
	fmt.Println("→ Elem() 的含义随 Kind 变：Ptr/Slice/Array/Map/Chan 各有意义，其他会 panic")
	fmt.Printf("  MySlice.Elem() = %v，*User.Elem() = %v\n",
		reflect.TypeFor[MySlice]().Elem(), reflect.TypeFor[*User]().Elem())
}

// ---------------------------------------------------------------------------
// 1.4 可设置性（settability）
// ---------------------------------------------------------------------------

func basicSettable() {
	section("1.4 可设置性")

	u := User{Name: "bob"}

	cases := []struct {
		desc string
		v    reflect.Value
	}{
		{"ValueOf(u)", reflect.ValueOf(u)},
		{"ValueOf(u).Field(0)", reflect.ValueOf(u).Field(0)},
		{"ValueOf(&u)", reflect.ValueOf(&u)},
		{"ValueOf(&u).Elem()", reflect.ValueOf(&u).Elem()},
		{"ValueOf(&u).Elem().Field(0)", reflect.ValueOf(&u).Elem().Field(0)},
		{"ValueOf(&u).Elem().Field(4) 未导出", reflect.ValueOf(&u).Elem().Field(4)},
	}
	for _, c := range cases {
		fmt.Printf("  %-38s CanAddr=%-5v CanSet=%-5v CanInterface=%v\n",
			c.desc, c.v.CanAddr(), c.v.CanSet(), c.v.CanInterface())
	}

	fmt.Println("→ CanSet = CanAddr && 字段是导出的")
	fmt.Println("→ 所以要改东西，reflect.ValueOf 必须传指针，再 .Elem()")

	// map 的元素不可寻址，只能整体 SetMapIndex
	m := map[string]int{"a": 1}
	mv := reflect.ValueOf(m)
	fmt.Printf("  map 元素 CanSet = %v（不可寻址），要用 SetMapIndex\n",
		mv.MapIndex(reflect.ValueOf("a")).CanSet())
	mv.SetMapIndex(reflect.ValueOf("a"), reflect.ValueOf(2))
	fmt.Println("  SetMapIndex 之后:", m)

	// slice 元素可寻址（因为底层数组在堆上）
	s := []int{1, 2, 3}
	sv := reflect.ValueOf(s)
	fmt.Printf("  slice 元素 CanSet = %v（底层数组可寻址）\n", sv.Index(0).CanSet())
	sv.Index(0).SetInt(99)
	fmt.Println("  Index(0).SetInt(99) 之后:", s)
}

// ---------------------------------------------------------------------------
// 1.5 struct tag
// ---------------------------------------------------------------------------

func basicStructTags() {
	section("1.5 struct tag")

	t := reflect.TypeFor[User]()
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Tag == "" {
			continue
		}
		jsonTag, jsonOK := f.Tag.Lookup("json")
		valTag := f.Tag.Get("validate")
		fmt.Printf("  %-6s json=%-18q(ok=%v) validate=%q\n", f.Name, jsonTag, jsonOK, valTag)
	}

	fmt.Println("→ tag 是一个字符串，约定格式 `key:\"value\" key2:\"v2\"`（空格分隔，值用双引号）")
	fmt.Println("→ Get 找不到返回 \"\"，Lookup 能区分'没有这个 key'和'值是空串'")
	fmt.Println("→ tag 里的逗号语义（omitempty 之类）是各个库自己解析的，reflect 不管")
	fmt.Println("→ ⚠️ tag 写错（比如用了单引号、key 后面有空格）编译器不报错，只会静默失效")
	fmt.Println("   go vet 的 structtag 检查能抓到一部分")

	// 用 StructOf 动态构造一个"tag 写错"的类型，避免源码里写坏 tag 被 go vet 拦下
	badType := reflect.StructOf([]reflect.StructField{{
		Name: "A",
		Type: reflect.TypeFor[string](),
		Tag:  reflect.StructTag(`json: "a"`), // ✗ 冒号后有空格
	}})
	fmt.Printf("  写错的 tag `json: \"a\"`: Get(\"json\")=%q ← 静默失效\n",
		badType.Field(0).Tag.Get("json"))
}

// ---------------------------------------------------------------------------
// 1.6 新 API（1.22 / 1.23 / 1.25 / 1.26）
// ---------------------------------------------------------------------------

func basicNewAPIs() {
	section("1.6 新 API")

	// 1.22: TypeFor[T]
	fmt.Printf("  1.22 TypeFor[[]User]() = %v\n", reflect.TypeFor[[]User]())

	// 1.23: Value.Seq / Seq2 —— 语义完全对齐 range 那个类型时的行为
	s := reflect.ValueOf([]int{10, 20, 30})
	fmt.Print("  1.23 slice.Seq():   ")
	for v := range s.Seq() { // 对齐 `for i := range s`，给的是**下标**
		fmt.Print(v.Int(), " ")
	}
	fmt.Print("  <- 是下标，因为 range slice 单变量给的就是下标")
	fmt.Println()

	fmt.Print("  1.23 slice.Seq2():  ")
	for i, v := range s.Seq2() { // 对齐 `for i, v := range s`
		fmt.Printf("%v=%v ", i, v)
	}
	fmt.Println()

	m := reflect.ValueOf(map[string]int{"a": 1})
	fmt.Print("  1.23 map.Seq2():    ")
	for k, v := range m.Seq2() {
		fmt.Printf("%v=%v ", k, v)
	}
	fmt.Println()

	// 1.25: TypeAssert[T] —— 省掉 Interface().(T) 的装箱
	u := User{Name: "bob"}
	if got, ok := reflect.TypeAssert[User](reflect.ValueOf(u)); ok {
		fmt.Printf("  1.25 TypeAssert[User]: %v（等价于 v.Interface().(User)，但少一次装箱）\n", got.Name)
	}

	// 1.26: 迭代器版的字段/方法遍历
	fmt.Print("  1.26 Type.Fields():  ")
	for f := range reflect.TypeFor[User]().Fields() {
		fmt.Print(f.Name, " ")
	}
	fmt.Println()

	fmt.Print("  1.26 Value.Fields(): ")
	for f, v := range reflect.ValueOf(u).Fields() {
		if f.IsExported() {
			fmt.Printf("%s=%v ", f.Name, v)
		}
	}
	fmt.Println()

	fmt.Print("  1.26 Type.Methods(): ")
	for m := range reflect.TypeFor[*User]().Methods() {
		fmt.Print(m.Name, " ")
	}
	fmt.Println()

	ft := reflect.TypeFor[func(int, string) (bool, error)]()
	fmt.Print("  1.26 Type.Ins()/Outs(): in=")
	for t := range ft.Ins() {
		fmt.Print(t, " ")
	}
	fmt.Print("out=")
	for t := range ft.Outs() {
		fmt.Print(t, " ")
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// 2.1 reflect.Value 的内部结构
// ---------------------------------------------------------------------------

func principleEface() {
	section("2.1 reflect.Value 的内部结构")

	fmt.Println("  type Value struct {")
	fmt.Println("      typ_ *abi.Type      // 类型描述符（和 eface 里的 _type 是同一个东西）")
	fmt.Println("      ptr  unsafe.Pointer // 数据指针")
	fmt.Println("      flag                // kind、是否可寻址、是否只读、是否间接寻址...")
	fmt.Println("  }")
	fmt.Printf("  Sizeof(reflect.Value) = %d 字节\n", unsafe.Sizeof(reflect.Value{}))

	fmt.Println()
	fmt.Println("  reflect.ValueOf(x any) 做了什么：")
	fmt.Println("    ① x 已经是接口值（调用方把具体值装箱进 eface）—— 这一步就有一次逃逸/分配")
	fmt.Println("    ② 从 eface 里拆出 (type, data)，填进 Value 的 typ_ / ptr")
	fmt.Println("    ③ 按类型计算 flag（kind + flagIndir 等）")
	fmt.Println()
	fmt.Println("  → 这解释了反射为什么慢：每次进出反射边界都要装箱/拆箱 + 查表 + 位运算，")
	fmt.Println("    而且值一旦逃进接口就上了堆（mem.md 2.2）")
	fmt.Println("  → 也解释了 Value 为什么可以按值传递：它就是 24 字节的三元组")
}

// ---------------------------------------------------------------------------
// 2.2 MakeFunc：动态造函数
// ---------------------------------------------------------------------------

func principleMakeFunc() {
	section("2.2 MakeFunc")

	// 造一个"把任意二元函数变成带日志版本"的通用包装器
	var add func(int, int) int = func(a, b int) int { return a + b }
	logged := wrapWithLog(add).(func(int, int) int)
	fmt.Println("  logged(3, 4) =", logged(3, 4))

	var concat func(string, string) string = func(a, b string) string { return a + b }
	logged2 := wrapWithLog(concat).(func(string, string) string)
	fmt.Println("  logged2(\"a\",\"b\") =", logged2("a", "b"))

	fmt.Println("→ MakeFunc 的签名: MakeFunc(typ Type, fn func([]Value) []Value) Value")
	fmt.Println("→ 用途：RPC 客户端桩、ORM 的动态查询、测试 mock、依赖注入容器")
	fmt.Println("→ 代价：每次调用都要构造 []Value（分配）+ 反射调用，比直接调用慢 1-2 个数量级")
}

func wrapWithLog(fn any) any {
	fv := reflect.ValueOf(fn)
	ft := fv.Type()
	return reflect.MakeFunc(ft, func(in []reflect.Value) []reflect.Value {
		args := make([]string, len(in))
		for i, a := range in {
			args[i] = fmt.Sprint(a.Interface())
		}
		out := fv.Call(in)
		fmt.Printf("    [log] call(%s) -> %v\n", strings.Join(args, ", "), out[0].Interface())
		return out
	}).Interface()
}

// ---------------------------------------------------------------------------
// 2.3 DeepEqual
// ---------------------------------------------------------------------------

func principleDeepEqual() {
	section("2.3 DeepEqual 的规则与坑")

	fmt.Println("  nil slice vs 空 slice:    ", reflect.DeepEqual([]int(nil), []int{}), "← false！")
	fmt.Println("  nil map vs 空 map:        ", reflect.DeepEqual(map[string]int(nil), map[string]int{}), "← false！")
	fmt.Println("  两个 NaN:                ", reflect.DeepEqual(nan(), nan()), "← false（浮点规则）")
	fmt.Println("  同一个 func:             ", reflect.DeepEqual(nan, nan), "← 函数只有都是 nil 才相等")
	fmt.Println("  不同类型的相同值 int/int64:", reflect.DeepEqual(int(1), int64(1)), "← 类型必须一致")

	type node struct{ next *node }
	a, b := &node{}, &node{}
	a.next, b.next = a, b // 各自成环
	fmt.Println("  循环引用:                ", reflect.DeepEqual(a, b), "（有 visited 记录，不会栈溢出）")

	fmt.Println("→ DeepEqual 会比较**未导出字段**，所以对 time.Time、sync.Mutex 之类容易误判")
	fmt.Println("→ 测试里优先用 google/go-cmp（cmp.Diff）：能配置忽略字段、能打印差异")
	fmt.Println("→ 1.21 起简单场景可以用 slices.Equal / maps.Equal，它们不走反射、快很多")
}

func nan() float64  { return float64(0) / zero() }
func zero() float64 { return 0 }

// ---------------------------------------------------------------------------
// 3.1 CanSet 相关 panic
// ---------------------------------------------------------------------------

func trapCanSet() {
	section("3.1 常见 panic")

	panics := []struct {
		desc string
		fn   func()
	}{
		{"改一个不可设置的值", func() { reflect.ValueOf(User{}).Field(0).SetString("x") }},
		{"Elem() 一个非指针", func() { reflect.ValueOf(1).Elem() }},
		{"Int() 一个 string", func() { _ = reflect.ValueOf("s").Int() }},
		{"Field() 越界", func() { _ = reflect.ValueOf(User{}).Field(99) }},
		{"Call 参数个数不对", func() { reflect.ValueOf(func(int) {}).Call(nil) }},
		{"读未导出字段的 Interface()", func() { _ = reflect.ValueOf(User{}).Field(4).Interface() }},
		{"用零值 Value", func() { _ = reflect.Value{}.Kind().String(); reflect.Value{}.Int() }},
	}
	for _, p := range panics {
		func() {
			defer func() { fmt.Printf("  %-26s -> panic: %v\n", p.desc, recover()) }()
			p.fn()
		}()
	}
	fmt.Println("→ 反射把编译期错误变成了运行时 panic，这是它最大的成本")
	fmt.Println("→ 所以反射代码必须：① 有充分的单元测试 ② 在边界处校验 Kind ③ 尽量收敛在一个包里")
}

// ---------------------------------------------------------------------------
// 3.2 nil 与零值
// ---------------------------------------------------------------------------

func trapNilAndZero() {
	section("3.2 nil 与零值")

	var nilPtr *User
	var nilIface any
	var nilSlice []int

	fmt.Printf("  reflect.ValueOf(nil):        IsValid=%v ← 零值 Value，什么方法都不能调\n",
		reflect.ValueOf(nilIface).IsValid())
	fmt.Printf("  reflect.ValueOf(nilPtr):     IsValid=%v IsNil=%v Kind=%v\n",
		reflect.ValueOf(nilPtr).IsValid(), reflect.ValueOf(nilPtr).IsNil(), reflect.ValueOf(nilPtr).Kind())
	fmt.Printf("  reflect.ValueOf(nilSlice):   IsNil=%v Len=%d\n",
		reflect.ValueOf(nilSlice).IsNil(), reflect.ValueOf(nilSlice).Len())

	fmt.Println("  IsNil 只对 chan/func/interface/map/ptr/slice 合法，其他 Kind 会 panic")
	fmt.Println("  IsZero（1.13+）对所有类型合法，判断是否等于类型零值")
	fmt.Printf("  ValueOf(User{}).IsZero() = %v，ValueOf(User{Name:\"a\"}).IsZero() = %v\n",
		reflect.ValueOf(User{}).IsZero(), reflect.ValueOf(User{Name: "a"}).IsZero())

	fmt.Println("→ 处理 any 参数的反射代码，第一件事就是 if !v.IsValid() { ... }")
}

// ---------------------------------------------------------------------------
// 3.3 未导出字段
// ---------------------------------------------------------------------------

func trapUnexported() {
	section("3.3 未导出字段")

	u := User{Name: "bob", private: "secret"}
	f := reflect.ValueOf(u).Field(4)

	fmt.Printf("  未导出字段: 能读到类型和 Kind = %v/%v\n", f.Type(), f.Kind())
	fmt.Printf("  CanInterface = %v，CanSet = %v\n", f.CanInterface(), f.CanSet())
	fmt.Println("  → Interface() 会 panic，String() 却能拿到值（因为 String 不走 Interface）")
	fmt.Printf("  f.String() = %q ← 只有 string kind 有这个'后门'\n", f.String())

	// 用 unsafe 突破（能做，但别在生产代码里做）
	pu := &User{private: "secret"}
	pv := reflect.ValueOf(pu).Elem().Field(4)
	real := reflect.NewAt(pv.Type(), unsafe.Pointer(pv.UnsafeAddr())).Elem()
	fmt.Printf("  reflect.NewAt + UnsafeAddr 突破: %q，CanSet=%v\n", real.String(), real.CanSet())
	real.SetString("changed")
	fmt.Printf("  改完: %q\n", pu.private)
	fmt.Println("→ 这个技巧在测试和序列化库里偶尔用；生产代码里出现基本等于设计有问题")
}

// ---------------------------------------------------------------------------
// 3.4 性能
// ---------------------------------------------------------------------------

func trapPerf() {
	section("3.4 性能（详见 bench_test.go）")

	for _, row := range [][2]string{
		{"u.Name 直接取字段", "0.83 ns/op    0 allocs"},
		{"Value.Field(0).String()", "3.50 ns/op    0 allocs   ~4x"},
		{"ValueOf(u).Field(0).String()", "4.43 ns/op    0 allocs（含 ValueOf）"},
		{"Value.FieldByName(\"Name\")", "59.5 ns/op    0 allocs   ~72x ← 要遍历+比字符串"},
		{"FieldByIndex(缓存的 []int)", "14.5 ns/op    0 allocs   ← 缓存索引路径的效果"},
		{"u.Age = 1 直接设", "0.63 ns/op"},
		{"Value.SetInt(1)", "3.04 ns/op    ~5x"},
		{"u.Greet() 直接调", "27.4 ns/op    1 alloc（返回值拼字符串）"},
		{"Value.Method.Call(nil)", "315.6 ns/op   4 allocs   ~11x"},
		{"MakeFunc 包装后调用", "470.2 ns/op   5 allocs"},
		{"直接调普通函数", "0.64 ns/op    0 allocs"},
		{"reflect.TypeOf(u)", "3.62 ns/op"},
		{"reflect.TypeFor[User]()", "0.88 ns/op    ← 编译期就知道类型，快 4 倍"},
		{"Value.Interface()", "4.93 ns/op"},
		{"v.Interface().(User)", "10.35 ns/op"},
		{"reflect.TypeAssert[User](v)", "20.00 ns/op   ← 实测反而更慢！"},
		{"a1 == a2（struct 直接比）", "6.87 ns/op    0 allocs"},
		{"reflect.DeepEqual(a1, a2)", "198.0 ns/op   2 allocs   ~29x"},
		{"reflect.DeepEqual(两个 []int)", "224.3 ns/op   2 allocs"},
	} {
		fmt.Printf("    %-30s %s\n", row[0], row[1])
	}
	fmt.Println()
	fmt.Println("  几个和直觉不符的点：")
	fmt.Println("    · 取字段只慢 4 倍，不是传说中的 100 倍——**FieldByName 才是真凶（72x）**")
	fmt.Println("    · TypeFor[T] 比 TypeOf(v) 快 4 倍：前者编译期确定，后者要装箱 + 读 eface")
	fmt.Println("    · TypeAssert[T]（1.25+）实测比 Interface().(T) 慢一倍，")
	fmt.Println("      它的价值在类型安全和省掉装箱分配（本例中两者都没分配），不在速度")
	fmt.Println("    · DeepEqual 有分配，热路径上不要用；能直接比就直接比")
	fmt.Println()
	fmt.Println("  四条优化手法（标准库和主流库都在用）：")
	fmt.Println("    ① 缓存 Type 级别的解析结果（encoding/json 的 cachedTypeFields，见 json.md 2.1）")
	fmt.Println("    ② 用 []int 索引路径代替 FieldByName（FieldByIndex 快得多）")
	fmt.Println("    ③ 走一次反射生成闭包/代码，之后走闭包（sqlx、gorm 的做法）")
	fmt.Println("    ④ 干脆代码生成（easyjson、protobuf-go）——零反射")
}

// ---------------------------------------------------------------------------
// 3.5 接口往返
// ---------------------------------------------------------------------------

func trapInterfaceRoundTrip() {
	section("3.5 反射与接口的往返")

	var s fmt.Stringer = User{Name: "bob"}

	// 反射一个接口变量：拿到的是**动态类型**，不是接口类型
	v := reflect.ValueOf(s)
	fmt.Printf("  reflect.ValueOf(接口变量): Type=%v Kind=%v ← 动态类型，接口这层没了\n",
		v.Type(), v.Kind())

	// 要拿到接口类型本身，得通过指针
	vp := reflect.ValueOf(&s).Elem()
	fmt.Printf("  reflect.ValueOf(&接口).Elem(): Type=%v Kind=%v ← 这才是接口类型\n",
		vp.Type(), vp.Kind())
	fmt.Printf("  再 .Elem() 才拿到动态值: %v\n", vp.Elem().Type())

	// 判断类型是否实现某接口
	stringerType := reflect.TypeFor[fmt.Stringer]()
	fmt.Printf("  User 实现 Stringer? %v\n", reflect.TypeFor[User]().Implements(stringerType))
	fmt.Printf("  *User 实现 Stringer? %v（值方法会被指针继承）\n",
		reflect.TypeFor[*User]().Implements(stringerType))
	fmt.Printf("  Address 实现 Stringer? %v\n", reflect.TypeFor[Address]().Implements(stringerType))

	fmt.Println("→ 这是写序列化/校验库时最容易搞错的一处：ValueOf 会自动'穿透'接口")
	fmt.Println("→ 想检测 json.Marshaler 之类的接口，用 Type.Implements 而不是类型断言")
}
