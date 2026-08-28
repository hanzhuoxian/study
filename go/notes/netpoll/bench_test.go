package main

import (
	"bufio"
	"io"
	"net"
	"testing"
	"time"
)

// echoPair 起一个 echo 服务并返回一条已建立的客户端连接。
func echoPair(tb testing.TB) net.Conn {
	tb.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		c, err := ln.Accept()
		if err != nil {
			return
		}
		io.Copy(c, c)
		c.Close()
	}()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		ln.Close()
		tb.Fatal(err)
	}
	if tc, ok := c.(*net.TCPConn); ok {
		tc.SetNoDelay(true) // 否则 Nagle 会把往返延迟拉到 40ms
	}
	tb.Cleanup(func() {
		c.Close()
		ln.Close()
		<-done
	})
	return c
}

// sinkConn 起一个只读不回的服务，用来单独测发送侧的开销。
func sinkConn(tb testing.TB) net.Conn {
	tb.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		c, err := ln.Accept()
		if err != nil {
			return
		}
		io.Copy(io.Discard, c)
		c.Close()
	}()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		ln.Close()
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		c.Close()
		ln.Close()
		<-done
	})
	return c
}

var payload = []byte("0123456789abcdef")

func roundTrip(b *testing.B, c net.Conn, buf []byte) {
	if _, err := c.Write(payload); err != nil {
		b.Fatal(err)
	}
	if _, err := io.ReadFull(c, buf); err != nil {
		b.Fatal(err)
	}
}

// 基线：一次 write + 一次 read 的本机往返。
func BenchmarkRoundTrip(b *testing.B) {
	c := echoPair(b)
	buf := make([]byte, len(payload))
	b.ResetTimer()
	for b.Loop() {
		roundTrip(b, c, buf)
	}
}

// 每次往返前设一次 SetDeadline：rd==wd 走 combo 路径，只挂一个 runtime timer。
func BenchmarkRoundTripSetDeadline(b *testing.B) {
	c := echoPair(b)
	buf := make([]byte, len(payload))
	b.ResetTimer()
	for b.Loop() {
		c.SetDeadline(time.Now().Add(time.Second))
		roundTrip(b, c, buf)
	}
}

// 分开设读写 deadline：两次 poll_runtime_pollSetDeadline 调用。
func BenchmarkRoundTripSetRWDeadline(b *testing.B) {
	c := echoPair(b)
	buf := make([]byte, len(payload))
	b.ResetTimer()
	for b.Loop() {
		now := time.Now()
		c.SetReadDeadline(now.Add(time.Second))
		c.SetWriteDeadline(now.Add(time.Second))
		roundTrip(b, c, buf)
	}
}

// 裸 Read：每次 16 字节都是一次 syscall。
func BenchmarkReadRaw(b *testing.B) {
	c := echoPair(b)
	go func() {
		big := make([]byte, 4096)
		for {
			if _, err := c.Write(big); err != nil {
				return
			}
		}
	}()
	buf := make([]byte, 16)
	b.SetBytes(int64(len(buf)))
	b.ResetTimer()
	for b.Loop() {
		if _, err := io.ReadFull(c, buf); err != nil {
			b.Fatal(err)
		}
	}
}

// bufio 包一层：syscall 次数降到 1/256，但 deadline 语义要自己管（见 4.2）。
func BenchmarkReadBufio(b *testing.B) {
	c := echoPair(b)
	go func() {
		big := make([]byte, 4096)
		for {
			if _, err := c.Write(big); err != nil {
				return
			}
		}
	}()
	r := bufio.NewReaderSize(c, 4096)
	buf := make([]byte, 16)
	b.SetBytes(int64(len(buf)))
	b.ResetTimer()
	for b.Loop() {
		if _, err := io.ReadFull(r, buf); err != nil {
			b.Fatal(err)
		}
	}
}

var head, body = []byte("HEADER|"), []byte("bodybodybodybody")

// 两次 Write = 两次 syscall。
func BenchmarkWriteTwice(b *testing.B) {
	c := sinkConn(b)
	b.SetBytes(int64(len(head) + len(body)))
	b.ResetTimer()
	for b.Loop() {
		if _, err := c.Write(head); err != nil {
			b.Fatal(err)
		}
		if _, err := c.Write(body); err != nil {
			b.Fatal(err)
		}
	}
}

// 先在用户态拼好再发：一次 syscall，但多一次拷贝 + 可能的分配。
func BenchmarkWriteConcat(b *testing.B) {
	c := sinkConn(b)
	buf := make([]byte, 0, len(head)+len(body))
	b.SetBytes(int64(len(head) + len(body)))
	b.ResetTimer()
	for b.Loop() {
		buf = append(append(buf[:0], head...), body...)
		if _, err := c.Write(buf); err != nil {
			b.Fatal(err)
		}
	}
}

// net.Buffers 走 writev：一次 syscall 发多段，不用拷贝。
func BenchmarkWritev(b *testing.B) {
	c := sinkConn(b)
	b.SetBytes(int64(len(head) + len(body)))
	b.ResetTimer()
	for b.Loop() {
		bufs := net.Buffers{head, body}
		if _, err := bufs.WriteTo(c); err != nil {
			b.Fatal(err)
		}
	}
}

// deadline 本身的成本：不做 IO，只反复设置。
// SetDeadline 一次调用同时写 rd 和 wd，rd==wd 时走 combo 路径只挂一个 runtime timer。
func BenchmarkSetDeadline(b *testing.B) {
	c := sinkConn(b)
	b.ResetTimer()
	for b.Loop() {
		c.SetDeadline(time.Now().Add(time.Second))
	}
}

// 分开设：两次 poll_runtime_pollSetDeadline，两个 timer 状态要维护。
func BenchmarkSetRWDeadline(b *testing.B) {
	c := sinkConn(b)
	b.ResetTimer()
	for b.Loop() {
		now := time.Now()
		c.SetReadDeadline(now.Add(time.Second))
		c.SetWriteDeadline(now.Add(time.Second))
	}
}
