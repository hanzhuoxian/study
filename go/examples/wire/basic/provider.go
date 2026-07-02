package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/wire"
)

type Foo struct {
	X int
}

func ProvideFoo() Foo {
	return Foo{X: 88}
}

type Bar struct {
	X int
}

func ProvideBar(foo Foo) Bar {
	return Bar{X: -foo.X}
}

type Baz struct {
	X int
}

func ProvideBaz(ctx context.Context, bar Bar) (Baz, error) {
	if bar.X == 0 {
		return Baz{}, errors.New("cann't provide baz when bar is zero")
	}
	return Baz{X: bar.X}, nil
}

var SuperSet = wire.NewSet(ProvideBar, ProvideFoo, ProvideBaz)

type Other struct {
	Y string
}

func ProvideOther(baz Baz) Other {
	return Other{Y: fmt.Sprintf("Other: %d", baz.X)}
}

var MegaSet = wire.NewSet(SuperSet, ProvideOther)
