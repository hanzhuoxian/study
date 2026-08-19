package eval

import (
	"bytes"
	"fmt"
	"strconv"
)

// String 方法把语法树还原成表达式字符串。
//
// 输出是「完全括号化」的：每个一元和二元运算符都带上自己的括号。
// 这样字符串就不依赖运算符优先级，重新解析后必然得到
// 结构完全相同的语法树（参见 TestString 的往返测试）。

func (v Var) String() string { return string(v) }

func (l literal) String() string {
	// 'g' 加精度 -1 表示「用最少的位数精确表示该 float64」，
	// 保证 ParseFloat(l.String()) 能还原出同一个值。
	return strconv.FormatFloat(float64(l), 'g', -1, 64)
}

func (u unary) String() string {
	return fmt.Sprintf("(%c%s)", u.op, u.x)
}

func (b binary) String() string {
	return fmt.Sprintf("(%s %c %s)", b.x, b.op, b.y)
}

func (c call) String() string {
	var buf bytes.Buffer
	buf.WriteString(c.fn)
	buf.WriteByte('(')
	for i, arg := range c.args {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(arg.String())
	}
	buf.WriteByte(')')
	return buf.String()
}
