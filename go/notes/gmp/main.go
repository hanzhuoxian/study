package main

import (
	"os"
	"runtime/trace"
)

func main() {
	f, err := os.Create("./tmp/trace.out")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()

}
