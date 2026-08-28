package tst

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// go test -bench . -benchmem ./tst
// 只跑一个：go test -bench BenchmarkSumLoop -benchtime 100x ./tst
// 稳定测量：go test -bench . -count 10 ./tst | tee old.txt && benchstat old.txt new.txt

var (
	intSink int
	strSink string
	errSink error
)

// ---------------------------------------------------------------------------
// 1. b.Loop（1.24+）vs 老的 b.N 写法
// ---------------------------------------------------------------------------

// 推荐写法：b.Loop 自动处理计时器和"防止被优化掉"
func BenchmarkSumLoop(b *testing.B) {
	lo, hi := 1, 1000 // setup 不计时（Loop 第一次调用时重置计时器）
	for b.Loop() {
		intSink = Sum(lo, hi)
	}
}

// 老写法：等价，但有两个历史包袱
func BenchmarkSumOldStyle(b *testing.B) {
	lo, hi := 1, 1000
	b.ResetTimer() // ① 要手动重置计时器
	for range b.N {
		intSink = Sum(lo, hi) // ② 要自己想办法防止被优化掉（赋值给包级变量）
	}
}

// b.Loop 的两个额外好处：
//   - 循环体内的参数/返回值/赋值会被 runtime.KeepAlive 保活，编译器无法整段消除
//   - 只执行一次 setup/cleanup 的语义更清晰（Loop 返回 false 时停表）
func BenchmarkLoopKeepsAlive(b *testing.B) {
	for b.Loop() {
		// 即使不赋值给包级变量，结果也不会被优化掉
		_ = Sum(1, 100)
	}
}

// ---------------------------------------------------------------------------
// 2. 子 benchmark：b.Run + 规模扫描
// ---------------------------------------------------------------------------

func BenchmarkSumSizes(b *testing.B) {
	for _, n := range []int{10, 1000, 100000} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs() // 等价于命令行 -benchmem，写在代码里更保险
			for b.Loop() {
				intSink = Sum(1, n)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. 自定义指标：b.ReportMetric
// ---------------------------------------------------------------------------

func BenchmarkCustomMetric(b *testing.B) {
	const n = 1000
	count := 0
	for b.Loop() {
		intSink = Sum(1, n)
		count += n
	}
	// 除了 ns/op，还能报自己的单位
	b.ReportMetric(float64(count)/b.Elapsed().Seconds()/1e6, "Mitem/s")
	b.ReportMetric(float64(n), "items/op")
}

// ---------------------------------------------------------------------------
// 4. RunParallel：测并发扩展性
// ---------------------------------------------------------------------------

func BenchmarkParseParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, errSink = ParseRange("3-7")
		}
	})
}

// ---------------------------------------------------------------------------
// 5. 常见的 benchmark 写错示范（注释形式，避免真的误导数据）
// ---------------------------------------------------------------------------

// ✗ 把 setup 写在循环里：测出来的几乎全是 setup 的耗时
func BenchmarkWrongSetupInLoop(b *testing.B) {
	for b.Loop() {
		pad := strings.Repeat("x", 10000) // ← 这行才是大头
		_, _, errSink = ParseRange("3-7")
		strSink = pad[:1]
	}
}

// ✓ setup 挪到循环外，测的才是 ParseRange
func BenchmarkRightSetupOutside(b *testing.B) {
	pad := strings.Repeat("x", 10000)
	for b.Loop() {
		_, _, errSink = ParseRange("3-7")
		strSink = pad[:1]
	}
}

// ✗ 结果没被使用（老写法下会被编译器整段删掉，b.Loop 保护了这种情况）
func BenchmarkResultUnusedOldStyle(b *testing.B) {
	for range b.N {
		_ = Sum(1, 100) // 用 b.N 时这行可能被优化掉，得到虚假的 0.3 ns/op
	}
}

// ---------------------------------------------------------------------------
// 6. Example：既是文档又是测试
// ---------------------------------------------------------------------------

// 有 Output 注释：go test 会执行并比对输出
func ExampleParseRange() {
	lo, hi, err := ParseRange("3-7")
	fmt.Println(lo, hi, err)
	// Output: 3 7 <nil>
}

// 无 Output 注释：只编译不执行（适合演示"会阻塞/需要网络"的用法）
func ExampleParseRange_noOutput() {
	lo, hi, _ := ParseRange("5")
	fmt.Println(lo, hi)
}

// 命名规则：ExampleF / ExampleT / ExampleT_M / ExampleF_suffix（suffix 必须小写开头）
// 这些函数会出现在 go doc 和 pkg.go.dev 的文档里
func ExampleSum() {
	fmt.Println(Sum(1, 100))
	// Output: 5050
}
