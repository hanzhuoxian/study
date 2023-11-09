package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// 从命令行中获取链接，请求页面的内容并将页面的内容打印出来
func main() {
	for _, url := range os.Args[1:] {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s error : %v", url, err)
			os.Exit(1)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s error : %v", url, err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		fmt.Println(string(body))
	}
}
