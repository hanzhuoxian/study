package main

import (
	"sync"

	"github.com/google/wire"
)

// Foo foo
type Foo int

// Bar bar
type Bar int

// ProvideFoo provide foo
func ProvideFoo() Foo {
	return 1
}

// ProvideBar provide bar
func ProvideBar() Bar {
	return 2
}

// FooBar foobar
type FooBar struct {
	mu    sync.Mutex `wire:"-"` //不需要wire来初始化
	MyFoo Foo
	MyBar Bar
}

// Set set
var Set = wire.NewSet(
	ProvideBar,
	ProvideFoo,
	// wire.Struct(new(FooBar), "MyFoo", "MyBar"),
	wire.Struct(new(FooBar), "*"), // *代表全部字段
)

func provoideFooBar() FooBar {
	return FooBar{MyFoo: Foo(2), MyBar: Bar(1)}
}
