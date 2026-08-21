package main

import "fmt"

func main() {
	naturals := make(chan int)
	squares := make(chan int)

	// Counter
	go counter(naturals)

	// Squarer
	go squarer(naturals, squares)

	// Printer (in main goroutine)
	printer(squares)
}

func printer(in <-chan int) {
	for x := range in {
		fmt.Println(x)
	}
}

func counter(out chan<- int) {
	for x := 0; x < 100; x++ {
		out <- x
	}
	close(out)
}

func squarer(in <-chan int, out chan<- int) {
	for x := range in {
		out <- x * x
	}
	close(out)
}
