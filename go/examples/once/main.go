package main

import "sync"

func main() {
	once := sync.Once{}
	done := make(chan struct{}, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() {
				done <- struct{}{}
			}()
			once.Do(func() {
				println("hello world")
			})
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
