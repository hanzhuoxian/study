# json

> 基于 Go 1.26（`GOEXPERIMENT=jsonv2` 关闭时的默认实现，即 `encode.go` / `decode.go` / `tags.go`）源码整理。

## 一、struct tag 用法

`json:"name,opt1,opt2"`，逗号前是自定义字段名，逗号后是选项列表。

### 字段名

- 不写 tag：默认使用 Go 字段名（必须是导出字段，首字母大写）。
- `json:"name"`：编解码时使用 `name` 作为 key。
- `json:"-"`：该字段完全不参与编解码（唯一例外：`json:"-,"` 表示字段名字面量就是 `-`）。

### omitempty

字段值为 **empty** 时序列化时省略该字段。
"empty" 的定义：`false`、`0`、`nil` 指针、`nil` interface，以及长度为 0 的 array/slice/map/string。
注意：**对 struct 类型不生效**（struct 永远不被认为是 empty，即使所有字段都是零值）—— 因为 `isEmptyValue` 只判断上述几种 `reflect.Kind`，`Struct` 落入 `default` 分支返回 `false`。

### omitzero

字段值为**零值**时省略，判断规则：
1. 如果类型实现了 `interface{ IsZero() bool }`（如 `time.Time`），优先调用该方法判断；
2. 否则用 `reflect.Value.IsZero()`（即字段的内存表示是否等于该类型的零值）判断。

与 `omitempty` 的关键区别：`omitzero` **对 struct 类型也生效**（只要整个 struct 等于零值，或其 `IsZero()` 返回 true）。
两者可同时使用：`json:",omitempty,omitzero"`，此时值为 empty 或 zero 都会省略。

### string

把字段值编码成 "JSON 字符串"（即再包一层引号），常用于和 JS 交互时避免大整数精度丢失等问题。
只对 string、浮点、整数、bool 类型生效，例如 `int64` 字段加了 `,string` 后会被编码成 `"123"` 而不是 `123`。

### 匿名（嵌入）字段

匿名 struct 字段的子字段会被"提升"到外层参与编解码，冲突消解规则见下文「typeFields 字段解析规则」。

---

## 二、Marshal 源码解析

### 整体调用链

```go
Marshal(v)
  → newEncodeState()           // 从 sync.Pool 取一个 encodeState（内嵌 bytes.Buffer），减少 GC 压力
  → e.marshal(v, opts)          // defer recover，把内部 panic（如 UnsupportedTypeError）转成 error 返回
    → e.reflectValue(reflect.ValueOf(v), opts)
      → valueEncoder(v)(e, v, opts)   // 取该类型对应的 encoderFunc 并执行
```

编码逻辑**不是一次性的递归函数**，而是「类型 → encoderFunc」的缓存表：每个 `reflect.Type` 只需要构建一次编码函数（`newTypeEncoder`），构建结果被缓存到全局 `sync.Map`：

```go
var encoderCache sync.Map // map[reflect.Type]encoderFunc

func cachedTypeEncoder(t reflect.Type) encoderFunc {
    if fi, ok := encoderCache.Load(t); ok {
        return fi.(encoderFunc)
    }
    // 用一个"占位"闭包处理并发下同一类型的自引用/递归定义（如链表节点），
    // 避免 newTypeEncoder 在解析自身类型时死循环
    fi, loaded := encoderCache.LoadOrStore(t, encoderFunc(func(e *encodeState, v reflect.Value, opts encOpts) {
        // wait & 调用真正的 encoder
    }))
    if !loaded {
        f := newTypeEncoder(t, true)
        encoderCache.Store(t, f)
    }
    return fi.(encoderFunc)
}
```

之后同类型的每次 `Marshal` 都直接走缓存的 `encoderFunc`，避免重复的反射类型判断——这是 `encoding/json` 性能优化的核心手段。

### encoder 的选择优先级（newTypeEncoder）

对一个类型 `t`，按以下顺序决定用哪个 `encoderFunc`：

