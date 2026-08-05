package main

// 练习 5.4： 扩展visit函数，使其能够处理其他类型的结点，如images、scripts和style sheets。

import (
	"fmt"
	"os"

	"golang.org/x/net/html"
)

func main() {
	doc, err := html.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "findlinks1: %v\n", err)
		os.Exit(1)
	}
	for _, link := range visit(nil, doc) {
		fmt.Println(link)
	}
}

func visit(links []string, n *html.Node) []string {
	if n.Type == html.ElementNode {
		if link := getLink(n); link != "" {
			links = append(links, link)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		links = visit(links, c)
	}
	return links
}

func getLink(n *html.Node) string {
	switch n.Data {
	case "a":
		for _, a := range n.Attr {
			if a.Key == "href" {
				return a.Val
			}
		}
	case "img":
		for _, a := range n.Attr {
			if a.Key == "src" {
				return a.Val
			}
		}
	case "script":
		for _, a := range n.Attr {
			if a.Key == "src" {
				return a.Val
			}
		}
	case "link":
		for _, a := range n.Attr {
			if a.Key == "href" && isStyleSheet(n) {
				return a.Val
			}
		}
	}
	return ""
}

func isStyleSheet(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "rel" && a.Val == "stylesheet" {
			return true
		}
	}
	return false
}
