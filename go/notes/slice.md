# Slice

> 环境：`go version go1.26.3`。扩容参数、`runtime.slice` 结构体等均以该版本源码（`runtime/slice.go`）为准，不同版本可能有细节差异（尤其是 1.18 前后的扩容阈值/因子）。

## 一、基础使用

### 1.1 声明与初始化

```go
var s1 []int              // nil slice：len=0, cap=0, 底层指针为 nil
s2 := []int{}              // 空 slice：len=0, cap=0，但底层指针非 nil（指向 zerobase）
s3 := []int{1, 2, 3}       // 字面量初始化
s4 := make([]int, 5)       // len=5, cap=5，元素置零
s5 := make([]int, 3, 10)   // len=3, cap=10
s6 := make([]int, 0, 10)   // 常用于“预分配容量，逐步 append”的场景
```

### 1.2 从数组/slice 切出新 slice

```go
arr := [5]int{0, 1, 2, 3, 4}
s := arr[1:3]     // len=2, cap=4（从下标1到数组末尾 4）：[1 2]
s2 := arr[1:3:4]  // 三索引切片 low:high:max，len=2, cap=3（4-1）
```

- `s[low:high]`：`len = high-low`，`cap = cap(s)-low`（默认延伸到底层数组末尾）。
- `s[low:high:max]`（三索引/full slice expression）：显式限制 `cap = max-low`，常用于防止 `append` 污染共享的底层数组（见 4.3）。

### 1.3 append / copy

```go
s := make([]int, 0, 2)
s = append(s, 1)       // append 必须接收返回值，因为底层数组可能被重新分配
s = append(s, 2, 3)    // 支持多个变长参数
s = append(s, s2...)   // 追加另一个 slice，需要展开（...）

dst := make([]int, len(s))
n := copy(dst, s)      // 返回实际拷贝的元素个数 = min(len(dst), len(src))
```

### 1.4 range 遍历

```go
for i, v := range s {
    // v 是元素的值拷贝，修改 v 不会影响 s[i]
    s[i] = v * 2 // 要修改原元素必须通过下标
}
```

### 1.5 多维 slice

```go
grid := make([][]int, rows)
for i := range grid {
    grid[i] = make([]int, cols) // 每一行需要单独分配，二维 slice 不是连续内存
}
```

### 1.6 字符串与 slice 互转

```go
b := []byte("hello")   // 会发生一次内存拷贝
r := []rune("你好")     // 按 UTF-8 解码后拷贝为 rune slice（每个 rune 4 字节）
s := string(b)         // 同样会拷贝
```

## 二、底层原理

### 2.1 数据结构

`slice` 在运行时其实就是一个「三元组」的结构体（`runtime/slice.go`）：

```go
type slice struct {
    array unsafe.Pointer // 指向底层数组的指针
    len   int             // 当前长度
    cap   int             // 底层数组从 array 起可用的最大长度
}
```

- **值类型是这个三元组本身**，不是底层数组。函数传参、赋值、range 迭代变量赋值，拷贝的都只是这 24 字节（64 位下 8+8+8）的 header，底层数组不会被拷贝。
- 正因为共享底层数组指针，**多个 slice 可能指向同一块内存**，通过其中一个修改元素会影响所有共享该数组的 slice（前提是索引在各自的 `[0, len)` 范围内）。

### 2.2 slice 与 array 的关系

- `array` 是值类型，长度是类型的一部分（`[5]int` 和 `[3]int` 是不同类型），赋值/传参会整体拷贝。
- `slice` 是对某个数组一段连续区间的「视图」（descriptor），本身不持有数据。`make([]T, n)` 只是运行时帮你分配了一个匿名数组，再返回指向它的 slice。

### 2.3 append 与扩容机制

`append` 的伪代码语义：

```go
if len(s)+n <= cap(s) {
    // 容量够，直接在原底层数组后面写入，返回的 slice 复用同一底层数组
} else {
    // 容量不够，调用 runtime.growslice 分配一块更大的新数组
    // 把旧元素拷贝过去，再追加新元素，返回指向新数组的 slice
}
```

**扩容策略**（`runtime.nextslicecap`，决定“需要多大的新容量”）：

```go
func nextslicecap(newLen, oldCap int) int {
    newcap := oldCap
    doublecap := newcap + newcap
    if newLen > doublecap {
        return newLen // 一次性追加很多元素时，直接按需要的长度扩容
    }

    const threshold = 256
    if oldCap < threshold {
        return doublecap // 旧容量 < 256：翻倍（2x）
    }
    for {
        // 旧容量 >= 256：从 2x 逐步过渡到 1.25x，避免大 slice 扩容浪费太多内存
        newcap += (newcap + 3*threshold) >> 2
        if uint(newcap) >= uint(newLen) {
            break
        }
    }
    return newcap
}
```

