// 重量单位:千克磅的转换 千克 = 磅 × 2.2046
package main

import (
	"flag"
	"fmt"
)

type Pound float64
type Kg float64

func (p Pound) String() string { return fmt.Sprintf("%g\n", p) }
func (k Kg) String() string    { return fmt.Sprintf("%g\n", k) }

// PoundToKg 将磅转换为千克
func PoundToKg(p Pound) Kg {
	return Kg(p / 2.2046)
}

// KgToPound 将千克转换为磅
func KgToPound(k Kg) Pound {
	return Pound(k * 2.2046)
}

var kgParams = flag.Float64("kg", 0.0, "please input float")
var poundParams = flag.Float64("pound", 0.0, "please input float")

func main() {
	flag.Parse()
	// TODO 从标准输入读取
	fmt.Println(KgToPound(Kg(*kgParams)), PoundToKg(Pound(*poundParams)))
}

var a = b + c //第三个初始化为3
var b = f()   // 第二个初始化为2
var c = 1     // 第一个初始化为1

func f() int { return c + 1 }