1. 若 `t` 不是指针，且 `*t` 实现了 `Marshaler`（值接收者场景），用 `condAddrEncoder`：能取地址就走地址版本的 encoder，取不了地址就退回普通类型的 encoder（这是为了避免"把值转成接口"时产生一次额外堆分配）。
2. 若 `t` 本身实现了 `Marshaler`（即有 `MarshalJSON() ([]byte, error)` 方法），直接用 `marshalerEncoder`。
3. 类似地检查 `encoding.TextMarshaler`（`MarshalText() ([]byte, error)`），编码结果会被当作 JSON 字符串（额外加引号转义）。
4. 都没有实现，按 `reflect.Kind` 走内置 encoder：`bool/int/uint/float/string` 是简单类型；`interface` 用 `interfaceEncoder`（运行时再取动态类型递归编码）；`struct/map/slice/array/pointer` 分别对应 `newStructEncoder/newMapEncoder/newSliceEncoder/newArrayEncoder/newPtrEncoder`。

即优先级：`Marshaler` > `TextMarshaler` > 类型默认规则。这也是为什么自定义 `MarshalJSON` 可以完全接管某个类型的序列化行为。

### struct 的编码（structEncoder）

`typeFields(t)` 只在类型首次编码时解析一次 tag、匿名嵌入、命名冲突等信息，结果同样被缓存（`cachedTypeFields`，也是 `sync.Map`）。之后 `structEncoder.encode` 只是遍历这份预解析好的 `[]field`：

```go
func (se structEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
    next := byte('{')
FieldLoop:
    for i := range se.fields.list {
        f := &se.fields.list[i]
        // 按 f.index（支持多层嵌入）逐层取到真正的字段值 fv
        fv := v
        for _, i := range f.index {
            if fv.Kind() == reflect.Pointer {
                if fv.IsNil() {
                    continue FieldLoop   // 嵌入指针为 nil，整个字段跳过
                }
                fv = fv.Elem()
            }
            fv = fv.Field(i)
        }
        if (f.omitEmpty && isEmptyValue(fv)) ||
            (f.omitZero && fv满足零值判断) {
            continue
        }
        // 写入预先计算好的、已转义的字段名（nameEscHTML / nameNonEsc）
        // 再调用该字段自己的 f.encoder 递归编码
        f.encoder(e, fv, opts)
    }
}
```

要点：字段名的 JSON 转义在 `typeFields` 阶段就预计算好并缓存了（`nameEscHTML`/`nameNonEsc`），编码时不会重复做字符串转义，进一步减少运行时开销。

### map 的编码（mapEncoder）—— key 会被排序

`encoding/json` 对 map 编码时会显式对 key 排序，保证同一个 map 每次序列化结果**确定（deterministic）**：

```go
sv := make([]reflectWithString, v.Len())
// 收集 key（转成 string：字符串 key 直接用；整数 key 格式化；实现了 TextMarshaler 的 key 调用 MarshalText）
slices.SortFunc(sv, func(i, j reflectWithString) int {
    return strings.Compare(i.ks, j.ks)
})
```

这也解释了为什么 map 序列化比 struct 序列化更慢：多了一次 key 提取 + 排序。

### 特殊情况

- `[]byte`：不会被当成普通 slice 编码，而是走 `encodeByteSlice`，Base64 编码后作为 JSON 字符串。
- 循环引用检测：`ptrEncoder`/`mapEncoder` 里有 `e.ptrLevel` 计数器，超过阈值（`startDetectingCyclesAfter`）后才开始用 `ptrSeen` 记录已访问指针，检测到环会返回 `UnsupportedValueError`（避免正常场景下每次都做昂贵的环检测，只有嵌套很深时才启用）。
- `nil` 指针 / `nil` interface / `nil` map / `nil` slice：编码为 `null`。

---

## 三、Unmarshal 源码解析

### 整体调用链

```go
Unmarshal(data, v)
  → d.unmarshal(v)
    → 校验 v 必须是非 nil 指针（否则返回 InvalidUnmarshalError）
    → d.value(rv.Elem())     // 顶层从指针指向的 Elem 开始解码（不是 rv 本身）
```

