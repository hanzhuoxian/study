package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hanzhuoxian/study/go/gopl/ch05/links"
)

var tokens = make(chan struct{}, 20)

func crawl(url string) []string {
	fmt.Println(url)
	tokens <- struct{}{}
	list, err := links.Extract(url)
	<-tokens
	if err != nil {
		log.Print(err)
	}
	return list
}

func main() {
	worklist := make(chan []string)

	n := 1
	// Start with the command-line arguments.
	go func() { worklist <- os.Args[1:] }()

	// Crawl the web concurrently

	seen := make(map[string]bool)
	for ; n > 0; n-- {
		list := <-worklist
		for _, link := range list {
			if !seen[link] {
				seen[link] = true
				n++
				go func(link string) {
					worklist <- crawl(link)
				}(link)
			}
		}
	}
}