要点：

1. **不是简单的“无脑翻倍”**：旧容量小于 256 时是 2 倍增长；达到 256 后，增长因子逐渐降到约 1.25 倍，是空间和拷贝次数之间的折中（1.18 版本之后才是这个平滑过渡策略，之前是 1024 为界直接从 2x 切到 1.25x）。
2. `nextslicecap` 算出的只是「期望容量」，真正的最终容量还要经过 `roundupsize`：Go 的内存分配器（mallocgc）会把申请的字节数对齐到预定义的 **size class**，所以实际 `cap` 往往比期望值略大，且和元素大小相关（这就是为什么同样 append 逻辑，不同元素类型算出来的 cap 增长曲线不完全一致）。
3. 扩容会导致：**新的底层数组、地址变化**，因此扩容之后，原来共享同一底层数组的其它 slice **不再和新 slice 共享内存**（见 4.3 的例子）。
4. `append` 必须用返回值赋值回原变量（`s = append(s, x)`），因为 slice header 本身是值传递，扩容后新的 `array/len/cap` 只有通过返回值才能让调用者感知到。

### 2.4 copy 的语义

`copy(dst, src)` 按 `min(len(dst), len(src))` 逐元素拷贝（对 `[]byte` 和 `string` 有特殊优化，用 `memmove`），不会自动扩容 `dst`，也允许 `dst` 和 `src` 有重叠（内部用 `memmove` 处理重叠区域，语义等价于先读完源再写，不会像手写循环那样在重叠时踩坏数据）。

## 三、常见陷阱

### 3.1 共享底层数组导致的“意外修改”

```go
a := []int{1, 2, 3, 4, 5}
b := a[1:3]   // b = [2 3]，与 a 共享底层数组
b[0] = 99     // a 变为 [1 99 3 4 5]
```

### 3.2 append 引发的“污染”

```go
a := make([]int, 3, 5)   // [0 0 0], cap=5
b := a[:2]                // [0 0]，与 a 共享底层数组，cap(b)=5
b = append(b, 100)        // 容量够用，直接写入底层数组第 3 个位置
fmt.Println(a)            // [0 0 100] —— a 被 b 的 append 污染了！
```

这是最经典的 slice 陷阱：**只要 `cap` 够用，append 就不会分配新数组，而是直接覆盖底层数组中“看起来空闲、实际上属于别的 slice 视图”的内存**。

解决办法：

- 使用三索引切片显式限制 `cap`，逼迫任何在该切片上的 append 都触发扩容：`b := a[:2:2]`，此时 `cap(b)==2==len(b)`，append 时必然重新分配，不会影响 `a`。
- 或者需要独立数据时显式 `copy`。

### 3.3 大 slice 截取小 slice 造成的内存泄漏

```go
func getFirstMB(data []byte) []byte {
    return data[:1<<20] // 只是截取视图，底层依然引用整个大数组
}
```

即使只用到了前 1MB，只要这个小 slice 还存活，GC 就无法回收背后的大数组。正确做法是显式 `copy` 出一份独立数据：

```go
func getFirstMB(data []byte) []byte {
    out := make([]byte, 1<<20)
    copy(out, data)
    return out
}
```

### 3.4 nil slice 与空 slice

```go
var a []int          // nil，a == nil 为 true
b := []int{}         // 非 nil，len(b)==0
```

- 两者 `len`、`cap` 都是 0，`append` 行为一致，绝大多数场景可以互换使用。
- 区别体现在：`== nil` 判断、`json.Marshal`（`nil` 编码为 `null`，`[]int{}` 编码为 `[]`）等场景。

### 3.5 并发不安全

多个 goroutine 并发 `append` 同一个 slice 变量（尤其是共享容量、可能触发扩容竞争）是不安全的，需要加锁或使用 `channel`/`sync` 原语保护；只读且不并发写的情况下才是安全的。

## 四、常见面试题

**1. slice 和 array 的区别？**
Array 是值类型，长度是类型的一部分，赋值/传参整体拷贝；slice 是对某段连续内存的引用视图（`{array, len, cap}` 三元组），赋值/传参只拷贝这个 header，共享底层数组。

**2. slice 的底层数据结构是什么？**
`struct { array unsafe.Pointer; len int; cap int }`，见 2.1。

**3. len 和 cap 的区别？**
`len` 是当前可访问的元素个数；`cap` 是从当前 `array` 起点到底层数组末尾还能容纳的元素个数上限（`cap >= len`）。

