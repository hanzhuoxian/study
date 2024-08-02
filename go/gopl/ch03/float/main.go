package main

import (
	"fmt"
	"math"
)

func main() {
	// 浮点数示例
	var f float32 = 34.32
	var f1 float64 = 43.43
	fmt.Printf("%f %[1]T\n%f %[2]T\n", f, f1)
	// 34.320000 float32
	// 43.430000 float64

	// +Inf 正无穷大 -Inf 负无穷大 NaN 非数（不是一个数）
	var z float64
	fmt.Println(z, -z, 1/z, -1/z, z/z)
	// 0 -0 +Inf -Inf NaN

	nan := math.NaN()
	fmt.Println(nan == nan, nan < nan, nan > nan, math.IsNaN(nan))
	// false false false true
}
