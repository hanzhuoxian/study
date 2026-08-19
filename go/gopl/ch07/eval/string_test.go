package eval

import (
	"reflect"
	"testing"
)

// TestString 验证 String 的输出能被重新解析，
// 且得到的语法树与原来的完全相同（练习 7.13）。
func TestString(t *testing.T) {
	tests := []struct {
		expr string
		want string // String() 的期望输出
	}{
		{"sqrt(A / pi)", "sqrt((A / pi))"},
		{"pow(x, 3) + pow(y, 3)", "(pow(x, 3) + pow(y, 3))"},
		{"5 / 9 * (F - 32)", "((5 / 9) * (F - 32))"},
		{"-x", "(-x)"},
		{"-1 - -2", "((-1) - (-2))"},
		{"a + b + c", "((a + b) + c)"},   // 加法左结合
		{"a + b * c", "(a + (b * c))"},   // 乘法优先级更高
		{"(a + b) * c", "((a + b) * c)"}, // 括号改变结合方式
		{"sin(-x) * pow(1.5, -r)", "(sin((-x)) * pow(1.5, (-r)))"},
		{"3.141592653589793", "3.141592653589793"},
	}
	for _, test := range tests {
		expr, err := Parse(test.expr)
		if err != nil {
			t.Errorf("Parse(%q) 失败: %v", test.expr, err)
			continue
		}

		// 1. String() 的输出符合预期
		got := expr.String()
		if got != test.want {
			t.Errorf("Parse(%q).String() = %q, want %q", test.expr, got, test.want)
		}

		// 2. 往返：重新解析 String() 的输出
		expr2, err := Parse(got)
		if err != nil {
			t.Errorf("重新解析 %q 失败: %v", got, err)
			continue
		}

		// 3. 两棵语法树深度相等
		if !reflect.DeepEqual(expr, expr2) {
			t.Errorf("往返后语法树不同:\n 原始 = %#v\n 重解析 = %#v", expr, expr2)
		}

		// 4. 再打印一次也应该完全一致（幂等）
		if got2 := expr2.String(); got2 != got {
			t.Errorf("String() 不幂等: %q -> %q", got, got2)
		}
	}
}

// TestStringEval 验证往返之后求值结果不变。
func TestStringEval(t *testing.T) {
	env := Env{"x": 2, "y": 3, "r": 5}
	for _, src := range []string{
		"x + y * 2", "pow(x, y) - sqrt(r)", "-x / (y + 1)", "sin(r) / r",
	} {
		expr, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		expr2, err := Parse(expr.String())
		if err != nil {
			t.Fatalf("重新解析 %q: %v", expr.String(), err)
		}
		if got, want := expr2.Eval(env), expr.Eval(env); got != want {
			t.Errorf("%s: 往返后求值 = %g, want %g", src, got, want)
		}
	}
}
