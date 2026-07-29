package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type Movie struct {
	Name      string `json:"name"`
	Author    string
	Password  string    `json:"-"`                             // 忽略字段
	Age       int       `json:"age,string"`                    // 强制转换成字符串
	Color     bool      `json:"color,omitempty" db:"is_color"` // 空值忽略字段
	CreatedAt time.Time `json:"created_at,omitzero"`
}

func main() {
	movie := Movie{
		Name:      "功夫女足",
		Author:    "周星驰",
		Age:       18,
		Color:     true,
		CreatedAt: time.Now(),
	}

	rt, ok := reflect.TypeOf(movie).FieldByName("Color")
	if !ok {

	}
	fmt.Println(rt.Tag.Get("json"))
	fmt.Println(strings.Cut(rt.Tag.Get("json"), ","))
	fmt.Println(rt.Tag.Get("db"))
	s, err := json.MarshalIndent(movie, " ", " ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(s))
}
