package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args[1:]) < 1 {
		fmt.Println("Please input fetch URL")
		return
	}
	ch := make(chan string, 2*len(os.Args[1:]))
	for _, url := range os.Args[1:] {
		go fetch(url, 2, ch)
	}

	for range os.Args[1:] {
		fmt.Println(<-ch)
		fmt.Println(<-ch)
	}

}

func fetch(url string, t int, ch chan<- string) {
	for i := 0; i < t; i++ {
		start := time.Now()
		resp, err := http.Get(url)
		if err != nil {
			ch <- fmt.Sprintf("url:%s get is failed", url)
		}
		f, err := os.OpenFile(fmt.Sprintf("./sites/%s_%d", strings.TrimLeft(url, "https://"), i), os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			ch <- fmt.Sprintf("open file is failed %v", err)
		}
		nbytes, err := io.Copy(f, resp.Body)
		if err != nil {
			ch <- fmt.Sprintf("copy file is failed")
		}
		resp.Body.Close()
		f.Close()
		ch <- fmt.Sprintf("%s(%d): size:%d time:%d", url, i, nbytes, time.Since(start).Milliseconds())

	}
}
