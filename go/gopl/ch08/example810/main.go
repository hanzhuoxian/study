package main

// **练习 8.10：** HTTP请求可能会因http.Request结构体中Cancel channel的关闭而取消。
// 修改8.6节中的web crawler来支持取消http请求。（提示：http.Get并没有提供方便地定制一个请求的方法。
// 你可以用http.NewRequest来取而代之，设置它的Cancel字段，然后用http.DefaultClient.Do(req)来进行这个http请求。）
//
// 说明：练习原文写于 Request.Cancel 时代，该字段自 Go 1.7 起已被废弃（Deprecated），
// 官方推荐改用 context。这里用 context.WithCancel 实现：取消信号从 main 一路传到
// http.NewRequestWithContext，既能中断正在飞行的 HTTP 请求（Transport 会直接关掉底层
// 连接并返回 context.Canceled），也能让 worker 和主循环一起退出。
//
// 用法：go run . [seed-url...]    运行后按回车即取消整个爬取过程。

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"golang.org/x/net/html"
)

// numWorkers 限制并发爬取的 goroutine 数量，即同时在飞的 HTTP 请求数上限。
const numWorkers = 20

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 从标准输入读到任意一个字节（按回车）即触发取消。
	go func() {
		_, _ = os.Stdin.Read(make([]byte, 1))
		fmt.Fprintln(os.Stderr, "canceling...")
		cancel()
	}()

	seeds := os.Args[1:]
	if len(seeds) == 0 {
		seeds = []string{"https://golang.org"}
	}

	seen := crawlAll(ctx, seeds)
	fmt.Fprintf(os.Stderr, "done: %d urls seen\n", len(seen))
}

// crawlAll 从 seeds 出发广度优先地爬取，返回所有见过的 URL。
// ctx 被取消后，所有 worker、所有在途 HTTP 请求和主循环都会退出，
// 函数在全部 goroutine 结束后才返回，不留悬挂的 goroutine。
func crawlAll(ctx context.Context, seeds []string) map[string]bool {
	workList := make(chan []string)  // 每个元素是一批待去重的链接
	unseenLinks := make(chan string) // 去重后待爬取的链接

	// wg 覆盖所有 worker 和所有向 workList 投递结果的 goroutine，
	// 保证取消后能等到它们全部退出。
	var wg sync.WaitGroup

	// 投递种子。注意不能直接 workList <- seeds，
	// 因为主循环还没开始接收，会死锁。
	wg.Add(1)
	go func() {
		defer wg.Done()
		sendList(ctx, workList, seeds)
	}()

	// worker 池：从 unseenLinks 取链接，爬取，把结果投回 workList。
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case link := <-unseenLinks:
					found := crawl(ctx, link)
					// 用独立的 goroutine 投递，避免 worker 卡在发送上
					// 而主循环又在等 worker 接收下一个链接（互相等待）。
					wg.Add(1)
					go func() {
						defer wg.Done()
						sendList(ctx, workList, found)
					}()
				}
			}
		}()
	}

	// 主循环：去重并把新链接分发给 worker。
	// 每一次收发都带上 ctx.Done()，取消时才能立刻跳出而不是卡在 channel 上。
	seen := make(map[string]bool)
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case list := <-workList:
			for _, link := range list {
				if seen[link] {
					continue
				}
				seen[link] = true
				select {
				case <-ctx.Done():
					break loop
				case unseenLinks <- link:
				}
			}
		}
	}

	wg.Wait()
	return seen
}

// sendList 把 list 发送到 ch，除非 ctx 先被取消。
func sendList(ctx context.Context, ch chan<- []string, list []string) {
	select {
	case ch <- list:
	case <-ctx.Done():
	}
}

// crawl 爬取单个 url 并返回其中的链接。
// 取消导致的错误不打印日志，否则一次取消会让 numWorkers 个 worker 同时刷屏。
func crawl(ctx context.Context, url string) []string {
	fmt.Println(url)
	list, err := Extract(ctx, url)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Print(err)
	}
	return list
}

// Extract 向 url 发起一次可取消的 GET 请求，把响应解析为 HTML，
// 返回文档中的所有链接。ctx 被取消时请求会立即中断。
func Extract(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getting %s: %s", url, resp.Status)
	}

	// 读取 body 期间同样受 ctx 控制：取消会让 Read 立刻返回 context.Canceled。
	// 这里用 %w 保留错误链，好让上层的 errors.Is 能识别出取消。
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %w", url, err)
	}

	var links []string

	visitNode := func(n *html.Node) {
		if n.Type != html.ElementNode || n.Data != "a" {
			return
		}
		for _, a := range n.Attr {
			if a.Key != "href" {
				continue
			}
			link, err := resp.Request.URL.Parse(a.Val)
			if err != nil {
				continue // 忽略不合法的 URL
			}
			links = append(links, link.String())
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
