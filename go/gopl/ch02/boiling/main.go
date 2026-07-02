// boiling 输出冰点的华氏和摄氏温度
package main

import "fmt"

// BoilingF 沸点 华氏温度
const BoilingF = 212.0

func main() {
	var f = BoilingF
	var c = (f - 32) * 5 / 9
	fmt.Printf("boiling point = %g° F or %g° C \n", BoilingF, c)
}
