package main

import (
	"errors"
	"fmt"
	"io"
)

func main() {
	f, cleanup, err := NewFile(Path("./wire.go"))
	if err != nil {
		panic(err)
	}
	defer cleanup()
	buf := make([]byte, 1024)
	for {
		_, err = f.Read(buf)
		if err != nil {
			break
		}
		fmt.Println(string(buf))
	}
	if err != nil && !errors.Is(err, io.EOF) {
		panic(err)
	}

	fmt.Println("done")
}
