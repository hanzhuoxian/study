package main

import "fmt"

type Point struct {
	X, Y int
}

type Circle struct {
	Point
	Radius int
}

type Wheel struct {
	Circle
	Spokes int
}

func main() {
	w := Wheel{Circle{Point{1, 2}, 3}, 4}

	w1 := Wheel{Circle: Circle{
		Point:  Point{1, 2},
		Radius: 3,
	}, Spokes: 4,
	}
	fmt.Printf("%#v\n", w)
	fmt.Printf("%#v\n", w1)

	fmt.Println(w == w1)
}
