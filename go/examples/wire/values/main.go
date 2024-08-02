package main

import (
	"bufio"
	"fmt"
)

func main() {
	foo := injectFoo()
	fmt.Println(foo)

	in := injectReader()
	scan := bufio.NewScanner(in)
	fmt.Println(scan.Text())

}