`decodeState` 内部先用一个轻量 `scanner`（状态机）扫描 JSON 文本，得到 token 边界（`scanBeginObject`/`scanBeginArray`/`scanBeginLiteral` 等 opcode），再驱动 `object()`/`array()`/`literalStore()` 做真正的反射赋值——即"扫描"和"赋值"是分离的两层。

### indirect：处理指针链、接口、Unmarshaler 优先级

这是 Unmarshal 里最精巧的一段逻辑，作用是把 `v` 一路解引用到一个可以真正赋值的 `reflect.Value`，期间：

- 如果某一层实现了 `Unmarshaler`（`UnmarshalJSON`），立刻停止解引用，把原始 JSON 片段整体交给它处理，**后续字段不再走反射赋值**。
- 否则检查 `encoding.TextUnmarshaler`。
- 遇到 `nil` 指针会自动 `New` 一个新值填充（这就是为什么 `*int`、`**Foo` 这类多级指针字段可以直接被 Unmarshal 出来，不需要预先分配)。
- 对 `decodingNull`（JSON 值是字面量 `null`）场景特殊处理：遇到第一个可设置的指针就停止，把它置为 `nil`，而不是继续解引用。

优先级：`Unmarshaler` > `TextUnmarshaler` > 反射默认规则。和 Marshal 对称。

### object 的解析（字段匹配规则）

```go
func (d *decodeState) object(v reflect.Value) error {
    u, ut, pv := indirect(v, false)
    if u != nil {                 // 实现了 Unmarshaler，整体转交
        return u.UnmarshalJSON(rawBytes)
    }
    ...
    fields = cachedTypeFields(t)  // 和 Marshal 共用同一份 typeFields 缓存/解析逻辑
    for 每个 JSON key {
        f := fields.byExactName[key]   // 1. 先按导出字段名/tag 精确匹配（大小写敏感）
        if f == nil {
            f = fields.byFoldedName[foldName(key)]  // 2. 精确匹配失败，退化为大小写不敏感匹配（ASCII fold）
        }
        if f == nil {
            // 找不到对应字段：DisallowUnknownFields() 开启时报错，否则直接丢弃这个 key
            continue
        }
        // destring: 字段带 ",string" tag 时，先把值当字符串解出来再二次解析
    }
}
```

要点：
- **精确匹配优先，大小写不敏感兜底**——所以 JSON 里的 `Name`/`name`/`NAME` 都能匹配到 Go 字段 `Name`，但如果两个字段仅大小写不同都能"模糊命中"同一个 key，会取 tag 更明确、层级更浅的那个（复用 typeFields 的冲突消解规则）。
- 目标是 `map[K]V` 时不用 `typeFields`，而是校验 key 类型（string/整数/`TextUnmarshaler`），每次新建/复用一个 `mapElem` 承载 value 后 `SetMapIndex`。
- 目标是 `nil` 接口（无方法接口 `any`）时不走反射，直接用 `objectInterface()` 递归解析成 `map[string]any`，效率更高也更简单。

### array 的解析

对 slice：容量不够会 `growslice` 扩容；JSON 数组元素比 slice 长度多时，多余的会被静默丢弃；数组（`[N]T`）多余的元素同样丢弃，元素不足则保留剩余位置的零值。

---

## 四、typeFields 字段解析规则（Marshal/Unmarshal 共用）

`typeFields(t)` 在类型第一次被编解码时执行一次并缓存结果（`cachedTypeFields`），核心是解决"匿名嵌入字段提升"和"命名冲突"：

1. **按 BFS 逐层展开**：从最外层 struct 开始，同一层的所有字段先收集，如果本层没有出现同名冲突，才会展开下一层的匿名字段（`next`/`current` 两个队列 + `nextCount` 计数）。也就是说，**浅层字段总是优先于深层同名字段**（类似 Go 语言里字段/方法遮蔽规则）。
2. **未导出的匿名非 struct 字段会被忽略**（因为无法从包外访问）。
3. **同一层出现同名冲突**（比如两个匿名字段都提升出一个 `Name`）：
   - 如果只有一个字段显式写了 tag，tag 优先，冲突消解为那个字段；
   - 如果都写了/都没写 tag，这些字段全部丢弃，谁都不参与编解码（Go 的"二义性即不存在"哲学）。
