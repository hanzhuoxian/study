//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
)

func initializeBaz(context.Context) (Baz, error) {
	wire.Build(SuperSet)
	return Baz{}, nil
}

func initializeOther(context.Context) (Other, error) {
	wire.Build(MegaSet)
	return Other{}, nil
}
