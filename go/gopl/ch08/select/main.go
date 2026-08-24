package main

func main() {
	ch := make(chan int, 1)
	for i := range 10 {
		select {
		case x := <-ch:
			println(x)
		case ch <- i:

		}
	}
}
