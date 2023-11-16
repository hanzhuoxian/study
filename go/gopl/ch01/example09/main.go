package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// 从命令行中获取链接，请求页面的内容并将页面的内容打印出来,使用io.Copy从网络结果复制到标准输出

func main() {
	for _, url := range os.Args[1:] {
		const urlPrefix = "http://"
		if !strings.HasPrefix(url, urlPrefix) {
			url = urlPrefix + url
		}
		resp, err := http.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "get %s error : %v", url, err)
			os.Exit(1)
		}

		fmt.Println("http status code : ", resp.Status)
		_, err = io.Copy(os.Stdout, resp.Body)
		defer resp.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s error : %v", url, err)
			os.Exit(1)
		}

	}
}
