package main

import (
	"sync"
	"testing"
)

type TypedPool[T any] struct {
	p   sync.Pool
	new func() *T
}

func New[T any](newFn func() *T) *TypedPool[T] {
	tp := &TypedPool[T]{
		new: newFn,
	}
	tp.p.New = func() any {
		return tp.new()
	}
	return tp
}

func (tp *TypedPool[T]) Get() *T {
	return tp.p.Get().(*T)
}
func (tp *TypedPool[T]) Put(t *T) {
	tp.p.Put(t)
}

var pool *TypedPool[[]byte]

func Run() {

	buf := pool.Get()
	*buf = (*buf)[:0]
	*buf = append(*buf, 'a')
	pool.Put(buf)
}

func TestRun(t *testing.T) {

	pool = New(func() *[]byte {
		return new([]byte)
	})

	Run()
	Run()
	Run()

	t.Fail()
}

func BenchmarkRun(b *testing.B) {
	pool = New(func() *[]byte {
		return new([]byte)
	})

	for b.Loop() {
		Run()
	}
}
