package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

func main() {
	l := rate.NewLimiter(20, 10)

	c, cancel := context.WithCancel(context.Background())
	defer cancel()
	fmt.Println(l.Limit(), l.Burst())

	for i := 0; i < 1000; i++ {
		l.Wait(c)
		//time.Sleep(200 * time.Millisecond)
		fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"))
	}
}
