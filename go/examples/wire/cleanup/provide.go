package main

import (
	"log"
	"os"
)

func provideFile() (*os.File, func(), error) {
	f, err := os.Open("./log.txt")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}
	return f, cleanup, nil
}
