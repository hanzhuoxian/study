package main

import (
	"fmt"
	"os"

	"golang.org/x/net/html"
)

// 练习 5.3： 编写函数输出所有text结点的内容。注意不要访问<script>和<style>元素，因为这些元素对浏览者是不可见的。
func main() {
	doc, err := html.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "text: %v\n", err)
		os.Exit(1)
	}
	text(doc)
}

func text(n *html.Node) {
	if n == nil {
		return
	}
	if n.Type == html.TextNode {
		fmt.Println(n.Data)
	}
	if n.Type != html.ElementNode || (n.Data != "script" && n.Data != "style") {
		text(n.FirstChild)
	}
	text(n.NextSibling)
}
