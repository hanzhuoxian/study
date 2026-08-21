package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
)

// **练习 8.3：** 在netcat3例子中，conn虽然是一个interface类型的值，但是其底层真实类型是`*net.TCPConn`，代表一个TCP连接。一个TCP连接有读和写两个部分，可以使用CloseRead和CloseWrite方法分别关闭它们。修改netcat3的主goroutine代码，只关闭网络连接中写的部分，这样的话后台goroutine可以在标准输入被关闭后继续打印从reverb1服务器传回的数据。（要在reverb2服务器也完成同样的功能是比较困难的；参考**练习 8.4**。）

var port = flag.String("port", "8000", "please input port")

func main() {
	flag.Parse()
	uri := net.JoinHostPort("localhost", *port)
	conn, err := net.Dial("tcp", uri)
	if err != nil {
		log.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		mustCopy(os.Stdout, conn)
		log.Println("done")
		done <- struct{}{}
	}()
	mustCopy(conn, os.Stdin)
	tcp, ok := conn.(*net.TCPConn)
	if !ok {
		log.Fatalf("not a TCP connection: %T", conn)
	}
	// 只关闭写方向：服务端会读到 EOF，而后台 goroutine 仍能继续接收回声。
	if err := tcp.CloseWrite(); err != nil {
		log.Fatal(err)
	}
	<-done

	conn.Close()
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}
