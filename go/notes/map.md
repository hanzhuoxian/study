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

两个关键常量：

- 最大平均负载因子 `maxAvgGroupLoad = 7`：这是**整张 table 的平均阈值，不是每个 group 的硬上限**。整表最多填到 `capacity × 7/8`（即平均每个 8-slot 的 group 用 7 个）就触发扩容——单个 group 完全可以被填满 8 个，满了的元素靠探测溢出到相邻 group；留出的空位是**全表层面**的，只要整表不 100% 填满，遍历所有 group 的探测序列就必然能遇到空槽而终止（否则会死循环）。触发 rehash 的判定是 `used + tombstones > 7/8 × capacity`（把墓碑也算进去，避免表被墓碑占满）。**特例**：单 group 的小表能填到 `capacity - 1`（8 个填 7 个），此时「留 1 个空位」才是字面意义上的单组留一空。
- 单张 table 的容量上限 `maxTableCapacity = 1024`（slot 数）。

**扩容分为两种算法，分界线是单张 table 是否已达到 1024 上限：**

#### 算法一：单表翻倍重排（table growth）

- **触发条件**：某张 table 负载超过 `maxAvgGroupLoad`，且当前容量 **< 1024**。
- **做法**：分配一张容量**翻倍**的新 table，把旧表所有元素按新探测序列重新摆放，顺带清掉墓碑，再替换 `Map` 内部指向该 table 的指针，旧表回收。
- 本质就是经典的“哈希表满了就整表 rehash”，只不过对象是一张 ≤1024 slot 的小表。

#### 算法二：分裂 + 可扩展哈希（table split + extendible hashing）

- **触发条件**：table 已达到 **1024 上限**还需继续增长，此时不再增大单表。
- **做法**：把这张 table **分裂成两张**（各自容量仍是 1024），元素按 hash 的某一位比特分流；通过 **directory（table 指针数组）** 路由——用 hash 的高 `globalDepth` 位在 directory 里选具体 table。
- **`globalDepth` / `localDepth`**：`localDepth` 记录某张 table 是在哪个深度被创建的。分裂时：
  - `localDepth == globalDepth`：directory 空间不够区分新旧表，需把 **directory 翻倍**（`globalDepth++`）。
  - `localDepth < globalDepth`：directory 已够大，只需让原本指向老表的一半 directory 项改指向新表，**不用翻倍 directory**。

#### 为什么分两种

| | 算法一（翻倍重排） | 算法二（分裂 + directory） |
|---|---|---|
| 适用 | 小 map（table < 1024） | 大 map（table 满 1024） |
| 单次开销 | 重排整张表 | **有上限**：最多重排 1024 个 slot |
| 目的 | 简单高效 | 避免几百万 key 的 map 一次性 rehash 造成的长尾延迟 |

核心动机：把“整个巨大 map 一次性扩容”拆成“一张张 1024 的小表可分片、可增量地扩容”，让**任何单次扩容的最坏耗时都封顶**，本质上是把“整体一次性扩容”变成了“可分片、可增量”的扩容。

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

### 2.6 关键源码解读

> 源码位置：`$GOROOT/src/internal/runtime/maps/`（本机为 `~/.asdf/installs/golang/1.26.3/go/src/internal/runtime/maps/`），核心文件 `map.go` / `table.go` / `group.go`。`runtime/map.go` 只是编译器与本包之间的薄封装桥。

本节按 map 的**生命周期顺序**解读：创建 → 插入 → 查找 → 扩容。整体调用链总览：

```text
make(map[K]V, hint)                     // 编译器改写成下面的 runtime 调用
   └─ runtime.makemap(t, hint, m)       // runtime/map.go，薄封装
        └─ maps.NewMap(t, hint, m, ...) // ① 创建：算容量、建 directory/table

m[k] = v                                // 编译器改写
   └─ maps.Map.PutSlot                  // ② 插入：三段式
        ├─ growToSmall   (nil → 单 group)
        ├─ putSlotSmall  (元素 < 8，装进单 group)
        ├─ growToTable   (第 8 个元素，小 map 升级为完整 table)
        └─ table.PutSlot (完整 table 插入；满了触发 rehash)
             └─ table.rehash            // ④ 扩容：grow / split 二选一

v := m[k]                               // maps.Map.Get → getWithKey → ③ 查找
```

