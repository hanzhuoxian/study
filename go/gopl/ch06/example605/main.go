package main

import "fmt"

// **练习 6.5：** 我们这章定义的IntSet里的每个字都是用的uint64类型，但是64位的数值可能在32位的平台上不高效。修改程序，使其使用uint类型，这种类型对于32位平台来说更合适。当然了，这里我们可以不用简单粗暴地除64，可以定义一个常量来决定是用32还是64，这里你可能会用到平台的自动判断的一个智能表达式：32 << (^uint(0) >> 63)

const wordSize = 32 << (^uint(0) >> 63) // Word size in bits

type IntSet struct {
	words []uint
}

func New(values ...int) *IntSet {
	s := &IntSet{}
	for _, v := range values {
		s.Add(v)
	}
	return s
}

func (w *IntSet) Has(x int) bool {
	word, bit := x/wordSize, uint(x%wordSize)
	return word < len(w.words) && w.words[word]&(1<<bit) != 0
}
func (w *IntSet) Add(x int) {
	word, bit := x/wordSize, uint(x%wordSize)
	for word >= len(w.words) {
		w.words = append(w.words, 0)
	}
	w.words[word] |= 1 << bit
}
func (s *IntSet) Elems() []int {
	n := len(s.words) * wordSize
	elements := make([]int, 0, n)
	for i, word := range s.words {
		if word == 0 {
			continue
		}
		for j := 0; j < wordSize; j++ {
			if word&(1<<uint(j)) != 0 {
				elements = append(elements, wordSize*i+j)
			}
		}
	}
	return elements
}
func main() {
	s := New(1, 2, 3)
	fmt.Println(s.Elems())
}
