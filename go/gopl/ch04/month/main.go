package main

import "fmt"

var months = [...]string{
	1:  "January",
	2:  "February",
	3:  "March",
	4:  "April",
	5:  "May",
	6:  "June",
	7:  "July",
	8:  "August",
	9:  "September",
	10: "October",
	11: "November",
	12: "December",
}

func main() {
	fmt.Println(len(months)) // 13 (index 0 is the empty string)
	for i, m := range months {
		fmt.Printf("%2d: %s\n", i, m)
	}

	Q2 := months[4:7]
	summer := months[6:9]
	fmt.Println(Q2)     // ["April" "May" "June"]
	fmt.Println(summer) // ["June" "July" "August"]

	endlessSummer := summer[:5] // 没有超出 cap 会扩展 slice
	fmt.Println(endlessSummer)

	var s []int
	if s == nil {
		fmt.Println("s = nil")
	}
	s = nil
	if s == nil {
		fmt.Println("s = nil")
	}
	s = []int(nil)
	if s == nil {
		fmt.Println("s = nil")
	}
	s = []int{}

	if s != nil {
		fmt.Println("s != nil")
	}
}
