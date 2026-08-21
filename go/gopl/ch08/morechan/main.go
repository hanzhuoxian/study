package main

import "fmt"

func main() {
	ch := make(chan string, 3)
	ch <- "hello"
	ch <- "world"
	ch <- "!"
	fmt.Println()
	fmt.Printf("len: %d; cap: %d; type: %T; %s\n", len(ch), cap(ch), ch, <-ch)
	fmt.Printf("len: %d; cap: %d; type: %T; %s\n", len(ch), cap(ch), ch, <-ch)
	fmt.Printf("len: %d; cap: %d; type: %T; %s\n", len(ch), cap(ch), ch, <-ch)
}
