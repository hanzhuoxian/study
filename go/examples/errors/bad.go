package main

import "fmt"

func main() {
	isValid("()}]")
}
func isValid(s string) bool {
	sLen := len(s)
	stack := []byte{}
	m := map[byte]byte{
		')': '(',
		'}': '{',
		']': '[',
	}
	for i := 0; i < sLen; i++ {
		fmt.Println(stack)
		if s[i] == '(' || s[i] == '{' || s[i] == '[' {
			stack = append(stack, s[i])
			continue
		}

		if left, ok := m[s[i]]; ok && len(stack) > 0 && stack[len(stack)-1] == left {
			stack = stack[:len(stack)-1]
		} else {
			return false
		}
	}

	return true
}
