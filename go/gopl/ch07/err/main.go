package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	fmt.Println(errors.New("My Error"))
	fmt.Println(fmt.Errorf("My %s", "Error"))

	_, err := os.Open("/no/such/file")
	fmt.Println(err) // "open /no/such/file: No such file or directory"
	if os.IsExist(err) {
		fmt.Println("file is exist")
	}
}
