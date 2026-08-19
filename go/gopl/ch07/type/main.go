package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

func main() {
	var w io.Writer
	w = os.Stdout
	f := w.(*os.File)
	// b := w.(*bytes.Buffer)

	fmt.Printf("%T\n", f)
	// fmt.Printf("%T", b)

	wr := w.(io.ReadWriter)
	fmt.Printf("%T\n", wr)

	w = wr.(io.Writer)

	if f, ok := w.(*os.File); ok {
		f.Write([]byte("write"))
	}

	if b, ok := w.(*bytes.Buffer); !ok {
		fmt.Printf("%T is not ", b)
	}
}
