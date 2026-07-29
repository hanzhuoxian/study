// json_ref 用一组可运行的小例子验证 notes/json.md 里整理的 encoding/json 行为：
// struct tag 选项、Marshaler/TextMarshaler 优先级、匿名字段提升与冲突消解、
// map key 排序、[]byte base64、Unmarshal 的字段匹配规则、RawMessage/Number、
// 以及流式 Encoder/Decoder。
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

type Movie struct {
	Name      string `json:"name"`
	Author    string
	Password  string    `json:"-"`                             // 忽略字段
	Age       int       `json:"age,string"`                    // 强制转换成字符串
	Color     bool      `json:"color,omitempty" db:"is_color"` // 空值忽略字段
	CreatedAt time.Time `json:"created_at,omitzero"`           // 零值（IsZero）忽略字段
}

func main() {
	sections := []struct {
		title string
		fn    func()
	}{
		{"一、struct tag 与 reflect 解析", tagAndReflect},
		{"二、omitempty vs omitzero", omitEmptyVsOmitZero},
		{"三、Marshaler / TextMarshaler 优先级", marshalerPriority},
		{"四、匿名字段提升与命名冲突", embedAndConflict},
		{"五、map key 排序与 []byte base64", mapAndByteSlice},
		{"六、Unmarshal 字段匹配 / 未知字段 / 指针要求", unmarshalRules},
		{"七、RawMessage 延迟解析与 json.Number", rawMessageAndNumber},
		{"八、流式 Encoder / Decoder", streaming},
		{"九、循环引用检测", cycleDetect},
	}
	for _, s := range sections {
		fmt.Printf("\n==================== %s ====================\n", s.title)
		s.fn()
	}
}

// ---------------------------------------------------------------- 一

func tagAndReflect() {
	movie := Movie{
		Name:      "功夫女足",
		Author:    "周星驰",
		Password:  "should-not-appear",
		Age:       18,
		Color:     true,
		CreatedAt: time.Date(2004, 12, 23, 0, 0, 0, 0, time.UTC),
	}

	// tag 本身只是字符串，encoding/json 在 typeFields 阶段用 strings.Cut 之类的方式
	// 把 "名字,选项1,选项2" 切开；同一个字段上可以并存多个库的 tag（json / db / ...）。
	ft, ok := reflect.TypeOf(movie).FieldByName("Color")
	if !ok {
		panic("Movie 没有 Color 字段")
	}
	tag := ft.Tag.Get("json")
	name, opts, found := strings.Cut(tag, ",")
	fmt.Printf("json tag = %q\n", tag)
	fmt.Printf("  name=%q opts=%q found=%v\n", name, opts, found)
	fmt.Printf("db   tag = %q\n", ft.Tag.Get("db"))

	// Author 无 tag → 直接用 Go 字段名；Password 是 "-" → 不参与编解码；
	// Age 带 ",string" → 编码成 "18"。
	s, err := json.MarshalIndent(movie, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Marshal:\n%s\n", s)

	// 对称地，Unmarshal 时 age 也必须是字符串形式的 "18"，给数字 18 会报错。
	var back Movie
	if err := json.Unmarshal(s, &back); err != nil {
		panic(err)
	}
	fmt.Printf("Unmarshal 回来: name=%s author=%s age=%d password=%q\n",
		back.Name, back.Author, back.Age, back.Password)

	fmt.Printf(`age 给数字而非 "18" 时: %v`+"\n",
		json.Unmarshal([]byte(`{"age":18}`), &Movie{}))
}

// ---------------------------------------------------------------- 二

type Inner struct {
	A int `json:"a"`
}

type EmptyZero struct {
	// omitempty 只覆盖 bool/数字/指针/interface/len==0 的容器，struct 落到 default 分支
	// 永远不算 empty，所以下面这个字段即使是零值也会被输出。
	StructEmpty Inner `json:"struct_empty,omitempty"`
	// omitzero 走 IsZero()/reflect.Value.IsZero()，对 struct 生效。
	StructZero Inner `json:"struct_zero,omitzero"`

	Slice     []int          `json:"slice,omitempty"`      // nil / len==0 → 省略
	Map       map[string]int `json:"map,omitempty"`        // nil / len==0 → 省略
	EmptySlc  []int          `json:"empty_slice,omitzero"` // 注意：非 nil 的空 slice 不是零值
	Zero      int            `json:"zero,omitempty"`
	TimeZero  time.Time      `json:"time_zero,omitzero"`   // time.Time 实现了 IsZero()
	TimeEmpty time.Time      `json:"time_empty,omitempty"` // struct，omitempty 无效
}

func omitEmptyVsOmitZero() {
	v := EmptyZero{EmptySlc: []int{}}
	s, _ := json.MarshalIndent(v, "", "  ")
	fmt.Printf("全零值结构体:\n%s\n", s)
	fmt.Println("→ struct_empty / time_empty 说明 omitempty 对 struct 不生效；")
	fmt.Println("→ empty_slice 说明 omitzero 判断的是「等于类型零值」，[]int{} != nil 所以保留。")
}

// ---------------------------------------------------------------- 三

// Celsius 实现 json.Marshaler：可以产出任意合法 JSON（这里是一个对象）。
type Celsius float64

func (c Celsius) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `{"unit":"C","value":%g}`, float64(c)), nil
}

