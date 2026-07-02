package main

import "fmt"

// N 数组长度
const N int = 3

var arr [N]int
var count = 0

func main() {
	insert(5)
	insert(4)
	insert(3)

	insert(10)
	insert(11)
	insert(12)
	insert(0)
	insert(0)
	fmt.Println(arr, count)
}

func insert(val int) {

	if count == N {
		sum := 0
		for i := 0; i < N; i++ {
			sum += arr[i]
		}
		arr[0] = sum
		count = 1
	}

	arr[count] = val
	count++
}
