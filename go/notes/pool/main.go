// pool 示例：对应 notes/pool.md
// 运行：go run ./pool
// 压测：go test -bench . -benchmem ./pool
package main

import (
	"bytes"
	"fmt"
	"io"
	"math/bits"
	"os"
	"runtime"
	"strings"
	"sync"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	basicMinimal()
	basicNewSemantics()
	basicThreeStyles()
	basicTypedPool()
	basicNotAResourcePool()

	victimCache()

	trapPutValue()
	trapForgetReset()
	trapUseAfterPut()
	trapBigObject()
	trapCopyPool()
	trapGetIsNotYours()
}

// ---------------------------------------------------------------------------
// 1.1 最小示例
// ---------------------------------------------------------------------------

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) }, // 可选：池空时兜底构造
}

func handle(w io.Writer, data []string) {
	buf := bufPool.Get().(*bytes.Buffer) // 取（可能是复用的，也可能是 New 出来的）
	buf.Reset()                          // 关键：必须自己重置状态
	defer bufPool.Put(buf)               // 还

	for _, s := range data {
		buf.WriteString(s)
	}
	_, _ = w.Write(buf.Bytes())
}

func basicMinimal() {
	section("1.1 最小示例")

	fmt.Print("handle -> ")
	handle(os.Stdout, []string{"hello", " ", "pool"})
	fmt.Println()
	fmt.Println("三步：Get -> Reset -> defer Put")
}

// ---------------------------------------------------------------------------
// 1.2 New 的语义
// ---------------------------------------------------------------------------

func basicNewSemantics() {
	section("1.2 New 的语义")

	var p sync.Pool                              // New == nil
	fmt.Println("New==nil 且池空，Get() =", p.Get()) // <nil>

	p.Put(new(int))
	fmt.Printf("Put 之后再 Get() = %v（非 nil）\n", p.Get() != nil)

	calls := 0
	q := sync.Pool{New: func() any { calls++; return new(int) }}
	_, _ = q.Get(), q.Get()
	fmt.Printf("New 被调了 %d 次：池空时每次 Get 都会造一个新的\n", calls)
	fmt.Println("→ New 只是兜底构造，不是'初始化池'；池永远不会被预热")
}

// ---------------------------------------------------------------------------
// 1.3 三种典型写法对比
// ---------------------------------------------------------------------------

var (
	// 写法一：New 返回指针（推荐）
	p1 = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	// 写法二：New 返回值类型（❌ Put 时会额外分配，见 4.1）
	p2 = sync.Pool{New: func() any { return make([]byte, 0, 4096) }}
	// 写法三：值类型用指针包一层（✅ 正确的切片池写法）
	p3 = sync.Pool{New: func() any { s := make([]byte, 0, 4096); return &s }}
)

func basicThreeStyles() {
	section("1.3 三种典型写法")

	buf := p1.Get().(*bytes.Buffer)
	buf.Reset()
	fmt.Printf("① *bytes.Buffer: cap=%d\n", buf.Cap())
	p1.Put(buf)

	b := p2.Get().([]byte)
	b = b[:0]
	fmt.Printf("② []byte（值类型）: len=%d cap=%d -> Put 时装箱，1 alloc\n", len(b), cap(b))
	p2.Put(b) //nolint:staticcheck // SA6002：故意演示

	sp := p3.Get().(*[]byte)
	*sp = (*sp)[:0]
	fmt.Printf("③ *[]byte: cap=%d -> Put 零分配\n", cap(*sp))
	p3.Put(sp)

	fmt.Println("→ 池化对象一律用指针类型：指针能直接塞进 eface 的 data 字段")
}

// ---------------------------------------------------------------------------
// 1.4 泛型封装（消掉调用方的类型断言）
// ---------------------------------------------------------------------------

type TypedPool[T any] struct {
	p   sync.Pool
	new func() *T
}

func NewTypedPool[T any](newFn func() *T) *TypedPool[T] {
	tp := &TypedPool[T]{new: newFn}
	tp.p.New = func() any { return tp.new() }
	return tp
}

func (tp *TypedPool[T]) Get() *T  { return tp.p.Get().(*T) } // 断言收敛到一处
func (tp *TypedPool[T]) Put(x *T) { tp.p.Put(x) }

type Req struct {
	ID   int
	Tags []string
}

func (r *Req) Reset() {
	r.ID = 0
	clear(r.Tags)       // 元素是指针/含指针时，必须先 clear 断开引用
	r.Tags = r.Tags[:0] // 再截断，保留底层数组
}

var reqPool = NewTypedPool(func() *Req { return &Req{Tags: make([]string, 0, 8)} })

func basicTypedPool() {
	section("1.4 泛型封装")

	r := reqPool.Get()
	r.Reset()
	r.ID, r.Tags = 1, append(r.Tags, "a", "b")
	fmt.Printf("TypedPool[Req].Get() -> %+v（调用方不用写 .(*Req)）\n", *r)
	reqPool.Put(r)

	fmt.Println("→ 泛型只消除调用方样板，内部仍是 any，装箱没消掉，所以类型参数得是 *T")
	fmt.Println("→ struct 的 Reset 要逐字段考虑：s[:0] 不会清掉底层数组里的指针引用")
}

