//go:build wireinject
// +build wireinject

package main

import (
	"os"

	"github.com/google/wire"
)

func injectFile() (*os.File, func(), error) {
	wire.Build(provideFile)
	return nil, nil, nil
}
