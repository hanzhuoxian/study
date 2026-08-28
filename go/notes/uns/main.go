// unsafe 示例：对应 notes/unsafe.md
// 运行：go run ./uns
// 开 checkptr（-race 自动开）：go run -race ./uns
// 关掉 checkptr 看差异：      go run -gcflags=all=-d=checkptr=0 ./uns
package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"unsafe"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	if mode := os.Getenv("UNS_DEMO"); mode != "" {
		runChild(mode)
		return
	}

	basicSizeof()
	basicPointerOps()
	basicNewAPIs()

	patternReinterpret()
	patternArithmetic()
	patternReflect()

	trapUintptrStore()
	trapCheckptr()
	trapStringWrite()
	trapHeaderLayout()
	trapLinkname()
	whenToUse()
}

// ---------------------------------------------------------------------------
// 1.1 Sizeof / Alignof / Offsetof —— 三个编译期常量函数
// ---------------------------------------------------------------------------

type mixed struct {
	a bool    // 1
	b int64   // 8（需要 8 字节对齐 -> 前面填 7 字节）
	c int16   // 2
	d [3]byte // 3
}

type packed struct {
	b int64   // 8
	c int16   // 2
	d [3]byte // 3
	a bool    // 1
}

func basicSizeof() {
	section("1.1 Sizeof / Alignof / Offsetof")

	var m mixed
	fmt.Printf("  mixed:  Sizeof=%d Alignof=%d\n", unsafe.Sizeof(m), unsafe.Alignof(m))
	for _, f := range []struct {
		name string
		off  uintptr
		size uintptr
	}{
		{"a bool", unsafe.Offsetof(m.a), unsafe.Sizeof(m.a)},
		{"b int64", unsafe.Offsetof(m.b), unsafe.Sizeof(m.b)},
		{"c int16", unsafe.Offsetof(m.c), unsafe.Sizeof(m.c)},
		{"d [3]byte", unsafe.Offsetof(m.d), unsafe.Sizeof(m.d)},
	} {
		fmt.Printf("    %-10s offset=%-3d size=%d\n", f.name, f.off, f.size)
	}

	var p packed
	fmt.Printf("  packed（字段按大小降序）: Sizeof=%d ← 省了 %d 字节\n",
		unsafe.Sizeof(p), unsafe.Sizeof(m)-unsafe.Sizeof(p))

	fmt.Println("  常见类型的大小：")
	for _, row := range [][2]any{
		{"bool/int8", unsafe.Sizeof(true)},
		{"int/int64/指针", unsafe.Sizeof(int(0))},
		{"string", unsafe.Sizeof("")},
		{"[]byte", unsafe.Sizeof([]byte(nil))},
		{"map", unsafe.Sizeof(map[int]int(nil))},
		{"chan", unsafe.Sizeof(chan int(nil))},
		{"func", unsafe.Sizeof(func() {})},
		{"any（接口）", unsafe.Sizeof(any(nil))},
		{"error（接口）", unsafe.Sizeof(error(nil))},
	} {
		fmt.Printf("    %-16v %v 字节\n", row[0], row[1])
	}

	fmt.Println("→ 这三个是**编译期常量**：可以用在 const、数组长度里")
	const sz = unsafe.Sizeof(mixed{})
	var arr [sz]byte
	fmt.Printf("  const sz = unsafe.Sizeof(mixed{}) = %d，能开 [sz]byte（len=%d）\n", sz, len(arr))
	fmt.Println("→ Sizeof 只算类型本身，不含指针指向的数据（string 永远是 16，不管多长）")
	fmt.Println("→ Offsetof 的参数必须是**字段选择表达式**（x.f），不能是 (*p).f 之外的形式")
}

// ---------------------------------------------------------------------------
// 1.2 unsafe.Pointer 的四种转换
// ---------------------------------------------------------------------------

