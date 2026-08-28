// json 示例：对应 notes/json.md
// 运行：go run ./json
//
// v2 部分（第六节）需要实验开关才存在：
//
//	GOEXPERIMENT=jsonv2 go doc encoding/json/v2
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	tagName()
	tagOmitEmpty()
	tagOmitZero()
	tagString()
	tagEmbedded()

	marshalerPriority()
	marshalSpecialCases()
	marshalMapSorted()
	marshalCycle()

	unmarshalBasics()
	unmarshalIndirect()
	unmarshalFieldMatch()
	unmarshalContainers()

	typeFieldsRules()

	streamDecoder()
	streamEncoder()
	streamToken()
}

// ---------------------------------------------------------------------------
// 一、struct tag —— 字段名
// ---------------------------------------------------------------------------

type Named struct {
	ID       int    // 不写 tag：用 Go 字段名
	UserName string `json:"user_name"` // 自定义 key
	Secret   string `json:"-"`         // 完全不参与编解码
	Dash     string `json:"-,"`        // 唯一例外：字段名字面量就是 "-"
	unexp    string // 未导出字段：反射拿不到，直接忽略
}

func tagName() {
	section("1.1 字段名")

	b, _ := json.Marshal(Named{ID: 1, UserName: "bob", Secret: "s", Dash: "d", unexp: "x"})
	fmt.Println("Marshal:", string(b))

	var n Named
	_ = json.Unmarshal([]byte(`{"ID":9,"user_name":"alice","Secret":"leak","-":"dd"}`), &n)
	fmt.Printf("Unmarshal: %+v（Secret 没被写入，- 命中 Dash）\n", n)
}

// ---------------------------------------------------------------------------
// 1.2 omitempty：empty 只覆盖几种 Kind，对 struct 不生效
// ---------------------------------------------------------------------------

type Inner2 struct{ A int }

type OmitEmpty struct {
	Bool   bool           `json:"bool,omitempty"`
	Int    int            `json:"int,omitempty"`
	Str    string         `json:"str,omitempty"`
	Ptr    *int           `json:"ptr,omitempty"`
	Iface  any            `json:"iface,omitempty"`
	Slice  []int          `json:"slice,omitempty"`
	Map    map[string]int `json:"map,omitempty"`
	Arr    [0]int         `json:"arr,omitempty"`
	Struct Inner2         `json:"struct,omitempty"` // ⚠️ 不生效
	Time   time.Time      `json:"time,omitempty"`   // ⚠️ time.Time 是 struct，也不生效
}

func tagOmitEmpty() {
	section("1.2 omitempty")

	b, _ := json.Marshal(OmitEmpty{})
	fmt.Println("全零值:", string(b))
	fmt.Println("→ 只剩 struct/time：isEmptyValue 只判断 bool/数值/指针/接口/len==0 的容器")
	fmt.Println("  reflect.Struct 落到 default 分支返回 false，所以永远不 empty")

	// 长度为 0 但非 nil 的 slice/map 也算 empty
	b, _ = json.Marshal(OmitEmpty{Slice: []int{}, Map: map[string]int{}})
	fmt.Println("空 slice/map:", string(b))
}

// ---------------------------------------------------------------------------
// 1.3 omitzero（Go 1.24+）：对 struct 也生效，优先用 IsZero()
// ---------------------------------------------------------------------------

// 自定义 IsZero：只要 Amount==0 就算零值，哪怕 Currency 有值
type Money struct {
	Amount   int
	Currency string
}

func (m Money) IsZero() bool { return m.Amount == 0 }

type OmitZero struct {
	Struct Inner2    `json:"struct,omitzero"` // 整个 struct 等于零值就省略
	Time   time.Time `json:"time,omitzero"`   // time.Time 有 IsZero()
	Money  Money     `json:"money,omitzero"`  // 走自定义 IsZero()
	Both   *int      `json:"both,omitempty,omitzero"`
}

