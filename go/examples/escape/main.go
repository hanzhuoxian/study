package main

func main() {
	p := foo()
	println(*p) // 输出 10
}

func foo() *int {
	x := 10
	return &x // x 逃逸到堆
}
