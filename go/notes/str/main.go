// string 示例：对应 notes/string.md
// 运行：go run ./str
// 看零拷贝优化：go build -gcflags=-m ./str 2>&1 | grep -i 'does not escape\|escapes'
// 压测：go test -bench . -benchmem ./str
package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
	"unique"
	"unsafe"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

// note 打印含 % 动词的说明文字（直接给 fmt.Println 会被 go vet 判为疑似格式化指令）
func note(s string) { fmt.Println(s) }

func main() {
	basicLayout()
	basicImmutable()
	basicBytesRunes()
	basicRangeSemantics()
	basicUTF8()
	basicBuilder()
	basicNewAPIs()

	convCopy()
	convZeroCopy()
	convUnsafe()
	internDemo()

	trapIndexByte()
	trapLenNotCount()
	trapUpperLower()
	trapSubstringMemory()
	trapCompare()
	trapNumberConv()
}

// ---------------------------------------------------------------------------
// 1.1 内存布局
// ---------------------------------------------------------------------------

func basicLayout() {
	section("1.1 内存布局")

	s := "hello"
	fmt.Printf("  unsafe.Sizeof(string) = %d（两个字：data 指针 + len）\n", unsafe.Sizeof(s))
	fmt.Println("  runtime.stringStruct{ str unsafe.Pointer; len int }")

	// 用 unsafe 看内部结构
	type stringHeader struct {
		data unsafe.Pointer
		len  int
	}
	h := (*stringHeader)(unsafe.Pointer(&s))
	fmt.Printf("  \"hello\" -> data=%p len=%d\n", h.data, h.len)

	// 和 slice 的区别：string 没有 cap
	b := []byte("hello")
	fmt.Printf("  unsafe.Sizeof([]byte) = %d（三个字：data + len + cap）\n", unsafe.Sizeof(b))

	fmt.Println("→ 字符串字面量存在只读段（.rodata），多个相同字面量会被链接器合并")
	s1, s2 := "abc", "abc"
	h1 := (*stringHeader)(unsafe.Pointer(&s1))
	h2 := (*stringHeader)(unsafe.Pointer(&s2))
	fmt.Printf("  两个 \"abc\" 字面量的 data 指针相同？%v\n", h1.data == h2.data)
}

// ---------------------------------------------------------------------------
// 1.2 不可变
// ---------------------------------------------------------------------------

func basicImmutable() {
	section("1.2 不可变")

	s := "hello"
	// s[0] = 'H'  // ✗ cannot assign to s[0] (neither addressable nor a map index expression)
	fmt.Println("  s[0] = 'H' 是编译错误：字符串不可变")
	fmt.Println("  改字符串的唯一办法：转成 []byte 改完再转回来（两次拷贝）")
	b := []byte(s)
	b[0] = 'H'
	fmt.Println("  ", string(b))

	fmt.Println("→ 不可变带来的三个好处：")
	fmt.Println("  ① 可以安全共享底层数组（切片操作 s[1:3] 是零拷贝的！）")
	fmt.Println("  ② 可以做 map key（哈希值稳定）")
	fmt.Println("  ③ 并发读天然安全，不需要锁")
	fmt.Println("→ 代价：任何修改都要重新分配；拼接在循环里就是 O(n²)（见 bench）")

	sub := s[1:3]
	h := func(x string) unsafe.Pointer {
		return unsafe.Pointer(unsafe.StringData(x))
	}
	fmt.Printf("  s[1:3] 零拷贝验证: &s[1]=%p, &sub[0]=%p, 相同=%v\n",
		unsafe.Add(h(s), 1), h(sub), unsafe.Add(h(s), 1) == h(sub))
}

// ---------------------------------------------------------------------------
// 1.3 string / []byte / []rune 三者关系
// ---------------------------------------------------------------------------

func basicBytesRunes() {
	section("1.3 string / []byte / []rune")

	s := "Go语言"
	fmt.Printf("  s          = %q\n", s)
	fmt.Printf("  len(s)     = %d（字节数！不是字符数）\n", len(s))
	fmt.Printf("  []byte(s)  = %v\n", []byte(s))
	fmt.Printf("  []rune(s)  = %v（每个 rune 是一个 Unicode 码点）\n", []rune(s))
	fmt.Printf("  utf8.RuneCountInString = %d\n", utf8.RuneCountInString(s))

	fmt.Println("  s[0] 的类型是 byte:", s[0], string(rune(s[0])))
	fmt.Println("  s[2] 是'语'的第一个字节，单独拿出来是乱码:", s[2])

	fmt.Println("→ string ↔ []byte：都要拷贝（除了下面 2.2 的编译器特例）")
	fmt.Println("→ string ↔ []rune：一定拷贝且要解码/编码 UTF-8，比 []byte 贵得多")
}

