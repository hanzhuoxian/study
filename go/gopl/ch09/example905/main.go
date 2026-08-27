// **练习 9.5:** 写一个有两个goroutine的程序，两个goroutine会向两个无buffer channel反复地发送ping-pong消息。
// 这样的程序每秒可以支持多少次通信？
//
// 用法:
//
//	go run . [-rounds 1000000]
//
// 一次「往返」(round) 包含两次通信: 发球方 ping-> 回球方, 回球方 pong-> 发球方。
// 因为两个 channel 都是无缓存的, 每次发送都必须等到对方接收, 两个 goroutine
// 严格交替运行, 任何时刻只有一个在跑 —— 这也是为什么 GOMAXPROCS=1 反而最快。
package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"
)

// pingPong 让两个 goroutine 在两个无缓存 channel 上来回传递 rounds 个来回,
// 返回这些往返总共花费的时间。
//
// 计时放在发球方(即当前 goroutine)内部, 且循环里不做任何额外工作
// (不取时间、不累加共享计数器), 以免测量开销淹没 channel 通信本身的开销。
func pingPong(rounds int) time.Duration {
	ping := make(chan struct{})
	pong := make(chan struct{})

	// 回球方: 收到一个 ping 就回一个 pong, 正好 rounds 次后退出, 不泄漏 goroutine。
	go func() {
		for range rounds {
			<-ping
			pong <- struct{}{}
		}
	}()

	// 发球方。
	start := time.Now()
	for range rounds {
		ping <- struct{}{}
		<-pong
	}
	return time.Since(start)
}

func main() {
	rounds := flag.Int("rounds", 1_000_000, "ping-pong 往返次数")
	flag.Parse()

	if *rounds <= 0 {
		fmt.Println("rounds 必须大于 0")
		return
	}

	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(0)) // 结束时恢复原值

	fmt.Printf("NumCPU = %d, 往返次数 = %d (每次往返 2 次通信)\n\n",
		runtime.NumCPU(), *rounds)
	fmt.Printf("%-10s %12s %16s %16s\n", "GOMAXPROCS", "耗时", "通信/秒", "每次通信")

	for _, procs := range procsToTest() {
		runtime.GOMAXPROCS(procs)

		pingPong(*rounds / 10) // 预热, 让调度器和内存分配稳定下来

		d := pingPong(*rounds)
		comms := float64(2 * *rounds)
		fmt.Printf("%-10d %12s %16.0f %16s\n",
			procs,
			d.Round(time.Millisecond),
			comms/d.Seconds(),
			(d / time.Duration(2**rounds)).String(),
		)
	}
}

// procsToTest 返回要对比的 GOMAXPROCS 取值(去重且保持升序)。
func procsToTest() []int {
	var out []int
	for _, p := range []int{1, 2, runtime.NumCPU()} {
		if p >= 1 && (len(out) == 0 || p > out[len(out)-1]) {
			out = append(out, p)
		}
	}
	return out
}
