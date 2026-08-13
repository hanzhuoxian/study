package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/net/html"
)

// **练习 7.4：** strings.NewReader函数通过读取一个string参数返回一个满足io.Reader接口类型的值（和其它值）。
// 实现一个简单版本的NewReader，用它来构造一个接收字符串输入的HTML解析器（§5.2）

type Reader struct {
	s string
	i int64
}

func NewReader(s string) *Reader {
	return &Reader{s, 0}
}

func (r *Reader) Read(b []byte) (n int, err error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n = copy(b, r.s[r.i:])
	r.i += int64(n)
	return n, nil
}

func main() {
	const input = `<html><body><h1>Title</h1><p>Hello, <b>world</b>!</p></body></html>`

	doc, err := html.Parse(NewReader(input))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}
	outline(nil, doc)
}

func outline(stack []string, n *html.Node) {
	if n.Type == html.ElementNode {
		stack = append(stack, n.Data)
		fmt.Println(stack)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		outline(stack, c)
	}
}
