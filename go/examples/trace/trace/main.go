package main

import (
	"fmt"
	"log"
	"os"
	"runtime/trace"
)

func main() {
	f, err := os.Create("./trace.out")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	trace.Start(f)
	fmt.Println("hello")
	trace.Stop()
}

// go tool trace trace.out
// 查看trace信息
