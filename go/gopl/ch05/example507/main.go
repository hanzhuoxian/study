package main

// **练习 5.7：** 完善startElement和endElement函数，使其成为通用的HTML输出器。要求：输出注释结点，文本结点以及每个元素的属性（< a href='...'>）。使用简略格式输出没有孩子结点的元素（即用`<img/>`代替`<img></img>`）。编写测试，验证程序输出的格式正确。（详见11章）

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/net/html"
)

var (
	depth int
	out   io.Writer = os.Stdout
)

func startElement(n *html.Node) {
	switch n.Type {
	case html.ElementNode:
		fmt.Fprintf(out, "%*s<%s", depth*2, "", n.Data)
		for _, attr := range n.Attr {
			fmt.Fprintf(out, " %s='%s'", attr.Key, attr.Val)
		}
		if n.FirstChild == nil {
			fmt.Fprintf(out, "/>\n")
		} else {
			fmt.Fprintf(out, ">\n")
			depth++
		}
	case html.CommentNode:
		fmt.Fprintf(out, "%*s<!--%s-->\n", depth*2, "", n.Data)
	case html.TextNode:
		fmt.Fprintf(out, "%*s%s\n", depth*2, "", n.Data)
	}
}
func endElement(n *html.Node) {
	if n.Type == html.ElementNode && n.FirstChild != nil {
		depth--
		fmt.Fprintf(out, "%*s</%s>\n", depth*2, "", n.Data)
	}
}
func main() {
	doc, err := html.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "outline: %v\n", err)
		os.Exit(1)
	}
	forEachNode(doc, startElement, endElement)
}

func forEachNode(n *html.Node, pre, post func(n *html.Node)) {
	if pre != nil {
		pre(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		forEachNode(c, pre, post)
	}
	if post != nil {
		post(n)
	}
}
