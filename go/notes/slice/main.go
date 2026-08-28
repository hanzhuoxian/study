// slice 示例：对应 notes/slice.md
// 运行：go run ./slice
package main

import (
	"fmt"
	"reflect"
	"sort"
	"unsafe"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

// header 打印 slice 的三元组 {array, len, cap}
//
// reflect.SliceHeader 在 Go 1.20 起已废弃（它把指针当 uintptr 存，GC 不可见，
// 结构布局也不受兼容性保证）。取底层数组地址的正确姿势：
//   - unsafe.SliceData(s) → *T，Go 1.20+ 官方入口，本文件采用
//   - reflect.ValueOf(s).UnsafePointer() → unsafe.Pointer，纯反射、无需 unsafe 语义
//   - &s[0]（len>0 时）→ 最朴素，但空 slice 会 panic
//
// unsafe.SliceData 的边界语义：len==cap==0 且 s==nil 时返回 nil；
// cap==0 但 s!=nil 时返回一个不可解引用的非 nil 地址。
func header[T any](name string, s []T) {
	ptr := unsafe.Pointer(unsafe.SliceData(s))
	fmt.Printf("%-10s ptr=%#x len=%d cap=%d nil=%t\n", name, uintptr(ptr), len(s), cap(s), s == nil)
}

func main() {
	basicDeclare()
	basicReslice()
	basicAppendCopy()
	basicRange()
	basicMatrix()
	basicStringConv()

	growth()
	copyOverlap()

	trapShareArray()
	trapAppendPollution()
	trapMemoryLeak()
	trapNilVsEmpty()
}

// ---------------------------------------------------------------------------
// 1.1 声明与初始化
// ---------------------------------------------------------------------------

func basicDeclare() {
	section("1.1 声明与初始化")

	var s1 []int         // nil slice：len=0 cap=0，底层指针为 nil
	s2 := []int{}        // 空 slice：非 nil，指向 zerobase
	s3 := []int{1, 2, 3} // 字面量
	s4 := make([]int, 5) // len=cap=5，元素置零
	s5 := make([]int, 3, 10)
	s6 := make([]int, 0, 10) // 预分配容量，逐步 append

	header("s1(var)", s1)
	header("s2({})", s2)
	header("s3(lit)", s3)
	header("s4", s4)
	header("s5", s5)
	header("s6", s6)

	// nil slice 也能直接 append / len / range，不会 panic（只有写下标才 panic）
	s1 = append(s1, 1)
	fmt.Println("nil slice append 后:", s1)
}

// ---------------------------------------------------------------------------
// 1.2 从数组/slice 切出新 slice
// ---------------------------------------------------------------------------

func basicReslice() {
	section("1.2 切片表达式")

	arr := [5]int{0, 1, 2, 3, 4}
	s := arr[1:3]    // len=2, cap=cap(arr)-1=4
	s2 := arr[1:3:4] // 三索引 low:high:max，cap=max-low=3

	fmt.Printf("arr[1:3]   = %v len=%d cap=%d\n", s, len(s), cap(s))
	fmt.Printf("arr[1:3:4] = %v len=%d cap=%d\n", s2, len(s2), cap(s2))

	// slice 只是数组的视图：改视图 = 改数组
	s[0] = 99
	fmt.Println("s[0]=99 之后 arr =", arr)

	// 可以切到 len 之外、cap 之内，把"隐藏"的元素重新暴露出来
	fmt.Println("s[:cap(s)] =", s[:cap(s)])
}

// ---------------------------------------------------------------------------
// 1.3 append / copy
// ---------------------------------------------------------------------------

func basicAppendCopy() {
	section("1.3 append / copy")

	s := make([]int, 0, 2)
	s = append(s, 1)    // 必须接收返回值：可能重新分配底层数组
	s = append(s, 2, 3) // 变长参数
	other := []int{7, 8}
	s = append(s, other...) // 追加另一个 slice 要展开

	dst := make([]int, 3)
	n := copy(dst, s) // 返回 min(len(dst), len(src))
	fmt.Println("s =", s, "dst =", dst, "copy n =", n)

	// []byte 与 string 之间 copy/append 有特化：可以直接 append 字符串
	b := append([]byte("hello "), "world"...)
	fmt.Println("append string...:", string(b))

	// 不接收返回值的错误写法（这里演示后果）
	bad := make([]int, 0, 1)
	appendWrong(bad)
	fmt.Println("值传递 append 后调用方看不到:", bad, "len =", len(bad))
}

func appendWrong(s []int) { _ = append(s, 42) } //nolint // 仅演示：返回值被丢弃

// ---------------------------------------------------------------------------
// 1.4 range 遍历
// ---------------------------------------------------------------------------

func basicRange() {
	section("1.4 range 遍历")

	s := []int{1, 2, 3}
	for i, v := range s {
		fmt.Printf("i=%d &v=%p &s[i]=%p\n", i, &v, &s[i]) // v 是拷贝，地址与 s[i] 不同
		v *= 10                                           // 改拷贝，无效
		_ = v
	}
	fmt.Println("改 v 之后:", s)

	for i := range s {
		s[i] *= 10 // 要改原元素必须通过下标
	}
	fmt.Println("改 s[i] 之后:", s)

	// Go 1.22+ 每轮迭代都是新变量，取地址不再共享同一个 v
	var ptrs []*int
	for _, v := range s {
		ptrs = append(ptrs, &v)
	}
	fmt.Print("闭包/取址捕获: ")
	for _, p := range ptrs {
		fmt.Print(*p, " ")
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// 1.5 多维 slice
// ---------------------------------------------------------------------------

func basicMatrix() {
	section("1.5 多维 slice")

	const rows, cols = 2, 3
	grid := make([][]int, rows)
	for i := range grid {
		grid[i] = make([]int, cols) // 每行单独分配，行与行之间不连续
	}
	grid[1][2] = 7
	fmt.Println("独立分配:", grid)
	fmt.Printf("row0=%p row1=%p（地址不连续）\n", &grid[0][0], &grid[1][0])

	// 想要连续内存：先开一整块，再按行切片
	flat := make([]int, rows*cols)
	grid2 := make([][]int, rows)
	for i := range grid2 {
		grid2[i] = flat[i*cols : (i+1)*cols : (i+1)*cols]
	}
	grid2[0][0] = 1
	fmt.Println("共享底层数组:", grid2, "flat =", flat)
}

// ---------------------------------------------------------------------------
// 1.6 字符串与 slice 互转
// ---------------------------------------------------------------------------

func basicStringConv() {
	section("1.6 字符串与 slice 互转")

	s := "你好, go"
	b := []byte(s) // 一次拷贝
	r := []rune(s) // UTF-8 解码后拷贝，每个 rune 4 字节

	fmt.Printf("len(string)=%d len([]byte)=%d len([]rune)=%d\n", len(s), len(b), len(r))
	fmt.Println("b[0:3] =", string(b[0:3]), " r[0] =", string(r[0]))

	// range string 按 rune 迭代，下标是字节偏移
	for i, ch := range s {
		fmt.Printf("(%d,%c) ", i, ch)
	}
	fmt.Println()

	// 零拷贝转换（unsafe）：前提是绝不修改 b，否则破坏 string 不可变语义
	zero := unsafe.String(unsafe.SliceData(b), len(b))
	fmt.Println("unsafe.String:", zero)
}

// ---------------------------------------------------------------------------
// 2.3 append 与扩容机制：观察 nextslicecap + roundupsize 的实际效果
// ---------------------------------------------------------------------------

func growth() {
	section("2.3 扩容曲线（<256 翻倍，>=256 逐步降到 ~1.25x）")

	printGrowth[int]("[]int   ")
	printGrowth[byte]("[]byte  ")
	printGrowth[[3]int]("[][3]int")

	// 一次 append 很多元素：新长度 > 2*oldCap，直接按需要的长度分配（再 roundupsize）
	s := make([]int, 0, 4)
	s = append(s, make([]int, 100)...)
	fmt.Printf("一次追加 100 个: len=%d cap=%d\n", len(s), cap(s))
}

func printGrowth[T any](name string) {
	var s []T
	prev := cap(s)
	fmt.Printf("%s cap 变化: 0", name)
	for range 2000 {
		var zero T
		s = append(s, zero)
		if cap(s) != prev {
			ratio := 0.0
			if prev > 0 {
				ratio = float64(cap(s)) / float64(prev)
			}
			fmt.Printf(" -> %d(%.2fx)", cap(s), ratio)
			prev = cap(s)
		}
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// 2.4 copy 的语义：允许重叠，等价于 memmove
// ---------------------------------------------------------------------------

func copyOverlap() {
	section("2.4 copy 处理重叠内存")

	a := []int{1, 2, 3, 4, 5}
	copy(a[1:], a[:4])           // 目标与源重叠，向后移动一格
	fmt.Println("copy 重叠结果:", a) // [1 1 2 3 4]

	b := []int{1, 2, 3, 4, 5}
	for i := range b[:4] { // 手写循环在重叠时会踩坏数据
		b[i+1] = b[i]
	}
	fmt.Println("手写循环结果:", b) // [1 1 1 1 1]
}

// ---------------------------------------------------------------------------
// 3.1 共享底层数组导致的"意外修改"
// ---------------------------------------------------------------------------

func trapShareArray() {
	section("3.1 共享底层数组")

	a := []int{1, 2, 3, 4, 5}
	b := a[1:3]
	b[0] = 99
	fmt.Println("b[0]=99 之后 a =", a) // [1 99 3 4 5]

	// sort 是原地排序，同样会影响所有共享该数组的 slice
	sort.Sort(sort.Reverse(sort.IntSlice(a)))
	fmt.Println("排序后 a =", a, " b =", b)
}

// ---------------------------------------------------------------------------
// 3.2 append 引发的"污染"
// ---------------------------------------------------------------------------

func trapAppendPollution() {
	section("3.2 append 污染共享数组")

	a := make([]int, 3, 5)
	b := a[:2]         // cap(b) 仍然是 5
	b = append(b, 100) // 容量够 -> 原地写入 a[2]
	fmt.Println("污染: a =", a, " b =", b, " cap(b) =", cap(b))

	// 修复一：三索引切片把 cap 收窄到 len，append 必然扩容
	a2 := make([]int, 3, 5)
	b2 := a2[:2:2]
	b2 = append(b2, 100)
	fmt.Println("三索引: a2 =", a2, " b2 =", b2, " cap(b2) =", cap(b2))

	// 修复二：需要独立数据时显式 copy
	a3 := make([]int, 3, 5)
	b3 := make([]int, 2)
	copy(b3, a3[:2])
	b3 = append(b3, 100)
	fmt.Println("copy:  a3 =", a3, " b3 =", b3)
}

// ---------------------------------------------------------------------------
// 3.3 大 slice 截取小 slice 造成的内存泄漏
// ---------------------------------------------------------------------------

func trapMemoryLeak() {
	section("3.3 截取视图导致大数组无法回收")

	big := make([]byte, 4<<20) // 4MB
	leak := badPrefix(big)
	safe := goodPrefix(big)

	// 两者内容一样，但 leak 的底层数组仍然是那 4MB
	fmt.Printf("leak len=%d cap=%d（cap 暴露了它背后是大数组）\n", len(leak), cap(leak))
	fmt.Printf("safe len=%d cap=%d（独立的小数组，大数组可被 GC 回收）\n", len(safe), cap(safe))
}

func badPrefix(data []byte) []byte { return data[:1<<10] }

func goodPrefix(data []byte) []byte {
	out := make([]byte, 1<<10)
	copy(out, data)
	return out
}

// ---------------------------------------------------------------------------
// 3.4 nil slice 与空 slice
// ---------------------------------------------------------------------------

func trapNilVsEmpty() {
	section("3.4 nil slice vs 空 slice")

	var a []int
	b := []int{}

	fmt.Printf("a==nil: %t  len=%d cap=%d\n", a == nil, len(a), cap(a))
	fmt.Printf("b==nil: %t  len=%d cap=%d\n", b == nil, len(b), cap(b))

	// 区别主要体现在 == nil 判断和 json 编码上（见 json.md）
	fmt.Println("reflect.DeepEqual(a, b) =", reflect.DeepEqual(a, b))

	// slice 不可比较，只能和 nil 比：a == b 无法编译
	// slice 也不能做 map key（不可比较类型）
}
