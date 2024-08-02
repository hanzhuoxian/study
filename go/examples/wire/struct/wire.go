//go:build wireinject
// +build wireinject

package main

import "github.com/google/wire"

// 返回指针
func injectFooBarPointer() *FooBar {
	wire.Build(Set)
	return &FooBar{}
}

// 返回struct
func injectFooBar() FooBar {
	wire.Build(Set)
	return FooBar{}
}

// 返回struct feild
func injectFooBarOfFoo() Foo {
	wire.Build(provoideFooBar, wire.FieldsOf(new(FooBar), "MyFoo"))
	return Foo(0)
}
