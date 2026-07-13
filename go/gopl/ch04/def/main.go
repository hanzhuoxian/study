package main

import "fmt"

func main() {
	var a [3]int
	fmt.Println(a[0])
	fmt.Println(a[len(a)-1])

	for i, v := range a {
		fmt.Printf("index: %d, value: %d\n", i, v)
	}

	for _, v := range a {
		fmt.Printf("value: %d\n", v)
	}

	var q [3]int = [3]int{1, 2, 3}
	fmt.Println(q[2]) // "3"
	var r [3]int = [3]int{1, 2}
	fmt.Println(r[2]) // "0"

	// ... 省略数组长度，由初始值个数计算
	qq := [...]int{1, 2, 3}
	fmt.Printf("%T\n", qq) // "[3]int"

	// 指定某个索引的初始值，其余初始化为 0 值
	rr := [...]int{99: -1}
	fmt.Printf("1 : %d, 0: %d, type: %T", rr[1], rr[99], rr)
}
