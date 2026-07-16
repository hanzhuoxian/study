package main

import (
	"fmt"
	"sort"
)

func main() {
	// 创建 1
	ages := make(map[string]int)
	ages["alice"] = 31
	ages["charlie"] = 34

	// 创建 2 等价与 创建 1
	ages1 := map[string]int{
		"alice":   31,
		"charlie": 34,
	}

	fmt.Println(ages1)

	// 删除
	delete(ages1, "alice")
	fmt.Println(ages1)

	// 不能进行取地址
	// _ = &ages1["alice"]

	// 遍历全部元素
	fmt.Println("for range ages start")
	for k, v := range ages {
		fmt.Println(k, v)
	}
	fmt.Println("for range ages end")

	// map 有序输出
	fmt.Println("order by map start")
	names := make([]string, len(ages))
	for k := range ages {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, n := range names {
		fmt.Println(n, ages[n])
	}

	fmt.Println("order by map end")

	var m map[int]int
	fmt.Println(m == nil)
	fmt.Println(len(m) == 0)

	m1 := map[int]int{
		1: 1,
	}
	fmt.Println(m1)

	if i, ok := m1[1]; ok {
		fmt.Println(i)
	}

	if i, ok := m1[2]; !ok {
		fmt.Println("no", i)
	}

	// map 也不能使用等于号进行比较

}
