package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) }, // 可选：池空时兜底构造
}

func handle(w io.Writer, data []string) {
	buf := bufPool.Get().(*bytes.Buffer) // 取（可能是复用的，也可能是 New 出来的）
	buf.Reset()                          // 关键：必须自己重置状态
	defer bufPool.Put(buf)               // 还

	for _, s := range data {
		buf.WriteString(s)
	}
	w.Write(buf.Bytes())
}

func main() {
	fmt.Println()
	handle(os.Stdout, []string{"hello"})
	fmt.Println()
	var p sync.Pool      // New == nil
	fmt.Println(p.Get()) // <nil>：池空且没有 New，直接返回 nil
}

func work(b *bytes.Buffer) {
	for _ = range 100 {
		_, err := b.WriteString("hello world ")
		if err != nil {
			panic(err)
		}
	}
}

func workNoPool() {
	buf := new(bytes.Buffer)
	work(buf)
}

var workBufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func workWithPool() {
	buf := workBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer workBufPool.Put(buf)

	work(buf)
}
