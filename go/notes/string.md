# string

> 环境：`go version go1.26.3 darwin/amd64`。源码：`runtime/string.go`、`strings/builder.go`、`unicode/utf8`。配套代码：`notes/str/`（目录名避开 `string`）。
>
> 版本相关：
> - **1.18**：`strings.Clone`。
> - **1.18/1.20**：`strings.Cut`、`CutPrefix`、`CutSuffix`。
> - **1.20**：**`unsafe.String` / `unsafe.StringData`**——零拷贝转换终于有了合法写法。
> - **1.21**：`strings.ContainsFunc`。
> - **1.23**：**`unique.Make`**（字符串驻留），底层就是后来 `sync.Map` 换用的 hash-trie。
> - **1.24**：迭代器版 `strings.SplitSeq`/`FieldsSeq`/`Lines`/`SplitAfterSeq`。

## 一、基础

### 1.1 内存布局

```go
// runtime/string.go:290
type stringStruct struct {
    str unsafe.Pointer   // 指向字节数组
    len int              // 字节长度
}
```

```text
unsafe.Sizeof(string) = 16   （两个字：data + len）
unsafe.Sizeof([]byte) = 24   （三个字：data + len + cap）
```

**string 没有 cap**——因为它不可变，不需要预留增长空间。这也决定了 `s[i:j]` 是**零拷贝**的：

```text
s[1:3] 零拷贝验证: &s[1]=0xee86c4f, &sub[0]=0xee86c4f, 相同=true
```

字符串字面量存在只读段（`.rodata`），**相同字面量会被链接器合并**：

```text
两个 "abc" 字面量的 data 指针相同？true
```

### 1.2 不可变意味着什么

```go
s := "hello"
s[0] = 'H'   // ✗ 编译错误：cannot assign to s[0]
```

三个好处：

1. **可以安全共享底层数组**——切片零拷贝，函数间传递只拷 16 字节的 header；
2. **可以做 map key**（哈希值稳定）；
3. **并发读天然安全**，不需要锁。

代价：**任何修改都要重新分配**。循环里 `s += x` 是 O(n²)（见 3.5）。

### 1.3 string / []byte / []rune

```go
s := "Go语言"
len(s)                      // 8  ← 字节数！
[]byte(s)                   // [71 111 232 175 173 232 168 128]
[]rune(s)                   // [71 111 35821 35328]  ← Unicode 码点
utf8.RuneCountInString(s)   // 4
s[0]                        // 71 (byte 'G')
s[2]                        // 232 ← "语"的第一个字节，单独拿出来是乱码
```

| 转换 | 成本 |
| --- | --- |
| `[]byte(s)` / `string(b)` | 分配 + `memmove`（除了 2.2 的编译器特例） |
| `[]rune(s)` / `string(rs)` | 分配 + **逐个解码/编码 UTF-8**，比 `[]byte` 贵一个数量级 |

实测（12KB 字符串）：

```text
BenchmarkStringToBytesLong-8         1392 ns/op   12288 B/op   1 allocs/op
BenchmarkBytesToStringLong-8         1535 ns/op   12288 B/op   1 allocs/op
BenchmarkStringToRunesLong-8        17586 ns/op   49152 B/op   1 allocs/op   ← 慢 12 倍、内存 4 倍
```

`[]rune` 的 49152 = 12288 × 4，因为 `rune` 是 `int32`：**转 `[]rune` 内存翻 4 倍**。

### 1.4 `range` 的语义

```go
for i, r := range "Aé中" {
    // i=0 r='A' (U+0041) 占 1 字节
    // i=1 r='é' (U+00E9) 占 2 字节
    // i=3 r='中' (U+4E2D) 占 3 字节
}
```

- `i` 是**字节下标**（不连续），`r` 是解码出来的 `rune`；
- **非法 UTF-8 字节**得到 `U+FFFD`（`utf8.RuneError`）且 `i` 只前进 1；
- `for i := 0; i < len(s); i++` 则是按**字节**遍历，`s[i]` 是 `byte`。

### 1.5 UTF-8 要点

| 字符 | 码点 | 字节数 |
| --- | --- | --- |
| `A` | U+0041 | 1（ASCII） |
| `é` | U+00E9 | 2 |
| `中` | U+4E2D | 3 |
| `🎉` | U+1F389 | 4 |