#### ① 顶层数据结构 `Map` / `table` / `group`（map.go:195、table.go:33）

```go
type Map struct {
    used        uint64        // 元素个数，必须是第一个字段：len() 直接读它
    seed        uintptr       // 每个 map 独立的随机哈希种子
    dirPtr      unsafe.Pointer // directory：*[1<<globalDepth]*table；小 map 时直接指向单个 group
    dirLen      int           // directory 长度；== 0 表示处于“小 map”优化态
    globalDepth uint8         // directory 索引用的比特数，dirLen == 1<<globalDepth
    globalShift uint8         // 64 - globalDepth，取 hash 高位时用
    writing     uint8         // 写入中标志（XOR 1 翻转），并发写检测靠它
    tombstonePossible bool    // 该 map 是否可能存在墓碑
    clearSeq    uint64        // Clear 计数器，遍历期间检测 clear
}

type table struct {
    used       uint16 // 本表元素个数
    capacity   uint16 // 本表总 slot 数（2^N），≤ maxTableCapacity(1024)
    growthLeft uint16 // 还能再插几个才需 rehash（= 7/8·capacity - used - tombstones）
    localDepth uint8  // 本表是在哪个深度被创建的（extendible hashing 用）
    index      int    // 在 directory 里的首个下标；-1 表示已失效
    groups     groupsReference // 一段连续的 group 数组
}
```

`used` 放 `Map` 第一个字段是硬约束——`len(m)` 由编译器直接读该偏移，`cmd/compile/internal/reflectdata/map.go` 依赖这个布局。`growthLeft` 是插入路径判断是否要扩容的核心计数器（见 ②④）。

#### ② 创建 map：`make` → `makemap` → `NewMap`（runtime/map.go:62、map.go:264）

`make(map[K]V, hint)` 是编译器内建，会被改写成对 `runtime.makemap`（三个变体之一）的调用，参数由编译器**合成**：

```go
// runtime/map.go：薄封装，转发给真正实现
func makemap(t *abi.MapType, hint int, m *Map) *maps.Map {
    if hint < 0 { hint = 0 }
    return maps.NewMap(t, uintptr(hint), m, maxAlloc)
}
```

- `t *abi.MapType`：编译器为 `map[K]V` 这个**具体类型**静态生成的类型描述符（含 `Hasher`、`Key.Equal`、各字段大小、`IndirectKey/Elem` 等），只读数据段里。
- `hint`：`make` 的第二个参数（预估元素数），省略则为 0。
- `m *Map`：**逃逸分析**产物——若 map 不逃逸，编译器在栈上预留 `Map`（甚至第一个 group）并传地址进来，省一次堆分配；逃逸则传 nil。

真正干活的 `NewMap`：

```go
func NewMap(mt *abi.MapType, hint uintptr, m *Map, maxAlloc uintptr) *Map {
    if m == nil { m = new(Map) }
    m.seed = uintptr(rand())                 // 每个 map 独立随机种子（遍历随机化、防哈希碰撞攻击）

    if hint <= abi.MapGroupSlots {           // hint ≤ 8：小 map，压根不预分配
        return m                             // 保持 dirLen==0，等第一次写入再懒分配（见 ③）
    }

    // 完整 map：把 hint 换算成 slot 容量
    targetCapacity := (hint * abi.MapGroupSlots) / maxAvgGroupLoad // = hint*8/7
    if targetCapacity < hint { return m }    // uintptr 乘法溢出保护 → 放弃预分配，返回空 map

    dirSize := (uint64(targetCapacity) + maxTableCapacity - 1) / maxTableCapacity
    dirSize, overflow := alignUpPow2(dirSize)                     // directory 长度向上取 2 的幂
    // ... 再做若干溢出 / 超 maxAlloc 检查，任一失败都 return m（退化成空 map）...

    m.globalDepth = uint8(sys.TrailingZeros64(dirSize))
    m.globalShift = depthToShift(m.globalDepth)
    directory := make([]*table, dirSize)
    for i := range directory {
        directory[i] = newTable(mt, uint64(targetCapacity)/dirSize, i, m.globalDepth)
    }
    m.dirPtr = unsafe.Pointer(&directory[0])
    m.dirLen = len(directory)
    return m
}
```

