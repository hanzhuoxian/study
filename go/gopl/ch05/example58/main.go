package main

// **练习 5.8：** 修改pre和post函数，使其返回布尔类型的返回值。返回false时，中止forEachNode的遍历。使用修改后的代码编写ElementByID函数，根据用户输入的id查找第一个拥有该id元素的HTML元素，查找成功后，停止遍历。

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/net/html"
)

var (
	out io.Writer = os.Stdout
)

func ElementByID(n *html.Node, id string) *html.Node {
	var found *html.Node
	pre := func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == id {
					found = n
					return false // Stop traversal
				}
			}
		}
		return true // Continue traversal
	}
	forEachNode(n, pre, nil)
	return found
}

func main() {
	doc, err := html.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "outline: %v\n", err)
		os.Exit(1)
	}
	e := ElementByID(doc, "target-id") // Replace "target-id" with the desired ID
	if e != nil {
		fmt.Fprintf(out, "Found element: <%s>\n", e.Data)
	} else {
		fmt.Fprintf(out, "Element with ID 'target-id' not found.\n")
	}
}

func forEachNode(n *html.Node, pre, post func(n *html.Node) bool) bool {
	if pre != nil {
		if !pre(n) {
			return false
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if !forEachNode(c, pre, post) {
			return false
		}
	}
	if post != nil {
		if !post(n) {
			return false
		}
	}
	return true
}
