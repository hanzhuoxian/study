package main

import (
	"fmt"
	"os"

	"golang.org/x/net/html"
)

// 练习 5.2： 编写函数，记录在HTML树中出现的同名元素的次数。
func main() {

	doc, err := html.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "countElements: %v\n", err)
		os.Exit(1)
	}

	counts := make(map[string]int)
	countElements(doc, counts)

	for tag, count := range counts {
		fmt.Printf("%s: %d\n", tag, count)
	}
}

func countElements(n *html.Node, counts map[string]int) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode {
		counts[n.Data]++
	}
	countElements(n.FirstChild, counts)
	countElements(n.NextSibling, counts)
}
