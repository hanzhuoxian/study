package main

import (
	"fmt"
	"log"
)

// **练习5.10：** 重写topoSort函数，用map代替切片并移除对key的排序代码。验证结果的正确性（结果不唯一）。

var preReqs = map[string][]string{
	"algorithms": {"data structures"},
	"calculus":   {"linear algebra"},
	"compilers": {
		"data structures",
		"formal languages",
		"computer organization",
	},
	"data structures":       {"discrete math"},
	"databases":             {"data structures"},
	"discrete math":         {"intro to programming"},
	"formal languages":      {"discrete math"},
	"networks":              {"operating systems"},
	"operating systems":     {"data structures", "computer organization"},
	"programming languages": {"data structures", "computer organization"},
}

func topoSort(m map[string][]string) map[int]string {
	order := make(map[int]string)
	var i int
	seen := make(map[string]bool)
	var visitAll func(items []string)
	visitAll = func(items []string) {
		for _, item := range items {
			if !seen[item] {
				seen[item] = true
				visitAll(m[item])
				i++
				order[i] = item
			}
		}
	}
	for key := range m {
		if !seen[key] {
			seen[key] = true
			visitAll(m[key])
			i++
			order[i] = key
		}
	}
	return order
}

func main() {
	order := topoSort(preReqs)
	l := len(order)
	seen := make(map[string]bool)
	for i := 1; i <= l; i++ {
		fmt.Printf("%d:%s\n", i, order[i])
		seen[order[i]] = true
		for _, p := range preReqs[order[i]] {
			if !seen[p] {
				log.Fatal("sort failed")
			}
		}

	}
}
