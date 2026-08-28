// iter 示例：对应 notes/iter.md（range over func、iter.Seq/Pull）
// 运行：go run ./iter
package main

import (
	"bufio"
	"fmt"
	"iter"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	basicSignatures()
	basicWriteIterator()
	basicContainerIterator()
	basicControlFlow()
	basicStdlib()
	basicPull()
	basicCombinators()
	basicSingleUse()

	trapIgnoreYieldResult()
	trapSwallowPanic()
	trapConcurrentYield()
	trapPullLeak()
	trapConsumeTwice()
	trapSeqIsNotCollection()
	trapModifyWhileIterating()
	trapLazyErrors()
}

// ---------------------------------------------------------------------------
// 1.1 / 1.2 三种合法签名
// ---------------------------------------------------------------------------

func tick(yield func() bool) { // func(func() bool)：0 个迭代变量
	for range 3 {
		if !yield() {
			return
		}
	}
}

func basicSignatures() {
	section("1.1 三种签名")

	n := 0
	for range tick { // 0 个迭代变量
		n++
	}
	fmt.Println("func(func() bool) 迭代次数:", n)

	seq := Count(3)                       // iter.Seq[int]  = func(func(V) bool)
	seq2 := Enumerate([]string{"a", "b"}) // iter.Seq2[K,V] = func(func(K,V) bool)

	fmt.Print("Seq:  ")
	for v := range seq {
		fmt.Print(v, " ")
	}
	fmt.Print("\nSeq2: ")
	for k, v := range seq2 {
		fmt.Printf("%d=%s ", k, v)
	}
	fmt.Print("\nSeq2 只取第一个值: ")
	for k := range seq2 { // 合法：Seq2 可以只接收 K
		fmt.Print(k, " ")
	}
	fmt.Println()
	// for i, v := range seq // 编译错误：permits only one iteration variable
}

// ---------------------------------------------------------------------------
// 1.3 写一个迭代器
// ---------------------------------------------------------------------------

func Count(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range n {
			if !yield(i) { // 必须检查返回值
				return // 提前结束：这里做清理
			}
		}
	}
}

func Enumerate[V any](s []V) iter.Seq2[int, V] {
	return func(yield func(int, V) bool) {
		for i, v := range s {
			if !yield(i, v) {
				return
			}
		}
	}
}

// 带资源清理：defer 覆盖 break / return / panic 三条退出路径
func Lines(name string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		f, err := os.Open(name)
		if err != nil {
			yield("", err) // 错误也是序列的一部分
			return
		}
		defer func() {
			f.Close()
			fmt.Println("  (defer 执行了：文件已关闭)")
		}()

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if !yield(sc.Text(), nil) {
				return
			}
		}
		if err := sc.Err(); err != nil {
			yield("", err)
		}
	}
}