func basicPointerOps() {
	section("1.2 unsafe.Pointer 的四种转换能力")

	x := int64(0x1122334455667788)
	p := unsafe.Pointer(&x) // ① 任意类型指针 -> Pointer

	pi := (*int64)(p) // ② Pointer -> 任意类型指针
	fmt.Printf("  *int64 读回: 0x%x\n", *pi)

	u := uintptr(p) // ③ Pointer -> uintptr（只应该用来打印）
	fmt.Printf("  地址（uintptr）: 0x%x\n", u)

	// ④ uintptr -> Pointer：语言允许，但只在极少数模式下正确（见 2.2、3.1）。
	//    这里故意不演示 `unsafe.Pointer(u)`——go vet 会直接报
	//    "possible misuse of unsafe.Pointer"，而它报得对。
	fmt.Println("  uintptr -> Pointer: 语言允许，但 go vet 会报 possible misuse")
	fmt.Println("    只有'转换和算术在同一个表达式里'这种形态才是安全的（见 2.2）")

	fmt.Println()
	fmt.Println("  ⚠️ unsafe.Pointer 和 uintptr 的根本区别：")
	fmt.Println("    unsafe.Pointer 是**指针**：GC 认它，会保活指向的对象，栈移动时会被修正")
	fmt.Println("    uintptr        是**整数**：GC 不认，不保活、不修正")
	fmt.Println("  → 所以 uintptr 只能作为'临时的算术中间值'存在于一个表达式里")
}

// ---------------------------------------------------------------------------
// 1.3 1.17 / 1.20 的新 API：让常见用法有了安全写法
// ---------------------------------------------------------------------------

func basicNewAPIs() {
	section("1.3 Add / Slice / String（1.17、1.20）")

	arr := [5]int32{10, 20, 30, 40, 50}

	// 1.17 unsafe.Add：替代 uintptr 算术
	third := (*int32)(unsafe.Add(unsafe.Pointer(&arr[0]), 2*unsafe.Sizeof(arr[0])))
	fmt.Printf("  unsafe.Add: arr[2] = %d\n", *third)
	fmt.Println("    老写法: (*int32)(unsafe.Pointer(uintptr(unsafe.Pointer(&arr[0])) + 2*4))")

	// 1.17 unsafe.Slice：从指针 + 长度造 slice
	s := unsafe.Slice(&arr[0], len(arr))
	fmt.Printf("  unsafe.Slice: %v（和 arr 共享内存）\n", s)
	s[0] = 99
	fmt.Printf("    改 s[0] 之后 arr = %v\n", arr)

	// 1.20 unsafe.String / StringData / SliceData
	b := []byte("hello")
	str := unsafe.String(unsafe.SliceData(b), len(b))
	fmt.Printf("  unsafe.String: %q（零拷贝，见 string.md 2.2）\n", str)
	fmt.Printf("  unsafe.StringData(%q) = %p\n", str, unsafe.StringData(str))
	fmt.Printf("  unsafe.SliceData(b) = %p（两者相同: %v）\n",
		unsafe.SliceData(b), unsafe.StringData(str) == unsafe.SliceData(b))

	fmt.Println()
	fmt.Println("→ 这四个 API 的意义：把最常见的三种 unsafe 用法（指针算术、指针转 slice、")
	fmt.Println("  []byte 转 string）变成了**编译器认识的、能被 checkptr 检查的**写法。")
	fmt.Println("→ 1.20 之后 reflect.SliceHeader / StringHeader 已废弃，别再用（见 3.4）")
}

// ---------------------------------------------------------------------------
// 2.1 合法模式 ①：*T1 -> Pointer -> *T2（重新解释内存）
// ---------------------------------------------------------------------------

func patternReinterpret() {
	section("2.1 合法模式 ①：类型重新解释")

	// 标准库自己就这么写（math.Float64bits）
	f := 3.14
	bits := *(*uint64)(unsafe.Pointer(&f))
	fmt.Printf("  float64 %v 的 bit pattern = 0x%016x\n", f, bits)
	fmt.Printf("  math.Float64bits 的结果一致: %v\n", bits == math.Float64bits(f))

	back := *(*float64)(unsafe.Pointer(&bits))
	fmt.Printf("  转回来: %v\n", back)

	fmt.Println()
	fmt.Println("  前提（文档原文）：T2 不大于 T1，且两者内存布局等价")
	fmt.Println("  ✗ 反例：把 *[2]byte 当 *int64 读 —— 越界读到别人的内存")
	fmt.Println("  ✗ 反例：忽略字节序（大小端）—— 跨平台会得到不同结果")

	// 一个真实用途：批量转换 []int32 -> []byte（零拷贝写文件）
	nums := []int32{1, 2, 3}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(nums))), len(nums)*4)
	fmt.Printf("  []int32%v 的原始字节（小端）: %v\n", nums, raw)
}

