package main

import (
	"testing"
	"time"
)

// go test -bench . -benchmem ./tm
// 对比 1.23 之前的 timer 行为：GODEBUG=asynctimerchan=1 go test -bench . -benchmem ./tm

var (
	timeSink time.Time
	intSink  int64
	strSink  string
	boolSink bool
)

// ---------------------------------------------------------------------------
// ① 取当前时间的成本
// ---------------------------------------------------------------------------

func BenchmarkTimeNow(b *testing.B) {
	for b.Loop() {
		timeSink = time.Now()
	}
}

func BenchmarkTimeSince(b *testing.B) {
	start := time.Now()
	for b.Loop() {
		intSink = int64(time.Since(start))
	}
}

func BenchmarkTimeNowUnixNano(b *testing.B) {
	for b.Loop() {
		intSink = time.Now().UnixNano()
	}
}

// runtime.nanotime 的等价物：只要单调时钟，不要墙上时间
func BenchmarkMonotonicOnly(b *testing.B) {
	var zero time.Time
	for b.Loop() {
		intSink = int64(time.Since(zero))
	}
}

// ---------------------------------------------------------------------------
// ② select + time.After vs 复用 Timer
// ---------------------------------------------------------------------------

func BenchmarkSelectAfter(b *testing.B) {
	ch := make(chan int)
	for b.Loop() {
		select {
		case <-ch:
		case <-time.After(time.Hour):
		default:
		}
	}
}

func BenchmarkSelectReuseTimer(b *testing.B) {
	ch := make(chan int)
	t := time.NewTimer(time.Hour)
	defer t.Stop()
	for b.Loop() {
		t.Reset(time.Hour)
		select {
		case <-ch:
		case <-t.C:
		default:
		}
	}
}

// 连 Reset 都省掉：只在真的需要超时的那一次才碰 timer
func BenchmarkSelectNoTimer(b *testing.B) {
	ch := make(chan int)
	for b.Loop() {
		select {
		case <-ch:
		default:
		}
	}
}

// ---------------------------------------------------------------------------
// ③ Timer 的创建/停止成本
// ---------------------------------------------------------------------------

func BenchmarkNewTimerStop(b *testing.B) {
	for b.Loop() {
		t := time.NewTimer(time.Hour)
		t.Stop()
	}
}

func BenchmarkTimerReset(b *testing.B) {
	t := time.NewTimer(time.Hour)
	defer t.Stop()
	for b.Loop() {
		t.Reset(time.Hour)
	}
}

func BenchmarkAfterFuncStop(b *testing.B) {
	f := func() {}
	for b.Loop() {
		t := time.AfterFunc(time.Hour, f)
		t.Stop()
	}
}

// ---------------------------------------------------------------------------
// ④ 格式化与解析
// ---------------------------------------------------------------------------

var t0 = time.Date(2026, 8, 28, 15, 4, 5, 0, time.UTC)

func BenchmarkFormatRFC3339(b *testing.B) {
	for b.Loop() {
		strSink = t0.Format(time.RFC3339)
	}
}

func BenchmarkAppendFormat(b *testing.B) {
	buf := make([]byte, 0, 64)
	for b.Loop() {
		buf = t0.AppendFormat(buf[:0], time.RFC3339)
	}
	strSink = string(buf)
}

func BenchmarkParseRFC3339(b *testing.B) {
	s := "2026-08-28T15:04:05Z"
	for b.Loop() {
		timeSink, _ = time.Parse(time.RFC3339, s)
	}
}

func BenchmarkUnixVsFormat(b *testing.B) {
	for b.Loop() {
		intSink = t0.Unix()
	}
}

// ---------------------------------------------------------------------------
// ⑤ 比较
// ---------------------------------------------------------------------------

var t1 = t0.Add(time.Second)

func BenchmarkTimeEqual(b *testing.B) {
	for b.Loop() {
		boolSink = t0.Equal(t1)
	}
}

func BenchmarkTimeBefore(b *testing.B) {
	for b.Loop() {
		boolSink = t0.Before(t1)
	}
}

func BenchmarkUnixNanoCompare(b *testing.B) {
	a, c := t0.UnixNano(), t1.UnixNano()
	for b.Loop() {
		boolSink = a < c
	}
}
