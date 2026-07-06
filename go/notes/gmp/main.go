package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"sync"
	"time"
)

// 运行后用 `go tool trace tmp/trace.out` 打开，看 Goroutine analysis / Proc / Synchronization blocking profile。
func main() {
	runtime.GOMAXPROCS(4) // 固定 P 的数量，方便在 trace 里数清楚有几个 P 在并行

	if err := os.MkdirAll("tmp", 0o755); err != nil {
		panic(err)
	}
	f, err := os.Create(filepath.Join("tmp", "trace.out"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := trace.Start(f); err != nil {
		panic(err)
	}
	defer trace.Stop()

	var wg sync.WaitGroup

	// goroutine 数 > GOMAXPROCS，且每个都有真实计算量：
	// 逼调度器把它们分散到多个 P 上并行跑，队列不均时触发 work-stealing，
	// 单个循环跑够久还会被异步抢占打断。
	const cpuWorkers = 8
	for range cpuWorkers {
		wg.Go(busyLoop)
	}

	// 阻塞在无缓冲 channel 上：这个 G 会进入 _Gwaiting，直到下面主 goroutine 发送数据唤醒它。
	ch := make(chan int)
	wg.Go(func() {
		v := <-ch
		fmt.Println("received:", v)
	})

	// time.Sleep 会挂到 P 的 timer 堆上，G 进入 _Gwaiting，到点由 sysmon/netpoller 唤醒。
	wg.Go(func() {
		time.Sleep(50 * time.Millisecond)
		fmt.Println("timer goroutine done")
	})

	// 阻塞式文件 I/O 会让 G 进入 _Gsyscall，对应的 M 会把 P 交接给别的 M 去跑其他 G。
	wg.Go(func() {
		if b, err := os.ReadFile(os.Args[0]); err == nil {
			fmt.Println("read binary bytes:", len(b))
		}
	})

	time.Sleep(5 * time.Millisecond) // 确保 channel 接收方先进入等待态，再发送
	ch <- 42

	wg.Wait() // 必须等所有 G 跑完再让 main 返回，否则 trace 会在中途被截断
}

func busyLoop() {
	sum := 0
	for i := range 300_000_000 {
		sum += i
	}
	_ = sum
}
