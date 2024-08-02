package main

import (
	"context"
	"fmt"
	"log"
)

func main() {
	baz, err := initializeBaz(context.Background())
	if err != nil {
		log.Fatalf("could not create baz %v\n", baz)
	}
	fmt.Printf("baz.X = %d\n", baz.X)

	bar, err := initializeBar(context.Background())
	if err != nil {
		log.Fatalf("could not crerte bar %v\n", bar)
	}
	fmt.Printf("bar.X = %d\n", bar.X)

	foo, err := initializeFoo(context.Background())
	if err != nil {
		log.Fatalf("could not crerte foo %v\n", foo)
	}
	fmt.Printf("foo.X = %d\n", foo.X)
}
