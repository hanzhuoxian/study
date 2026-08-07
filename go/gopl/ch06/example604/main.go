package main

// **练习 6.4：** 实现一个 Elems 方法，返回集合中的所有元素，用于做一些range之类的遍历操作。
func (s *IntSet) Elems() []int {
	n := len(s.words) * 64
	elements := make([]int, 0, n)
	for i, word := range s.words {
		if word == 0 {
			continue
		}
		for j := 0; j < 64; j++ {
			if word&(1<<uint(j)) != 0 {
				elements = append(elements, 64*i+j)
			}
		}
	}
	return elements
}

type IntSet struct {
	words []uint64
}

func (s *IntSet) Has(x int) bool {
	word, bit := x/64, x%64
	return word < len(s.words) && s.words[word]&(1<<bit) != 0
}

func (s *IntSet) Add(x int) {
	word, bit := x/64, x%64
	for word >= len(s.words) {
		s.words = append(s.words, 0)
	}
	s.words[word] |= 1 << bit
}
