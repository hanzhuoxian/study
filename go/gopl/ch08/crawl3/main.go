package main

import (
	"fmt"
	"log"

	"github.com/hanzhuoxian/study/go/gopl/ch05/links"
)

func crawl(url string) []string {
	fmt.Println(url)
	list, err := links.Extract(url)
	if err != nil {
		log.Print(err)
	}
	return list
}
func main() {
	worklist := make(chan []string)
	unseenLinks := make(chan string)

	go func() {
		worklist <- []string{"https://golang.org"}
	}()

	for range 20 {
		go func() {
			for link := range unseenLinks {
				foundLinks := crawl(link)
				go func() {
					worklist <- foundLinks
				}()
			}
		}()
	}

	seen := make(map[string]bool)
	for list := range worklist {
		for _, link := range list {
			if !seen[link] {
				seen[link] = true
				unseenLinks <- link
			}
		}
	}
}
