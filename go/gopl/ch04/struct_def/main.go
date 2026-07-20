package main

import (
	"fmt"
	"time"
)

type Employee struct {
	ID        int
	Name      string
	Address   string
	DoB       time.Time
	Position  string
	Salary    int
	ManagerID int
}

func main() {
	e := Employee{ID: 1, Name: "Bob", Position: "Dev"}
	fmt.Println(e)

	// 访问结构体字段
	e.Salary = 1000
	fmt.Println(e.Salary)

	// 访问结构体字段的指针
	position := &e.Position
	*position = "CEO-" + *position
	fmt.Println(e.Position)

	// 指向结构题的指针
	var pe *Employee = &e
	pe.Salary += 2000
	fmt.Println(e.Salary)

	(*pe).Salary += 3000
	fmt.Println(e.Salary)

	employees = append(employees, e)
	fmt.Println(GetEmployeeByID(1).Position)

	GetEmployeeByID(1).Position = "CTO"
	fmt.Println(GetEmployeeByID(1).Position)

	// 结构体嵌入
	wheel := Wheel{
		Circle: Circle{
			Center: Point{X: 1, Y: 1},
			Radius: 3,
		},
		Spokes: 26,
	}

	fmt.Println(wheel)
	fmt.Println("x = ", wheel.Circle.Center.X)

	// 匿名结构体
	var simpleWheel SimpleWheel
	simpleWheel.X = 1
	simpleWheel.Y = 2
	simpleWheel.Radius = 3
	simpleWheel.Spokes = 4
	fmt.Println(simpleWheel)

	// 可以直接访问叶子属性而不需要写出完整路径
	simpleWheel = SimpleWheel{
		SimpleCircle: SimpleCircle{
			Point:  Point{X: 1, Y: 1},
			Radius: 3,
		},
		Spokes: 26,
	}
	fmt.Println(simpleWheel)
}

var employees []Employee

func GetEmployeeByID(id int) *Employee {
	for i := range employees {
		if employees[i].ID == id {
			return &employees[i]
		}
	}
	return nil
}

type Point struct {
	X, Y int
}

type Circle struct {
	Center Point
	Radius int
}

type Wheel struct {
	Circle Circle
	Spokes int
}

type SimpleCircle struct {
	Point
	Radius int
}

type SimpleWheel struct {
	SimpleCircle
	Spokes int
}