4. 解析出的每个 `field` 会预先计算好：JSON 转义后的字段名（HTML 转义/非转义两个版本）、`omitempty`/`omitzero`/`string` 等选项、字段访问路径 `index []int`（支持多级嵌入）、对应的 `encoderFunc`。

这一整套解析只做一次，后续所有该类型的 Marshal/Unmarshal 都复用缓存结果，是 `encoding/json` 反射开销可控的关键设计。

---

## 五、流式编解码：Decoder / Encoder

`json.Marshal`/`json.Unmarshal` 是"一次性、全量在内存"的 API：入参/出参都是完整的 `[]byte`。
`json.NewDecoder(r io.Reader)`/`json.NewEncoder(w io.Writer)` 则面向 `io.Reader`/`io.Writer`，适合：

- 网络流（如 HTTP body）：不需要先 `io.ReadAll` 把整个 body 读进内存再 `Unmarshal`，可以边读边解析，节省内存、降低延迟。
- 连续多个 JSON 值（JSON Lines / streaming JSON）：同一个 `Decoder` 反复调用 `Decode()` 可以依次读出多个顶层 JSON 值，配合 `More()` 判断是否还有下一个值。

`Decoder` 内部维护一个 `bufio` 风格的缓冲区，`Decode` 时若当前 buffer 里的数据不足以构成一个完整 JSON 值，会调用 `refill()` 继续从底层 `io.Reader` 读取补充，而不是要求一次性传入完整数据。`Token()` 提供更底层的、按 token（`{`/`}`/`[`/`]`/key/value）遍历 JSON 的能力，用于流式处理超大 JSON 而不想一次性反射到具体的 struct。

`Encoder.Encode(v)` 相当于 `Marshal(v)` 后再写入 `w`，并在末尾追加换行符 `\n`（这是和 `Marshal` 明确不同的行为），适合按行输出多个 JSON 对象。

对比：

|                           | 输入/输出               | 是否要求完整数据 | 典型场景                                                          |
| ------------------------- | ----------------------- | ---------------- | ----------------------------------------------------------------- |
| `Marshal`/`Unmarshal`     | `[]byte`                | 是               | 内存中已有的数据、需要拿到完整 `[]byte` 做其他处理（如计算 hash） |
| `NewEncoder`/`NewDecoder` | `io.Writer`/`io.Reader` | 否，支持边读边解 | HTTP body、文件、多个连续 JSON 值                                 |

---

## 六、encoding/json/v2 与 jsontext（实验性新一代实现）

Go 1.26 的 `GOROOT/src/encoding/json/` 下还带了两个新包：`encoding/json/v2`（语义层）和 `encoding/json/jsontext`（语法层），只有用 `GOEXPERIMENT=jsonv2` 编译时才会生效。开启后，**标准库的 `encoding/json`（v1 API：`Marshal`/`Unmarshal`/`Decoder`/`Encoder`）本身会变成一层薄封装，内部转调 `jsonv2` 实现**（对应源码里 `v2_encode.go`/`v2_decode.go`/`v2_stream.go`/`v2_tags.go` 这批带 `//go:build goexperiment.jsonv2` 构建约束的文件）。也就是说 v2 不是另起炉灶的平行包，而是准备将来替换掉 v1 内部实现的下一代版本，v1 的函数签名保持不变，行为可能因为默认值不同而略有差异。

### 三层结构