// ---------------------------------------------------------------------------
// 2.2 合法模式 ③：Pointer -> uintptr 做算术 -> 立刻转回 Pointer
// ---------------------------------------------------------------------------

type record struct {
	ID   int64
	Name [8]byte
	Age  int32
}

func patternArithmetic() {
	section("2.2 合法模式 ③：指针算术")

	r := record{ID: 7, Age: 30}
	copy(r.Name[:], "bob")

	// ✓ 两次转换必须在**同一个表达式**里，中间只能有算术
	agePtr := (*int32)(unsafe.Pointer(uintptr(unsafe.Pointer(&r)) + unsafe.Offsetof(r.Age)))
	fmt.Printf("  通过偏移拿到 Age: %d\n", *agePtr)

	// ✓ 1.17 之后更推荐 unsafe.Add（可读性和安全性都更好）
	agePtr2 := (*int32)(unsafe.Add(unsafe.Pointer(&r), unsafe.Offsetof(r.Age)))
	fmt.Printf("  unsafe.Add 版本: %d\n", *agePtr2)

	fmt.Println()
	fmt.Println("  文档明确列出的三条禁忌：")
	fmt.Println("    ✗ 中间存进变量: u := uintptr(p); p = unsafe.Pointer(u + off)")
	fmt.Println("    ✗ 指到分配区之外: end = unsafe.Pointer(uintptr(p) + Sizeof(*p))  ← 尾后指针非法")
	fmt.Println("    ✗ 对 nil 做算术")
	fmt.Println("  （C 里合法的 one-past-the-end 指针在 Go 里是非法的，因为 GC 会误判所属对象）")
}

// ---------------------------------------------------------------------------
// 2.3 合法模式 ⑤：reflect.Value.UnsafeAddr / Pointer 必须立刻转换
// ---------------------------------------------------------------------------

type withPrivate struct {
	Public  string
	private int
}

func patternReflect() {
	section("2.3 合法模式 ⑤：配合 reflect")

	w := &withPrivate{Public: "p", private: 42}
	v := reflect.ValueOf(w).Elem().Field(1)

	// ✓ UnsafeAddr 的结果必须在同一表达式里转成 Pointer
	real := reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
	fmt.Printf("  读未导出字段: %d\n", real.Int())
	real.SetInt(100)
	fmt.Printf("  写未导出字段: %d\n", w.private)

	fmt.Println("  ✗ 错误写法（存进变量就可能被 GC 搬走）：")
	fmt.Println("      u := v.UnsafeAddr()")
	fmt.Println("      p := unsafe.Pointer(u)")
	fmt.Println("→ 1.18 起 reflect.Value.UnsafePointer() 直接返回 unsafe.Pointer，优先用它")
}

// ---------------------------------------------------------------------------
// 3.1 把 uintptr 存进变量：最经典的 unsafe bug
// ---------------------------------------------------------------------------

func trapUintptrStore() {
	section("3.1 uintptr 存进变量 = 悬垂地址")

	fmt.Println("  ✗ 错误：")
	fmt.Println("      u := uintptr(unsafe.Pointer(obj))   // GC 不认识 u，obj 可能被回收")
	fmt.Println("      // ... 中间发生了 GC 或栈增长 ...")
	fmt.Println("      p := (*T)(unsafe.Pointer(u))        // u 指向的可能已经是别的东西")
	fmt.Println()
	fmt.Println("  两个独立的失效原因：")
	fmt.Println("    ① GC 回收：uintptr 不是引用，不保活对象（gc.md 2.2）")
	fmt.Println("    ② 栈移动：goroutine 栈增长时会整体拷贝并修正**指针**，")
	fmt.Println("       但 uintptr 是整数，不会被修正（mem.md 3.2 实测有 4 次栈搬家）")
	fmt.Println()
	fmt.Println("  ✓ 唯一正确的形态：转换和算术在同一个表达式里，结果立刻变回 Pointer")
	fmt.Println("      p = unsafe.Pointer(uintptr(p) + offset)")
	fmt.Println("      p = unsafe.Add(p, offset)                  // 1.17+，更好")
	fmt.Println()
	fmt.Println("  ⚠️ 还有一个隐蔽变体：unsafe.Pointer 存在**只有 uintptr 字段的结构里**，")
	fmt.Println("     或者存进 map[uintptr]xxx —— 一样不保活")
}

