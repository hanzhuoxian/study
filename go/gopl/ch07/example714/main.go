package main

// **练习 7.14：** 定义一个新的满足Expr接口的具体类型并且提供一个新的操作
// 例如对它运算单元中的最小值的计算。因为Parse函数不会创建这个新类型的实例，
// 为了使用它你可能需要直接构造一个语法树（或者继承parser接口）。
//
// 新类型是 ch07/eval/min.go 里的 eval.Min：它表示 min(x, y, ...)，
// 与已有的 call 不同——call 的每个函数参数个数固定（见 numParams），
// 而 Min 接受任意数量（至少一个）的参数，是变参的。
//
// 本练习两条路都走通了：
//  1. 直接构造语法树（Min 及其字段都是导出的，包外可以自由拼装）；
//  2. 扩展 parser（parsePrimary 遇到 min 就返回 Min），这样字符串里也能写 min。
//
// 用法：go run ./ch07/example714

import (
	"fmt"
	"log"

	"github.com/hanzhuoxian/study/go/gopl/ch07/eval"
)

func main() {
	fmt.Println("== 1. 直接构造语法树 ==")
	// 手工搭出 min(x, y, 3) 这棵树，完全不经过 Parse。
	tree := eval.Min{Args: []eval.Expr{
		eval.Var("x"),
		eval.Var("y"),
		eval.Literal(3),
	}}
	if err := tree.Check(make(map[eval.Var]bool)); err != nil {
		log.Fatal(err)
	}
	for _, env := range []eval.Env{
		{"x": 1, "y": 2},
		{"x": 5, "y": 4}, // 常量 3 最小
		{"x": -7, "y": 0},
	} {
		fmt.Printf("  %s\t%v => %g\n", tree, env, tree.Eval(env))
	}

	fmt.Println("\n== 2. 与已有类型自由组合 ==")
	// 把 Parse 出来的子树塞进手工构造的 Min 里：(min(x, pow(x, 2)) + 1)
	sub, err := eval.Parse("pow(x, 2)")
	if err != nil {
		log.Fatal(err)
	}
	mixed := eval.Min{Args: []eval.Expr{eval.Var("x"), sub}}
	env := eval.Env{"x": 0.5}
	fmt.Printf("  %s\t%v => %g\n", mixed, env, mixed.Eval(env))

	fmt.Println("\n== 3. 扩展后的 Parse 也能识别 min ==")
	for _, src := range []string{
		"min(3)",
		"min(x, y)",
		"min(x, y, 1, -4)",
		"min(min(x, 0), y) * 2", // 可嵌套、可参与运算
	} {
		expr, err := eval.Parse(src)
		if err != nil {
			log.Fatal(err)
		}
		if err := expr.Check(make(map[eval.Var]bool)); err != nil {
			log.Fatal(err)
		}
		env := eval.Env{"x": 2, "y": 9}
		fmt.Printf("  %-22s => %-26s = %g\n", src, expr, expr.Eval(env))
	}

	fmt.Println("\n== 4. 错误处理 ==")
	// min 至少要有一个参数，这由 Check 而不是 numParams 保证。
	empty, err := eval.Parse("min()")
	if err != nil {
		log.Fatal(err)
	}
	if err := empty.Check(make(map[eval.Var]bool)); err != nil {
		fmt.Printf("  min() => %v\n", err)
	}
}