func tagOmitZero() {
	section("1.3 omitzero")

	b, _ := json.Marshal(OmitZero{})
	fmt.Println("全零值:", string(b), "→ 全部省略（对比 omitempty）")

	b, _ = json.Marshal(OmitZero{
		Struct: Inner2{A: 1},
		Time:   time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC),
		Money:  Money{Amount: 0, Currency: "CNY"}, // IsZero() 返回 true → 仍被省略
	})
	fmt.Println("部分赋值:", string(b))
	fmt.Println("→ Money 有 Currency 但 IsZero() 为 true，仍被省略：优先调 IsZero()")
}

// ---------------------------------------------------------------------------
// 1.4 string：把值再包一层引号
// ---------------------------------------------------------------------------

type WithString struct {
	ID    int64   `json:"id,string"`
	Score float64 `json:"score,string"`
	OK    bool    `json:"ok,string"`
	Name  string  `json:"name,string"`
	Bad   Inner2  `json:"bad,string"` // 对 struct 无效，静默忽略该选项
}

func tagString() {
	section("1.4 string")

	b, _ := json.Marshal(WithString{ID: 1 << 60, Score: 1.5, OK: true, Name: "n"})
	fmt.Println("Marshal:", string(b))
	fmt.Println("→ 常用于和 JS 交互时避免 int64 精度丢失（JS number 只有 53 位有效位）")

	var w WithString
	// 反序列化时会 destring：先当字符串解出来再二次解析
	_ = json.Unmarshal([]byte(`{"id":"123","score":"2.5","ok":"false","name":"\"q\""}`), &w)
	fmt.Printf("Unmarshal: %+v\n", w)

	err := json.Unmarshal([]byte(`{"id":123}`), &w) // 带 ,string 的字段不接受裸数字
	fmt.Println("裸数字给 ,string 字段:", err)
}

// ---------------------------------------------------------------------------
// 1.5 匿名（嵌入）字段：子字段被提升到外层
// ---------------------------------------------------------------------------

type Meta struct {
	CreatedAt string `json:"created_at"`
	Version   int    `json:"version"`
}

type Embedded struct {
	Meta          // 匿名：字段被提升
	Named2 Meta   `json:"named"` // 具名：多一层嵌套
	*Extra        // 匿名指针：nil 时整段跳过
	Name   string `json:"name"`
}

type Extra struct {
	Tag string `json:"tag"`
}

func tagEmbedded() {
	section("1.5 匿名（嵌入）字段")

	b, _ := json.Marshal(Embedded{Meta: Meta{"2026-08-27", 1}, Name: "x"})
	fmt.Println("嵌入指针为 nil:", string(b))
	fmt.Println("→ structEncoder 里 fv.IsNil() 时 continue FieldLoop，整个字段跳过")

	b, _ = json.Marshal(Embedded{Meta: Meta{"2026-08-27", 1}, Extra: &Extra{"t"}, Name: "x"})
	fmt.Println("嵌入指针非 nil:", string(b))
}

// ---------------------------------------------------------------------------
// 二、Marshal —— encoder 选择优先级：Marshaler > TextMarshaler > 默认规则
// ---------------------------------------------------------------------------

type OnlyText struct{ V string }

func (t OnlyText) MarshalText() ([]byte, error) { return []byte("text:" + t.V), nil }

type BothImpl struct{ V string }

func (b BothImpl) MarshalJSON() ([]byte, error) { return []byte(`"json:` + b.V + `"`), nil }
func (b BothImpl) MarshalText() ([]byte, error) { return []byte("text:" + b.V), nil }

// 指针接收者：值本身不实现 Marshaler，只有 *PtrOnly 实现
type PtrOnly struct{ V string }

func (p *PtrOnly) MarshalJSON() ([]byte, error) { return []byte(`"ptr:` + p.V + `"`), nil }

