//+build wireinject

package main

import (
	"context"
	"io"

	"github.com/google/wire"
)

func injectCtx() context.Context {
	panic(wire.Build(CtxSet))
}

func injectWirter() io.Writer {
	panic(wire.Build(WriterSet))
}

func injectMessages() []Message {
	panic(wire.Build(MessageSet))
}

func injectGreeter() *Greeter {
	panic(wire.Build(GreeterSet, CtxSet, MessageSet, WriterSet))
}
