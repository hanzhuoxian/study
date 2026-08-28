// map 示例：对应 notes/map.md（Go 1.24+ Swiss Table 实现）
// 运行：go run ./map
package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func section(title string) { fmt.Printf("\n===== %s =====\n", title) }

func main() {
	// 子进程模式：演示并发读写 map 触发的 fatal error（见 3.4）
	if os.Getenv("MAP_DEMO_CONCURRENT") == "1" {
		concurrentCrash()
		return
	}

	basicDeclare()
	basicCRUD()
	basicRange()
	basicAsParam()
	basicNested()
	basicNotAddressable()

	growthHint()
	iterationRandom()

	trapNilWrite()
	trapConcurrent()
	trapKeyComparable()
	trapNaNKey()
	trapDeleteNoShrink()
	trapZeroValue()
	safeConcurrentMap()
}

// ---------------------------------------------------------------------------
// 1.1 声明与初始化
// ---------------------------------------------------------------------------

func basicDeclare() {
	section("1.1 声明与初始化")

	var m1 map[string]int  // nil map：可读，写入 panic
	m2 := map[string]int{} // 空 map
	m3 := map[string]int{"a": 1, "b": 2}
	m4 := make(map[string]int)
	m5 := make(map[string]int, 100) // hint：预估元素个数，减少扩容次数

	fmt.Printf("m1==nil:%t len=%d  m2==nil:%t  m3=%v len(m3)=%d\n",
		m1 == nil, len(m1), m2 == nil, m3, len(m3))
	fmt.Println("读 nil map 返回零值:", m1["nothing"])
	_, _ = m4, m5
}

// ---------------------------------------------------------------------------
// 1.2 读写删除
// ---------------------------------------------------------------------------

func basicCRUD() {
	section("1.2 读写删除")

	m := map[string]int{"a": 1}

	fmt.Println("m[\"a\"] =", m["a"])
	fmt.Println("m[\"x\"] =", m["x"], "（不存在返回零值，不 panic）")

	if v, ok := m["x"]; !ok {
		fmt.Printf("comma-ok: v=%d ok=%t\n", v, ok)
	}

	m["b"] = 2
	delete(m, "a")
	delete(m, "not-exist") // 空操作，不 panic
	fmt.Println("m =", m, "len =", len(m))
}

// ---------------------------------------------------------------------------
// 1.3 range 遍历（顺序随机）
// ---------------------------------------------------------------------------

