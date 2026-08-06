package intset

import (
	"bytes"
	"fmt"
)

// **练习 6.2：** 定义一个变参方法(*IntSet).AddAll(...int)，这个方法可以添加一组IntSet，比如s.AddAll(1,2,3)。

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

func (s *IntSet) AddAll(xs ...int) {
	for _, x := range xs {
		s.Add(x)
	}
}

func (s *IntSet) UnionWith(t *IntSet) {
	for i, tword := range t.words {
		if i < len(s.words) {
			s.words[i] |= tword
		} else {
			s.words = append(s.words, tword)
		}
	}
}

// String returns the set as a string of the form "{1 2 3}".
func (s *IntSet) String() string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, word := range s.words {
		if word == 0 {
			continue
		}
		for j := range 64 {
			if word&(1<<uint(j)) != 0 {
				if buf.Len() > len("{") {
					buf.WriteByte(' ')
				}
				fmt.Fprintf(&buf, "%d", 64*i+j)
			}
		}
	}
	buf.WriteByte('}')
	return buf.String()
}

// Len returns the number of elements.
func (s *IntSet) Len() int {
	n := 0
	for _, word := range s.words {
		for word != 0 {
			word &= word - 1
			n++
		}
	}
	return n
}

// Remove removes x from the set.
func (s *IntSet) Remove(x int) {
	word, bit := x/64, x%64
	if word < len(s.words) {
		s.words[word] &^= 1 << bit
	}
}

// Clear removes all elements from the set.
func (s *IntSet) Clear() {
	s.words = nil
}

// Copy returns a copy of the set.
func (s *IntSet) Copy() *IntSet {
	words := make([]uint64, len(s.words))
	copy(words, s.words)
	return &IntSet{words: words}
}