func basicWriteIterator() {
	section("1.3 写一个迭代器")

	fmt.Print("break 提前结束: ")
	for v := range Count(5) {
		if v == 2 {
			break // 让 yield 返回 false
		}
		fmt.Print(v, " ")
	}
	fmt.Println()

	path := filepath.Join(os.TempDir(), "iter-demo.txt")
	os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644)
	defer os.Remove(path)

	fmt.Println("读文件，读到第二行就 break：")
	for line, err := range Lines(path) {
		if err != nil {
			fmt.Println("  错误:", err)
			break
		}
		fmt.Println("  ", line)
		if line == "line2" {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// 1.4 给自定义容器提供迭代器（All / Backward / Keys / Values 命名约定）
// ---------------------------------------------------------------------------

type Ring[V any] struct{ items []V }

func (r *Ring[V]) Push(v V) { r.items = append(r.items, v) }

// All returns an iterator over index-value pairs.
func (r *Ring[V]) All() iter.Seq2[int, V] {
	return func(yield func(int, V) bool) {
		for i, v := range r.items {
			if !yield(i, v) {
				return
			}
		}
	}
}

// Backward returns an iterator over values in reverse order.
func (r *Ring[V]) Backward() iter.Seq[V] {
	return func(yield func(V) bool) {
		for i := len(r.items) - 1; i >= 0; i-- {
			if !yield(r.items[i]) {
				return
			}
		}
	}
}

func basicContainerIterator() {
	section("1.4 容器迭代器")

	r := &Ring[string]{}
	for _, s := range []string{"a", "b", "c"} {
		r.Push(s)
	}

	fmt.Print("All():      ")
	for i, v := range r.All() {
		fmt.Printf("%d=%s ", i, v)
	}
	fmt.Print("\nBackward(): ")
	for v := range r.Backward() {
		fmt.Print(v, " ")
	}
	fmt.Println("\n方法返回的是迭代器（廉价的闭包），遍历发生在 range 里")
}

// ---------------------------------------------------------------------------
// 1.5 循环体里的控制流
// ---------------------------------------------------------------------------

func basicControlFlow() {
	section("1.5 循环体里的控制流")

	fmt.Println("continue/break/return/goto/带标签 break 都能正常工作：")
	fmt.Println("  找到的第一个偶数:", firstEven(Count(10)))

Outer:
	for x := range Count(3) {
		for y := range Count(3) {
			if x+y == 3 {
				fmt.Printf("  break Outer at (%d,%d)\n", x, y)
				break Outer
			}
		}
	}

	for v := range Count(10) {
		if v == 2 {
			goto done
		}
	}
done:
	fmt.Println("  goto 跳出成功")
}

func firstEven(seq iter.Seq[int]) int {
	for v := range seq {
		if v > 0 && v%2 == 0 {
			return v // 先记录返回值，再让 yield 返回 false
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// 1.6 标准库里的迭代器 API
// ---------------------------------------------------------------------------

func basicStdlib() {
	section("1.6 标准库迭代器")

	s := []int{3, 1, 4, 1, 5}
	fmt.Println("slices.Collect(slices.Values):", slices.Collect(slices.Values(s)))
	fmt.Println("slices.Sorted:               ", slices.Sorted(slices.Values(s)))
	fmt.Print("slices.Backward:              ")
	for i, v := range slices.Backward(s) {
		fmt.Printf("%d:%d ", i, v)
	}
	fmt.Println()
	fmt.Print("slices.Chunk(2):              ")
	for c := range slices.Chunk(s, 2) {
		fmt.Print(c, " ")
	}
	fmt.Println()

	m := map[string]int{"b": 2, "a": 1, "c": 3}
	fmt.Println("按 key 有序遍历 map（最常用的组合）:")
	for _, k := range slices.Sorted(maps.Keys(m)) {
		fmt.Printf("  %s=%d\n", k, m[k])
	}
	fmt.Println("maps.Collect(Seq2 -> map):", maps.Collect(Enumerate([]string{"x", "y"})))

	// Go 1.24 的字符串切分迭代器：不会一次性分配整个 []string
	fmt.Print("strings.SplitSeq: ")
	for f := range strings.SplitSeq("a,b,c", ",") {
		fmt.Print(f, " ")
	}
	fmt.Print("\nstrings.FieldsSeq: ")
	for f := range strings.FieldsSeq(" x  y  z ") {
		fmt.Print(f, " ")
	}
	fmt.Print("\nstrings.Lines: ")
	for l := range strings.Lines("l1\nl2\n") {
		fmt.Printf("%q ", l)
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// 1.7 iter.Pull：push 转 pull
// ---------------------------------------------------------------------------

func Zip[A, B any](a iter.Seq[A], b iter.Seq[B]) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		nextB, stop := iter.Pull(b)
		defer stop()
		for va := range a {
			vb, ok := nextB()
			if !ok || !yield(va, vb) {
				return
			}
		}
	}
}

func basicPull() {
	section("1.7 iter.Pull")

	next, stop := iter.Pull(Count(3))
	defer stop() // 约定：下一行就写 defer stop()
	for {
		v, ok := next()
		if !ok {
			break
		}
		fmt.Print(v, " ")
	}
	fmt.Println("<- 由外部驱动的 pull 模型")

	// 序列结束后继续调用不会 panic，还是 (零值, false)
	v, ok := next()
	fmt.Printf("结束后再 next(): v=%d ok=%t\n", v, ok)

	fmt.Print("Zip 两个序列: ")
	for a, b := range Zip(Count(3), slices.Values([]string{"x", "y", "z"})) {
		fmt.Printf("(%d,%s) ", a, b)
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// 1.8 常用组合子（标准库刻意没提供，模板固定）
// ---------------------------------------------------------------------------

func Map[A, B any](seq iter.Seq[A], f func(A) B) iter.Seq[B] {
	return func(yield func(B) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

func Filter[V any](seq iter.Seq[V], keep func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if keep(v) && !yield(v) {
				return
			}
		}
	}
}

func Take[V any](seq iter.Seq[V], n int) iter.Seq[V] {
	return func(yield func(V) bool) {
		if n <= 0 {
			return
		}
		i := 0
		for v := range seq {
			if !yield(v) {
				return
			}
			if i++; i >= n {
				return
			}
		}
	}
}

func Chain[V any](seqs ...iter.Seq[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, s := range seqs {
			for v := range s {
				if !yield(v) {
					return
				}
			}
		}
	}
}

func Naturals() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 1; ; i++ {
			if !yield(i) { // 靠调用方 break 终止
				return
			}
		}
	}
}

func basicCombinators() {
	section("1.8 组合子")

	sq := Map(Filter(Naturals(), func(n int) bool { return n%3 == 0 }),
		func(n int) int { return n * n })
	fmt.Println("无限序列 + Filter + Map + Take:", slices.Collect(Take(sq, 5)))

	fmt.Println("Chain:", slices.Collect(Chain(Count(2), Count(3))))
}

// ---------------------------------------------------------------------------
// 1.9 单次使用迭代器
// ---------------------------------------------------------------------------

// scannerSeq returns a single-use iterator.
func scannerSeq(r *strings.Reader) iter.Seq[string] {
	sc := bufio.NewScanner(r)
	return func(yield func(string) bool) {
		for sc.Scan() {
			if !yield(sc.Text()) {
				return
			}
		}
	}
}

func basicSingleUse() {
	section("1.9 单次使用迭代器")

	seq := scannerSeq(strings.NewReader("1\n2\n3\n"))
	fmt.Println("第一次 Collect:", slices.Collect(seq))
	fmt.Println("第二次 Collect:", slices.Collect(seq), "<- 空的！文档必须写明 single-use")
}

// ---------------------------------------------------------------------------
// 3.1 忽略 yield 的返回值
// ---------------------------------------------------------------------------

func Bad(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range n {
			yield(i) // 没检查返回值
		}
	}
}

func trapIgnoreYieldResult() {
	section("3.1 忽略 yield 返回值")

	defer func() { fmt.Println("recover:", recover()) }()

	for v := range Bad(5) {
		if v == 1 {
			break // yield 返回 false，但迭代器继续调用 -> 状态机 panic
		}
	}
}

// ---------------------------------------------------------------------------
// 3.2 迭代器吞掉循环体的 panic
// ---------------------------------------------------------------------------

func swallow() iter.Seq[int] {
	return func(yield func(int) bool) {
		defer func() { recover() }() // 想"保护"，结果吞掉了 body 的 panic
		for i := range 3 {
			yield(i)
		}
	}
}

func trapSwallowPanic() {
	section("3.2 迭代器吞掉循环体 panic")

	defer func() { fmt.Println("recover:", recover()) }()

	for v := range swallow() {
		if v == 1 {
			panic("boom") // 循环体的 panic 属于调用方，迭代器无权吞
		}
	}
}

// ---------------------------------------------------------------------------
// 3.3 并发调用 yield
// ---------------------------------------------------------------------------

func trapConcurrentYield() {
	section("3.3 yield 必须串行、同 goroutine 调用")

	fmt.Println("并发 yield 会被状态机抓到并 panic（RF_PANIC）")
	fmt.Println("更坏的情况：panic 发生在迭代器自己起的 goroutine 里并被 recover，")
	fmt.Println("主循环会静默少收元素，什么错误都看不到。")
	fmt.Println("结论：yield 只在调用迭代器的那个 goroutine 里串行调用。")
}

// ---------------------------------------------------------------------------
// 3.4 iter.Pull 忘记 stop
// ---------------------------------------------------------------------------

func trapPullLeak() {
	section("3.4 Pull 忘记 stop 泄漏 goroutine")

	// ① 正确用法：defer stop()，退出时协程被回收
	before := runtime.NumGoroutine()
	func() {
		next, stop := iter.Pull(Count(1_000_000))
		defer stop()
		next()
	}()
	runtime.GC()
	fmt.Printf("defer stop: goroutine %d -> %d（无泄漏）\n", before, runtime.NumGoroutine())

	// ② 忘记 stop：协程一直阻塞在 yield 上，GC 也回收不了
	before = runtime.NumGoroutine()
	next, _ := iter.Pull(Count(1_000_000))
	next() // 只取一个就不要了

	runtime.GC()
	runtime.GC()
	fmt.Printf("忘记 stop: goroutine %d -> %d（coro 不会被 GC 回收）\n",
		before, runtime.NumGoroutine())
	fmt.Println("另外：迭代器停在 yield 里，它的 defer f.Close() 也不会执行")
}

// ---------------------------------------------------------------------------
// 3.6 单次使用迭代器被消费两次
// ---------------------------------------------------------------------------

func trapConsumeTwice() {
	section("3.6 先统计再遍历的事故")

	seq := scannerSeq(strings.NewReader("a\nb\nc\n"))
	n := len(slices.Collect(seq)) // 第一次消费掉了
	got := 0
	for range seq {
		got++
	}
	fmt.Printf("统计到 %d 条，实际遍历到 %d 条 -> 要复用先 Collect 成 slice\n", n, got)
}

// ---------------------------------------------------------------------------
// 3.7 Seq 不是集合：每次 range 都重跑整条链路
// ---------------------------------------------------------------------------

func trapSeqIsNotCollection() {
	section("3.7 Seq 不缓存结果")

	calls := 0
	expensive := func(v int) int { calls++; return v * 2 }

	seq := Map(slices.Values([]int{1, 2, 3}), expensive)
	_ = slices.Collect(seq)
	fmt.Println("第一次遍历后 expensive 调用次数:", calls)
	for range seq {
	}
	fmt.Println("第二次遍历后:", calls, "<- 链路被重新执行了一遍")

	fixed := slices.Collect(seq) // 需要多次使用就固化成 slice
	fmt.Println("固化后随便用:", fixed, len(fixed))
}

// ---------------------------------------------------------------------------
// 3.8 遍历过程中修改底层容器
// ---------------------------------------------------------------------------

func trapModifyWhileIterating() {
	section("3.8 遍历中修改容器")

	s := []int{1, 2, 3}
	seq := slices.Values(s) // 拿的是 range 开始时的 slice 头
	out := []int{}
	for v := range seq {
		s = append(s, v*10) // 扩容后迭代器看到的还是旧数组
		out = append(out, v)
	}
	fmt.Println("遍历中 append:", out, " 底层 slice 变成:", s)

	m := map[int]int{1: 1, 2: 2, 3: 3}
	deleted := 0
	for k := range maps.Keys(m) {
		delete(m, k) // 删是安全的（Go map 规则）；新增是否被遍历到未定义
		deleted++
	}
	fmt.Printf("遍历中 delete: 删了 %d 个，剩余 %d 个\n", deleted, len(m))
}

// ---------------------------------------------------------------------------
// 3.10 / 3.12 惰性：错误与 ctx 要显式传；构造 ≠ 执行
// ---------------------------------------------------------------------------

func trapLazyErrors() {
	section("3.10/3.12 惰性")

	fmt.Println("构造迭代器时什么都不会执行：")
	seq := loud(3)
	fmt.Println("  已构造，但还没有任何输出")
	fmt.Print("  开始 range: ")
	for range seq {
	}
	fmt.Println()

	fmt.Println("错误随值一起产出（Seq2[T, error]），不要藏在闭包变量里：")
	for v, err := range mayFail(3) {
		if err != nil {
			fmt.Println("  遇到错误，break:", err)
			break
		}
		fmt.Println("  值:", v)
	}
}

func loud(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range n {
			fmt.Print("产出", i, " ")
			if !yield(i) {
				return
			}
		}
	}
}

func mayFail(n int) iter.Seq2[int, error] {
	return func(yield func(int, error) bool) {
		for i := range n {
			if i == 2 {
				yield(0, fmt.Errorf("第 %d 个元素出错", i))
				return
			}
			if !yield(i, nil) {
				return
			}
		}
	}
}
