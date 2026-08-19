package eval

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

// lexer 是词法分析器，它包装 text/scanner，
// 并始终保存一个向前看的记号（lookahead token）。
type lexer struct {
	scan  scanner.Scanner
	token rune // 当前向前看的记号
}

// next 读取下一个记号。
func (lex *lexer) next() { lex.token = lex.scan.Scan() }

// text 返回当前记号的字面文本。
func (lex *lexer) text() string { return lex.scan.TokenText() }

// lexPanic 是语法错误的 panic 值类型，
// 用它把解析过程深处的错误一次性抛回给 Parse。
type lexPanic string

// describe 返回描述当前记号的字符串，用于错误消息。
func (lex *lexer) describe() string {
	switch lex.token {
	case scanner.EOF:
		return "文件结束"
	case scanner.Ident:
		return fmt.Sprintf("标识符 %s", lex.text())
	case scanner.Int, scanner.Float:
		return fmt.Sprintf("数字 %s", lex.text())
	}
	return fmt.Sprintf("%q", rune(lex.token)) // 其他任意字符
}

// precedence 返回二元运算符的优先级，数值越大结合越紧。
// 非运算符返回 0。
func precedence(op rune) int {
	switch op {
	case '*', '/':
		return 2
	case '+', '-':
		return 1
	}
	return 0
}

// Parse 把输入字符串解析为算术表达式。
//
//	expr = num                         数值常量，例如 3.14159
//	     | id                          变量名，例如 x
//	     | id '(' expr ',' ... ')'     函数调用
//	     | '-' expr                    一元运算符（+-）
//	     | expr '+' expr               二元运算符（+-*/）
func Parse(input string) (_ Expr, err error) {
	defer func() {
		switch x := recover().(type) {
		case nil:
			// 没有 panic，正常返回
		case lexPanic:
			err = fmt.Errorf("%s", x)
		default:
			// 非预期的 panic：保持原有的 panic 状态继续向上抛
			panic(x)
		}
	}()
	lex := new(lexer)
	lex.scan.Init(strings.NewReader(input))
	lex.scan.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats
	lex.next() // 初始化向前看的记号
	e := parseExpr(lex)
	if lex.token != scanner.EOF {
		return nil, fmt.Errorf("表达式结尾出现意外的 %s", lex.describe())
	}
	return e, nil
}

func parseExpr(lex *lexer) Expr { return parseBinary(lex, 1) }

// binary = unary ('+' binary)*
// 当遇到优先级低于 prec1 的运算符时，parseBinary 停止解析。
func parseBinary(lex *lexer, prec1 int) Expr {
	lhs := parseUnary(lex)
	for prec := precedence(lex.token); prec >= prec1; prec-- {
		for precedence(lex.token) == prec {
			op := lex.token
			lex.next() // 消费运算符
			rhs := parseBinary(lex, prec+1)
			lhs = binary{op, lhs, rhs}
		}
	}
	return lhs
}

// unary = '+' expr | '-' expr | primary
func parseUnary(lex *lexer) Expr {
	if lex.token == '+' || lex.token == '-' {
		op := lex.token
		lex.next() // 消费 '+' 或 '-'
		return unary{op, parseUnary(lex)}
	}
	return parsePrimary(lex)
}

// primary = id
//
//	| id '(' expr ',' ... ',' expr ')'
//	| num
//	| '(' expr ')'
func parsePrimary(lex *lexer) Expr {
	switch lex.token {
	case scanner.Ident:
		id := lex.text()
		lex.next() // 消费标识符
		if lex.token != '(' {
			return Var(id) // 普通变量
		}
		// 函数调用
		lex.next() // 消费 '('
		var args []Expr
		if lex.token != ')' {
			for {
				args = append(args, parseExpr(lex))
				if lex.token != ',' {
					break
				}
				lex.next() // 消费 ','
			}
			if lex.token != ')' {
				msg := fmt.Sprintf("得到 %s，期望 ')'", lex.describe())
				panic(lexPanic(msg))
			}
		}
		lex.next() // 消费 ')'
		if id == "min" {
			// min 是变参函数，用专门的 Min 类型表示，而不是 call。
			return Min{Args: args}
		}
		return call{id, args}

	case scanner.Int, scanner.Float:
		f, err := strconv.ParseFloat(lex.text(), 64)
		if err != nil {
			panic(lexPanic(err.Error()))
		}
		lex.next() // 消费数字
		return literal(f)

	case '(':
		lex.next() // 消费 '('
		e := parseExpr(lex)
		if lex.token != ')' {
			msg := fmt.Sprintf("得到 %s，期望 ')'", lex.describe())
			panic(lexPanic(msg))
		}
		lex.next() // 消费 ')'
		return e
	}
	msg := fmt.Sprintf("意外的 %s", lex.describe())
	panic(lexPanic(msg))
}
