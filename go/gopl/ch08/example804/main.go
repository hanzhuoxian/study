package main

// **练习 8.4：** 修改reverb2服务器，在每一个连接中使用sync.WaitGroup来计数活跃的echo goroutine。
// 当计数减为零时，关闭TCP连接的写入，像练习8.3中一样。验证一下你的修改版netcat3客户端会一直等待所有的并发“喊叫”完成，
// 即使是在标准输入流已经关闭的情况下。
import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

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
	input := bufio.NewScanner(c)
	for input.Scan() {
		wg.Add(1)
		go func(shout string) {
			echo(c, shout, 1*time.Second)
			wg.Done()
		}(input.Text())

	}

	// 等所有 echo goroutine 写完，再关闭写方向：
	// 提前关会让它们的写入全部失败，客户端就收不到剩下的喊叫。
	wg.Wait()

	if hc, ok := c.(halfCloser); ok {
		if err := hc.CloseWrite(); err != nil {
			log.Print(err)
		}
	}

	if err := input.Err(); err != nil {
		fmt.Printf("%v", err)
	}
}