**自同步性**：看任何一个字节的高位就知道它是首字节（`0xxxxxxx`/`110xxxxx`/`1110xxxx`/`11110xxx`）还是续字节（`10xxxxxx`）。这让"从中间开始解析"和"反向扫描"成为可能。

一个常被忽略的事实：**"一个字符" ≠ "一个 rune"**。

```text
"é"(1 rune) vs "é"(2 rune)：看起来一样，但 == 为 false
```

真正的"用户感知字符"（grapheme cluster）要靠 `golang.org/x/text`。所以"截取前 10 个字"这种需求，`[]rune` 只是近似解。

### 1.6 `strings.Builder`

```go
var sb strings.Builder
sb.Grow(64)                    // 预分配，避免多次扩容
sb.WriteString("a")
sb.WriteByte('b')
sb.WriteRune('中')
fmt.Fprintf(&sb, "%d", 42)     // 实现了 io.Writer
s := sb.String()
```

三个实现细节：

1. **`String()` 是零拷贝的**（内部 `unsafe.String(&buf[0], len(buf))`）；
2. 因此 `String()` 之后再 `Write`，会先复制一份 buf，避免改到已返回的字符串；
3. **不能拷贝**——内部 `addr *Builder` 字段做自引用检查，拷贝后写入直接 panic：

```text
strings: illegal use of non-zero Builder copied by value
```

对比 `bytes.Buffer`：`Buffer` 的 `String()` 会拷贝，但支持读取（`Read`/`Next`）。**只拼字符串用 `Builder`，需要读写用 `Buffer`**。

### 1.7 值得知道的新 API

```go
before, after, found := strings.Cut("key=value", "=")       // 1.18，替代 SplitN(...,2)
rest, ok := strings.CutPrefix("prefix-body", "prefix-")     // 1.20，替代 HasPrefix+TrimPrefix
s2 := strings.Clone(s)                                       // 1.18，切断底层共享（见 3.4）
strings.ContainsFunc("abc1", unicode.IsDigit)                // 1.21

for part := range strings.SplitSeq("a,b,c", ",") { ... }     // 1.24，不产生中间 slice
for line := range strings.Lines("l1\nl2\n") { ... }          // 1.24，保留行尾 \n
```

`Cut` 是最该养成习惯的一个：它把"找分隔符 + 判断存在 + 切两半"合成一次操作，替代了大量 `strings.Index` + 手工切片的代码。

## 二、转换与零拷贝

### 2.1 编译器的零拷贝特例

`runtime/string.go:194` 的注释精确列出了 `slicebytetostringtmp`（即"不分配的临时字符串"）的**全部**应用场景：

```go
// Some internal compiler optimizations use this function.
//   - Used for m[T1{... Tn{..., string(k), ...} ...}] and m[string(k)]
//     where k is []byte, T1 to Tn is a nesting of struct and array literals.
//   - Used for "<"+string(b)+">" concatenation where b is []byte.
//   - Used for string(b)=="foo" comparison where b is []byte.
```

即三条：**map 查找、拼接的中间结果、和字面量比较**。实测：

```text
BenchmarkZeroCopyMapLookup-8    10.09 ns/op       0 B/op   0 allocs/op   ✓
BenchmarkZeroCopyCompare-8       1.09 ns/op       0 B/op   0 allocs/op   ✓
BenchmarkZeroCopyRange-8     31349    ns/op   12288 B/op   1 allocs/op   ✗ 会拷贝！
```

**这里纠正一个流传很广的说法**：`for range string(b)` **不在**优化列表里，实测会完整拷贝一次（12KB → 12288 B/op）。要零拷贝地按 rune 遍历 `[]byte`，用 `utf8.DecodeRune` 手写循环，或者 `unsafe.String`。

另外注意：

```text
BenchmarkNotZeroCopy-8           7.77 ns/op   0 B/op   0 allocs/op
```

把 `string(b)` 存进变量后仍然是 0 allocs——因为**小字符串（≤32 字节）不逃逸时用栈上 `tmpBuf`**。但它确实做了一次 `memmove`，比直接比较慢 7 倍。所以"0 allocs" 不等于"零拷贝"。

### 2.2 `unsafe` 零拷贝转换（1.20+ 的正确写法）

