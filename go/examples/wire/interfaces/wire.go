//+build wireinject

package main

import "github.com/google/wire"

// InitializeEvent 初始化事件
func InitializeFooer() Fooer {
	wire.Build(Set)
	return nil
}
