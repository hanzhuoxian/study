// +build wireinject
// The build tag makes sure the stub is not built in the final build.

package main

import (
	"context"

	"github.com/daymenu/gostudy/examples/wire/basics/foobarbaz"

	"github.com/google/wire"
)

// initializeBaz 初始化 baz
func initializeBaz(ctx context.Context) (foobarbaz.Baz, error) {
	wire.Build(foobarbaz.SuperSet)
	return foobarbaz.Baz{}, nil
}

// initializeBar 初始化 bar
func initializeBar(ctx context.Context) (foobarbaz.Bar, error) {
	wire.Build(foobarbaz.SuperSet)
	return foobarbaz.Bar{}, nil
}

// initializeFoo 初始化 Foo
func initializeFoo(ctx context.Context) (foobarbaz.Foo, error) {
	wire.Build(foobarbaz.SuperSet)
	return foobarbaz.Foo{}, nil
}
