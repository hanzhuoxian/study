package main

import (
	"bytes"
	"io"
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
