package main

import (
	"errors"
	"fmt"
	"time"
)

type Message string

type Greeter struct {
	Grumpy  bool
	Message Message
}

type Event struct {
	Greeter Greeter
}

func NewMessage(msg string) Message {
	return Message(msg)
}

func NewGreeter(m Message) Greeter {
	var grumpy bool
	if time.Now().Unix()%2 == 0 {
		grumpy = true
	}

	return Greeter{
		Grumpy:  grumpy,
		Message: m,
	}
}

func (g Greeter) Greet() Message {
	if g.Grumpy {
		return Message("Go away!")
	}
	return g.Message
}

func NewEvent(g Greeter) (Event, error) {
	if g.Grumpy {
		return Event{}, errors.New("could not create event: event greeter is grumpy")
	}
	return Event{Greeter: g}, nil
}

func (e Event) Start() {
	msg := e.Greeter.Greet()
	fmt.Println(msg)
}

func main() {
	event, err := InitializeEvent("hello world")
	if err != nil {
		panic(err)
	}
	event.Start()
}