**4. append 的扩容规则是什么？**
先看新长度是否超过旧容量的 2 倍，超过则直接按需要的长度分配；否则：旧容量 < 256 时翻倍（2x），>= 256 时按约 1.25x 递增（`newcap += (newcap+3*256)>>2` 迭代到够用为止）；最终容量还会被 `roundupsize` 对齐到内存分配器的 size class，因此实际 cap 常比理论值略大。详见 2.3。

**5. 为什么 append 必须写成 `s = append(s, x)`，而不能直接 `append(s, x)`？**
`append` 参数和返回值都是 slice header 的值拷贝。若发生扩容，函数内部会生成新的 `array/len/cap`，只有通过返回值赋值回调用者的变量，才能让调用者看到新的底层数组和容量；否则调用者手里的还是旧 header，指向旧数组。

**6. 两个 slice 共享同一个底层数组，其中一个 append 会不会影响另一个？**
看 `cap` 是否够用：够用则直接原地写入，会“污染”其他共享该数组、且索引落在被写位置的 slice（3.2）；不够用则触发扩容分配新数组，二者之后不再共享内存，互不影响。

**7. 如何避免 append 污染共享数组？**

- 用三索引切片 `s[a:b:c]` 显式收窄 `cap`，让 `cap == len`，任何 append 都强制走扩容分支。
- 需要真正独立的数据时用 `copy` 显式复制一份。

**8. 从一个大 slice 截取出小 slice 长期持有，会有什么问题？如何解决？**
底层数组不会因为只用了一部分而被回收，可能造成事实上的内存泄漏（3.3）。解决方式是 `copy` 出独立的小数组，断开与大数组的引用。

**9. `make([]T, len)` 和 `make([]T, len, cap)` 的区别？`new([]T)` 呢？**
前者 `len==cap`；后者显式指定容量，通常用于预分配，减少后续 append 时的多次扩容拷贝。`new([]T)` 返回的是一个指向 nil slice 的指针（`*[]T`），实践中几乎不用，和 `make` 不是一回事。

**10. slice 可以作为 map 的 key 吗？为什么？**
不可以。slice（以及 map、function）是不可比较类型，Go 规范规定它们不能作为 map key，也不能用 `==` 比较（只能和 `nil` 比较）。原因是 slice 语义上是“引用视图”，逐元素比较代价高且语义模糊（值相等 vs 引用相等），语言设计上直接禁止。

**11. for range 遍历 slice 时修改元素，为什么不生效？**
`for i, v := range s` 中的 `v` 是每次迭代时对 `s[i]` 的值拷贝，修改 `v` 只是修改了这个局部拷贝。要修改原 slice，必须通过下标 `s[i] = ...`，或者遍历指针/结构体指针 slice。

**12. `sort.Ints(s)` 之类的排序会不会影响“原数组”？**
会。`sort` 包是原地排序，直接操作 slice 底层数组的元素顺序；如果这个数组被其它 slice 共享，那些 slice 看到的元素顺序也会一起变。

**13. `copy` 和 `append` 在处理 `dst`、`src` 有重叠内存时的行为？**
`copy` 内部使用 `memmove` 语义，保证重叠区间拷贝正确（等价于先整体读出再写入）；直接手写 `for i := range src { dst[i] = src[i] }` 在有重叠且方向不对时可能覆盖尚未读取的数据，存在 bug 风险，应优先用内置 `copy`。

**14. `string` 与 `[]byte`/`[]rune` 互转的开销是什么？**
`string` 是不可变的只读字节序列，`[]byte(s)`/`string(b)` 互转都需要一次内存拷贝（保证 `string` 不可变的语义不被破坏）；`[]rune(s)` 需要按 UTF-8 解码，还涉及每个字符占 4 字节的膨胀。在明确不修改、且追求性能的场景，可以用 `unsafe.Pointer`/`unsafe.StringData`、`unsafe.String` 做零拷贝转换，但要非常清楚生命周期和不可变性前提，否则会破坏 `string` 只读的语义假设，产生未定义行为。

**15. 为什么切片扩容时超过阈值（256）会从 2x 降到 1.25x？**
小 slice 场景下翻倍能快速摊薄扩容次数（均摊 O(1) 的 append），代价是浪费的内存相对总量很小；但 slice 很大之后，翻倍会一次性浪费大量内存（比如 1GB 直接翻到 2GB），所以增长因子逐步降到 1.25x，用更多次、更小幅度的扩容换取内存利用率，这是时间和空间的权衡（amortized growth strategy）。
