# Map

> 环境：`go version go1.26.3`。**Go 1.24 起，map 的底层实现从传统的 bucket（数组 + 溢出桶）设计切换为基于 [Swiss Table](https://abseil.io/about/design/swisstables) 的新实现**（`internal/runtime/maps`，`runtime/map.go` 只是一层薄封装）。本文的“底层原理”一节以该新实现（源码见 `internal/runtime/maps/{map,table,group}.go`）为准；1.24 之前版本是 `hmap` + `bucket`（tophash 数组 + overflow 链表指针）的经典设计，细节不同，2.6 节简要对比。

## 一、基础使用

### 1.1 声明与初始化

```go
var m1 map[string]int                  // nil map：len=0，可读（返回零值），写入会 panic
m2 := map[string]int{}                 // 空 map：非 nil，可读可写
m3 := map[string]int{"a": 1, "b": 2}   // 字面量初始化
m4 := make(map[string]int)             // 等价于 m2
m5 := make(map[string]int, 100)        // hint：预估元素个数，用于一次性分配足够容量，减少后续扩容次数
```

### 1.2 读写删除

```go
m := map[string]int{"a": 1}

v := m["a"]        // 存在：1
v2 := m["x"]        // 不存在：返回 value 类型的零值（int 是 0），不会 panic

v3, ok := m["x"]    // comma-ok 写法：ok == false 表示 key 不存在
if v4, ok := m["a"]; ok {
    _ = v4
}

m["b"] = 2          // 写入/更新
delete(m, "a")      // 删除；key 不存在时 delete 也不会 panic，是空操作

n := len(m)          // 当前 key-value 对个数
```

### 1.3 range 遍历

```go
for k, v := range m {
    _ = k
    _ = v
    // 遍历顺序是随机的：运行时故意在每次遍历时随机化起始位置
    // 不要依赖 map 遍历顺序；需要稳定/有序遍历，请把 key 取出来单独排序后再遍历
}
```

### 1.4 map 作为函数参数

```go
func modify(m map[string]int) {
    m["x"] = 1 // 对调用方可见，因为 map 变量本身就是一个指向底层结构体的指针
}
```

与 slice 不同，map 变量的值就是一个指针（指向 `internal/runtime/maps.Map`），传参/赋值只拷贝指针本身。因此**不需要**像 slice 那样写 `m = insert(m, k, v)` 才能让调用者感知到变化——不管是否触发扩容，指针指向的始终是同一个 `Map` 对象（扩容只是替换该对象内部的 table，不会替换 `Map` 指针本身）。

### 1.5 嵌套 map

```go
grid := make(map[string]map[string]int)
grid["a"] = make(map[string]int) // 内层 map 必须单独初始化
grid["a"]["x"] = 1
// grid["b"]["y"] = 1  // panic：grid["b"] 是 nil map（未显式初始化），写入 nil map 直接 panic
```

### 1.6 map 元素不可寻址

```go
type Point struct{ X, Y int }

m := map[string]Point{"a": {1, 2}}

// m["a"].X = 10   // 编译错误：cannot assign to struct field m["a"].X

p := m["a"]        // 正确写法一：取出整个 value（值拷贝）
p.X = 10
m["a"] = p          // 修改后整体写回

// 正确写法二：value 直接用指针类型，规避这个限制
m2 := map[string]*Point{"a": {1, 2}}
m2["a"].X = 10      // 合法，因为修改的是指针指向的对象，不是 map 元素本身
```

Go 规范禁止对 map 元素取地址（`&m["a"]` 编译不过），因为 map 内部元素随时可能因为扩容/分裂被搬到别的位置，编译器无法保证这个地址长期有效。

## 二、底层原理

### 2.1 数据结构（Go 1.24+，Swiss Table 实现）

核心概念（术语与源码注释一致）：

- **Slot**：存放一个 key/value 的存储位置。
- **Group**：`abi.MapGroupSlots`（=8）个 slot 组成一组，外加一个 8 字节的 **控制字（ctrl word）**。
- **控制字节（ctrl byte）**：每个 slot 对应 1 个控制字节，标记该 slot 是 empty / deleted(墓碑) / full；若 full，还存着该 key hash 的低 7 位（**H2**）。
- **H1 / H2**：一个 key 的 hash 值被拆成高 57 位（H1，决定探测的起始 group）和低 7 位（H2，存进控制字节，用于同一 group 内快速比对）。
- **Table**：一个完整的 Swiss Table，由若干个 group 组成，附带扩容相关的元数据。
- **Map**：顶层类型，由一个或多个 Table 组成；用 hash 的高位比特从 **directory**（table 数组）里选出具体用哪张 table。

简化后的结构关系：

```text
Map
 └── directory []*table        // 长度 = 1 << globalDepth
        └── table
              └── groups []group
                     ├── ctrl (8 bytes)      // 8 个控制字节
                     └── slots [8]{key, elem}
```

查找一个 key 的过程：`hash(key)` → 用 H1 定位起始 group → 用 group 的控制字（8 字节一次性比较，天然适合 SIMD）并行比对 8 个 slot 的 H2 → 有命中的再逐个用 `==` 精确比较 key（因为 H2 只有 7 位，约 0.7% 概率假阳，必须二次确认）→ 没有命中且 group 未满则确定不存在；group 已满则按探测序列（quadratic probing）去下一个 group 继续找。

### 2.2 负载因子与扩容

- 最大平均负载因子 `maxAvgGroupLoad = 7`（即每个 8-slot 的 group 最多用到 7 个 slot，留 1 个空位保证探测序列必然能终止，表不会 100% 填满）。
- 单张 table 的容量上限 `maxTableCapacity = 1024`（slot 数）。
- **未达到 1024 上限时**：grow 就是简单地把这张 table 换成一张容量翻倍的新 table，把所有元素按新的探测序列重新摆放。
- **达到 1024 上限后**：不再继续增大单表容量，而是把这张 table **分裂成两张**（各自容量仍是 1024），通过 **directory + extendible hashing** 来路由：用 hash 的高 `globalDepth` 位选 directory 里的 table；`localDepth` 记录某张 table 是在哪个深度被创建的；当要分裂的 table 的 `localDepth == globalDepth` 时才需要把 directory 翻倍（`globalDepth++`），否则只是多个 directory 项指向同一张新表。

这种设计的好处：**单次扩容的开销有上限**（最大也就是重排 1024 个 slot 所在的表），避免了一次性 rehash 一个几百万 key 的巨大 map 造成的长尾延迟，本质上是把“整体一次性扩容”变成了“可分片、可增量”的扩容。

### 2.3 删除与墓碑（tombstone）

- 若删除时所在 group 还有空位（未满），直接把该 slot 标记为 **empty**。
- 若所在 group 已经满了，删除后不能直接标记 empty——因为探测序列的规则是“遇到 empty 就停止查找”，如果这里错误地标成 empty，会导致原本探测到这个 group 之后、真正存在的 key 被“提前截断”找不到。所以这种情况下标记为 **deleted（墓碑）**，探测时跳过墓碑继续往后找，但插入时会优先复用墓碑位置。
- 墓碑只有在整张 table 触发 grow 时才会被彻底清空（grow 会重建所有 slot）。**因此频繁增删同一批 key，map 占用的内存不会自动收缩**，需要真正回收内存要重新建一个 map 并搬运仍需要的 key。

### 2.4 遍历的随机化与一致性保证

Go 语言规范只保证 map 遍历顺序“未指定”，运行时更进一步**主动随机化**：每次 `range` 都从随机的 directory 偏移、随机的 group/slot 偏移开始遍历（源码里体现为遍历开始时调用 `rand()`），杜绝了任何“看起来稳定、其实只是实现巧合”的顺序被开发者误当作契约依赖。同时保证：

1. 同一个 key 不会被遍历返回两次；
2. 遍历期间新增的 key 可能出现也可能不出现；
3. 遍历期间被修改的 key，返回的是最新值；
4. 遍历期间被删除的 key 一定不会出现；

即使遍历过程中 map 发生了 grow/split，运行时也通过记录“遍历到的位置”做了兼容处理，保证以上语义。

### 2.5 与旧版本（bucket + tophash）的对比

Go 1.24 之前，map 是经典的 `hmap`：一个 bucket 数组，每个 bucket 固定 8 个 slot + 一个 `tophash` 数组（存 hash 高 8 位用于快速比对）+ 一个指向 overflow bucket 的链表指针（bucket 满了就挂一个新 bucket）。它的问题是：overflow 链表退化后查找要挨个遍历链表，且不同 bucket 各自独立扩容概念弱，扩容通常是整个 hmap 一次性搬迁（用 `oldbuckets` 做渐进式迁移，读写时顺带迁移一部分，均摊开销）。

Swiss Table 版本用「control word 并行匹配 + 无 overflow 链表退化」取代了 tophash + overflow bucket，用「directory + 可分裂的多 table」取代了「整体 hmap 渐进式迁移」，在缓存局部性、查找的 SIMD 友好性、扩容的分片粒度上都有改进。对使用者而言，语义（nil map、并发安全性、key 限制、遍历随机化等）完全不变，只是内部实现更换。

## 三、常见陷阱

### 3.1 nil map 只读不可写

```go
var m map[string]int
_ = m["a"]        // 合法，返回零值 0
m["a"] = 1        // panic: assignment to entry in nil map
```

### 3.2 map 元素不可寻址

见 1.6，`&m[k]`、`m[k].Field = x`（value 是 struct 时）都无法通过编译；需要整体取出再写回，或者 value 直接用指针类型。

### 3.3 遍历顺序不确定

不要依赖任何一次运行观察到的“规律”（比如插入顺序、字母序），运行时会主动打乱起始位置。需要确定顺序时，显式收集 key 后排序：

```go
keys := make([]string, 0, len(m))
for k := range m {
    keys = append(keys, k)
}
sort.Strings(keys)
```

### 3.4 并发读写不安全，且不是数据错乱而是直接崩溃

```go
// 一个 goroutine 写，一个 goroutine 读/写同一个 map，没有加锁保护
// fatal error: concurrent map writes
// fatal error: concurrent map read and map write
```

这类 `fatal error` 是运行时主动检测到后**直接终止进程**，属于 `fatal`，**不能用 `recover` 捕获**（有别于 `panic`）。多个 goroutine 只读不写是安全的；一旦涉及写，必须用 `sync.RWMutex` 保护，或改用 `sync.Map`（读多写少、key 集合稳定的场景更划算）。

### 3.5 key 类型必须可比较（comparable）

```go
// var m map[[]int]string     // 编译错误：invalid map key type []int
// var m map[map[string]int]string // 编译错误
// var m map[func()]string         // 编译错误
```

slice、map、function 都是不可比较类型，不能直接做 key；包含这些类型字段的 struct 同样不能做 key（编译期报错）。数组、可比较的 struct（字段均可比较）都可以做 key，比较时是**逐字段值比较**。

### 3.6 NaN 作为 key 的坑

```go
m := map[float64]string{}
nan := math.NaN()
m[nan] = "a"
m[nan] = "b"          // 因为 NaN != NaN，每次都被当成“新 key”插入，而不是覆盖
fmt.Println(len(m))    // 2，而不是 1
fmt.Println(m[nan])    // ""，永远查不到——查找时的 == 比较同样是 false
```

### 3.7 delete 不会收缩内存

即使 delete 掉了 map 里几乎所有的 key，`runtime` 也不会主动缩小底层 table 占用的内存（墓碑只在 grow 时清理，见 2.3）。长期持有一个“曾经很大、现在很空”的 map，会造成事实上的内存浪费；确实需要收紧内存时，需新建一个 map 把仍需要的数据搬过去。

### 3.8 零值和“不存在”容易混淆

```go
m := map[string]int{"a": 0}
v := m["a"]   // 0
v2 := m["x"]  // 也是 0，但 "x" 根本不存在
```

只要 value 类型的零值本身是一个有效业务值（0、""、false 等），就必须用 `v, ok := m[k]` 区分“真实存的零值”和“key 不存在”，不能只看返回值。

### 3.9 大 struct 做 key 的性能代价

struct 做 key 时每次查找/插入都要做逐字段比较，字段越多、越大，比较开销越高；如果只是需要唯一标识，优先用简单类型（string、整型、小 struct）做 key，或者对大 struct 做哈希/取关键字段拼出简单 key。

## 四、常见面试题

**1. map 和 slice 在“传参语义”上的本质区别？**
map 变量本身就是一个指针（指向 `internal/runtime/maps.Map`），传参/赋值只拷贝这个指针；slice 是 `{array, len, cap}` 三元组的值拷贝。这也是为什么 `func f(m map[K]V) { m[k]=v }` 对调用者可见，而 slice 扩容后必须 `s = append(s, x)` 才能让调用者感知变化。

**2. Go 1.24+ map 的底层结构是什么？**
基于 Swiss Table：`Map` 持有一个 `directory`（table 指针数组），每个 `table` 由若干个 `group` 组成，每个 group 是 8 个 slot + 一个 8 字节控制字（记录每个 slot 的 empty/deleted/full 状态和 key hash 的低 7 位）。详见 2.1。

**3. map 的扩容机制/负载因子是怎样的？**
每个 group 最大平均负载是 7/8（`maxAvgGroupLoad=7`，`MapGroupSlots=8`）。单表容量不超过 1024（`maxTableCapacity`）之前，grow 直接把该 table 容量翻倍重排；超过 1024 后不再继续增大单表，而是分裂成两张 1024 容量的 table，通过 directory 的 extendible hashing（`globalDepth`/`localDepth`）路由。详见 2.2。

**4. 为什么不能对 map 的元素取地址（`&m[k]`）？**
map 元素在扩容/分裂时可能被搬到新的 slot 位置，语言不保证这个地址长期稳定，所以直接禁止，编译期报错。需要修改元素字段时，要么整体取出再写回，要么 value 用指针类型。

**5. 为什么 map 遍历顺序是随机的？**
Go 规范本身只说顺序“未指定”，运行时为了防止开发者依赖某个具体实现细节（比如插入顺序恰好等于遍历顺序），故意在每次遍历时随机化起始的 group/slot 偏移。

**6. 并发读写 map 为什么是直接 crash 而不是数据错乱？**
运行时在读写路径上都会检测“是否有其他 goroutine 正在写”的标记，一旦发现冲突就主动调用 `fatal()` 终止进程（`concurrent map writes` / `concurrent map read and map write`），这是有意为之的快速失败设计，避免并发场景下 map 内部指针结构被破坏后产生更难排查的内存错误。注意这是 `fatal error`，`recover` 无法捕获。

**7. 哪些类型不能作为 map 的 key？为什么？**
slice、map、function 以及包含它们的 struct/数组不能作为 key，因为 Go 规范要求 key 类型必须是 `comparable`（可用 `==` 比较），而这几种类型语义上不支持真正意义的相等比较（或比较代价/语义模糊，如 slice 的引用语义）。

**8. `NaN` 作为 map key 会发生什么？**
可以插入多个 `NaN` key 而不会互相覆盖（因为 `NaN != NaN`，每次都被判定为不同的 key），但之后用 `NaN` 去查找永远查不到，因为查找同样要做 `==` 比较。

**9. nil map 和 `make` 出来的空 map 有什么区别？**
两者 `len` 都是 0，读取都返回零值；区别在于 nil map 不能写入（写入 panic），`make`/字面量创建的空 map 可以正常写入。此外 `json.Marshal` 对 nil map 编码为 `null`，对空 map 编码为 `{}`。

**10. delete 之后 map 占用的内存会释放/收缩吗？**
不会自动收缩。被删除位置在 group 未满时标记为 empty、group 已满时标记为墓碑（deleted），墓碑只有在整表触发 grow 时才会被清理；如果之后不再插入新数据触发 grow，这些空间会一直占用着，直到整个 map 被回收。真正需要收紧内存要显式建新 map 搬运。

**11. 如何实现按 key 有序地遍历 map？**
map 本身不提供有序遍历；标准做法是把 key 取出放进 slice、排序后再按顺序访问 map，例如用 `sort.Strings`/`sort.Slice` 对 key 排序。

**12. `v := m[k]` 和 `v, ok := m[k]` 有什么区别？什么场景必须用后者？**
前者不存在时返回零值，无法区分“真实存的零值”和“key 不存在”；后者的 `ok` 明确标识 key 是否存在。当 value 类型的零值本身是合法业务值（如 `int` 的 0、`string` 的空串）时，必须用 comma-ok 写法才能正确判断。

**13. 为什么 map 传参不需要像 slice 那样 `m = insert(m, k, v)`？**
因为 map 变量本身是指针语义（指向同一个 `Map` 结构体），无论内部 table 如何扩容/替换，调用者手里的指针始终指向同一个对象，运行时内部的写入天然对调用者可见；而 slice 是值语义的 header，扩容会产生新的 `array/len/cap`，必须通过返回值赋值回调用者才能感知。

**14. Go 1.24 之前的 map（hmap + bucket）和现在的 Swiss Table 实现核心区别是什么？**
旧版本是 bucket 数组，每个 bucket 8 个 slot + `tophash` 数组 + 指向 overflow bucket 的链表指针，扩容是整个 hmap 通过 `oldbuckets` 做渐进式搬迁；新版本用 control word 做 8 slot 一组的并行匹配（SIMD 友好，无 overflow 链表退化问题），扩容通过 directory + 可分裂的多个 table 做分片式增量扩容，单次 grow 开销有上限。对使用者语义完全一致，只是内部实现更高效。详见 2.5。

**15. 什么场景该用加锁的普通 map，什么场景该用 `sync.Map`？**
`sync.Map` 针对“key 集合相对稳定、大量并发读、少量并发写”或“多个 goroutine 各自读写不相交 key 集合”的场景做了优化；对于读写都频繁、key 集合变化大的通用场景，`sync.RWMutex` + 普通 map 通常更简单也更快（`sync.Map` 有额外的内部结构开销，并非总是比加锁 map 快）。
