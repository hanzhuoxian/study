package main

func mkslice() []int {
	s := make([]int, 0, 10)
	s = append(s, 100)
	return s
}

func mkmap() map[string]int {
	m := make(map[string]int)
	m["a"] = 1
	return m
}

// go build -gcflags "-l" && go tool objdump -s "main\.mk" reference_type
func main() {
	m := mkmap()
	println(m["a"])
	s := mkslice()
	println(s[0])
}
