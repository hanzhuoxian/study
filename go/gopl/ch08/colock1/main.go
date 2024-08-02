package main

import (
	"io"
	"log"
	"net"
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

		handleConn(conn)

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
