package main

import (
	"fmt"
	"time"
)

func main() {
	defer trace("main")()
	time.Sleep(1 * time.Second)

	bigSlowOperation()
}
func trace(name string) func() {
	start := time.Now()
	fmt.Printf("enter : %s", name)
	return func() {
		fmt.Printf("%s: %s\n", name, time.Since(start))
	}
}

func bigSlowOperation() {
	defer trace("bigSlowOperation")() // don't forget the extra parentheses
	// ...lots of work…
	time.Sleep(10 * time.Second) // simulate slow operation by sleeping
}
