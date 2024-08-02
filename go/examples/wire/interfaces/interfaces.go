package main

import "fmt"

func main() {
	var foo Fooer
	foo = InitializeFooer()
	fmt.Println(foo.Foo())
}
