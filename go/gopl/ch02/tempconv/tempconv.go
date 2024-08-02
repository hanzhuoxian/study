// Package 摄氏温度及华氏温度相互转换
package tempconv

import "fmt"

type Celsius float64    //摄氏温度
type Fahrenheit float64 //华氏温度
type Kelvin float64     //

// 典型温度值定义
const (
	AbsoluteZeroC Celsius = -273.15 //绝对零度
	FreezingC     Celsius = 0       // 结冰点温度
	BoilingC      Celsius = 100     //沸点温度
)

func (c Celsius) String() string    { return fmt.Sprintf("%g° C", c) }
func (f Fahrenheit) String() string { return fmt.Sprintf("%g° F", f) }
func (k Kelvin) String() string     { return fmt.Sprintf("%g° K", k) }