// ---------------------------------------------------------------------------
// 1.7 什么时候不该用 sync.Pool
// ---------------------------------------------------------------------------

func basicNotAResourcePool() {
	section("1.7 sync.Pool 不是资源池")

	for _, row := range [][2]string{
		{"限制池内对象总数", "❌ 没有容量上限，Put 多少存多少"},
		{"保证对象一定能复用", "❌ GC 随时清空（may be removed at any time）"},
		{"空闲超时回收", "❌ 只跟 GC 周期挂钩，与时间无关"},
		{"对象销毁钩子 Close()", "❌ 被丢弃时不会通知你 -> 句柄泄漏"},
		{"阻塞等待可用对象", "❌ Get 永不阻塞"},
		{"公平性 / FIFO", "❌ Get 返回哪个对象完全不确定"},
	} {
		fmt.Printf("  %-22s %s\n", row[0], row[1])
	}
	fmt.Println("→ 只放'纯内存、无外部资源、可被随时丢弃'的对象")
}

// ---------------------------------------------------------------------------
// 2.6 victim cache 与 GC：两轮 GC 之后对象一定丢失
// ---------------------------------------------------------------------------

func victimCache() {
	section("2.6 victim cache 与 GC")

	// 不设 New，这样 Get 返回 nil 就说明对象真的没了
	// 放多个：第一个进 private，其余进 shared（shared 能被其他 P 偷到，
	// private 只有原 P 上的 goroutine 能取回，见 4.10）
	fill := func(p *sync.Pool, n int) {
		for range n {
			p.Put(new(int))
		}
	}
	drain := func(p *sync.Pool) int {
		n := 0
		for p.Get() != nil {
			n++
		}
		return n
	}

	var p1 sync.Pool
	fill(&p1, 4)
	runtime.GC() // local -> victim
	fmt.Printf("放 4 个，一轮 GC 后捞回 %d 个（从 victim 里）\n", drain(&p1))

	var p2 sync.Pool
	fill(&p2, 4)
	runtime.GC() // local -> victim
	runtime.GC() // 老 victim 直接丢弃
	fmt.Printf("放 4 个，两轮 GC 后捞回 %d 个\n", drain(&p2))

	fmt.Println("→ poolCleanup 在 STW 阶段执行：victim = local，local 清空，老 victim 丢给 GC")
	fmt.Println("→ 所以低频使用的 Pool 命中率会很难看（4.10），直接 new 更可预测")
}

// ---------------------------------------------------------------------------
// 4.1 Put 值类型会额外分配一次
// ---------------------------------------------------------------------------

func trapPutValue() {
	section("4.1 Put 值类型会额外分配")

	fmt.Println("Put(x any) 的参数是接口：")
	fmt.Println("  []byte 是 24 字节 header，装箱进 eface 必须堆分配 -> 1 alloc")
	fmt.Println("  *[]byte 是一个字，直接塞进 eface.data -> 0 alloc")
	fmt.Println("staticcheck SA6002: argument should be pointer-like to avoid allocations")
	fmt.Println("实测：PutSliceValue 43ns/24B/1alloc  vs  PutSlicePtr 12ns/0B/0alloc")
	fmt.Println("4.7 Get/Put 固定开销 ~13ns：对象本来能栈分配就是纯亏（见 BenchmarkSmall*）")
}

// ---------------------------------------------------------------------------
// 4.2 忘记 Reset：脏数据与跨请求信息泄漏
// ---------------------------------------------------------------------------

var leakPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func handleDirty(userInput string) string {
	buf := leakPool.Get().(*bytes.Buffer)
	// 忘了 buf.Reset()
	buf.WriteString(userInput)
	defer leakPool.Put(buf)
	return buf.String()
}

func handleClean(userInput string) string {
	buf := leakPool.Get().(*bytes.Buffer)
	buf.Reset() // 流派 A：Get 后立刻 Reset，防御性更强
	buf.WriteString(userInput)
	defer leakPool.Put(buf)
	return buf.String()
}

func trapForgetReset() {
	section("4.2 忘记 Reset")

	fmt.Printf("请求 1（用户 A）: %q\n", handleDirty("secret-of-A"))
	fmt.Printf("请求 2（用户 B）: %q  ⚠️ A 的内容泄漏给了 B\n", handleDirty("data-of-B"))

	fmt.Printf("Reset 之后:      %q\n", handleClean("data-of-C"))
	fmt.Println("→ Reset 放 Get 后还是 Put 前都行，但必须全局统一（推荐 Get 后）")
}

// ---------------------------------------------------------------------------
// 4.3 Put 之后继续使用对象
// ---------------------------------------------------------------------------

// ❌ 返回的 slice 指向池内 buffer 的底层数组
func renderBad() []byte {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	buf.WriteString("payload")
	return buf.Bytes()
}

