package main

import "fmt"

func main() {
	foobar := injectFooBar()
	fmt.Printf("%d", foobar.MyBar)

	fb := injectFooBarPointer()
	fmt.Println(fb)
}
