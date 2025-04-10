// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"context"
	"github.com/daymenu/gostudy/examples/wire/basics/foobarbaz"
)

// Injectors from wire.go:

// initializeBaz 初始化 baz
func initializeBaz(ctx context.Context) (foobarbaz.Baz, error) {
	foo := foobarbaz.ProvideFoo()
	bar := foobarbaz.ProvideBar(foo)
	baz, err := foobarbaz.ProvideBaz(ctx, bar)
	if err != nil {
		return foobarbaz.Baz{}, err
	}
	return baz, nil
}

// initializeBar 初始化 bar
func initializeBar(ctx context.Context) (foobarbaz.Bar, error) {
	foo := foobarbaz.ProvideFoo()
	bar := foobarbaz.ProvideBar(foo)
	return bar, nil
}

// initializeFoo 初始化 Foo
func initializeFoo(ctx context.Context) (foobarbaz.Foo, error) {
	foo := foobarbaz.ProvideFoo()
	return foo, nil
}
