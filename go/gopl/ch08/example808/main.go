package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// **练习 8.8：** 使用select来改造8.3节中的echo服务器，为其增加超时，
// 这样服务器可以在客户端10秒中没有任何喊话时自动断开连接。

const idleTimeout = 10 * time.Second

func main() {
	linster, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := linster.Accept()
		if err != nil {
			log.Print(err)
			continue
		}

		go handleConn(conn)

	}
}

// halfCloser 由支持半关闭的连接实现，例如 *net.TCPConn 和 *net.UnixConn。
type halfCloser interface {
	CloseWrite() error
}

func echo(c net.Conn, shout string, delay time.Duration) {
	fmt.Fprintln(c, "\t", strings.ToUpper(shout))
	time.Sleep(delay)
	fmt.Fprintln(c, "\t", shout)
	time.Sleep(delay)
	fmt.Fprintln(c, "\t", strings.ToLower(shout))
	time.Sleep(delay)
}

func handleConn(c net.Conn) {
	var wg sync.WaitGroup

	fmt.Printf("conn start: %v\n", c.RemoteAddr())
	defer func() {
		fmt.Printf("conn close: %v\n", c.RemoteAddr())
		c.Close()
	}()

	// 读取必须放在独立的 goroutine 里：Scan 会阻塞，
	// 主 goroutine 要腾出来跑 select，才能同时等超时和等喊话。
	shout := make(chan string)
	// done 用于在主 goroutine 退出时唤醒可能卡在 shout <- 上的读取 goroutine，
	// 避免 goroutine 泄漏。
	done := make(chan struct{})
	defer close(done)

	go func() {
		input := bufio.NewScanner(c)
		for input.Scan() {
			select {
			case shout <- input.Text():
			case <-done:
				return
			}
		}
		if err := input.Err(); err != nil {
			// 主 goroutine 退出时会 Close 连接，此时 Read 报的
			// "use of closed network connection" 不是真错误，忽略即可。
			select {
			case <-done:
			default:
				log.Print(err)
			}
		}
		close(shout) // 客户端关闭了写方向（EOF）
	}()

	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()

loop:
	for {
		select {
		case s, ok := <-shout:
			if !ok { // 客户端主动结束，正常退出
				break loop
			}
			// 每收到一次喊话就重置空闲计时器
			if !timer.Stop() {
				// 计时器已触发但尚未被读取，先排空再重置
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(idleTimeout)

			wg.Add(1)
			go func(shout string) {
				defer wg.Done()
				echo(c, shout, 1*time.Second)
			}(s)
		case <-timer.C:
			fmt.Printf("conn idle timeout: %v\n", c.RemoteAddr())
			break loop
		}
	}

	// 等所有 echo goroutine 写完，再关闭写方向：
	// 提前关会让它们的写入全部失败，客户端就收不到剩下的喊叫。
	wg.Wait()

	if hc, ok := c.(halfCloser); ok {
		if err := hc.CloseWrite(); err != nil {
			log.Print(err)
		}
	}
}
