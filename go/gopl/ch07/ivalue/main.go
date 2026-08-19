package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

func main() {
	var w io.Writer
	fmt.Printf("%T\n", w) // "<nil>"

	w = os.Stdout
	fmt.Printf("%T\n", w) // "*os.File"

	w = new(bytes.Buffer)
	fmt.Printf("%T\n", w) // "*bytes.Buffer"

	w = nil
	fmt.Printf("%T\n", w) // "<nil>"

}