func basicRange() {
	section("1.3 range 遍历")

	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	for range 3 {
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		fmt.Println("一次遍历顺序:", keys)
	}

	// 需要稳定顺序：取出 key 排序后再访问
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Print("有序遍历: ")
	for _, k := range keys {
		fmt.Printf("%s=%d ", k, m[k])
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// 1.4 map 作为函数参数：指针语义
// ---------------------------------------------------------------------------

func modify(m map[string]int) {
	m["x"] = 1
	for i := range 1000 { // 内部触发多次扩容，调用方依旧看得到
		m[strconv.Itoa(i)] = i
	}
}

func basicAsParam() {
	section("1.4 map 传参是指针语义")

	m := map[string]int{}
	modify(m)
	fmt.Printf("调用方可见: m[\"x\"]=%d len=%d（扩容也不影响可见性）\n", m["x"], len(m))

	// 对照：给形参重新赋值不会影响调用方（拷贝的是那个指针）
	reassign(m)
	fmt.Println("重新赋值形参后调用方 len =", len(m))
}

func reassign(m map[string]int) { m = map[string]int{}; _ = m }

// ---------------------------------------------------------------------------
// 1.5 嵌套 map
// ---------------------------------------------------------------------------

func basicNested() {
	section("1.5 嵌套 map")

	grid := make(map[string]map[string]int)
	grid["a"] = make(map[string]int) // 内层必须单独初始化
	grid["a"]["x"] = 1
	fmt.Println("grid =", grid)

	// grid["b"]["y"] = 1 // panic: assignment to entry in nil map
	if _, ok := grid["b"]; !ok {
		fmt.Println("grid[\"b\"] 是 nil map，写入会 panic；先判断再初始化")
		grid["b"] = map[string]int{}
		grid["b"]["y"] = 2
	}
	fmt.Println("grid =", grid)
}

// ---------------------------------------------------------------------------
// 1.6 map 元素不可寻址
// ---------------------------------------------------------------------------

type Point struct{ X, Y int }

func basicNotAddressable() {
	section("1.6 map 元素不可寻址")

	m := map[string]Point{"a": {1, 2}}
	// m["a"].X = 10 // 编译错误：cannot assign to struct field
	// p := &m["a"]  // 编译错误：invalid operation, cannot take address

	p := m["a"] // 取出（值拷贝）
	p.X = 10
	m["a"] = p // 写回
	fmt.Println("取出-改-写回:", m["a"])

	m2 := map[string]*Point{"a": {1, 2}}
	m2["a"].X = 10 // 改的是指针指向的对象，合法
	fmt.Println("value 用指针:", *m2["a"])
}

// ---------------------------------------------------------------------------
// 2.2 扩容：预分配 hint 的收益
// ---------------------------------------------------------------------------

func growthHint() {
	section("2.2 预分配 hint 的收益")

	const n = 1 << 20

	start := time.Now()
	m1 := make(map[int]int) // 不给 hint：反复 grow / split
	for i := range n {
		m1[i] = i
	}
	noHint := time.Since(start)

	start = time.Now()
	m2 := make(map[int]int, n) // 一次性分配足够容量
	for i := range n {
		m2[i] = i
	}
	withHint := time.Since(start)

	fmt.Printf("插入 %d 个 key: 无 hint %v, 有 hint %v (%.2fx)\n",
		n, noHint.Round(time.Millisecond), withHint.Round(time.Millisecond),
		float64(noHint)/float64(withHint))
}

// ---------------------------------------------------------------------------
// 2.4 遍历随机化：统计首个 key 的分布
// ---------------------------------------------------------------------------

func iterationRandom() {
	section("2.4 遍历起始位置随机化")

	m := map[int]int{}
	for i := range 8 {
		m[i] = i
	}

	first := map[int]int{}
	for range 10000 {
		for k := range m {
			first[k]++
			break
		}
	}
	keys := make([]int, 0, len(first))
	for k := range first {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	fmt.Print("10000 次遍历中各 key 作为第一个出现的次数: ")
	for _, k := range keys {
		fmt.Printf("%d:%d ", k, first[k])
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// 3.1 nil map 只读不可写
// ---------------------------------------------------------------------------

func trapNilWrite() {
	section("3.1 nil map 写入 panic")

	var m map[string]int
	fmt.Println("读 nil map:", m["a"], "len =", len(m))
	for range m { // range nil map 是零次循环，也不 panic
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recover:", r) // assignment to entry in nil map
		}
	}()
	m["a"] = 1
}

// ---------------------------------------------------------------------------
// 3.4 并发读写：fatal error，recover 抓不住
// ---------------------------------------------------------------------------

func trapConcurrent() {
	section("3.4 并发读写 map 是 fatal error")

	// 在子进程里跑，避免把当前进程搞崩
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "MAP_DEMO_CONCURRENT=1")
	out, err := cmd.CombinedOutput()

	firstLine := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	fmt.Printf("子进程退出: %v\n输出首行: %s\n", err, firstLine)
	fmt.Println("注意：这是 fatal error（运行时主动终止），defer/recover 无法捕获")
}

func concurrentCrash() {
	defer func() { recover() }() // 拦不住 fatal error

	m := map[int]int{}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for i := range 1_000_000 {
				m[i] = i
				_ = m[i]
			}
		})
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// 3.5 key 必须可比较
// ---------------------------------------------------------------------------

type Key struct{ A, B int } // 字段都可比较 -> 可以做 key

func trapKeyComparable() {
	section("3.5 key 必须可比较")

	// var bad map[[]int]string          // 编译错误：invalid map key type []int
	// var bad2 map[map[int]int]string   // 编译错误
	// var bad3 map[func()]string        // 编译错误

	m := map[Key]string{{1, 2}: "a"}
	fmt.Println("struct 做 key:", m[Key{1, 2}], "（逐字段值比较）")

	arr := map[[2]int]string{{1, 2}: "b"} // 数组可比较
	fmt.Println("数组做 key:", arr[[2]int{1, 2}])

	// any 做 key 可以编译，但塞进不可比较的动态类型会运行时 panic
	defer func() { fmt.Println("recover:", recover()) }()
	anyKey := map[any]int{}
	anyKey[[]int{1}] = 1
}

// ---------------------------------------------------------------------------
// 3.6 NaN 作为 key
// ---------------------------------------------------------------------------

func trapNaNKey() {
	section("3.6 NaN 作为 key")

	m := map[float64]string{}
	nan := math.NaN()
	m[nan] = "a"
	m[nan] = "b" // NaN != NaN，被当成新 key
	fmt.Printf("len=%d（不是 1）  m[nan]=%q（永远查不到）\n", len(m), m[nan])

	for k, v := range m {
		fmt.Printf("实际存了: k=%v v=%s\n", k, v)
	}
}

// ---------------------------------------------------------------------------
// 3.7 delete 不会收缩内存
// ---------------------------------------------------------------------------

func trapDeleteNoShrink() {
	section("3.7 delete 不收缩内存")

	m := make(map[int][128]byte)
	for i := range 200_000 {
		m[i] = [128]byte{}
	}
	full := heapMB()

	for i := range 200_000 {
		delete(m, i)
	}
	runtime.GC()
	afterDelete := heapMB()

	// 真正要回收内存：新建 map 搬运存活数据
	m2 := make(map[int][128]byte, len(m))
	for k, v := range m {
		m2[k] = v
	}
	m = m2
	runtime.GC()
	afterRebuild := heapMB()

	fmt.Printf("满载 %.1fMB -> 全部 delete 后 %.1fMB -> 重建 map 后 %.1fMB (len=%d)\n",
		full, afterDelete, afterRebuild, len(m))
}

func heapMB() float64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.HeapAlloc) / (1 << 20)
}