两个要点：

1. **`hint*8/7` 换算**：因为负载因子只有 7/8（2.2 节），要「无扩容装下 hint 个元素」就得预留 `hint × 8/7` 的容量，只分配 `hint` 个会一装就满。
2. **小 map 懒分配**：`hint ≤ 8` 直接返回，连 group 都不分配——省掉「创建后从没写入」场景的开销，第一次写入时才由 `growToSmall` 补上（见 ③）。所有异常路径（溢出、超内存上限）也都是 `return m` 退化成空 map，而非 panic——hint 只是提示。

#### ③ 插入：`PutSlot` 三段式路径（map.go:486）

`m[k]=v` 改写成 `PutSlot`（返回 elem 槽地址，编译器再往里写 value）。它按 map 当前规模分三段走：

```go
func (m *Map) PutSlot(typ *abi.MapType, key unsafe.Pointer) unsafe.Pointer {
    if m.writing != 0 { fatal("concurrent map writes") }
    hash := typ.Hasher(key, m.seed)
    m.writing ^= 1

    if m.dirPtr == nil {           // 第一次写入一个从没分配过的 map
        m.growToSmall(typ)         //   → 分配「单个 group」（8 slot），dirLen 仍为 0
    }
    if m.dirLen == 0 {             // 处于小 map 态
        if m.used < abi.MapGroupSlots {
            elem := m.putSlotSmall(typ, hash, key)  // 段①：装进单 group，无探测无墓碑
            m.writing ^= 1
            return elem
        }
        m.growToTable(typ)         // 段②：第 8 个元素，把单 group 升级为完整 table
    }
    for {                          // 段③：完整 table
        idx := m.directoryIndex(hash)               // hash 高位选 table
        elem, ok := m.directoryAt(idx).PutSlot(typ, m, hash, key)
        if !ok { continue }        // ok==false：表刚被 split，重新选表再来
        m.writing ^= 1
        return elem
    }
}
```

- **段①小 map（`putSlotSmall`）**：只有一个 group，`matchH2` 找现有 key（有则更新），否则 `matchEmptyOrDeleted` 找空位插入。小 map 没有探测序列、**永远没有墓碑**（删除直接置空）。
- **`growToSmall`**（map.go:598）：`newGroups(typ, 1)` 只分配一个 group、`setEmpty()` 把 8 个控制字节全置空。
- **`growToTable`**（map.go:608）：当第 8 个元素来时，新建一张 `2*MapGroupSlots`（=16 slot）的 table，把原单 group 的元素 `uncheckedPutSlot` 搬进去，建一个长度为 1 的 directory 挂上，`dirLen` 变 1，从此进入完整 table 世界。
- **段③完整 table（`table.PutSlot`，table.go:266）**：真正的探测插入，逻辑见下。

`table.PutSlot` 里决定「扩容」的关键片段：

```go
// ...沿探测序列找到 key（更新）或找到落脚的空/墓碑位...
if t.growthLeft == 0 {
    t.pruneTombstones(typ, m)   // 先试着清理墓碑腾地方
}
if t.growthLeft > 0 {
    // 有空间：写入 key/elem，设置控制字节，growthLeft--、used++、m.used++
    return slotElem, true
}
t.rehash(typ, m)               // 实在没空间了 → 扩容
return nil, false              // 通知上层 PutSlot 重新选表重试（表可能已 split）
```

即：`growthLeft`（初值 `7/8·capacity`）耗尽时，先尝试 `pruneTombstones` 回收墓碑；仍不够才 `rehash`。插入时若沿途遇到过墓碑，会优先复用墓碑位（不消耗 `growthLeft`）——对应 2.3 节「插入优先复用墓碑」。

#### ④ hash 拆分 H1 / H2（map.go:183）

查找和插入都要先把 hash 拆成两段：

```go
func h1(h uintptr) uintptr { return h >> 7 }   // 高 57 位：定位起始 group
func h2(h uintptr) uintptr { return h & 0x7f }  // 低 7 位：存进控制字节做快速比对
```

#### ⑤ 控制字节与 SIMD 并行匹配（group.go:14, 152）

