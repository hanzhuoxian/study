package main

import "fmt"

func main() {
	// use wire
	event := InitializeEvent()
	event.Start()
	fmt.Println(`main`)
	// don't use wire
	// message := NewMessage()
	// greeter := NewGreeter(message)
	// event := NewEvent(greeter)
	// event.Start()
}
