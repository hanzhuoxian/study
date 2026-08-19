package main

import (
	"bytes"
	"io"
)

const debug = false

func main() {
	var buf io.Writer
	if debug {
		buf = new(bytes.Buffer)
	}
	f(buf)
	if debug {
		buf = buf.(*bytes.Buffer)
	}
}

func f(out io.Writer) {
	if out != nil {
		out.Write([]byte("hello"))
	}
}
