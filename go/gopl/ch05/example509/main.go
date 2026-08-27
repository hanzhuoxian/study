package main

// **练习 5.9：** 编写函数expand，将s中的"foo"替换为f("foo")的返回值。

func expand(s string, f func(string) string) string {
	return f("foo")
}

func main() {
	result := expand("foo", func(s string) string {
		return "bar"
	})
	println(result) // Output: bar
}