```go
const (
    ctrlEmpty   ctrl = 0b10000000  // 空槽：最高位 1
    ctrlDeleted ctrl = 0b11111110  // 墓碑
    // full 槽：最高位 0，低 7 位 = 该 key 的 H2
    bitsetLSB = 0x0101010101010101
    bitsetMSB = 0x8080808080808080
)

// 一次比较整组 8 个控制字节，返回“H2 命中”的 slot 位集
func ctrlGroupMatchH2(g ctrlGroup, h uintptr) bitset {
    v := uint64(g) ^ (bitsetLSB * uint64(h))       // 逐字节异或目标 H2
    return bitset(((v - bitsetLSB) &^ v) & bitsetMSB) // 字节为 0（即命中）的位被置 1
}
```

这段无分支位运算就是 Swiss Table 的精髓：一条指令级别的操作同时比对 8 个 slot（AMD64 上进一步被替换成 SSE 内建指令），取代了旧版逐个遍历 `tophash` 数组。命中后再用 `match.first()`（本质是 `TrailingZeros64`）取最低命中位。

**对比旧版（Go 1.23 及以前，`runtime/map.go`，1.26 已删除）的查找热路径 `mapaccess1`：**

```go
// 旧版数据结构：hmap + bmap（bucket）
type hmap struct {
    count      int            // 元素个数，== len()，也必须是第一个字段
    B          uint8          // 桶数量 = 2^B
    hash0      uint32         // 哈希种子
    buckets    unsafe.Pointer // 2^B 个 bmap
    oldbuckets unsafe.Pointer // 渐进式扩容时的旧桶数组
    // ...
}
type bmap struct {
    tophash [bucketCnt]uint8  // bucketCnt=8，存每个 key hash 的“高 8 位”
    // 后面紧跟 8 个 key、8 个 elem、1 个 overflow 指针（溢出桶链表）
}

// tophash 取 hash 高 8 位（作用类似新版的 H2，但是 8 位且没有 SIMD 并行）
func tophash(hash uintptr) uint8 {
    top := uint8(hash >> (goarch.PtrSize*8 - 8))
    if top < minTopHash { top += minTopHash } // 低值预留给“迁移状态”标记
    return top
}

func mapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {
    // ... nil / 并发写检测 ...
    hash := t.hasher(key, uintptr(h.hash0))
    m := bucketMask(h.B)
    b := (*bmap)(add(h.buckets, (hash&m)*uintptr(t.bucketsize))) // 低位选桶
    // ... 若正在扩容，可能要去 oldbuckets 找 ...
    top := tophash(hash)
bucketloop:
    for ; b != nil; b = b.overflow(t) {        // ← 外层：遍历 overflow 链表（退化点）
        for i := uintptr(0); i < bucketCnt; i++ { // ← 内层：逐个字节比 tophash
            if b.tophash[i] != top {
                if b.tophash[i] == emptyRest {
                    break bucketloop              // 后面全空，提前结束
                }
                continue                          // 一个一个跳
            }
            k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
            if t.indirectkey() { k = *((*unsafe.Pointer)(k)) }
            if t.key.equal(key, k) {              // tophash 命中后再精确比 key
                e := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.elemsize))
                if t.indirectelem() { e = *((*unsafe.Pointer)(e)) }
                return e
            }
        }
    }
    return unsafe.Pointer(&zeroVal[0]) // 没找到返回零值
}
```

两版对照：

| | 旧版 `bmap.tophash`（1.23-） | 新版 `ctrlGroup.matchH2`（1.24+） |
|---|---|---|
| 快速比对位 | hash 高 8 位，存在 `[8]uint8` | hash 低 7 位（H2），存在 8 字节控制字 |
| 组内比对方式 | `for i:=0;i<8;i++` **逐字节** `!=` 比较 | 一条位运算/SSE 指令 **8 个并行** |
| 桶满后的处理 | 挂 **overflow 桶链表**，查找时 `b = b.overflow(t)` 顺链遍历（可能退化成长链） | 无 overflow，按二次探测走**下一个 group**（表内寻址，缓存局部性好） |
| 结束条件 | `tophash[i] == emptyRest` | `matchEmpty() != 0` |

