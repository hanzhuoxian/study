package main

import (
	"fmt"
	"log"
)

func main() {
	f, c, err := injectFile()
	if err != nil {
		log.Fatal(err)
	}
	defer c()
	stat, err := f.Stat()

	fmt.Println(stat.Name())
}
