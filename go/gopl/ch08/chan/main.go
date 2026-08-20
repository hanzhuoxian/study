package main

import "fmt"

func main() {
	ch := make(chan int)
	fmt.Printf("ch : %T", ch) // ch : chan int

	x := 1
	go func() { ch <- x }() // a send statement
	x = <-ch                // a receive expression in a assignment statement
	go func() { ch <- x }()
	<-ch // a receive statement; result is discarded
}
