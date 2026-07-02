//go:build wireinject
// +build wireinject

package main

import "github.com/google/wire"

type Foo struct {
	S   string
	N   int
	F   float64
	Bar Bar
}

type Bar struct {
	S string
}

func getS(foo Foo) string {
	return foo.S
}

func ProvideFoo() Foo {
	return Foo{
		S: "Hello",
		N: 88,
		F: 3.14,
		Bar: Bar{
			S: "Bar",
		},
	}
}

func injectMessage() string {
	wire.Build(ProvideFoo, getS)
	return ""
}

func injectField() string {
	wire.Build(ProvideFoo, wire.FieldsOf(new(Foo), "S"))
	return ""
}

func injectBar() Bar {
	wire.Build(ProvideFoo, wire.FieldsOf(new(Foo), "Bar"))
	return Bar{}
}
