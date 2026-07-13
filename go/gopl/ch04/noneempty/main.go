package main

import "fmt"

func main() {
	s := []string{"hello", "", "world", "", "!"}
	fmt.Println(noneEmpty(s))
	fmt.Println(s)
}

func noneEmpty(strings []string) []string {
	i := 0
	for _, s := range strings {
		if s != "" {
			strings[i] = s
			i++
		}
	}
	return strings[:i]
}