// ---------------------------------------------------------------------------
// 3.2 checkptr：编译器给 unsafe 加的运行时检查
// ---------------------------------------------------------------------------

func trapCheckptr() {
	section("3.2 checkptr 检查")

	fmt.Println("  -race 或 -gcflags=all=-d=checkptr 会插入运行时检查，抓两类错误：")
	fmt.Println("    ① checkptrAlignment：转换后的指针没有满足目标类型的对齐要求")
	fmt.Println("    ② checkptrArithmetic：算出来的指针不在原对象的分配范围内")
	fmt.Println()

	out, _ := selfRun("checkptr")
	first := strings.SplitN(strings.TrimSpace(out), "\n", 4)
	fmt.Println("  子进程（-race）跑一个越界的指针算术：")
	for i, l := range first {
		if i == 3 {
			fmt.Println("    ...")
			break
		}
		fmt.Println("    " + l)
	}
	fmt.Println()
	fmt.Println("→ 所以 unsafe 代码的测试**必须跑 -race**，否则这类错误只会表现为随机的数据损坏")
	fmt.Println("→ 但它只能抓一部分。实测：把 []byte 的第 1 个字节当 *int64 读（未对齐）")
	fmt.Println("  在本机 amd64 + -race 下**没有**被 checkptr 拦住，读出来是 0。")
	fmt.Println("  文档原话：silence from go vet is not a guarantee that the code is valid")
}

// ---------------------------------------------------------------------------
// 3.3 写只读内存 = SIGSEGV（不是 panic）
// ---------------------------------------------------------------------------

func trapStringWrite() {
	section("3.3 写字符串的内存 = 段错误")

	out, _ := selfRun("writestring")
	lines := strings.SplitN(strings.TrimSpace(out), "\n", 3)
	fmt.Println("  子进程尝试写字面量字符串的内存：")
	for i, l := range lines {
		if i == 2 {
			fmt.Println("    ...")
			break
		}
		fmt.Println("    " + l)
	}
	fmt.Println()
	fmt.Println("→ 字符串字面量在 .rodata（只读段），写入触发 SIGSEGV")
	fmt.Println("→ 这是**信号级错误，不是 panic**：recover 救不了，进程直接死")
	fmt.Println("→ 所以 unsafe.Slice(unsafe.StringData(s), len(s)) 拿到的 []byte 只能读")
}

// ---------------------------------------------------------------------------
// 3.4 依赖 header 布局
// ---------------------------------------------------------------------------

func trapHeaderLayout() {
	section("3.4 别依赖 header 布局")

	b := []byte("hello")

	// 老代码常见写法（1.20 起 reflect.SliceHeader 已 deprecated）
	type sliceHeader struct {
		Data uintptr
		Len  int
		Cap  int
	}
	h := (*sliceHeader)(unsafe.Pointer(&b))
	fmt.Printf("  手工 header: Data=0x%x Len=%d Cap=%d（今天恰好对）\n", h.Data, h.Len, h.Cap)

	fmt.Println()
	fmt.Println("  为什么不该这么写：")
	fmt.Println("    ① Go 从未承诺 slice/string 的内存布局，理论上随时能改")
	fmt.Println("    ② reflect.SliceHeader.Data 是 uintptr —— 一旦你构造了一个 header 变量，")
	fmt.Println("       里面的 Data 就不再保活对象（3.1 的坑）")
	fmt.Println("    ③ 1.20 起 reflect.SliceHeader/StringHeader 明确标注 deprecated")
	fmt.Println("  ✓ 正确写法：unsafe.Slice / unsafe.String / unsafe.SliceData / unsafe.StringData")
}

// ---------------------------------------------------------------------------
// 3.5 //go:linkname
// ---------------------------------------------------------------------------