// ---------------------------------------------------------------------------
// 1.4 range 的语义
// ---------------------------------------------------------------------------

func basicRangeSemantics() {
	section("1.4 range string 的语义")

	s := "Aé中"
	fmt.Println("  for i, r := range s  —— i 是字节下标，r 是 rune：")
	for i, r := range s {
		fmt.Printf("    i=%d r=%q (U+%04X) 占 %d 字节\n", i, r, r, utf8.RuneLen(r))
	}

	fmt.Println("  for i := 0; i < len(s); i++ —— 按字节：")
	for i := 0; i < len(s); i++ {
		fmt.Printf("    i=%d b=%d(0x%02X)\n", i, s[i], s[i])
	}

	fmt.Println("→ range 自动解码 UTF-8；非法字节序列会得到 U+FFFD（RuneError）且 i 前进 1")
	bad := string([]byte{0xff, 'a'})
	for i, r := range bad {
		fmt.Printf("    非法序列: i=%d r=%q isRuneError=%v\n", i, r, r == utf8.RuneError)
	}
}

// ---------------------------------------------------------------------------
// 1.5 UTF-8 与 unicode
// ---------------------------------------------------------------------------

func basicUTF8() {
	section("1.5 UTF-8")

	for _, s := range []string{"A", "é", "中", "🎉"} {
		r, size := utf8.DecodeRuneInString(s)
		fmt.Printf("  %-4s U+%04X 占 %d 字节 %v\n", s, r, size, []byte(s))
	}

	fmt.Println("  UTF-8 编码规则：1 字节 ASCII / 2 字节 拉丁扩展 / 3 字节 中日韩 / 4 字节 emoji")
	fmt.Println("  自同步性：任何字节的高位就能看出它是首字节(0xxx/110x/1110/11110)还是续字节(10xx)")

	fmt.Println("  utf8.ValidString(\"\\xff\") =", utf8.ValidString("\xff"))
	fmt.Println("  strings.ToValidUTF8 替换非法字节:", strconv.Quote(strings.ToValidUTF8("a\xffb", "?")))

	// 组合字符：一个"字符"可能是多个 rune
	e1, e2 := "é", "e\u0301"
	fmt.Printf("  %q(%d rune) vs %q(%d rune)：看起来一样，但不相等 %v\n",
		e1, utf8.RuneCountInString(e1), e2, utf8.RuneCountInString(e2), e1 == e2)
	fmt.Println("  → 真正的'字符'（grapheme cluster）需要 golang.org/x/text 处理")
}

// ---------------------------------------------------------------------------
// 1.6 strings.Builder
// ---------------------------------------------------------------------------

func basicBuilder() {
	section("1.6 strings.Builder")

	var sb strings.Builder
	sb.Grow(64) // 预分配，避免多次扩容
	for i := range 5 {
		fmt.Fprintf(&sb, "%d,", i) // Builder 实现了 io.Writer
	}
	sb.WriteString("done")
	sb.WriteByte('!')
	sb.WriteRune('中')
	fmt.Println("  结果:", sb.String())
	fmt.Printf("  Len=%d Cap=%d\n", sb.Len(), sb.Cap())

	fmt.Println("→ Builder 的 String() 是零拷贝的（unsafe.String 直接指向内部 buf）")
	fmt.Println("→ 所以 String() 之后再 Write 会先复制一份 buf（copyCheck），避免改到已返回的字符串")
	fmt.Println("→ Builder 不能拷贝：内部有 addr 字段做自引用检查，拷贝后 Write 会 panic")

	func() {
		defer func() { fmt.Println("  拷贝后写入 -> panic:", recover()) }()
		var a strings.Builder
		a.WriteString("x")
		b := a // 拷贝
		b.WriteString("y")
	}()
}

// ---------------------------------------------------------------------------
// 1.7 新 API（1.18 ~ 1.24）
// ---------------------------------------------------------------------------

