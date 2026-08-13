package main

import (
	"fmt"
	"io"
	"os"
)

// **练习 7.2：** 写一个带有如下函数签名的函数 CountingWriter，传入一个 io.Writer 接口类型，
// 返回一个把原来的Writer封装在里面的新的Writer类型和一个表示新的写入字节数的int64类型指针。

type CountWriter struct {
	counter int64
	w       io.Writer
}

func (c *CountWriter) Write(b []byte) (n int, err error) {
	n, err = c.w.Write(b)
	c.counter += int64(n)
	return n, err
}

func CountingWriter(w io.Writer) (io.Writer, *int64) {
	c := CountWriter{
		counter: 0,
		w:       w,
	}
	return &c, &c.counter
}

func main() {
	w, count := CountingWriter(os.Stdout)
	fmt.Fprintf(w, "hello, world\n")
	fmt.Printf("wrote %d bytes\n", *count)
}
