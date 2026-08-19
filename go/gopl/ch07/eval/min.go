package eval

import (
	"bytes"
	"fmt"
)

// Min 表示对若干运算数取最小值的表达式，例如 min(x, y, 3)。
//
// 这是练习 7.14 要求的「新的具体类型」。它和已有的 call 不同：
// call 的每个函数都有固定的参数个数（见 numParams），
// 而 Min 接受任意数量（至少一个）的参数。
//
// Min 及其字段都是导出的，因此包外代码可以直接构造语法树：
//
//	eval.Min{Args: []eval.Expr{eval.Var("x"), eval.Var("y")}}
type Min struct {
	Args []Expr
}

func (m Min) Eval(env Env) float64 {
	// Check 已保证 Args 非空。
	result := m.Args[0].Eval(env)
	for _, arg := range m.Args[1:] {
		// 内置的 min 在任一操作数为 NaN 时返回 NaN，
		// 这正是我们想要的传播行为。
		result = min(result, arg.Eval(env))
	}
	return result
}

func (m Min) Check(vars map[Var]bool) error {
	if len(m.Args) == 0 {
		return fmt.Errorf("call to min has 0 args, want at least 1")
	}
	for _, arg := range m.Args {
		if err := arg.Check(vars); err != nil {
			return err
		}
	}
	return nil
}

func (m Min) String() string {
	var buf bytes.Buffer
	buf.WriteString("min(")
	for i, arg := range m.Args {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(arg.String())
	}
	buf.WriteByte(')')
	return buf.String()
}
