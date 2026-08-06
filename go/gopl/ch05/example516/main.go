package main

import "fmt"

func main() {
	fmt.Println()
	fmt.Println(Join(": ", "a", "b", "c"))
}

func Join(sep string, vals ...string) string {
	if len(vals) == 0 {
		return ""
	}
	result := vals[0]
	for _, v := range vals[1:] {
		result += sep + v
	}
	return result
}