func marshalerPriority() {
	section("2.1 encoder 选择优先级")

	b, _ := json.Marshal(OnlyText{"a"})
	fmt.Println("只有 MarshalText:", string(b), "→ 结果被当作 JSON 字符串（额外加引号转义）")

	b, _ = json.Marshal(BothImpl{"a"})
	fmt.Println("两个都有:        ", string(b), "→ Marshaler 优先")

	// 值可寻址时走 condAddrEncoder → 地址版本的 encoder
	b, _ = json.Marshal(struct{ P PtrOnly }{PtrOnly{"a"}})
	fmt.Println("struct 字段可寻址:", string(b), "→ condAddrEncoder 走指针版本")

	// map 的 value 不可寻址，退回普通类型 encoder
	b, _ = json.Marshal(map[string]PtrOnly{"k": {"a"}})
	fmt.Println("map value 不可寻址:", string(b), "→ 退回默认 struct encoder")
}

// ---------------------------------------------------------------------------
// 2.2 特殊情况
// ---------------------------------------------------------------------------

func marshalSpecialCases() {
	section("2.2 Marshal 特殊情况")

	b, _ := json.Marshal([]byte("hello"))
	fmt.Println("[]byte:      ", string(b), "→ 走 encodeByteSlice，Base64 后当字符串")

	b, _ = json.Marshal([]int8{1, 2})
	fmt.Println("[]int8:      ", string(b), "→ 只有 []byte/[]uint8 特殊，其他数值切片是普通数组")

	type Nils struct {
		Ptr   *int           `json:"ptr"`
		Iface any            `json:"iface"`
		Map   map[string]int `json:"map"`
		Slice []int          `json:"slice"`
	}
	b, _ = json.Marshal(Nils{})
	fmt.Println("各种 nil:    ", string(b), "→ 统一编码成 null")

	// HTML 转义：Marshal 默认开启
	b, _ = json.Marshal("<a href=\"x\">&")
	fmt.Println("HTML 转义:   ", string(b))

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode("<a href=\"x\">&")
	fmt.Print("关掉转义:     ", buf.String())

	// 不支持的类型直接 error（内部 panic 被 recover 转成 error）
	_, err := json.Marshal(map[string]any{"f": func() {}})
	fmt.Println("func 字段:   ", err)
	_, err = json.Marshal(map[string]any{"c": make(chan int)})
	fmt.Println("chan 字段:   ", err)
}

// ---------------------------------------------------------------------------
// 2.3 map 的 key 会被排序
// ---------------------------------------------------------------------------

type Key struct{ N int }

func (k Key) MarshalText() ([]byte, error) { return []byte(fmt.Sprintf("k%02d", k.N)), nil }

func marshalMapSorted() {
	section("2.3 map key 排序")

	m := map[string]int{"z": 1, "a": 2, "m": 3}
	for range 3 { // 多跑几次，结果始终一致
		b, _ := json.Marshal(m)
		fmt.Println("string key:", string(b))
	}

	// 整数 key 会被格式化成字符串，再按字符串排序（注意不是数值序）
	b, _ := json.Marshal(map[int]string{1: "a", 2: "b", 10: "c", 20: "d"})
	fmt.Println("int key:   ", string(b), "→ 字符串序：\"1\" < \"10\" < \"2\" < \"20\"")

	// 实现了 TextMarshaler 的 key 调 MarshalText
	b, _ = json.Marshal(map[Key]int{{3}: 3, {1}: 1})
	fmt.Println("TextMarshaler key:", string(b))

	fmt.Println("→ 多了一次 key 提取 + 排序，所以 map 序列化比 struct 慢")
}

// ---------------------------------------------------------------------------
// 2.4 循环引用检测
// ---------------------------------------------------------------------------

type Node struct {
	Name string `json:"name"`
	Next *Node  `json:"next,omitempty"`
}

