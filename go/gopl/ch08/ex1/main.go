package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

var port = flag.String("port", "8000", "please input port")

func main() {
	flag.Parse()
	uri := fmt.Sprintf("%s:%s", "localhost", *port)
	linster, err := net.Listen("tcp", uri)
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

func handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		t := time.Now().Format("15:04:05\n")
		_, err := io.WriteString(conn, t+"")
		if err != nil {
			return
		}
		time.Sleep(1 * time.Second)
	}
}
