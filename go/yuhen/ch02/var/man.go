package main

var x = 100

func main() {
	println(&x, x)
	x := "abc" // 会定义新变量而不是覆盖全局变量
	println(&x, x)
}
