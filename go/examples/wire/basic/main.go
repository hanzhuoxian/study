package main

import (
	"context"
	"fmt"
)

func main() {
	baz, err := initializeBaz(context.Background())
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(baz.X)

	other, err := initializeOther(context.Background())
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(other.Y)
}