// ---------------------------------------------------------------------------
// 3.8 零值和"不存在"
// ---------------------------------------------------------------------------

func trapZeroValue() {
	section("3.8 零值 vs 不存在")

	m := map[string]int{"a": 0}
	fmt.Println("m[\"a\"] =", m["a"], " m[\"x\"] =", m["x"], "（看不出区别）")

	for _, k := range []string{"a", "x"} {
		if v, ok := m[k]; ok {
			fmt.Printf("%q 存在，值为 %d\n", k, v)
		} else {
			fmt.Printf("%q 不存在\n", k)
		}
	}
}

// ---------------------------------------------------------------------------
// 并发安全的两种写法：RWMutex vs sync.Map
// ---------------------------------------------------------------------------

type SafeMap struct {
	mu sync.RWMutex
	m  map[string]int
}

func (s *SafeMap) Get(k string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[k]
	return v, ok
}

func (s *SafeMap) Set(k string, v int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = v
}

func safeConcurrentMap() {
	section("并发安全：RWMutex vs sync.Map")

	sm := &SafeMap{m: map[string]int{}}
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() { sm.Set(strconv.Itoa(i%10), i) })
	}
	wg.Wait()
	v, ok := sm.Get("1")
	fmt.Printf("RWMutex map: len=%d get(\"1\")=%d,%t\n", len(sm.m), v, ok)

	// sync.Map：读多写少、key 集合稳定时更划算
	var syncMap sync.Map
	for i := range 100 {
		wg.Go(func() { syncMap.Store(i%10, i) })
	}
	wg.Wait()
	cnt := 0
	syncMap.Range(func(_, _ any) bool { cnt++; return true })
	got, loaded := syncMap.Load(1)
	fmt.Printf("sync.Map:    len=%d load(1)=%v,%t\n", cnt, got, loaded)
}
