//+build wireinject

package main

import "github.com/google/wire"

// InitializeEvent 初始化事件
func InitializeEvent() (Event, error) {
	wire.Build(NewEvent, NewGreeter, NewMessage)
	return Event{}, nil
}
