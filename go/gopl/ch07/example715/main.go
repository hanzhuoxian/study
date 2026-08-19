// **练习 7.15：** 编写一个从标准输入中读取一个单一表达式的程序，
// 用户及时地提供对于任意变量的值，然后在结果环境变量中计算表达式的值。
// 优雅的处理所有遇到的错误。
//
// 本程序从标准输入读取一个表达式，依次询问其中每个变量的值，
// 然后在得到的环境中求值并打印结果。
//
// 「优雅处理错误」体现在三处：解析/校验失败只中止本次循环并提示，
// 不会让程序崩溃；变量值不是合法数字时就地重问，不丢弃整个表达式；
// 结果为 NaN/±Inf 时给出人话解释。
//
// 用法：
//
//	go run ./ch07/example715
//	表达式> pow(x, 3) + pow(y, 3)
//	  x = 9
//	  y = 10
//	= 1729
//
// 读到 EOF（Ctrl-D）时退出。
package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hanzhuoxian/study/go/gopl/ch07/eval"
)

func main() {
	in := bufio.NewScanner(os.Stdin)
	fmt.Println("输入表达式后回车；Ctrl-D 退出。")
	for {
		// 一次循环处理一个表达式。任何错误都只中止本次循环，
		// 打印提示后回到这里继续，不会让程序崩溃。
		if err := once(in); err != nil {
			if err == io.EOF {
				fmt.Println("\n再见。")
				return
			}
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		}
	}
}

// once 读取并求值一个表达式。返回 io.EOF 表示输入结束。
func once(in *bufio.Scanner) error {
	line, err := prompt(in, "表达式> ")
	if err != nil {
		return err
	}
	if line == "" {
		return nil // 空行：什么也不做
	}

	expr, err := eval.Parse(line)
	if err != nil {
		return fmt.Errorf("无法解析: %w", err)
	}

	// Check 既校验函数名和参数个数，又顺带收集出所有变量名。
	vars := make(map[eval.Var]bool)
	if err := expr.Check(vars); err != nil {
		return fmt.Errorf("表达式非法: %w", err)
	}

	env, err := readEnv(in, vars)
	if err != nil {
		return err
	}

	result := expr.Eval(env)
	fmt.Printf("  %s\n= %s\n\n", expr, format(result))
	return nil
}

// readEnv 依次询问每个变量的值，并组装成求值环境。
func readEnv(in *bufio.Scanner, vars map[eval.Var]bool) (eval.Env, error) {
	// 排序后再询问，保证每次运行的提问顺序稳定。
	names := make([]string, 0, len(vars))
	for v := range vars {
		names = append(names, string(v))
	}
	sort.Strings(names)

	env := make(eval.Env, len(names))
	for _, name := range names {
		// 输入不是合法数字时就地重问，不丢弃整个表达式。
		for {
			text, err := prompt(in, "  "+name+" = ")
			if err != nil {
				return nil, err
			}
			f, err := strconv.ParseFloat(text, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    %q 不是合法的数字，请重新输入\n", text)
				continue
			}
			env[eval.Var(name)] = f
			break
		}
	}
	return env, nil
}

// prompt 打印提示符并读取一行，返回去掉首尾空白的内容。
// 输入结束时返回 io.EOF。
func prompt(in *bufio.Scanner, s string) (string, error) {
	fmt.Print(s)
	if !in.Scan() {
		if err := in.Err(); err != nil {
			return "", fmt.Errorf("读取输入失败: %w", err)
		}
		return "", io.EOF
	}
	return strings.TrimSpace(in.Text()), nil
}

// format 打印结果，并对非有限数给出解释。
func format(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN（未定义，例如 0/0 或负数开方）"
	case math.IsInf(f, 1):
		return "+Inf（正无穷，例如除以 0）"
	case math.IsInf(f, -1):
		return "-Inf（负无穷）"
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}
