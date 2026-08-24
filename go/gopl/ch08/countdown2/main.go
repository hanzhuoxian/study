package main

import (
	"fmt"
	"os"
	"time"
)

func main() {

	fmt.Println("Commencing countdown.")

	tick := time.Tick(1 * time.Second)
	abort := make(chan struct{})

	go func() {
		os.Stdin.Read(make([]byte, 1))
		close(abort)
	}()
	for countdown := 10; countdown > 0; countdown-- {
		fmt.Println(countdown)
		select {
		case <-tick:
		case <-abort:
			fmt.Println("Aborted!")
			return
		}
	}
	fmt.Println("Lift off!")
}