func basicNewAPIs() {
	section("1.7 值得知道的新 API")

	before, after, found := strings.Cut("key=value", "=")
	fmt.Printf("  Cut(1.18):        before=%q after=%q found=%v\n", before, after, found)

	rest, ok := strings.CutPrefix("prefix-body", "prefix-")
	fmt.Printf("  CutPrefix(1.20):  rest=%q ok=%v\n", rest, ok)

	fmt.Printf("  Clone(1.18):      %q（切断和大字符串的共享，见 3.4）\n", strings.Clone("abc"))
	fmt.Printf("  ContainsFunc(1.21): %v\n", strings.ContainsFunc("abc1", unicode.IsDigit))

	// 1.24 的迭代器版本：不产生中间 slice
	fmt.Print("  SplitSeq(1.24):   ")
	for part := range strings.SplitSeq("a,b,c", ",") {
		fmt.Print(part, " ")
	}
	fmt.Println()

	fmt.Print("  Lines(1.24):      ")
	for line := range strings.Lines("l1\nl2\n") {
		fmt.Printf("%q ", line)
	}
	fmt.Println()

	fmt.Println("  FieldsSeq / FieldsFuncSeq / SplitAfterSeq 同理（见 iter.md）")
}

// ---------------------------------------------------------------------------
// 2.1 转换要不要拷贝
// ---------------------------------------------------------------------------

func convCopy() {
	section("2.1 转换的拷贝成本")

	fmt.Println("  一般情况都要拷贝（因为 string 不可变，而 []byte 可变）：")
	fmt.Println("    []byte(s)   -> runtime.stringtoslicebyte  分配 + memmove")
	fmt.Println("    string(b)   -> runtime.slicebytetostring  分配 + memmove")
	fmt.Println("    []rune(s)   -> runtime.stringtoslicerune  分配 + 逐个解码")
	fmt.Println("    string(rs)  -> runtime.slicerunetostring  分配 + 逐个编码")
	fmt.Println("  小字符串（≤32 字节）有栈上 tmpBuf 优化，不逃逸时可以零分配")
}

// ---------------------------------------------------------------------------
// 2.2 编译器的零拷贝特例（runtime/string.go:194 注释里列了三条）
// ---------------------------------------------------------------------------

var lookupMap = map[string]int{"key": 1}

func convZeroCopy() {
	section("2.2 编译器的零拷贝特例")

	b := []byte("key")

	fmt.Println("  ① m[string(b)]           —— map 查找，不分配")
	fmt.Println("     lookupMap[string(b)] =", lookupMap[string(b)])

	fmt.Println("  ② string(b) == \"literal\" —— 比较，不分配")
	fmt.Println("     string(b) == \"key\" =", string(b) == "key")

	fmt.Println("  ③ \"<\" + string(b) + \">\" —— 拼接，不为中间结果分配")
	fmt.Println("     结果 =", "<"+string(b)+">")

	fmt.Println("  以上三条就是 runtime/string.go:194 slicebytetostringtmp 注释里列出的全部场景")
	fmt.Println()
	fmt.Println("  ⚠️ 实测纠正一个流传很广的说法：for range string(b) **会**拷贝")
	fmt.Print("     ")
	for _, r := range string(b) {
		fmt.Printf("%c", r)
	}
	fmt.Println()
	fmt.Println("     BenchmarkZeroCopyRange 显示 12288 B/op 1 allocs/op（12KB 的 []byte）")
	fmt.Println("     要零拷贝地按 rune 遍历 []byte，用 utf8.DecodeRune 手写循环，或者 unsafe.String")
	fmt.Println()
	fmt.Println("→ 前提：那个'临时字符串'不能逃逸出表达式，否则退化为真拷贝")
	fmt.Println("→ 另外小字符串（≤32 字节）不逃逸时会用栈上 tmpBuf，也是 0 allocs，")
	fmt.Println("  但仍然做了一次 memmove（实测 7.8ns vs 直接比较的 1.1ns）")
	fmt.Println("→ 验证：go test -bench Zero -benchmem ./str")
}

// ---------------------------------------------------------------------------
// 2.3 unsafe 零拷贝转换（1.20 之后的正确写法）
// ---------------------------------------------------------------------------