- **`encoding/json/jsontext`**（语法层，syntactic）：只关心 JSON 文本本身的语法——`Encoder`/`Decoder` 按 `Token`（`{`、`}`、`[`、`]`、字面量、字符串、数字）或完整的 `Value`（`[]byte`，一段完整合法的 JSON 值）读写，不涉及"这段 JSON 对应哪个 Go 类型"。
- **`encoding/json/v2`**（语义层，semantic）：负责 Go 值 ↔ JSON 值的映射，`Marshal`/`Unmarshal` 建立在 `jsontext` 之上，另有 `MarshalWrite`/`UnmarshalRead`（对接 `io.Writer`/`io.Reader`）、`MarshalEncode`/`UnmarshalDecode`（直接对接 `jsontext.Encoder`/`Decoder`，可以插在别的编码流程中间）。
- **`encoding/json`**（v1 兼容层）：`goexperiment.jsonv2` 开启时，退化为调用 `jsonv2.Marshal`/`jsonv2.Unmarshal` 并做少量参数转换。

这套"语法/语义分层"是 v2 相对 v1 最大的架构变化：v1 里 scanner（语法扫描）和 decodeState（语义赋值）是耦合在一个内部包里的，v2 把语法层独立导出成 `jsontext`，使得"只处理 token 流、不反射到具体类型"这类需求（比如透传/转发 JSON、只改其中一个字段再转发）可以直接用 `jsontext.Encoder`/`Decoder`，不必先反序列化成 struct 再序列化回去。

### 更安全的默认行为（v1 vs v2）

v2 文档专门有一节 "Security Considerations"，明确列出了几个 v1/v2 默认值不同、且容易被利用做协议混淆攻击（同一份 JSON 被两个服务解析出不同语义）的地方：

| 行为                             | v1 默认                                             | v2 默认                                                              |
| -------------------------------- | --------------------------------------------------- | -------------------------------------------------------------------- |
| JSON 字符串里出现非法 UTF-8 字节 | 替换成 Unicode 替换字符（静默"修复"，属于数据损坏） | 直接报错拒绝                                                         |
| 对象里出现重复 key               | 允许，后出现的覆盖/合并前面的                       | 默认拒绝（可用 `jsontext.AllowDuplicateNames` 打开）                 |
| struct 字段名匹配                | 大小写不敏感（"宽松"匹配）                          | 大小写敏感（"严格"匹配，可用 `MatchCaseInsensitiveNames` 放宽）      |
| 未知字段                         | 默认忽略                                            | 默认仍然忽略（可用 `RejectUnknownMembers` 显式拒绝，两版本行为一致） |

可以看到 v2 选择的默认值整体更"严格"、更不容易产生歧义，这也是标准库明确写在文档里、建议新代码优先使用 v2 的原因之一。

### struct tag 的变化

v2 里 `json` tag 语法整体沿用逗号分隔，但语义更明确、也新增了几个选项：

- `omitzero`/`omitempty` 依然都支持，但语义边界更清晰：`omitzero` 完全按 Go 类型系统定义（`IsZero()` 或反射零值判断，`nil` slice/map 才算零值），`omitempty` 完全按"编码结果是否为 JSON null/空字符串/空对象/空数组"定义——两者不再是 v1 里那种"omitempty 覆盖几种 reflect.Kind、struct 例外"的实现细节耦合写法，而是分别对应"Go 语义"和"JSON 语义"两个正交维度。
- `case:strict` / `case:ignore`：显式指定某个字段反序列化时的大小写匹配策略，可以字段级别覆盖全局的 `MatchCaseInsensitiveNames` 选项。
- `inline`：把这个字段（必须是 struct、`map[~string]T` 或 `jsontext.Value`）当前字段"内联"进父 struct，效果类似 v1 里匿名嵌入的隐式提升，但可以显式加在**具名字段**上，不要求一定是匿名字段。
- `unknown`：标记某个 `map[~string]T`/`jsontext.Value` 字段专门用来兜底承接所有未匹配到具体字段的"未知成员"，避免用 `RejectUnknownMembers` 时把它们直接拒绝掉。
- `format`：为字段值指定格式化方式（比如时间用 `format:RFC3339`、`format:'2006-01-02'`），把"这个字段该按什么格式转字符串"这种原本要靠自定义 `MarshalJSON` 才能表达的需求，下沉成了 tag 选项。

