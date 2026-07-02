//go:build wireinject
// +build wireinject

package main

import "github.com/google/wire"

func initializeFooBar() (FooBar, error) {
	wire.Build(Set)
	return FooBar{}, nil
}

func initializeFooOnly() (FooOnly, error) {
	wire.Build(Set)
	return FooOnly{}, nil
}

func initializeFooBarAll() (FooBarAll, error) {
	wire.Build(Set)
	return FooBarAll{}, nil
}

func initializeFooBarAllPointer() (*FooBarAll, error) {
	wire.Build(Set)
	return &FooBarAll{}, nil
}

func initializeFooIngore() (FooIngore, error) {
	wire.Build(Set)
	return FooIngore{}, nil
}