func marshalCycle() {
	section("2.4 循环引用检测")

	a := &Node{Name: "a"}
	b := &Node{Name: "b"}
	a.Next, b.Next = b, a

	_, err := json.Marshal(a)
	fmt.Println("err:", err)

	var uv *json.UnsupportedValueError
	fmt.Println("errors.As(*UnsupportedValueError):", errors.As(err, &uv))
	fmt.Println("→ 嵌套超过 startDetectingCyclesAfter(1000) 层后才启用 ptrSeen 记录，避免常规路径付代价")
}

// ---------------------------------------------------------------------------
// 三、Unmarshal 基础
// ---------------------------------------------------------------------------

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func unmarshalBasics() {
	section("3.1 Unmarshal 基础")

	var u User
	var notPtr any = u // 绕开 go vet 的静态检查，演示运行时错误
	err := json.Unmarshal([]byte(`{"name":"bob","age":3}`), notPtr)
	fmt.Println("传非指针:", err)

	var pnil *User
	err = json.Unmarshal([]byte(`{}`), pnil)
	fmt.Println("传 nil 指针:", err)

	err = json.Unmarshal([]byte(`{"name":"bob","age":"3"}`), &u) // 类型不匹配
	fmt.Println("类型不匹配:", err)

	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		fmt.Printf("  UnmarshalTypeError: Value=%q Type=%v Field=%q Offset=%d\n",
			ute.Value, ute.Type, ute.Field, ute.Offset)
	}
	fmt.Printf("  部分字段仍被写入：%+v（Unmarshal 不保证原子性）\n", u)

	err = json.Unmarshal([]byte(`{"name":`), &u)
	fmt.Println("语法错误:", err)

	// 解到 any：不走反射，直接 objectInterface() 递归成 map[string]any
	var v any
	_ = json.Unmarshal([]byte(`{"a":1,"b":[true,null,"s"]}`), &v)
	fmt.Printf("解到 any: %#v\n", v)
	fmt.Println("→ 数字默认解成 float64；用 Decoder.UseNumber() 可以保留 json.Number")
}

// ---------------------------------------------------------------------------
// 3.2 indirect：指针链自动分配 + Unmarshaler 优先级
// ---------------------------------------------------------------------------

type Deep struct {
	P   *int    `json:"p"`
	PP  **int   `json:"pp"`
	PPP ***User `json:"ppp"`
}

// 实现 Unmarshaler：拿到原始 JSON 片段，后续字段不再走反射赋值
type Upper struct{ V string }

func (u *Upper) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	u.V = strings.ToUpper(s)
	return nil
}

// TextUnmarshaler 优先级低于 Unmarshaler
type Dur struct{ D time.Duration }

func (d *Dur) UnmarshalText(text []byte) error {
	v, err := time.ParseDuration(string(text))
	d.D = v
	return err
}

func unmarshalIndirect() {
	section("3.2 indirect：指针链与 Unmarshaler 优先级")

	var d Deep
	_ = json.Unmarshal([]byte(`{"p":1,"pp":2,"ppp":{"name":"bob"}}`), &d)
	fmt.Printf("多级指针自动分配: p=%d pp=%d ppp=%+v\n", *d.P, **d.PP, ***d.PPP)

	// null 遇到第一个可设置的指针就停下来置 nil
	d2 := Deep{P: new(int)}
	_ = json.Unmarshal([]byte(`{"p":null}`), &d2)
	fmt.Println("null 置空指针: p == nil ?", d2.P == nil)

	var u Upper
	_ = json.Unmarshal([]byte(`"hello"`), &u)
	fmt.Printf("Unmarshaler:     %+v\n", u)

	var dur Dur
	_ = json.Unmarshal([]byte(`"1h30m"`), &dur)
	fmt.Println("TextUnmarshaler:", dur.D)

	fmt.Println("→ 优先级 Unmarshaler > TextUnmarshaler > 反射默认规则，和 Marshal 对称")
}

// ---------------------------------------------------------------------------
// 3.3 字段匹配规则：精确优先，大小写不敏感兜底
// ---------------------------------------------------------------------------