字段名冲突消解规则和 v1 基本一致（BFS 找同名字段，浅层优先，同层冲突看谁显式打了 tag），但要求非内联字段的 JSON 名字必须唯一，否则会直接报 `SemanticError`，而不是像 v1 那样"冲突就全部丢弃、静默忽略"。

### 接口与自定义序列化的新能力

v2 把 v1 的 `MarshalJSON()([]byte, error)` / `UnmarshalJSON([]byte) error` 保留下来（分别叫 `Marshaler`/`Unmarshaler`），但新增了效率更高的版本：

```go
type MarshalerTo interface {
    MarshalJSONTo(*jsontext.Encoder) error   // 直接写 token/value，省掉一次 []byte 中转分配
}
type UnmarshalerFrom interface {
    UnmarshalJSONFrom(*jsontext.Decoder) error
}
```

如果一个类型同时实现了 `Marshaler` 和 `MarshalerTo`，`MarshalerTo` 优先——因为它可以直接对着 `jsontext.Encoder` 流式写，而不需要先在内存里拼出一段完整 `[]byte` 再拷贝进外层 buffer，减少了一次分配和一次拷贝。`UnmarshalerFrom` 相对 `Unmarshaler` 同理。

更进一步，v2 提供了**"不改动类型定义、也能自定义序列化"**的方式，这是 v1 完全没有的能力：

```go
opts := json.WithMarshalers(json.MarshalFunc(func(t time.Time) ([]byte, error) {
    return []byte(`"` + t.Format(time.RFC3339) + `"`), nil
}))
json.Marshal(v, opts)
```

`MarshalFunc[T]`/`MarshalToFunc[T]`/`UnmarshalFunc[T]`/`UnmarshalFromFunc[T]` 允许调用方针对某个具体类型 `T`（哪怕是第三方包里的类型，没法给它加方法）注入自定义编解码逻辑，通过 `Options` 参数传进 `Marshal`/`Unmarshal`，而不用像 v1 那样只能靠"给类型定义方法"或者"包一层 wrapper 类型"来定制。

### 函数式 Options

v1 的行为定制主要靠 struct tag 加 `Encoder`/`Decoder` 上少数几个方法（`SetIndent`、`DisallowUnknownFields`、`UseNumber`）。v2 统一成一套**可组合的 `Options` 参数**，直接传给 `Marshal`/`Unmarshal` 等函数：

```go
json.Marshal(v, json.Deterministic(true), json.FormatNilSliceAsNull(true))
json.Unmarshal(data, &v, json.RejectUnknownMembers(true))
```

`Options` 本质上是一组"属性名 → 值"的集合（类似不可变 map），`JoinOptions` 可以把多组选项合并，后设置的覆盖先设置的；`GetOption` 可以在自定义 Marshaler 内部读取当前生效的选项。这让"全局默认行为"和"某次调用的临时行为"都能用同一套 API 表达，不需要像 v1 那样为每个新行为单独加一个 `Decoder` 方法。

### 现状与建议

截至 Go 1.26，`encoding/json/v2` 和 `encoding/json/jsontext` 仍标注为**实验性（experimental）**，不在 Go 1 兼容性承诺范围内，API 后续可能调整，必须显式加 `GOEXPERIMENT=jsonv2` 编译标记才存在。标准库文档的建议是：新代码如果不受历史行为约束，优先直接使用 `encoding/json/v2`（默认值更安全），存量代码继续用 `encoding/json`（v1），等 v2 稳定转正后再迁移。

---

## 七、面试问题

1. **`json.Marshal` 对未导出字段（小写字段名）会怎么处理？为什么？**
   答：完全忽略，不会出现在输出中。因为 `reflect` 包无法读取/设置未导出字段的值（`CanInterface()`/`CanSet()` 为 false），`encoding/json` 在 `typeFields` 阶段就跳过了未导出字段。