```go
// []byte -> string：只读场景，零拷贝
func bytesToString(b []byte) string {
    if len(b) == 0 { return "" }
    return unsafe.String(unsafe.SliceData(b), len(b))
}

// string -> []byte：更危险
func stringToBytes(s string) []byte {
    if len(s) == 0 { return nil }
    return unsafe.Slice(unsafe.StringData(s), len(s))
}
```

```text
BenchmarkBytesToStringLong-8         1535 ns/op   12288 B/op   1 allocs/op
BenchmarkBytesToStringUnsafeLong-8      0.93 ns/op     0 B/op   0 allocs/op   ← 1650 倍
```

**使用条件**（违反任何一条就是 UB）：

1. 转换后**绝不修改**原 `[]byte`——实测改了 `b[0]`，"不可变"的字符串内容真的变了；
2. 原 `[]byte` 的生命周期必须覆盖字符串的使用期；
3. `string → []byte` 方向**几乎不该用**：字面量在只读段，写入是 **SIGSEGV 进程崩溃**（不是 panic，recover 不了）。

1.20 之前的老写法（`*(*string)(unsafe.Pointer(&b))`）依赖 header 内存布局，现在应该全部换掉（见 unsafe.md）。

**什么时候值得用**：热路径上 KB 级以上的转换，典型是 HTTP body / 文件内容的解析——注意此时 `[]byte` 通常来自 `sync.Pool`，一旦回池字符串就悬空了（pool.md 4.3）。

### 2.3 字符串驻留（interning）

场景：解析大量重复字符串（日志 tag、JSON key、监控维度值），同一份内容存了几万份。

```go
// 1.23+
h1 := unique.Make("some-tag")       // Handle[string]，8 字节，可比较
h2 := unique.Make("some-tag")
h1 == h2                             // true，一次指针比较
h1.Value()                           // 拿回字符串
```

相比自己用 `map[string]string` 做池：

| | 手写 map 池 | `unique.Make` |
| --- | --- | --- |
| 并发 | 要自己加锁 | 内置（hash-trie，无锁读） |
| 回收 | **永不释放**，得自己管 | 没人引用时**被 GC 自动清理**（weak pointer） |
| 比较 | 还是比字符串内容 | Handle 是指针，`==` 一次比较 |

`sync.Map` 在 1.24 换成 hash-trie，最初的动机就是给 `unique` 包用（见 sync.md 2.9）。

## 三、常见陷阱

### 3.1 `s[i]` 是字节，不是字符

```go
s := "中文abc"           // len = 9
s[0]                     // 228 —— "中"的第一个字节
string(s[0])             // "ä" 乱码
string([]rune(s)[0])     // "中" ✓

// 更省的写法：不建整个 []rune
r, size := utf8.DecodeRuneInString(s)   // '中', 3
```

截取同理：

```go
s[:3]                     // "中"  碰巧对（"中"正好 3 字节）
s[:2]                     // "\xe4\xb8"  截断了一个 rune
string([]rune(s)[:2])     // "中文"  ✓
```

### 3.2 `len(s)` 不是字符数

```text
"abc"      len=3  RuneCount=3
"中文"      len=6  RuneCount=2
"🎉🎊"      len=8  RuneCount=2
"é"  len=3  RuneCount=2   ← 而"看起来"只有 1 个字符
```

区分两类需求：

- **展示/校验层面**（"昵称不超过 10 个字"）→ `utf8.RuneCountInString`；
- **存储/协议层面**（数据库字段、长度字段）→ `len`（字节数）。

### 3.3 大小写与比较

```go
strings.ToLower(a) == strings.ToLower(b)   // ✗ 两次分配
strings.EqualFold(a, b)                    // ✓ 零分配 + Unicode case folding
```

```text
BenchmarkEqualFold-8        13.64 ns/op    0 B/op   0 allocs/op
BenchmarkToLowerCompare-8   72.27 ns/op   16 B/op   1 allocs/op
```

另外：

- `strings.ToUpper("straße")` → `STRASSE`（一个字符变两个），**大小写转换会改变长度**；
- 土耳其语的 `i`/`İ` 需要 `ToUpperSpecial(unicode.TurkishCase, ...)`；
- **安全相关的比较**（token、签名、密码哈希）必须用 `crypto/subtle.ConstantTimeCompare`，`==` 是短路的、会泄漏时序信息。

