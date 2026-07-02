//go:build wireinject
// +build wireinject

package main

import "github.com/google/wire"

type Foo struct {
	X int
}

func ProvideFoo() Foo {
	wire.Build(wire.Value(Foo{X: 88}))
	return Foo{}
}
