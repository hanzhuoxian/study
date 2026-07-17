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
