package main

import "github.com/google/wire"

// Fooer foo 接口
type Fooer interface {
	Foo() string
}

// MyFooer foo
type MyFooer string

// Foo foo
func (b *MyFooer) Foo() string {
	return string(*b)
}

func provideMyFooer() *MyFooer {
	b := new(MyFooer)
	*b = "Hello, World!"
	return b
}

// Bar bar
type Bar string

func provideBar(f Fooer) string {
	// f will be a *MyFooer.
	return f.Foo()
}

// Set set
var Set = wire.NewSet(
	provideMyFooer,
	// Bind 的第一个参数是接口指针，第二个是类型指针
	wire.Bind(new(Fooer), new(*MyFooer)),
	provideBar)
