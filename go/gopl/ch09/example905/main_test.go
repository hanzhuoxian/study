package main

import "testing"

// BenchmarkPingPong 用 go 自带的基准测试工具回答"每秒多少次通信":
//
//	go test -bench . -benchtime 2s
//	go test -bench . -cpu 1,2,8
//
// 一次 b.Loop 迭代 = 一个往返 = 2 次通信, 结果里额外报告 comm/s。
func BenchmarkPingPong(b *testing.B) {
	ping := make(chan struct{})
	pong := make(chan struct{})
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ping:
				pong <- struct{}{}
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	for b.Loop() {
		ping <- struct{}{}
		<-pong
	}

	b.ReportMetric(float64(2*b.N)/b.Elapsed().Seconds(), "comm/s")
}