// 只读场景：把 []byte 当 string 用，不拷贝
func bytesToStringUnsafe(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// 危险：返回的 []byte 指向字符串的只读内存，写入会 SIGSEGV
func stringToBytesUnsafe(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func convUnsafe() {
	section("2.3 unsafe 零拷贝转换")

	b := []byte("hello")
	s := bytesToStringUnsafe(b)
	fmt.Printf("  unsafe.String: %q（和 b 共享内存）\n", s)
	b[0] = 'H'
	fmt.Printf("  改了 b[0] 之后，s 变成了 %q ← 字符串被'改'了！\n", s)

	fmt.Println("  1.20 之前的老写法（现在别用）：")
	fmt.Println("    *(*string)(unsafe.Pointer(&b))                  // 依赖 header 布局")
	fmt.Println("    (*[1<<30]byte)(unsafe.Pointer(&s))[:len(s):len(s)]")
	fmt.Println("  1.20+ 的正确写法：")
	fmt.Println("    unsafe.String(unsafe.SliceData(b), len(b))      // []byte -> string")
	fmt.Println("    unsafe.Slice(unsafe.StringData(s), len(s))      // string -> []byte（危险）")

	fmt.Println("→ 使用条件：转换后**绝不修改**原 []byte，且生命周期覆盖字符串的使用期")
	fmt.Println("→ string -> []byte 更危险：字面量在只读段，写入直接段错误（不是 panic，是进程崩）")
	fmt.Println("→ 什么时候值得用：热路径上 KB 级以上的转换（比如 HTTP body 解析）")
	_ = stringToBytesUnsafe
}

// ---------------------------------------------------------------------------
// 2.4 unique.Make：字符串驻留（1.23+）
// ---------------------------------------------------------------------------

func internDemo() {
	section("2.4 字符串驻留（interning）")

	fmt.Println("  场景：解析大量重复的字符串（日志 tag、JSON key、维度值），")
	fmt.Println("       同一个内容存了几万份，浪费内存且 GC 要扫更多对象。")
	fmt.Println()
	fmt.Println("  ① 自己用 map 做池（最常见）：")
	fmt.Println("     var pool = map[string]string{}")
	fmt.Println("     func intern(s string) string { if v, ok := pool[s]; ok { return v }; pool[s]=s; return s }")
	fmt.Println("     缺点：永不释放，得自己管生命周期 + 并发要加锁")
	fmt.Println()
	fmt.Println("  ② unique.Make[T]（1.23+，底层是 HashTrieMap + 弱引用）：")
	fmt.Println("     h := unique.Make(\"some-tag\")   // 返回 Handle[string]，可比较")
	fmt.Println("     h.Value()                       // 拿回字符串")
	fmt.Println("     优点：Handle 可以直接 == 比较（一次指针比较，不比字符串内容）；")
	fmt.Println("           没人引用的条目会被 GC 自动清理（用了 weak pointer）")
	// 实测
	h1 := unique.Make(strings.Repeat("tag", 3))
	h2 := unique.Make(strings.Repeat("tag", 3))
	fmt.Printf("\n  实测: h1 == h2 ? %v（两次 Make 同样内容 -> 同一个 Handle）\n", h1 == h2)
	fmt.Printf("        h1.Value() = %q\n", h1.Value())
	fmt.Printf("        Sizeof(Handle[string]) = %d（就一个指针，比较是指针比较）\n",
		unsafe.Sizeof(h1))
	fmt.Println("  → sync.Map 换成 hash-trie 的原始动机就是给 unique 包用（见 sync.md 2.9）")
}

// ---------------------------------------------------------------------------
// 3.1 按字节索引 vs 按字符
// ---------------------------------------------------------------------------

func trapIndexByte() {
	section("3.1 s[i] 是字节，不是字符")

	s := "中文abc"
	fmt.Printf("  s = %q, len = %d\n", s, len(s))
	fmt.Printf("  s[0] = %d，string(s[0]) = %q ← 乱码\n", s[0], string(s[0]))
	fmt.Printf("  正确取第一个字符: %q\n", string([]rune(s)[0]))

	r, size := utf8.DecodeRuneInString(s)
	fmt.Printf("  更省的写法（不建整个 []rune）: %q，占 %d 字节\n", r, size)

	fmt.Println("  截取前 N 个字符也是同理：")
	fmt.Printf("    s[:3]           = %q（碰巧对，因为'中'正好 3 字节）\n", s[:3])
	fmt.Printf("    s[:2]           = %q ← 截断了一个 rune\n", s[:2])
	fmt.Printf("    string([]rune(s)[:2]) = %q ← 正确\n", string([]rune(s)[:2]))
}

// ---------------------------------------------------------------------------
// 3.2 len 不是字符数
// ---------------------------------------------------------------------------

func trapLenNotCount() {
	section("3.2 len(s) 不是字符数")

	for _, s := range []string{"abc", "中文", "🎉🎊", "e\u0301"} {
		fmt.Printf("  %-8q len=%d RuneCount=%d\n", s, len(s), utf8.RuneCountInString(s))
	}
	fmt.Println("→ 校验'昵称不超过 10 个字'要用 utf8.RuneCountInString，不是 len")
	fmt.Println("→ 但存储限制（数据库字段、协议长度）用的是字节数，两者要分清")
}

// ---------------------------------------------------------------------------
// 3.3 大小写转换不是 ASCII 那么简单
// ---------------------------------------------------------------------------

func trapUpperLower() {
	section("3.3 大小写与比较")

	fmt.Println("  strings.ToUpper(\"straße\") =", strings.ToUpper("straße"))
	fmt.Println("  土耳其语 i 的特殊规则需要 ToUpperSpecial:")
	fmt.Println("    ToUpper(\"i\")                        =", strings.ToUpper("i"))
	fmt.Println("    ToUpperSpecial(TurkishCase, \"i\")    =", strings.ToUpperSpecial(unicode.TurkishCase, "i"))

	fmt.Println("  忽略大小写比较：")
	fmt.Println("    ✗ strings.ToLower(a) == strings.ToLower(b)  两次分配")
	fmt.Println("    ✓ strings.EqualFold(a, b)                   零分配，且做 Unicode case folding")
	fmt.Println("      EqualFold(\"Go\", \"GO\") =", strings.EqualFold("Go", "GO"))
	fmt.Println("  ⚠️ 安全相关的比较（token、签名）要用 crypto/subtle.ConstantTimeCompare")
}

// ---------------------------------------------------------------------------
// 3.4 子串持有整个大字符串
// ---------------------------------------------------------------------------

func trapSubstringMemory() {
	section("3.4 子串会持有整个底层数组")

	big := strings.Repeat("x", 10<<20) // 10MB
	small := big[:10]                  // 零拷贝，但引用着那 10MB

	fmt.Printf("  big 10MB, small = big[:10] len=%d\n", len(small))
	fmt.Printf("  两者 data 指针相同: %v ← small 活着，10MB 就回收不了\n",
		unsafe.StringData(big) == unsafe.StringData(small))

	cloned := strings.Clone(small)
	fmt.Printf("  strings.Clone 之后指针不同: %v ← 断开引用\n",
		unsafe.StringData(cloned) != unsafe.StringData(big))

	fmt.Println("→ 和 slice 的同一个坑（slice.md 3.3）。长期保存的子串一律 strings.Clone")
	fmt.Println("→ 典型场景：解析大 JSON/日志行后只留几个字段，结果整个 buffer 泄漏")
}

// ---------------------------------------------------------------------------
// 3.5 字符串比较与拼接
// ---------------------------------------------------------------------------

func trapCompare() {
	section("3.5 比较与拼接")

	fmt.Println("  == 比较：先比长度和指针（相同直接 true），再 memequal —— 短路很快")
	fmt.Println("  < > 比较：字典序，按字节（不是按 Unicode 码点排序，但 UTF-8 保证两者一致）")

	fmt.Println("  拼接的四种写法（见 bench 实测）：")
	fmt.Println("    a + b              少量拼接最快（编译器 concatstrings，一次分配）")
	fmt.Println("    循环里 s += x      O(n²)，每次都重新分配整个字符串")
	fmt.Println("    strings.Builder    循环拼接的标准答案")
	fmt.Println("    strings.Join       已有 slice 时最省（一次算总长，一次分配）")
	fmt.Println("    []byte + append    需要 append 语义时用；最后 string(buf) 一次拷贝")
}

// ---------------------------------------------------------------------------
// 3.6 数字与字符串转换
// ---------------------------------------------------------------------------

func trapNumberConv() {
	section("3.6 数字转换的坑")

	fmt.Println("  string(65) —— Go 1.15 起是 vet 错误（想要 \"65\" 却得到 \"A\"）")
	fmt.Println("    string(rune(65)) =", string(rune(65)), "  strconv.Itoa(65) =", strconv.Itoa(65))

	note("  性能：strconv.Itoa 比 fmt.Sprintf(\"%d\") 快好几倍（见 bench）")

	n, err := strconv.Atoi("12a")
	fmt.Printf("  Atoi(\"12a\") -> %d, %v\n", n, err)
	f, _ := strconv.ParseFloat("1.5e3", 64)
	fmt.Println("  ParseFloat(\"1.5e3\") =", f)
	fmt.Println("  Quote:", strconv.Quote("a\"b\n中"))
	fmt.Println("  AppendInt 复用 buffer（零分配的关键）:",
		string(strconv.AppendInt(make([]byte, 0, 8), 42, 10)))
}