### 3.4 子串持有整个底层数组

```go
big := strings.Repeat("x", 10<<20)   // 10MB
small := big[:10]                    // 零拷贝，但引用着那 10MB
// big 不可达之后，10MB 依然回收不了，因为 small 还活着
small = strings.Clone(small)          // ✓ 断开引用
```

和 slice 的同一个坑（slice.md 3.3）。**长期保存的子串一律 `strings.Clone`**。典型事故：解析大 JSON / 日志行，只留几个字段却让整个 buffer 泄漏。

### 3.5 拼接的五种写法

```text
BenchmarkConcatOperator-8       4972 ns/op   9744 B/op   99 allocs/op   ← 循环里 s += x，O(n²)
BenchmarkConcatSprintf-8       13876 ns/op  11332 B/op  198 allocs/op   ← 最差
BenchmarkConcatBuilder-8         745 ns/op    504 B/op    6 allocs/op
BenchmarkConcatBuilderGrow-8     553 ns/op    320 B/op    1 allocs/op   ← 最优（已知长度）
BenchmarkConcatJoin-8            731 ns/op    192 B/op    1 allocs/op   ← 已有 slice 时
BenchmarkConcatAppendBytes-8     615 ns/op    192 B/op    1 allocs/op
```

选择规则：

- **少量、固定个数**：直接 `a + b + c`——编译器生成一次 `concatstrings`，只分配一次，最快；
- **循环拼接**：`strings.Builder` + `Grow`；
- **已有 `[]string`**：`strings.Join`（内部先算总长度，一次分配）；
- **需要 `append` 语义 / 要复用 buffer**：`[]byte` + `strconv.AppendXxx`，最后 `string(buf)`。

### 3.6 数字转换

```go
string(65)              // ✗ Go 1.15 起是 vet 错误：得到 "A" 而不是 "65"
string(rune(65))        // "A"
strconv.Itoa(65)        // "65" ✓
```

```text
BenchmarkItoa-8         24.30 ns/op   8 B/op   1 allocs/op
BenchmarkSprintfD-8     66.69 ns/op   8 B/op   1 allocs/op   ← 慢 2.7 倍（反射 + 接口装箱）
BenchmarkAppendInt-8    12.67 ns/op   0 B/op   0 allocs/op   ← 复用 buffer，零分配
```

`fmt.Sprintf` 在热路径上是常见的性能杀手：参数要装箱（mem.md 2.2）、格式串要解析、内部还有 `sync.Pool`。能用 `strconv` 就别用 `fmt`。

### 3.7 `utf8.RuneCountInString` vs `len([]rune(s))`

```text
BenchmarkRuneCount-8           8251 ns/op   0 B/op   0 allocs/op
BenchmarkRuneCountViaRunes-8   8123 ns/op   0 B/op   0 allocs/op
```

有意思的是 `len([]rune(s))` 也是 0 allocs——编译器识别了"只取长度"这个模式，把整个转换优化掉了。但**只有 `len()` 直接套在转换外面时**才成立；一旦存进变量就会真的分配 4 倍内存。

### 3.8 `strings.Split` 的边界行为

```go
strings.Split("", ",")        // [""]  长度 1，不是 0！
strings.Split("a", "")        // ["a"] → 实际是 ["a"]，空分隔符按 rune 切
strings.Split("a,b", "")      // ["a" "," "b"]
strings.Fields("  a  b  ")    // ["a" "b"]  按空白切且自动去空
```

`Split("", sep)` 返回长度 1 的切片是最常见的踩坑点（写 `if len(parts) > 0` 永远为真）。用 `strings.Fields` 或先判 `s == ""`。

## 四、常见面试题

**1. Go 的 string 底层结构是什么？为什么没有 cap？**
`stringStruct{str unsafe.Pointer; len int}`，16 字节。没有 cap 是因为不可变——不需要预留增长空间。这也让 `s[i:j]` 天然零拷贝（见 1.1）。

**2. 字符串为什么设计成不可变？**
换来三件事：切片/传参零拷贝、可做 map key（哈希稳定）、并发读免锁。代价是任何修改都要重新分配，循环拼接是 O(n²)（见 1.2、3.5）。

