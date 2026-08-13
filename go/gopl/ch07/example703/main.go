package main

// **练习 7.3：** 为在gopl.io/ch4/treesort（§4.4）中的*tree类型实现一个String方法去展示tree类型的值序列。

import (
	"bytes"
	"fmt"
)

type tree struct {
	value       int
	left, right *tree
}

func (t *tree) String() string {
	if t == nil {
		return ""
	}
	buf := new(bytes.Buffer)
	var walk func(t *tree)
	walk = func(t *tree) {
		if t == nil {
			return
		}
		walk(t.left)
		if buf.Len() > 0 {
			buf.WriteString(" ")
		}
		fmt.Fprintf(buf, "%d", t.value)
		walk(t.right)
	}
	walk(t)
	return buf.String()
}

func main() {
	numbers := []int{2, 1, 3, 5, 4}
	Sort(numbers)
	fmt.Println(numbers)

	var root *tree
	for _, v := range numbers {
		root = add(root, v)
	}
	fmt.Println(root) // 会自动调用 root.String()
}

func Sort(values []int) {
	var root *tree

	for _, v := range values {
		root = add(root, v)
	}

	appendValues(values[:0], root)
}

func appendValues(values []int, t *tree) []int {
	if t != nil {
		values = appendValues(values, t.left)
		values = append(values, t.value)
		values = appendValues(values, t.right)
	}
	return values
}

func add(t *tree, value int) *tree {
	if t == nil {
		t = new(tree)
		t.value = value
		return t
	}

	if value < t.value {
		t.left = add(t.left, value)
	} else {
		t.right = add(t.right, value)
	}

	return t
}