func unmarshalFieldMatch() {
	section("3.3 字段匹配规则")

	for _, data := range []string{
		`{"name":"exact"}`,
		`{"Name":"different case"}`,
		`{"NAME":"upper"}`,
		`{"nAmE":"mixed"}`,
	} {
		var u User
		_ = json.Unmarshal([]byte(data), &u)
		fmt.Printf("%-26s -> %q\n", data, u.Name)
	}
	fmt.Println("→ byExactName 精确匹配（大小写敏感）失败后，退化到 byFoldedName（ASCII fold）")

	// 未知字段：默认丢弃
	var u User
	_ = json.Unmarshal([]byte(`{"name":"bob","unknown":1}`), &u)
	fmt.Printf("默认忽略未知字段: %+v\n", u)

	dec := json.NewDecoder(strings.NewReader(`{"name":"bob","unknown":1}`))
	dec.DisallowUnknownFields()
	fmt.Println("DisallowUnknownFields:", dec.Decode(&u))
}

// ---------------------------------------------------------------------------
// 3.4 容器：slice / array / map 的长度与复用行为
// ---------------------------------------------------------------------------

func unmarshalContainers() {
	section("3.4 slice / array / map")

	s := []int{9, 9, 9, 9, 9}
	_ = json.Unmarshal([]byte(`[1,2]`), &s)
	fmt.Println("slice 变长（复用底层数组，长度被截断）:", s, "cap =", cap(s))

	var arr [4]int
	_ = json.Unmarshal([]byte(`[1,2]`), &arr)
	fmt.Println("array 元素不足，剩余保留零值:", arr)

	arr = [4]int{}
	_ = json.Unmarshal([]byte(`[1,2,3,4,5,6]`), &arr)
	fmt.Println("array 元素过多，多余静默丢弃:", arr)

	// map 是合并而不是覆盖
	m := map[string]int{"keep": 1, "over": 1}
	_ = json.Unmarshal([]byte(`{"over":2,"new":3}`), &m)
	fmt.Println("map 合并语义:", m)

	// struct 同理：JSON 里没有的字段保留原值
	u := User{Name: "old", Age: 30}
	_ = json.Unmarshal([]byte(`{"name":"new"}`), &u)
	fmt.Printf("struct 保留原值: %+v\n", u)

	// 非 string/整数/TextUnmarshaler 的 map key 不支持
	var bad map[float64]int
	fmt.Println("float64 作 map key:", json.Unmarshal([]byte(`{"1.5":1}`), &bad))

	// UseNumber：避免大整数被 float64 精度截断
	var v1, v2 any
	raw := `{"n":12345678901234567890}`
	_ = json.Unmarshal([]byte(raw), &v1)
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	_ = dec.Decode(&v2)
	fmt.Printf("默认 float64: %v\nUseNumber:    %v\n",
		v1.(map[string]any)["n"], v2.(map[string]any)["n"])
}

// ---------------------------------------------------------------------------
// 四、typeFields 字段解析规则（Marshal / Unmarshal 共用）
// ---------------------------------------------------------------------------

type L1 struct {
	Name string `json:"name"`
	Only string `json:"only"`
}

// 规则 1：BFS 逐层展开，浅层字段遮蔽深层同名字段
type ShallowWins struct {
	L1
	Name string `json:"name"` // 外层，赢
}

type NameA struct{ Name string }
type NameB struct{ Name string }

// 规则 3-b：同层冲突且 tag 情况相同 → 全部丢弃
type BothDropped struct {
	NameA
	NameB
	Keep string
}

type NameTagged struct {
	Name string `json:"Name"`
}

// 规则 3-a：同层冲突但只有一个显式写了 tag → tag 赢
type TagWins struct {
	NameA
	NameTagged
}

// 规则 2：未导出的匿名非 struct 字段被忽略
type myInt int