func (c *Celsius) UnmarshalJSON(data []byte) error {
	var raw struct {
		Value float64 `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = Celsius(raw.Value)
	return nil
}

// Level 只实现 encoding.TextMarshaler：结果被当成字符串再加引号转义。
type Level int

const (
	LevelLow Level = iota
	LevelHigh
)

func (l Level) MarshalText() ([]byte, error) {
	if l == LevelHigh {
		return []byte("high"), nil
	}
	return []byte("low"), nil
}

func (l *Level) UnmarshalText(text []byte) error {
	switch string(text) {
	case "high":
		*l = LevelHigh
	case "low":
		*l = LevelLow
	default:
		return fmt.Errorf("unknown level %q", text)
	}
	return nil
}

// Both 同时实现两个接口，用来验证 Marshaler 优先级更高。
type Both struct{}

func (Both) MarshalJSON() ([]byte, error) { return []byte(`"from-MarshalJSON"`), nil }
func (Both) MarshalText() ([]byte, error) { return []byte("from-MarshalText"), nil }

func marshalerPriority() {
	type payload struct {
		Temp  Celsius `json:"temp"`
		Level Level   `json:"level"`
		Both  Both    `json:"both"`
		// TextMarshaler 也用于 map 的 key：Level 会被 MarshalText 转成字符串
		ByLevel map[Level]int `json:"by_level"`
	}
	p := payload{Temp: 36.6, Level: LevelHigh, ByLevel: map[Level]int{LevelHigh: 1, LevelLow: 2}}
	s, _ := json.Marshal(p)
	fmt.Printf("Marshal: %s\n", s)
	fmt.Println("→ both 字段说明 Marshaler > TextMarshaler；")
	fmt.Println("→ by_level 说明 map key 也走 TextMarshaler（并按转换后的字符串排序）。")

	var back struct {
		Temp  Celsius `json:"temp"`
		Level Level   `json:"level"`
	}
	if err := json.Unmarshal([]byte(`{"temp":{"unit":"C","value":40.5},"level":"low"}`), &back); err != nil {
		panic(err)
	}
	fmt.Printf("Unmarshal: temp=%g level=%d（分别走 Unmarshaler / TextUnmarshaler）\n",
		float64(back.Temp), back.Level)
}

// ---------------------------------------------------------------- 四

type Base struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Meta struct {
	Name string `json:"meta_name"` // 改了 json 名，和 Base.Name 不冲突
}

// Promoted：匿名嵌入的字段被提升到外层，输出是平铺的。
type Promoted struct {
	Base
	Meta
	Extra string `json:"extra"`
}

// 注意：冲突判断看的是**JSON 名字**，不是 Go 字段名。
// 所以 `Tag string`（JSON 名 "Tag"）和 `X string \`json:"tag"\`` 并不冲突，
// 下面几个例子刻意让它们的 JSON 名完全相同。

type L1 struct {
	Tag string // 无 tag，JSON 名 "Tag"
}

type L2 struct {
	Tag string // 无 tag，JSON 名 "Tag"，和 L1.Tag 同层同名
}

// AmbiguousConflict：同层两个匿名字段都提升出 "Tag"，且都没有显式 tag → 全部丢弃。
// （两个都写了 tag 的情况结果一样，但 go vet 的 structtag 检查会在编译期就报重复
// json tag，所以这里不写成代码。）
type AmbiguousConflict struct {
	L1
	L2
	Keep string `json:"keep"`
}

type T1 struct {
	Tag string // 无 tag，JSON 名 "Tag"
}

type T2 struct {
	Label string `json:"Tag"` // 显式 tag 也叫 "Tag"，只有它写了 tag → 冲突消解为它
}

type TagWins struct {
	T1
	T2
}

// ShallowWins：外层深度 0 的 "Tag" 遮蔽掉嵌入层深度 1 的同名字段（BFS 浅层优先）。
type ShallowWins struct {
	L1
	Tag string
}

func embedAndConflict() {
	p := Promoted{Base: Base{ID: 1, Name: "base"}, Meta: Meta{Name: "meta"}, Extra: "x"}
	s, _ := json.Marshal(p)
	fmt.Printf("提升（平铺）      : %s\n", s)

	a := AmbiguousConflict{L1: L1{Tag: "l1"}, L2: L2{Tag: "l2"}, Keep: "k"}
	s, _ = json.Marshal(a)
	fmt.Printf("同层同名都无 tag  : %s  ← Tag 被整体丢弃\n", s)

	w := TagWins{T1: T1{Tag: "t1"}, T2: T2{Label: "t2"}}
	s, _ = json.Marshal(w)
	fmt.Printf("同层同名只一个 tag: %s  ← 带 tag 的胜出\n", s)

	sw := ShallowWins{L1: L1{Tag: "deep"}, Tag: "shallow"}
	s, _ = json.Marshal(sw)
	fmt.Printf("浅层优先          : %s\n", s)

	// 嵌入指针为 nil 时，整组提升字段被跳过（structEncoder 里的 continue FieldLoop）。
	type WithPtrEmbed struct {
		*Base
		Extra string `json:"extra"`
	}
	s, _ = json.Marshal(WithPtrEmbed{Extra: "only"})
	fmt.Printf("nil 嵌入指针      : %s\n", s)
}

// ---------------------------------------------------------------- 五

func mapAndByteSlice() {
	m := map[string]int{"zebra": 1, "apple": 2, "mango": 3, "banana": 4}
	for i := range 3 {
		s, _ := json.Marshal(m)
		fmt.Printf("第 %d 次 Marshal map: %s\n", i+1, s)
	}
	fmt.Println("→ key 按字典序排序，结果确定（代价是一次 key 提取 + 排序）。")

	type blob struct {
		Data  []byte `json:"data"` // 特殊处理：base64 字符串
		Ints  []int  `json:"ints"` // 普通 slice：数字数组
		NilSl []int  `json:"nil"`  // nil slice → null
		Arr   [2]int `json:"arr"`
	}
	s, _ := json.Marshal(blob{Data: []byte("hello"), Ints: []int{1, 2}})
	fmt.Printf("[]byte / slice / nil: %s\n", s)

	var b blob
	if err := json.Unmarshal([]byte(`{"data":"aGVsbG8=","arr":[1,2,3,4]}`), &b); err != nil {
		panic(err)
	}
	fmt.Printf("base64 解码回来: %q；数组多余元素被丢弃: %v\n", b.Data, b.Arr)
}

// ---------------------------------------------------------------- 六

func unmarshalRules() {
	type user struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	// 1. 精确匹配优先，失败后大小写不敏感兜底（v1 行为；v2 默认大小写敏感）。
	for _, data := range []string{`{"name":"a"}`, `{"NAME":"b"}`, `{"Name":"c"}`, `{"nAmE":"d"}`} {
		var u user
		if err := json.Unmarshal([]byte(data), &u); err != nil {
			panic(err)
		}
		fmt.Printf("%-16s → Name=%q\n", data, u.Name)
	}

	// 2. 未知字段默认丢弃；DisallowUnknownFields 只有 Decoder 上有。
	var u user
	fmt.Printf("Unmarshal 未知字段: 静默丢弃, err=%v\n",
		json.Unmarshal([]byte(`{"name":"a","unknown":1}`), &u))

	dec := json.NewDecoder(strings.NewReader(`{"name":"a","unknown":1}`))
	dec.DisallowUnknownFields()
	fmt.Printf("DisallowUnknownFields: %v\n", dec.Decode(&u))

	// 3. 必须传指针，否则 InvalidUnmarshalError（反射需要可 Set 的值）。
	//    这里先塞进 any 变量，否则 go vet 会在编译期就报「passes non-pointer」。
	var notPointer any = user{}
	errVal := json.Unmarshal([]byte(`{}`), notPointer)
	var invalid *json.InvalidUnmarshalError
	fmt.Printf("传值而非指针: %v (是 InvalidUnmarshalError: %v)\n", errVal, errors.As(errVal, &invalid))

	// 4. 多级指针字段会被自动 New 出来（indirect 的行为）。
	var target struct {
		P **int `json:"p"`
	}
	if err := json.Unmarshal([]byte(`{"p":42}`), &target); err != nil {
		panic(err)
	}
	fmt.Printf("**int 自动分配: %d\n", **target.P)

	// 5. 目标是 any 时不走反射，直接解析成 map[string]any / []any / float64 / string / bool / nil。
	var anyV any
	if err := json.Unmarshal([]byte(`{"n":1,"s":"x","b":true,"arr":[1,"2"],"nil":null}`), &anyV); err != nil {
		panic(err)
	}
	m := anyV.(map[string]any)
	for _, k := range []string{"n", "s", "b", "arr", "nil"} {
		fmt.Printf("  any[%q] = %#v (%T)\n", k, m[k], m[k])
	}

	// 6. 类型不匹配返回 UnmarshalTypeError，且不影响其他字段已完成的赋值。
	var u2 user
	errType := json.Unmarshal([]byte(`{"name":"kept","age":"not-a-number"}`), &u2)
	var te *json.UnmarshalTypeError
	if errors.As(errType, &te) {
		fmt.Printf("UnmarshalTypeError: value=%s field=%s type=%s；Name 仍被赋值=%q\n",
			te.Value, te.Field, te.Type, u2.Name)
	}
}

// ---------------------------------------------------------------- 七

func rawMessageAndNumber() {
	// RawMessage 底层就是 []byte，只是自带 Marshal/Unmarshal 方法把字节原样透传，
	// 用来延迟解析：先按公共字段 type 分发，再用对应类型解析 payload。
	type envelope struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	data := []byte(`[
		{"type":"movie","payload":{"name":"功夫","age":"20"}},
		{"type":"level","payload":"high"}
	]`)
	var envs []envelope
	if err := json.Unmarshal(data, &envs); err != nil {
		panic(err)
	}
	for _, e := range envs {
		switch e.Type {
		case "movie":
			var m Movie
			if err := json.Unmarshal(e.Payload, &m); err != nil {
				panic(err)
			}
			fmt.Printf("movie: name=%s age=%d\n", m.Name, m.Age)
		case "level":
			var l Level
			if err := json.Unmarshal(e.Payload, &l); err != nil {
				panic(err)
			}
			fmt.Printf("level: %d\n", l)
		}
	}

	// 默认所有数字解成 float64，大整数会丢精度；UseNumber 保留原始字面量。
	big := []byte(`{"id":12345678901234567890}`)
	var f any
	if err := json.Unmarshal(big, &f); err != nil {
		panic(err)
	}
	fmt.Printf("默认 float64: %v\n", f.(map[string]any)["id"])

	dec := json.NewDecoder(bytes.NewReader(big))
	dec.UseNumber()
	var n any
	if err := dec.Decode(&n); err != nil {
		panic(err)
	}
	num := n.(map[string]any)["id"].(json.Number)
	fmt.Printf("UseNumber   : %s (原始字面量，可再转 Int64/Float64)\n", num)
	if _, err := num.Int64(); err != nil {
		fmt.Printf("  Int64() 溢出: %v\n", err)
	}
}

// ---------------------------------------------------------------- 八

func streaming() {
	// Encoder：写 io.Writer，每个值后面自动加 '\n'（和 Marshal 的明确差异）。
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // 默认会把 < > & 转义成 < 之类
	for _, m := range []Movie{{Name: "a<b>", Age: 1}, {Name: "c&d", Age: 2}} {
		if err := enc.Encode(m); err != nil {
			panic(err)
		}
	}
	fmt.Printf("Encoder 输出（JSON Lines，注意结尾换行）:\n%s", buf.String())

	// Decoder：同一个 Decoder 反复 Decode，配合 More() 依次读多个顶层值。
	dec := json.NewDecoder(strings.NewReader(buf.String()))
	for dec.More() {
		var m Movie
		if err := dec.Decode(&m); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		fmt.Printf("Decode 出一个值: name=%q age=%d\n", m.Name, m.Age)
	}

	// Token()：按 token 遍历，适合处理超大 JSON 而不想反射到具体类型。
	tok := json.NewDecoder(strings.NewReader(`{"a":[1,"two",null]}`))
	fmt.Print("Token 流: ")
	for {
		t, err := tok.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}
		// json.Delim 是 rune 的别名类型，%#v 会打成数字，这里用 %v 走它的 String()
		fmt.Printf("%v(%T) ", t, t)
	}
	fmt.Println()
}

// ---------------------------------------------------------------- 九

type node struct {
	Name string `json:"name"`
	Next *node  `json:"next,omitempty"`
}

func cycleDetect() {
	a := &node{Name: "a"}
	b := &node{Name: "b"}
	a.Next, b.Next = b, a // 成环

	_, err := json.Marshal(a)
	var ue *json.UnsupportedValueError
	fmt.Printf("循环引用: %v\n", err)
	fmt.Printf("是 UnsupportedValueError: %v\n", errors.As(err, &ue))
	fmt.Println("→ ptrLevel 超过 startDetectingCyclesAfter 后才启用 ptrSeen 记录，")
	fmt.Println("  正常浅层结构不付环检测代价。")
}
