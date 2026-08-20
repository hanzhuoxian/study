package main

// 练习 8.2： 实现一个并发FTP服务器。服务器应该解析客户端发来的一些命令，比如cd命令来切换目录，ls来列出目录内文件，get和send来传输文件，close来关闭连接。
// 你可以用标准的ftp命令来作为客户端，或者也可以自己实现一个。
//
// 这里实现的是 RFC 959 的一个子集：控制连接上收命令，数据传输走 PASV 模式另开的数据连接。
// 支持的命令：USER PASS PWD CWD CDUP TYPE PASV LIST NLST RETR STOR SYST FEAT NOOP QUIT。

import (
	"flag"
	"fmt"
	"log"
	"net"
	"path/filepath"
)

var (
	port = flag.Int("port", 8000, "port")
	root = flag.String("root", ".", "服务的根目录")
)

func main() {
	flag.Parse()

	abs, err := filepath.Abs(*root)
	if err != nil {
		log.Fatal(err)
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ftp 服务已启动: %s，根目录 %s", l.Addr(), abs)
	serve(l, abs)
}

// serve 接受连接，每个控制连接交给一个独立的 goroutine。
// 会话状态全部私有（见 session），goroutine 之间不共享任何可变数据，所以这里一把锁都不需要。
func serve(l net.Listener, root string) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return // 监听器已关闭
		}
		go newSession(conn, root).serve()
	}
}
