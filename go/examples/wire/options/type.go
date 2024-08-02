package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/google/wire"
)

// Message message
type Message string

// Options option
type Options struct {
	Message []Message
	Writer  io.Writer
}

// GoodMessages 推荐greeter话语
func GoodMessages() []Message {
	return []Message{"一路向西", "五十度灰", "三生三世十里桃花"}
}

// Writer wirter
func provideWriter() io.Writer {
	// return os.Stdout
	return nil
}

//
func providerCtx() context.Context {
	return context.Background()
}

// Greeter greeter
type Greeter struct {
	Message Message
	Writer  io.Writer
}

// Greet 打招呼
func (g *Greeter) Greet() {
	g.Writer.Write([]byte(g.Message))
}

// NewGreeter new greeter
func NewGreeter(ctx context.Context, opts *Options) *Greeter {
	mLen := len(opts.Message)
	rand.Seed(time.Now().Unix())
	i := rand.Intn(mLen)
	fmt.Println(i)
	msg := opts.Message[i]

	var w io.Writer = opts.Writer
	if opts.Writer == nil {
		w = os.Stdout
	}
	return &Greeter{Message: msg, Writer: w}
}

// GreeterSet greeter set
var GreeterSet = wire.NewSet(wire.Struct(new(Options), "*"), NewGreeter)

// MessageSet message set
var MessageSet = wire.NewSet(GoodMessages)

// CtxSet ctx set
var CtxSet = wire.NewSet(providerCtx)

// WriterSet wirter set
var WriterSet = wire.NewSet(provideWriter)