// ✅ 返回值必须拷贝
func renderGood() []byte {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	buf.WriteString("payload")
	return bytes.Clone(buf.Bytes())
}

func trapUseAfterPut() {
	section("4.3 Put 之后继续使用对象")

	bad := renderBad()
	// 模拟"别的 goroutine 已经 Get 到同一个 buffer 并写入"
	other := bufPool.Get().(*bytes.Buffer)
	other.Reset()
	other.WriteString("XXXXXXX")
	bufPool.Put(other)
	fmt.Printf("renderBad 的返回值被改写成: %q\n", bad)

	good := renderGood()
	other = bufPool.Get().(*bytes.Buffer)
	other.Reset()
	other.WriteString("YYYYYYY")
	bufPool.Put(other)
	fmt.Printf("renderGood 的返回值不受影响: %q\n", good)

	fmt.Println("→ Put 是所有权转移；低并发下几乎不复现，上线才炸")
}

// ---------------------------------------------------------------------------
// 4.4 大对象回池导致内存不降：设上限 or 分级
// ---------------------------------------------------------------------------

const maxBufCap = 64 << 10

// 解法一：设上限，超了就丢（fmt 包的做法）
func putCapped(buf *bytes.Buffer) bool {
	if buf.Cap() > maxBufCap {
		return false // 直接不回池，交给 GC
	}
	buf.Reset()
	bufPool.Put(buf)
	return true
}

// 解法二：按大小分级（net/http2 的做法）
var sizedPools [7]sync.Pool // 16K/32K/64K/128K/256K/512K/∞

func poolIndex(size int) int {
	if size <= 16<<10 {
		return 0
	}
	i := bits.Len(uint(size-1)) - 14
	return min(i, len(sizedPools)-1)
}

func trapBigObject() {
	section("4.4 大对象回池")

	small := new(bytes.Buffer)
	small.Grow(1 << 10)
	big := new(bytes.Buffer)
	big.Grow(1 << 20)

	fmt.Printf("cap=%dKB 回池 ? %v\n", small.Cap()>>10, putCapped(small))
	fmt.Printf("cap=%dKB 回池 ? %v（交给 GC）\n", big.Cap()>>10, putCapped(big))

	for _, n := range []int{1 << 10, 16 << 10, 20 << 10, 64 << 10, 1 << 20} {
		fmt.Printf("  poolIndex(%7d) = %d\n", n, poolIndex(n))
	}
	fmt.Println("→ 4.5 文档要求每个 entry 内存成本大致相同，否则命中即浪费")
}

// ---------------------------------------------------------------------------
// 4.8 不能拷贝 Pool
// ---------------------------------------------------------------------------

type badHolder struct{ p sync.Pool }   // ⚠️ 这个 struct 一旦被拷贝就出问题
type goodHolder struct{ p *sync.Pool } // ✅ 用指针

func trapCopyPool() {
	section("4.8 不能拷贝 Pool")

	// var p sync.Pool
	// q := p // go vet: assignment copies lock value to q: sync.Pool contains sync.noCopy
	fmt.Println("go vet: assignment copies lock value: sync.Pool contains sync.noCopy")

	h := goodHolder{p: &sync.Pool{New: func() any { return new(int) }}}
	fmt.Println("用 *sync.Pool 做字段:", *h.p.Get().(*int))
	fmt.Printf("反例类型（别这么写）: %T\n", badHolder{})
	fmt.Println("→ 总是用 *sync.Pool 或包级变量")
}

// ---------------------------------------------------------------------------
// 4.9 Get 拿到的不一定是你 Put 的
// ---------------------------------------------------------------------------

func trapGetIsNotYours() {
	section("4.9 Get 拿到的不一定是你 Put 的")

	p := sync.Pool{New: func() any { s := "new"; return &s }}

	mine := "mine"
	p.Put(&mine)

	var wg sync.WaitGroup
	got := make([]string, 4)
	for i := range got {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i] = *p.Get().(*string)
		}(i)
	}
	wg.Wait()

	fmt.Println("4 个 goroutine 各自 Get 到:", got)
	fmt.Println("→ 可能来自其他 P（偷取）、victim，或干脆是 New() 现造的")
	fmt.Println("→ -race 模式下还会随机丢弃 25%，专门打'Put 进去一定 Get 得到'的假设")
	fmt.Println("   验证：go run -race ./pool")
}

// ---------------------------------------------------------------------------
// benchmark 用到的工作负载（见 bench_test.go）
// ---------------------------------------------------------------------------

func work(b *bytes.Buffer) {
	for range 100 {
		b.WriteString("hello world ")
	}
}

func workNoPool() {
	buf := new(bytes.Buffer)
	work(buf)
	sink = buf.Len()
}

var workBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func workWithPool() {
	buf := workBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer workBufPool.Put(buf)

	work(buf)
	sink = buf.Len()
}

var sink int

var _ = strings.TrimSpace // 占位：保留 import 便于随手加示例
