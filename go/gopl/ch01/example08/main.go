package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	for _, url := range os.Args[1:] {
		resp, err := http.Get(addHTTPPrefix(url))
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
			os.Exit(1)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		fmt.Println(string(body))
	}
}

func addHTTPPrefix(url string) string {
	if strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	return url
}
