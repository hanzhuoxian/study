//go:build wireinject
// +build wireinject

package main

import (
	"io"
	"os"

	"github.com/google/wire"
)

func injectFoo() Foo {
	wire.Build(wire.Value(Foo{X: 42}))
	return Foo{}
}

func injectReader() io.Reader {
	wire.Build(wire.InterfaceValue(new(io.Reader), os.Stdin))
	return nil
}