// 拿到 runtime 里未导出的函数：需要 import _ "unsafe" 且函数体留空（由链接器绑定）
//
//go:linkname runtimeNanotime runtime.nanotime
func runtimeNanotime() int64

func trapLinkname() {
	section("3.5 //go:linkname")

	t1 := runtimeNanotime()
	t2 := runtimeNanotime()
	fmt.Printf("  runtime.nanotime() 两次调用差 %d ns（单调时钟，比 time.Now 便宜）\n", t2-t1)

	fmt.Println()
	fmt.Println("  用法三要素：")
	fmt.Println("    ① import _ \"unsafe\"（不导入编译报错）")
	fmt.Println("    ② //go:linkname 本地名 目标符号")
	fmt.Println("    ③ 本地函数只声明不实现（或者用 //go:linkname 导出自己的符号）")
	fmt.Println()
	fmt.Println("  ⚠️ 这是**完全绕过类型系统和包边界**的后门：")
	fmt.Println("    · runtime 内部函数没有兼容性承诺，升级 Go 随时可能崩")
	fmt.Println("    · Go 1.23 起对 linkname 收紧了限制（--checklinkname，拉黑了一批符号）")
	fmt.Println("    · 曾经大量库靠它偷 runtime 的 g 结构、fastrand 等，1.22+ 陆续失效")
	fmt.Println("  → runtime 源码里能看到官方的抱怨清单：'Notable members of the hall of shame'")
	fmt.Printf("  当前 Go 版本: %s\n", runtime.Version())
}

// ---------------------------------------------------------------------------
// 4. 什么时候真的该用 unsafe
// ---------------------------------------------------------------------------

func whenToUse() {
	section("4. 该用与不该用")

	fmt.Println("  ✓ 值得用：")
	fmt.Println("    · 热路径上 KB 级以上的 []byte <-> string 零拷贝（string.md 2.2，实测 1650x）")
	fmt.Println("    · 类型重新解释：math.Float64bits 这类位操作")
	fmt.Println("    · 和 C / 系统调用 / mmap 内存互操作（syscall 模式 ④）")
	fmt.Println("    · 序列化库里按偏移直接读写字段（避开 reflect 的开销）")
	fmt.Println("    · 测试里读写未导出字段")
	fmt.Println()
	fmt.Println("  ✗ 不该用：")
	fmt.Println("    · 只为了省一次拷贝而在业务代码里散落 unsafe")
	fmt.Println("    · 绕过未导出字段做生产逻辑（设计问题）")
	fmt.Println("    · 自己拼 slice/string header（用 unsafe.Slice/String）")
	fmt.Println("    · 通过 linkname 依赖 runtime 内部（升级即炸）")
	fmt.Println()
	fmt.Println("  用了 unsafe 之后的三条纪律：")
	fmt.Println("    ① 收敛到一个包/一个文件里，函数名带 Unsafe 前缀，注释写清前提条件")
	fmt.Println("    ② 测试必须跑 -race（checkptr）")
	fmt.Println("    ③ 每次升级 Go 版本都重新跑一遍完整测试")
}

// ---------------------------------------------------------------------------
// 子进程演示
// ---------------------------------------------------------------------------

func runChild(mode string) {
	switch mode {
	case "checkptr":
		childOutOfBounds()
	case "writestring":
		s := "hello world"
		b := unsafe.Slice(unsafe.StringData(s), len(s))
		b[0] = 'H' // 写只读段 -> SIGSEGV
		fmt.Println("居然写成功了:", s)
	}
}

// 越界的指针算术：从一个 8 字节对象往后跳 64 字节。
// 注意结果要真的被用到（存进包级变量），否则可能被优化掉、检查也就不会插入。
var childSink []byte

func childOutOfBounds() {
	b := make([]byte, 8)
	p := unsafe.Add(unsafe.Pointer(&b[0]), 64) // checkptr 在这里 throw
	childSink = unsafe.Slice((*byte)(p), 8)
	fmt.Println("读到了", childSink[0])
}

func selfRun(mode string) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	cmd := exec.Command("go", "run", "-race", ".")
	cmd.Dir = filepath.Dir(file)
	cmd.Env = append(os.Environ(), "UNS_DEMO="+mode)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
