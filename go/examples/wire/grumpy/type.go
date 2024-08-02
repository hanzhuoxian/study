package main

import (
	"errors"
	"fmt"
	"time"
)

// Message message
type Message string

// NewMessage new message
func NewMessage() Message {
	return Message("Hi there!")
}

// Greeter greeter
type Greeter struct {
	Grumpy  bool
	Message Message
}

// NewGreeter new greeter
func NewGreeter(m Message) Greeter {
	var grumpy bool
	if time.Now().Unix()%2 == 0 {
		grumpy = true
	}
	return Greeter{Message: m, Grumpy: grumpy}
}

// Greet greet
func (g Greeter) Greet() Message {
	if g.Grumpy {
		return Message("Go away!")
	}
	return g.Message
}

// Event event
type Event struct {
	Greeter Greeter
}

// NewEvent new event
func NewEvent(g Greeter) (Event, error) {
	if g.Grumpy {
		return Event{}, errors.New("could not create event: event greeter is grumpy")
	}
	return Event{Greeter: g}, nil
}

// Start event start
func (e Event) Start() {
	msg := e.Greeter.Greet()
	fmt.Println(msg)
}
