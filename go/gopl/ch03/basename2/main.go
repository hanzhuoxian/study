package main

import (
	"fmt"
	"strings"
)

func main() {
	fmt.Println(basename("/"))        //
	fmt.Println(basename("a/b/c.go")) //c
	fmt.Println(basename("c.d.go"))   //c.d
	fmt.Println(basename("abc"))      // abc
}

func basename(s string) string {
	start := strings.LastIndex(s, "/")
	s = s[start+1:]
	end := strings.LastIndex(s, ".")
	if end >= 0 {
		s = s[:end]
	}

	return s
}
