// **练习 7.16：** 编写一个基于web的计算器程序。
//
// webcalc 是练习 7.15 的 web 版本：同一个 eval 包，换一层 HTTP 界面。
//
// 用法：
//
//	go run ./ch07/example716
//	打开 http://localhost:8080/
//
// 页面分两步：先提交表达式，服务端解析出其中的变量后
// 生成对应的输入框；填好变量值再提交即得到结果。
// 表单用 GET 提交，因此任何一次计算的 URL 都可以直接分享，例如
//
//	http://localhost:8080/?expr=pow%28x%2C3%29%2Bpow%28y%2C3%29&v_x=9&v_y=10
package main

import (
	"html/template"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/hanzhuoxian/study/go/gopl/ch07/eval"
)

// varField 是页面上的一个变量输入框。
type varField struct {
	Name  string
	Value string
}

// page 是模板的渲染数据。
type page struct {
	Expr   string     // 用户输入的表达式原文
	Tree   string     // 规范化后的语法树（String 方法的输出）
	Vars   []varField // 表达式中出现的变量
	Result string     // 求值结果
	Error  string     // 错误信息
}

func main() {
	http.HandleFunc("/", calc)
	log.Println("listening on http://localhost:8080/")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func calc(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	p := build(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, p); err != nil {
		log.Printf("渲染模板失败: %v", err)
	}
}

// build 根据请求参数完成解析、校验和求值，组装出页面数据。
// 它从不返回 error：所有问题都写进 page.Error 展示给用户。
func build(r *http.Request) *page {
	p := &page{Expr: strings.TrimSpace(r.FormValue("expr"))}
	if p.Expr == "" {
		return p // 首次访问：只显示空表单
	}

	expr, err := eval.Parse(p.Expr)
	if err != nil {
		p.Error = "无法解析表达式：" + err.Error()
		return p
	}

	// Check 校验函数名与参数个数，同时收集所有变量名。
	vars := make(map[eval.Var]bool)
	if err := expr.Check(vars); err != nil {
		p.Error = "表达式非法：" + err.Error()
		return p
	}
	p.Tree = expr.String()

	names := make([]string, 0, len(vars))
	for v := range vars {
		names = append(names, string(v))
	}
	sort.Strings(names)

	// 逐个读取变量值。缺失和非法要区别对待：
	// 缺失说明用户还没填（正常的第一步），非法才是错误。
	env := make(eval.Env, len(names))
	missing := false
	for _, name := range names {
		raw := strings.TrimSpace(r.FormValue("v_" + name))
		p.Vars = append(p.Vars, varField{Name: name, Value: raw})
		if raw == "" {
			missing = true
			continue
		}
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			p.Error = "变量 " + name + " 的值 " + strconv.Quote(raw) + " 不是合法的数字"
			return p
		}
		env[eval.Var(name)] = f
	}
	if missing {
		return p // 渲染变量输入框，等用户填完再算
	}

	p.Result = format(expr.Eval(env))
	return p
}

// format 把结果转成字符串，并对非有限数给出解释。
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

// tmpl 用 html/template 渲染，所有插值都会自动转义，
// 因此用户输入的表达式不会造成 XSS。
var tmpl = template.Must(template.New("calc").Parse(`<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>表达式计算器</title>
<style>
  :root {
    --bg: #f6f5f1; --card: #fff; --ink: #1b1b1b; --dim: #6b6b6b;
    --line: #e2e0da; --accent: #7a1f1f; --ok: #1f5c2e;
  }
  @media (prefers-color-scheme: dark) {
    :root {
      --bg: #16171a; --card: #1e2024; --ink: #e8e6e1; --dim: #9a9891;
      --line: #2e3238; --accent: #e08a7a; --ok: #7fc48f;
    }
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; padding: 3rem 1.25rem; background: var(--bg); color: var(--ink);
    font: 16px/1.6 ui-sans-serif, system-ui, "PingFang SC", sans-serif;
  }
  main { max-width: 34rem; margin: 0 auto; }
  h1 { font-size: 1.35rem; letter-spacing: -.01em; margin: 0 0 .25rem; }
  .sub { color: var(--dim); font-size: .875rem; margin: 0 0 2rem; }
  form {
    background: var(--card); border: 1px solid var(--line);
    border-radius: 10px; padding: 1.5rem;
  }
  label { display: block; font-size: .8rem; color: var(--dim);
          text-transform: uppercase; letter-spacing: .06em; margin-bottom: .4rem; }
  input {
    width: 100%; padding: .6rem .7rem; font: 15px/1.4 ui-monospace, SFMono-Regular, Menlo, monospace;
    color: var(--ink); background: transparent;
    border: 1px solid var(--line); border-radius: 6px;
  }
  input:focus { outline: 2px solid var(--accent); outline-offset: 1px; border-color: transparent; }
  .vars { display: grid; grid-template-columns: repeat(auto-fit, minmax(7rem, 1fr));
          gap: .75rem; margin-top: 1.25rem; }
  button {
    margin-top: 1.25rem; padding: .55rem 1.4rem; font-size: .95rem; cursor: pointer;
    color: var(--card); background: var(--ink); border: 0; border-radius: 6px;
  }
  button:hover { background: var(--accent); }
  .out { margin-top: 1.5rem; padding-top: 1.25rem; border-top: 1px solid var(--line); }
  .tree { font-family: ui-monospace, Menlo, monospace; font-size: .85rem; color: var(--dim);
          word-break: break-all; }
  .result { font-family: ui-monospace, Menlo, monospace; font-size: 1.6rem;
            color: var(--ok); margin-top: .4rem; }
  .err { margin-top: 1.25rem; padding: .7rem .85rem; border-radius: 6px; font-size: .9rem;
         color: var(--accent); border: 1px solid var(--accent); }
  .hint { margin-top: 2rem; font-size: .8rem; color: var(--dim); }
  .hint code { font-family: ui-monospace, Menlo, monospace; }
</style>
</head>
<body>
<main>
  <h1>表达式计算器</h1>
  <p class="sub">支持 + - * / 、括号、函数 sqrt sin pow 与变参 min</p>

  <form method="GET" action="/">
    <label for="expr">表达式</label>
    <input id="expr" name="expr" value="{{.Expr}}" autofocus
           placeholder="pow(x, 3) + pow(y, 3)">

    {{if .Vars}}
    <div class="vars">
      {{range .Vars}}
      <div>
        <label for="v_{{.Name}}">{{.Name}}</label>
        <input id="v_{{.Name}}" name="v_{{.Name}}" value="{{.Value}}" placeholder="0">
      </div>
      {{end}}
    </div>
    {{end}}

    <button type="submit">计算</button>

    {{if .Error}}<p class="err">{{.Error}}</p>{{end}}

    {{if .Result}}
    <div class="out">
      <div class="tree">{{.Tree}}</div>
      <div class="result">= {{.Result}}</div>
    </div>
    {{end}}
  </form>

  <p class="hint">
    试试 <code>sqrt(A / pi)</code>、<code>5 / 9 * (F - 32)</code>、<code>min(x, y, 2)</code>。
    表单用 GET 提交，结果链接可以直接分享。
  </p>
</main>
</body>
</html>
`))