type IgnoreUnexported struct {
	myInt // 未导出的匿名非 struct 字段
	V     int
}

func typeFieldsRules() {
	section("4 typeFields 规则")

	b, _ := json.Marshal(ShallowWins{L1: L1{Name: "deep", Only: "o"}, Name: "shallow"})
	fmt.Println("浅层优先:      ", string(b))

	b, _ = json.Marshal(BothDropped{NameA{"a"}, NameB{"b"}, "keep"})
	fmt.Println("同层冲突全丢弃:", string(b), "→ Go 的'二义性即不存在'哲学")

	var bd BothDropped
	_ = json.Unmarshal([]byte(`{"Name":"x","Keep":"y"}`), &bd)
	fmt.Printf("反序列化同理:   %+v\n", bd)

	b, _ = json.Marshal(TagWins{NameA{"untagged"}, NameTagged{"tagged"}})
	fmt.Println("有 tag 的赢:   ", string(b))

	b, _ = json.Marshal(IgnoreUnexported{myInt: 1, V: 2})
	fmt.Println("忽略未导出匿名:", string(b))

	fmt.Println("→ 解析结果（转义后的字段名 / 选项 / index 路径 / encoderFunc）缓存在 cachedTypeFields")
}

// ---------------------------------------------------------------------------
// 五、流式编解码
// ---------------------------------------------------------------------------

func streamDecoder() {
	section("5.1 Decoder：连续多个 JSON 值")

	// JSON Lines：一个 Decoder 反复 Decode
	r := strings.NewReader(`{"name":"a","age":1}
{"name":"b","age":2}
{"name":"c","age":3}`)
	dec := json.NewDecoder(r)
	for {
		var u User
		if err := dec.Decode(&u); err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("err:", err)
			break
		}
		fmt.Printf("  decode: %+v (offset=%d)\n", u, dec.InputOffset())
	}

	// More()：数组元素逐个流式处理，不用一次性反射整个数组
	dec = json.NewDecoder(strings.NewReader(`[{"name":"x"},{"name":"y"}]`))
	_, _ = dec.Token() // 读掉 '['
	for dec.More() {
		var u User
		_ = dec.Decode(&u)
		fmt.Println("  array item:", u.Name)
	}
	_, _ = dec.Token() // 读掉 ']'
}

func streamEncoder() {
	section("5.2 Encoder：自动补换行 + SetIndent")

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, u := range []User{{"a", 1}, {"b", 2}} {
		_ = enc.Encode(u) // 每次 Encode 末尾追加 '\n'
	}
	fmt.Printf("Encode 输出（%d 字节，含换行）:\n%s", buf.Len(), buf.String())

	b, _ := json.Marshal(User{"a", 1})
	fmt.Printf("对比 Marshal（%d 字节，无换行）: %s\n", len(b), b)

	enc = json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	fmt.Println("SetIndent:")
	_ = enc.Encode(User{"a", 1})

	b, _ = json.MarshalIndent(User{"a", 1}, "", "  ")
	fmt.Println("MarshalIndent 等价物长度:", len(b))
}

func streamToken() {
	section("5.3 Token：不反射到具体类型，逐 token 遍历")

	dec := json.NewDecoder(strings.NewReader(`{"a":1,"b":[true,"s"]}`))
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("err:", err)
			break
		}
		fmt.Printf("  %-6T %v (offset=%d)\n", t, t, dec.InputOffset())
	}
	fmt.Println("→ 适合流式处理超大 JSON；json.RawMessage 则用于延迟解析某个子树")

	// RawMessage：先只解外层，内层原样留着
	var envelope struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal([]byte(`{"type":"user","data":{"name":"bob","age":3}}`), &envelope)
	fmt.Println("RawMessage 延迟解析:", envelope.Type, string(envelope.Data))
	var u User
	_ = json.Unmarshal(envelope.Data, &u)
	fmt.Printf("  再按 type 解析: %+v\n", u)
}
