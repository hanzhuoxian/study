package main

import (
	"fmt"
	"sync"
)

type singleton struct {
}

var ins *singleton
var once sync.Once

type Person interface {
	Greet()
}

type person struct {
	name string
	age  int
}

func (p person) Greet() {
	fmt.Printf("Hi! My name is %s", p.name)
}

// Here, NewPerson returns an interface, and not the person struct itself
func NewPerson(name string, age int) Person {
	return person{
		name: name,
		age:  age,
	}
}

func GetInsOr() *singleton {
	once.Do(func() {
		ins = &singleton{}
	})
	return ins
}