**3. `len(s)` 返回什么？怎么数字符数？**
字节数。字符数用 `utf8.RuneCountInString(s)`。注意"字符"还有第三层——grapheme cluster（`"é"` 是 2 个 rune 但看起来 1 个字符），要靠 `x/text`（见 1.5、3.2）。

**4. `string` 和 `[]byte` 互转要拷贝吗？有哪些例外？**
一般都要拷贝。编译器有三条零拷贝特例（`runtime/string.go:194` 注释）：`m[string(b)]`、`"<"+string(b)+">"`、`string(b)=="literal"`。**`for range string(b)` 不在其中**，实测会拷贝（见 2.1）。

**5. 怎么零拷贝地在 string 和 []byte 之间转换？有什么风险？**
1.20 起用 `unsafe.String(unsafe.SliceData(b), len(b))` 和 `unsafe.Slice(unsafe.StringData(s), len(s))`。风险：转换后修改 `[]byte` 会让"不可变"的字符串变化；`string → []byte` 方向写入字面量会 SIGSEGV（不可 recover）。实测能省 1650 倍（12KB：1535ns → 0.93ns），只在热路径上大块数据时值得（见 2.2）。

**6. `for i, r := range s` 里 i 和 r 分别是什么？**
`i` 是字节下标（不连续），`r` 是解码出的 rune。非法 UTF-8 得到 `U+FFFD` 且 i 只前进 1（见 1.4）。

**7. `s[0]` 为什么打印出来是数字/乱码？**
`s[i]` 的类型是 `byte`（`uint8`），不是字符。对多字节字符取单个字节没有意义。正确做法是 `utf8.DecodeRuneInString` 或 `[]rune(s)[0]`（见 3.1）。

**8. 拼接字符串有哪些方式？分别什么时候用？**
固定少量 → `a+b+c`（一次 `concatstrings`）；循环 → `strings.Builder`+`Grow`；已有 slice → `strings.Join`；需要 append 语义/复用 buffer → `[]byte`+`strconv.AppendXxx`。实测 `Builder+Grow` 比循环 `+=` 快 9 倍、分配少 99 倍（见 3.5）。

**9. `strings.Builder` 和 `bytes.Buffer` 有什么区别？**
`Builder` 只写不读，`String()` 零拷贝（`unsafe.String` 指向内部 buf），不能拷贝（有 `addr` 自引用检查）。`Buffer` 支持读写，`String()` 会拷贝。只拼字符串用前者（见 1.6）。

**10. 为什么子串会造成内存泄漏？**
`s[i:j]` 共享底层数组，一个 10 字节的子串能让 10MB 的原字符串无法回收。长期保存的子串要 `strings.Clone`（见 3.4）。

**11. 忽略大小写比较字符串的正确方式？**
`strings.EqualFold`（零分配，做 Unicode case folding）。`ToLower(a)==ToLower(b)` 有两次分配且慢 5 倍。安全敏感的比较用 `crypto/subtle.ConstantTimeCompare`（见 3.3）。

**12. `string(65)` 得到什么？**
`"A"`（把 65 当 rune）。Go 1.15 起 `go vet` 会报错提示。要 `"65"` 用 `strconv.Itoa`（见 3.6）。

**13. `strconv.Itoa` 和 `fmt.Sprintf("%d", n)` 差多少？**
实测 24.3ns vs 66.7ns（2.7 倍）。`fmt` 要装箱参数、解析格式串、走反射。零分配版本是 `strconv.AppendInt(buf[:0], n, 10)`（12.7ns，0 allocs）（见 3.6）。

**14. `[]rune(s)` 的开销是多少？**
分配 `4 × RuneCount` 字节 + 逐个解码。实测 12KB 字符串转 `[]rune` 是转 `[]byte` 的 12 倍时间、4 倍内存。只在真的需要随机访问字符时才转（见 1.3）。

**15. 什么是字符串驻留？Go 里怎么做？**
把内容相同的字符串收敛到同一份内存，减少重复占用。1.23 起用 `unique.Make[T]` —— 返回可比较的 8 字节 `Handle`，无引用时会被 GC 自动清理（weak pointer）。自己写 map 池的问题是永不释放且要加锁（见 2.3）。
