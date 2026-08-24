package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/hanzhuoxian/study/go/gopl/ch05/links"
)

// **练习 8.6：** 为并发爬虫增加深度限制。也就是说，如果用户设置了depth=3，
// 那么只有从首页跳转三次以内能够跳到的页面才能被抓取到。
//
// 深度是相对命令行给出的起始 URL 计算的最短跳数：种子页面深度为 0，
// 从种子页面上抽出的链接深度为 1，依此类推。跨域跳转和同域跳转一样都算一跳。

// work 是一个待抓取的页面及它所处的深度。
type work struct {
	depth int
	url   string
}

// batch 是从某个页面上抽出的一组链接，depth 是这些链接所处的深度。
type batch struct {
	depth int
	links []string
}

var (
	depth   = flag.Int("depth", 3, "最大抓取深度，0 表示只抓命令行给出的页面")
	workers = flag.Int("workers", 20, "并发抓取的 goroutine 数量")
)

func crawl(w work) []string {
	fmt.Printf("%d\t%s\n", w.depth, w.url)
	list, err := links.Extract(w.url)
	if err != nil {
		log.Print(err)
	}
	return list
}

func main() {
	flag.Parse()

	urls := flag.Args()
	if len(urls) == 0 {
		log.Fatalf("usage: %s [-depth=n] [-workers=n] url...", "crawl")
	}

	fmt.Printf("depth is : %d\ninit urls: %v\n", *depth, urls)

	worklist := make(chan batch)
	unseenLinks := make(chan work)

	// n 记录还有多少个 batch 尚未从 worklist 中取出，用它判断何时抓完收工。
	n := 1
	go func() { worklist <- batch{0, urls} }()

	for range *workers {
		go func() {
			for w := range unseenLinks {
				found := crawl(w)
				// 抽出的链接比当前页面深一层。
				// 用单独的 goroutine 发送，避免和主循环互相等待造成死锁。
				go func() { worklist <- batch{w.depth + 1, found} }()
			}
		}()
	}

	seen := make(map[string]bool)
	for ; n > 0; n-- {
		b := <-worklist
		if b.depth > *depth {
			continue // 超出深度限制，这批链接不再入队
		}
		for _, link := range b.links {
			if seen[link] {
				continue
			}
			seen[link] = true
			n++
			unseenLinks <- work{b.depth, link}
		}
	}
}
