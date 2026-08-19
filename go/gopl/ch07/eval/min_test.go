package eval

import (
	"math"
	"reflect"
	"testing"
)

// TestMinDirect 直接构造语法树来使用 Min（练习 7.14）。
func TestMinDirect(t *testing.T) {
	// 手工搭出 min(x, y, 3) 这棵树，不经过 Parse。
	expr := Min{Args: []Expr{Var("x"), Var("y"), literal(3)}}

	if err := expr.Check(make(map[Var]bool)); err != nil {
		t.Fatalf("Check: %v", err)
	}

	tests := []struct {
		env  Env
		want float64
	}{
		{Env{"x": 1, "y": 2}, 1},
		{Env{"x": 5, "y": 4}, 3}, // 常量 3 最小
		{Env{"x": -7, "y": 0}, -7},
	}
	for _, test := range tests {
		if got := expr.Eval(test.env); got != test.want {
			t.Errorf("%s.Eval(%v) = %g, want %g", expr, test.env, got, test.want)
		}
	}

	if got, want := expr.String(), "min(x, y, 3)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// TestMinNested 把 Min 嵌进由 Parse 构造的树里，
// 验证新类型和已有类型可以自由组合。
func TestMinNested(t *testing.T) {
	sub, err := Parse("pow(x, 2)")
	if err != nil {
		t.Fatal(err)
	}
	// 相当于 (min(x, pow(x, 2)) + 1)
	expr := binary{'+', Min{Args: []Expr{Var("x"), sub}}, literal(1)}

	if got, want := expr.String(), "(min(x, pow(x, 2)) + 1)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got, want := expr.Eval(Env{"x": 0.5}), 1.25; got != want {
		t.Errorf("Eval = %g, want %g", got, want) // min(0.5, 0.25) + 1
	}
}

// TestMinParse 验证扩展后的 Parse 能识别变参的 min。
func TestMinParse(t *testing.T) {
	tests := []struct {
		expr string
		env  Env
		want float64
	}{
		{"min(3)", Env{}, 3},
		{"min(x, y)", Env{"x": 2, "y": 9}, 2},
		{"min(x, y, 1, -4)", Env{"x": 2, "y": 9}, -4},
		{"min(x, y) * 2", Env{"x": 2, "y": 9}, 4},
		{"min(min(x, 0), y)", Env{"x": 2, "y": 9}, 0}, // 可嵌套
	}
	for _, test := range tests {
		expr, err := Parse(test.expr)
		if err != nil {
			t.Errorf("Parse(%q): %v", test.expr, err)
			continue
		}
		if err := expr.Check(make(map[Var]bool)); err != nil {
			t.Errorf("Check(%q): %v", test.expr, err)
			continue
		}
		if got := expr.Eval(test.env); got != test.want {
			t.Errorf("%s = %g, want %g", test.expr, got, test.want)
		}
	}
}

// TestMinRoundTrip 确认 Min 也满足 7.13 的往返性质。
func TestMinRoundTrip(t *testing.T) {
	expr, err := Parse("min(x, y, 3) + 1")
	if err != nil {
		t.Fatal(err)
	}
	expr2, err := Parse(expr.String())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expr, expr2) {
		t.Errorf("往返后语法树不同:\n %#v\n %#v", expr, expr2)
	}
}

// TestMinErrors 验证参数个数与 NaN 的处理。
func TestMinErrors(t *testing.T) {
	// 零个参数应当被 Check 拒绝。
	expr, err := Parse("min()")
	if err != nil {
		t.Fatalf("Parse(%q): %v", "min()", err)
	}
	if err := expr.Check(make(map[Var]bool)); err == nil {
		t.Error("min() 应当报错，但 Check 通过了")
	} else {
		t.Logf("min() 的报错信息: %v", err)
	}

	// NaN 应当向外传播，而不是被 < 比较悄悄忽略。
	nan, _ := Parse("min(1, sqrt(-1), 2)")
	if got := nan.Eval(Env{}); !math.IsNaN(got) {
		t.Errorf("min(1, NaN, 2) = %g, want NaN", got)
	}
}
