package main

import (
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"io"
	"os"
)

var n = flag.Int("n", 256, "sha num")

func main() {
	flag.Parse()
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s := ""
	switch *n {
	case 512:
		s = fmt.Sprintf("%x", sha512.Sum512(data))
	case 384:
		s = fmt.Sprintf("%x", sha512.Sum384(data))
	case 256:
		s = fmt.Sprintf("%x", sha256.Sum256(data))
	default:
		fmt.Println("input invalid sha num")
		os.Exit(2)
	}

	fmt.Println(s)
}