2. **`omitempty` 和 `omitzero` 有什么区别？为什么 `omitempty` 对 struct 不生效而 `omitzero` 生效？**
   答：`omitempty` 判断的是"空值"语义（依赖 `reflect.Kind`，只覆盖 bool/数字/指针/interface/长度为 0 的容器），struct 类型没有归入这个判断集合，所以 struct 永远不被认为是 empty；`omitzero` 判断的是"是否等于零值"（或类型自定义的 `IsZero()`），可以对任意可比较其零值的类型生效，包括 struct（比如全零的 `time.Time`）。

3. **map 类型序列化后 key 的顺序是怎样的？为什么要这样设计？**
   答：`encoding/json` 会先把所有 key 转成字符串后按字典序排序，再输出。这样保证同一个 map 无论内部遍历顺序如何随机（Go map 遍历本身是无序的），序列化结果始终确定，方便做 diff、缓存、幂等重试等。

4. **`json:",string"` 这个 tag 选项的作用是什么？什么类型可以用？**
   答：让该字段的值被编码成"JSON 字符串"形式（即整数/浮点/bool/string 值外面再包一层引号）。常用于避免 JS/前端对超过 `2^53` 的整数精度丢失问题。只对 string、整数、浮点、bool 类型生效，对 struct/slice/map 无效。

5. **`encoding/json` 是如何避免每次 Marshal 都重新做反射类型分析的？**
   答：内部维护一个全局的 `sync.Map`（`encoderCache`/字段信息缓存），以 `reflect.Type` 为 key 缓存该类型对应的 `encoderFunc`（以及 struct 的字段解析结果 `structFields`）。首次遇到某个类型时才会做完整的反射分析（tag 解析、方法集检查、匿名嵌入展开），此后同类型的所有编解码都直接复用缓存的函数，只有实际编码时的字段取值/写 buffer 是运行时开销。

6. **`Marshaler`/`Unmarshaler` 接口和 `encoding.TextMarshaler`/`TextUnmarshaler` 接口哪个优先级更高？为什么要有两套接口？**
   答：`(Text)Marshaler`/`Unmarshaler` 优先级更高。`Marshaler` 直接生产/消费完整的 JSON 值（可以是对象、数组等任意合法 JSON），而 `TextMarshaler`/`TextUnmarshaler` 只是把值编码成一个字符串再套上 JSON 引号（常用于 `time.Duration`、枚举、`net.IP` 这类"本质是字符串表示"的类型），两者语义不同，所以并存且互斥，`Marshaler` 覆盖面更广所以优先。

7. **一个值类型 `T` 的方法用值接收者实现了 `MarshalJSON`，`Marshal(t)` 和 `Marshal(&t)` 的编码结果一样吗？内部实现上有什么区别？**
   答：结果一样（Go 方法集规则：值接收者方法对值和指针都可调用）。但内部实现上，`newTypeEncoder` 会优先检查 `*T` 是否实现了 `Marshaler`（哪怕原始类型是值类型），如果能取地址就直接用地址调用（`addrMarshalerEncoder`），这是为了避免把值 `T` 转成 `interface{}` 时产生一次逃逸到堆的拷贝分配，是一个性能层面的实现细节，不影响语义。

8. **两个匿名嵌入字段在同一层级都提升出同名字段时，`encoding/json` 是如何处理冲突的？**
   答：按 BFS 逐层展开，同一层如果出现命名冲突：如果只有一个字段显式写了 `json` tag，则以它为准，其余丢弃；如果都写了 tag 或都没写 tag（无法区分优先级），这些冲突字段全部被丢弃，都不参与编解码。浅层字段总是优先于更深层的同名字段（不会展开到下一层）。

9. **反序列化时，JSON 里出现了 struct 里不存在的字段，默认行为是什么？如何让它报错？**
   答：默认直接静默丢弃未知字段。调用 `Decoder.DisallowUnknownFields()` 后，遇到匹配不到任何字段（且不是 tag 为 `-`）的 JSON key 会返回错误（一次性 `Unmarshal` 没有这个开关，只能用 `NewDecoder` + `DisallowUnknownFields`）。

