package main

import "fmt"

// Message message
type Message string

// NewMessage new message
func NewMessage() Message {
	return Message("Hi Here")
}

// Greeter greeter
type Greeter struct {
	Message Message
}

// NewGreeter new greeter
func NewGreeter(m Message) Greeter {
	return Greeter{Message: m}
}

// Greet greet
func (g Greeter) Greet() Message {
	return g.Message
}

// Event event
type Event struct {
	Greeter Greeter
}

// NewEvent new event
func NewEvent(g Greeter) Event {
	return Event{Greeter: g}
}

// Start event start
func (e Event) Start() {
	msg := e.Greeter.Greet()
	fmt.Println(msg)
}
