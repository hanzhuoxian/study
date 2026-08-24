package main

import (
	"flag"
	"fmt"
)

// **练习 8.7：** 完成一个并发程序来创建一个线上网站的本地镜像，把该站点的所有可达的页面都抓取到本地硬盘。
// 为了省事，我们这里可以只取出现在该域下的所有页面（比如golang.org开头，译注：外链的应该就不算了。）当然了，
// 出现在页面里的链接你也需要进行一些处理，使其能够在你的镜像站点上进行跳转，而不是指向原始的链接。

var workers = flag.Int("workers", 20, "并发抓取的 goroutine 数量")
var url = flag.String("url", "https://golang.org", "要抓取的网站 URL")

func main() {
	flag.Parse()
	fmt.Printf("workers is: %d\n", *workers)
	fmt.Printf("url is: %s\n", *url)

}