10. **`json.Unmarshal(data, &v)` 里，为什么要求传入指针？如果传值会怎样？**
    答：反序列化本质是"写入"，需要通过反射拿到可寻址（`CanSet`）的 `reflect.Value` 才能修改目标内存。如果直接传值，`reflect.ValueOf(v)` 得到的是值的拷贝且不可设置，Unmarshal 无法把结果写回调用者的变量。传值会直接返回 `*json.InvalidUnmarshalError`。

11. **`json.NewDecoder(r).Decode(&v)` 和 `json.Unmarshal(data, &v)` 有什么本质区别？分别适合什么场景？**
    答：`Unmarshal` 要求一次性拿到完整 `[]byte`；`Decoder.Decode` 基于 `io.Reader`，内部维护缓冲区，数据不够时会自动 `refill` 继续读，支持从一个流里连续解析多个 JSON 值（配合 `More()`）。处理 HTTP body、大文件、JSON Lines 场景优先用 `Decoder`，避免一次性把整个响应体读入内存；已经拿到完整字节切片、或需要复用/校验原始字节（如算 hash）时用 `Unmarshal` 更直接。

12. **为什么 `[]byte` 类型的字段序列化后是一个字符串而不是数字数组？**
    答：`encoding/json` 对 `[]byte` 做了特殊处理（`encodeByteSlice`），会先做标准 Base64 编码，再作为 JSON 字符串输出；反序列化时对应地做 Base64 解码。这是标准库刻意的设计，因为把每个字节都编码成一个 JSON 数字元素会让体积膨胀且可读性差。

13. **`encoding/json/v2` 和 v1 相比，在默认行为上做了哪些更"安全"的改变？**
    答：主要几点：JSON 字符串里出现非法 UTF-8 时 v1 默认静默替换成替换字符，v2 默认直接报错；对象里出现重复 key 时 v1 默认允许（后者覆盖前者），v2 默认拒绝；struct 字段名匹配 v1 默认大小写不敏感，v2 默认大小写敏感。这些改动都是为了消除"同一份 JSON 被不同实现解析出不同语义"的安全隐患（如两个微服务分别用 v1/v2 解析同一请求却得到不同结果）。

14. **`encoding/json`、`encoding/json/v2`、`encoding/json/jsontext` 三者是什么关系？**
    答：`jsontext` 是语法层，只处理 JSON token/value 的读写，不关心 Go 类型；`json/v2` 是语义层，建立在 `jsontext` 之上，负责 Go 值与 JSON 值的相互映射（`Marshal`/`Unmarshal` 等）；`encoding/json`（v1）在 `GOEXPERIMENT=jsonv2` 编译时会变成对 `json/v2` 的一层兼容封装。三者是"底层→上层→兼容层"的分层关系，不是三个互相独立的实现。

15. **`MarshalerTo`/`UnmarshalerFrom` 相比 v1 就有的 `Marshaler`/`Unmarshaler`，优势是什么？**
    答：`Marshaler.MarshalJSON` 需要先在内存里拼出一段完整的 `[]byte` 再拷贝进外层输出 buffer；`MarshalerTo.MarshalJSONTo(*jsontext.Encoder)` 可以直接对着 encoder 流式写 token，省掉这次中间分配和拷贝，同类型如果两个接口都实现了，`MarshalerTo`/`UnmarshalerFrom` 优先。

16. **如果想在不修改某个第三方类型定义的前提下，自定义它的 JSON 序列化方式，v1 和 v2 分别怎么做？**
    答：v1 做不到直接定制，只能给这个类型包一层自己的 wrapper 类型再给 wrapper 定义 `MarshalJSON`/`UnmarshalJSON`。v2 提供了 `json.MarshalFunc[T]`/`json.UnmarshalFunc[T]` 等函数，可以针对具体类型 `T` 注册编解码函数，通过 `Options`（如 `json.WithMarshalers(...)`）传给 `Marshal`/`Unmarshal`，不需要改动类型定义或额外包装类型。
