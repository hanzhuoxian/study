package main

// **练习 8.10：** HTTP请求可能会因http.Request结构体中Cancel channel的关闭而取消。
// 修改8.6节中的web crawler来支持取消http请求。（提示：http.Get并没有提供方便地定制一个请求的方法。
// 你可以用http.NewRequest来取而代之，设置它的Cancel字段，然后用http.DefaultClient.Do(req)来进行这个http请求。）

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	body := strings.NewReader("")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.baidu.com", body)

	go func() {
		os.Stdin.Read(make([]byte, 1))
		fmt.Println("call cancel")
		cancel()
	}()

	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

}

// Extract makes an HTTP GET request to the specified URL, parses
// the response as HTML, and returns the links in the HTML document.
func Extract(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("getting %s: %s", url, resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %v", url, err)
	}

	var links []string

	var visitNode func(*html.Node)
	visitNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key != "href" {
					continue
				}

				link, err := resp.Request.URL.Parse(a.Val)
				if err != nil {
					continue // ignore bad URLs
				}
				links = append(links, link.String())
			}
		}
	}
	forEachNode(doc, visitNode, nil)
	return links, nil
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