关键差异就在那两层 `for` 循环：旧版内层是**逐字节比 tophash**、外层是**顺着 overflow 指针遍历链表**；新版把内层 8 次比较压成一条无分支位运算，把外层链表换成表内的二次探测。这就是 2.5 节所说「control word 并行匹配 + 无 overflow 链表退化」的代码级体现。

#### ⑥ 查找：小 map 直查 + 完整表二次探测（map.go:407/446、table.go:164）

查找入口 `getWithKey` 先按 `dirLen` 分流：`dirLen==0` 走小 map 直查，否则用 hash 高位选 table 再探测。

**小 map 直查（`getWithKeySmall`）**——只有一个 group，无 directory、无探测序列：

```go
func (m *Map) getWithKeySmall(...) {
    g := groupReference{data: m.dirPtr}
    match := g.ctrls().matchH2(h2(hash))
    for match != 0 {
        i := match.first()
        if typ.Key.Equal(key, g.key(typ, i)) { return ...true }
        match = match.removeFirst()
    }
    return nil, nil, false // 单 group 无需探测、无需查 empty
}
```

**完整表查找（`table.getWithKey`）——二次探测 + H2 过滤**：

```go
seq := makeProbeSeq(h1(hash), t.groups.lengthMask)
for ; ; seq = seq.next() {
    g := t.groups.group(typ, seq.offset)
    match := g.ctrls().matchH2(h2Hash)
    for match != 0 {                 // H2 命中的 slot 逐个精确比较
        i := match.first()
        if typ.Key.Equal(key, g.key(typ, i)) { return ...true }
        match = match.removeFirst()
    }
    if g.ctrls().matchEmpty() != 0 { // 遇到空槽 = 探测序列到头，确定不存在
        return nil, nil, false
    }
}
```

探测序列是**三角数二次探测**（table.go:1261）：`p(i) = hash + (i²+i)/2 mod (mask+1)`，当 group 数是 2 的幂时保证不重不漏遍历每个 group。注意墓碑 `ctrlDeleted` 既不匹配 H2、又不匹配 empty，所以探测“跳过墓碑继续走”天然成立——这正是 2.3 节“group 满时删除只能标墓碑”的代码级原因。

#### ⑦ 扩容的两种算法 —— `rehash` 的分叉（table.go:1145）

对应 2.2 节，两种算法就在 `rehash` 一个 if 里分叉：

```go
func (t *table) rehash(typ *abi.MapType, m *Map) {
    newCapacity := 2 * t.capacity
    if newCapacity <= maxTableCapacity { // maxTableCapacity = 1024
        t.grow(typ, m, newCapacity)      // 算法一：单表翻倍重排
        return
    }
    t.split(typ, m)                      // 算法二：分裂 + 可扩展哈希
}
```

`split`（table.go:1179）把 `localDepth++`，按“从高位数第 localDepth 位”把元素分流到 left/right 两张新表，再 `installTableSplit` 挂进 directory（必要时翻倍 directory）：

```go
mask := localDepthMask(localDepth)   // 1 << (64 - localDepth)
if hash&mask == 0 { newTable = left } else { newTable = right }
newTable.uncheckedPutSlot(typ, hash, key, elem) // 已知不重复，免去查重
```

`grow` 和 `split` 都通过 `uncheckedPutSlot` 逐个重放元素，重放过程顺带丢弃了所有墓碑（源码注释解释：之所以不做 Abseil 那种“原地 rehash”消除墓碑，是因为会打乱 slot 顺序、破坏遍历语义，见 table.go:1146）。

#### ⑧ 并发写检测（map.go:487, 495）

```go
func (m *Map) PutSlot(...) {
    if m.writing != 0 { fatal("concurrent map writes") }
    hash := typ.Hasher(key, m.seed) // 先算 hash（可能 panic），再置标志
    m.writing ^= 1                  // 用 XOR 翻转而非直接置 1
    ...
    if m.writing == 0 { fatal("concurrent map writes") } // 收尾再查一次
    m.writing ^= 1
}
```

用 **XOR 翻转**而不是简单赋值，是为了在多个并发写入者互相踩踏时，让 `writing` 更容易落到非预期值，从而提高冲突被检测到的概率。读路径（`getWithKey`）同样会检查 `m.writing != 0` 并 `fatal("concurrent map read and map write")`。这就是 3.4 节“并发读写是 `fatal` 不可 `recover`”的实现来源。

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
