//go:build wireinject
// +build wireinject

package main

import (
	"fmt"
	"os"

	"github.com/google/wire"
)

type Path string

func provideFile(path Path) (*os.File, func(), error) {
	f, err := os.Open(string(path))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		if err := f.Close(); err != nil {
			fmt.Println("close")
		}
	}
	return f, cleanup, nil
}

func NewFile(path Path) (*os.File, func(), error) {
	wire.Build(provideFile)
	return nil, nil, nil
}
