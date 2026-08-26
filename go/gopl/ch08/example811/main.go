package main

import (
	"context"
	"fmt"
	"net/http"
)

// **练习 8.11：** 紧接着8.4.4中的mirroredQuery流程，实现一个并发请求url的fetch的变种。当第一个请求返回时，直接取消其它的请求。
func main() {
	fmt.Println(mirroredQuery())
}
func mirroredQuery() string {
	res := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case res <- request(ctx, "asia.gopl.io"):
			fmt.Println("fetch  asia.gopl.io")
			cancel()
		case <-ctx.Done():
			fmt.Println("done 1")
		}
	}()
	go func() {
		select {
		case res <- request(ctx, "europe.gopl.io"):
			fmt.Println("fetch  europe.gopl.io")
			cancel()
		case <-ctx.Done():
			fmt.Println("done 2")
		}
	}()
	go func() {
		select {
		case res <- request(ctx, "americas.gopl.io"):
			fmt.Println("fetch  americas.gopl.io")
			cancel()
		case <-ctx.Done():
			fmt.Println("done 3")
		}
	}()
	return <-res // return the quickest response
}

func request(ctx context.Context, hostname string) (response string) {
	hostname = "http://" + hostname
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hostname, nil)
	if err != nil {
		return err.Error()
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	return resp.Status
}
