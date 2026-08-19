package main

// **练习 7.13：** 为Expr增加一个String方法来打印美观的语法树。
// 当再一次解析的时候，检查它的结果是否生成相同的语法树。
//
// 实现在 ch07/eval/string.go：每个一元、二元运算符都带上自己的括号，
// 输出「完全括号化」的表达式，因此重新解析时不依赖运算符优先级，
// 必然得到结构完全相同的语法树。往返测试见 ch07/eval/string_test.go。
//
// 用法：go run ./ch07/example713

import (
	"fmt"
	"os"
	"reflect"

	"github.com/hanzhuoxian/study/go/gopl/ch07/eval"
)

func main() {
	exprs := []string{
		"sqrt(A / pi)",
		"pow(x, 3) + pow(y, 3)",
		"5 / 9 * (F - 32)",
		"a + b + c",   // 加法左结合
		"a + b * c",   // 乘法优先级更高
		"(a + b) * c", // 括号改变结合方式
		"-1 - -2",     // 一元运算符
		"sin(-x) * pow(1.5, -r)",
		"3.141592653589793",
	}

	failed := false
	for _, src := range exprs {
		if err := roundTrip(src); err != nil {
			fmt.Fprintf(os.Stderr, "%-24s %v\n", src, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

// roundTrip 打印语法树，再解析一次，并检查两棵树是否完全相同。
func roundTrip(src string) error {
	expr, err := eval.Parse(src)
	if err != nil {
		return fmt.Errorf("解析失败: %w", err)
	}
	printed := expr.String()

	expr2, err := eval.Parse(printed)
	if err != nil {
		return fmt.Errorf("重新解析 %q 失败: %w", printed, err)
	}

	// DeepEqual 逐层比较接口里的具体类型和字段值，
	// 能真正验证「相同的语法树」，而不只是相同的字符串。
	if !reflect.DeepEqual(expr, expr2) {
		return fmt.Errorf("往返后语法树不同:\n 原始 = %#v\n 重解析 = %#v", expr, expr2)
	}

	fmt.Printf("%-24s => %s\n", src, printed)
	return nil
}
