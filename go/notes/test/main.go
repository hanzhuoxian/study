package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func main() {
	var x, y atomic.Int32
	both0 := 0
	const rounds = 20000

	for range rounds {
		x.Store(0)
		y.Store(0)
		var r1, r2 int32
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); x.Store(1); r1 = y.Load() }()
		go func() { defer wg.Done(); y.Store(1); r2 = x.Load() }()
		wg.Wait()
		if r1 == 0 && r2 == 0 {
			both0++
		}
	}
	fmt.Printf("  atomic 版本跑 %d 轮，两边同时读到 0 的次数: %d（顺序一致性保证为 0）\n", rounds, both0)
	fmt.Println("→ 如果用普通变量 + 编译器/CPU 重排，理论上可以两边都读到 0（x86 的 store buffer）")
	fmt.Println("→ atomic 除了'不撕裂'，还插入了必要的内存屏障（amd64 上 Store 是 XCHG，自带 full barrier）")
	fmt.Println("→ Go 只提供 seq_cst 一种强度，没有 C++ 的 relaxed/acquire/release 可选")
	fmt.Println("  （提案 golang/go#68578 讨论过 weak atomics，至今未进；理由是太容易写错）")
}
