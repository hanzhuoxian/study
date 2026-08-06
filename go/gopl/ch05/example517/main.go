package main

import (
	"fmt"
	"log"
	"strings"

	"golang.org/x/net/html"
)

// **练习5.17：** 编写多参数版本的ElementsByTagName，函数接收一个HTML结点树以及任意数量的标签名，返回与这些标签名匹配的所有元素。下面给出了2个例子：

func ElementsByTagName(doc *html.Node, name ...string) []*html.Node {
	var results []*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, n := range name {
				if n == node.Data {
					results = append(results, node)
					break
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	return results
}

func main() {
	doc, err := html.Parse(strings.NewReader(`
<html>
	<head>
		<title>Example</title>
	</head>
	<body>
		<h1>Example</h1>
		<img src="example.png" />
	</body>
</html>`))
	if err != nil {
		log.Fatal(err)
	}
	images := ElementsByTagName(doc, "img")
	for _, img := range images {
		fmt.Println(img.Data)
	}
	headings := ElementsByTagName(doc, "h1", "h2", "h3", "h4")
	for _, heading := range headings {
		fmt.Println(heading.Data)
	}
}
