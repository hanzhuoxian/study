package eval

import (
	"fmt"
	"math"
	"strings"
)

// Var 表示一个变量，例如 x。
type Var string

// literal 表示一个数值常量，例如 3.141。
type literal float64

// Literal 把一个数值包装成表达式。
//
// literal 本身是不导出的，而练习 7.14 需要在包外直接构造语法树
// （例如 eval.Min{Args: []eval.Expr{eval.Var("x"), eval.Literal(3)}}），
// 所以这里提供一个导出的构造函数。
func Literal(f float64) Expr { return literal(f) }

// unary 表示一元运算符表达式，例如 -x。
type unary struct {
	op rune // '+' 或 '-' 之一
	x  Expr
}

// binary 表示二元运算符表达式，例如 x+y。
type binary struct {
	op   rune // '+'、'-'、'*'、'/' 之一
	x, y Expr
}

// call 表示函数调用表达式，例如 sin(x)。
type call struct {
	fn   string // "pow"、"sin"、"sqrt" 之一
	args []Expr
}

type Env map[Var]float64

// Expr 表示一个算术表达式。
type Expr interface {
	Eval(env Env) float64
	Check(vars map[Var]bool) error

	// String 返回该表达式的字符串形式，重新解析它可得到相同的语法树。
	String() string
}

func (v Var) Eval(env Env) float64 {
	return env[v]
}

func (l literal) Eval(_ Env) float64 {
	return float64(l)
}

func (u unary) Eval(env Env) float64 {
	switch u.op {
	case '+':
		return u.x.Eval(env)
	case '-':
		return -u.x.Eval(env)
	default:
		panic("invalid unary operator")
	}
}

func (b binary) Eval(env Env) float64 {
	switch b.op {
	case '+':
		return b.x.Eval(env) + b.y.Eval(env)
	case '-':
		return b.x.Eval(env) - b.y.Eval(env)
	case '*':
		return b.x.Eval(env) * b.y.Eval(env)
	case '/':
		return b.x.Eval(env) / b.y.Eval(env)
	default:
		panic("invalid binary operator")
	}
}

func (c call) Eval(env Env) float64 {
	switch c.fn {
	case "pow":
		return math.Pow(c.args[0].Eval(env), c.args[1].Eval(env))
	case "sin":
		return math.Sin(c.args[0].Eval(env))
	case "sqrt":
		return math.Sqrt(c.args[0].Eval(env))
	default:
		panic("invalid function")
	}
}

func (v Var) Check(vars map[Var]bool) error {
	vars[v] = true
	return nil
}

func (literal) Check(vars map[Var]bool) error {
	return nil
}

func (u unary) Check(vars map[Var]bool) error {
	if !strings.ContainsRune("+-", u.op) {
		return fmt.Errorf("unexpected unary op %q", u.op)
	}
	return u.x.Check(vars)
}

func (b binary) Check(vars map[Var]bool) error {
	if !strings.ContainsRune("+-*/", b.op) {
		return fmt.Errorf("unexpected binary op %q", b.op)
	}
	if err := b.x.Check(vars); err != nil {
		return err
	}
	return b.y.Check(vars)
}

func (c call) Check(vars map[Var]bool) error {
	arity, ok := numParams[c.fn]
	if !ok {
		return fmt.Errorf("unknown function %q", c.fn)
	}
	if len(c.args) != arity {
		return fmt.Errorf("call to %s has %d args, want %d",
			c.fn, len(c.args), arity)
	}
	for _, arg := range c.args {
		if err := arg.Check(vars); err != nil {
			return err
		}
	}
	return nil
}

var numParams = map[string]int{"pow": 2, "sin": 1, "sqrt": 1}
