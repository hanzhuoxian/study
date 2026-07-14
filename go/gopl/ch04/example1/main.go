package main

import (
	"crypto/sha256"
	"fmt"
	"math/bits"
)

func main() {
	s1 := sha256.Sum256([]byte("x"))
	s2 := sha256.Sum256([]byte("X"))

	fmt.Printf("s1 s2 diff bit: %d", DiffCount(s1, s2))
}

func DiffCount(a [32]byte, b [32]byte) int {
	n := 0
	for i, v := range a {
		// v ^ b[i] 中为 1 的位,就是这一字节里两者不同的 bit
		n += bits.OnesCount8(v ^ b[i])
	}
	return n
}
