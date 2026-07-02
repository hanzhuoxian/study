package main

import "fmt"

func main() {
	bar, err := initializeBar()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(bar)
}
