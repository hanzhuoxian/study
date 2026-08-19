package main

import (
	"flag"
	"fmt"
)

// **练习 7.6：** 对tempFlag加入支持开尔文温度。

type Celsius float64    //摄氏温度
type Fahrenheit float64 //华氏温度
type Kelvin float64     //开尔文温度

func (c Celsius) String() string { return fmt.Sprintf("%g° C", c) }

// *celsiusFlag satisfies the flag.Value interface.
type celsiusFlag struct{ Celsius }

func (f *celsiusFlag) Set(s string) error {
	var unit string
	var value float64
	fmt.Sscanf(s, "%f%s", &value, &unit) // no error check needed
	switch unit {
	case "C", "°C":
		f.Celsius = Celsius(value)
		return nil
	case "F", "°F":
		f.Celsius = FToC(Fahrenheit(value))
		return nil
	case "K", "°K":
		f.Celsius = KToC(Kelvin(value))
		return nil
	}
	return fmt.Errorf("invalid temperature %q", s)
}

// CelsiusFlag 摄氏温度标志
func CelsiusFlag(name string, value Celsius, usage string) *celsiusFlag {
	f := celsiusFlag{value}
	flag.Var(&f, name, usage)
	return &f
}

// CToF 摄氏温度转华氏温度
func CToF(c Celsius) Fahrenheit { return Fahrenheit(c*9/5 + 32) }

// FToC 华氏温度转摄氏温度
func FToC(f Fahrenheit) Celsius { return Celsius((f - 32) * 5 / 9) }

// CToK 摄氏温度转开尔文温度
func CToK(c Celsius) Kelvin { return Kelvin(c + 273.15) }

// KToC 开尔文温度转摄氏温度
func KToC(k Kelvin) Celsius { return Celsius(k - 273.15) }

func main() {
	c := CelsiusFlag("celsius", 20, "celsius temperature")
	flag.Parse()
	fmt.Println(c)
}
