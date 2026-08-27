package main

import "fmt"

// 练习 4.4： 编写一个 rotate 函数，通过一次循环完成旋转。
// 循环替换法（juggling 算法）：每个元素只移动一次，共移动 len 次。

func main() {
	s := []int{1, 2, 3, 4, 5}
	rotate(s, 2)
	fmt.Println(s) // [3 4 5 1 2]
}

// rotate 将 s 向左旋转 n 位
func rotate(s []int, n int) {
	length := len(s)
	if length == 0 {
		return
	}
	// 归一化：处理 n 大于长度、n 为负的情况
	n = ((n % length) + length) % length
	if n == 0 {
		return
	}

	// 环的条数 = gcd(n, length)，需要从多个起点各走一条环
	for start := 0; start < gcd(n, length); start++ {
		temp := s[start] // 暂存环起点的值，最后回填
		j := start
		for {
			k := j + n // 下一个要搬过来的位置
			if k >= length {
				k -= length
			}
			if k == start { // 绕回起点，环走完
				break
			}
			s[j] = s[k] // 把后面的元素往前搬
			j = k
		}
		s[j] = temp // 环的最后一格填入暂存值
	}
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
