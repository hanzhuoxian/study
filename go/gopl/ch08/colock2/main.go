package main

import (
	"fmt"
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

		go handleConn(conn)

	}
}

func handleConn(conn net.Conn) {
	fmt.Printf("\naccept %v\n", conn.RemoteAddr())
	defer func() {
		conn.Close()
		fmt.Printf("\nhandle done %v\n", conn.RemoteAddr())
	}()
	for {
		t := time.Now().Format("15:04:05\n")
		_, err := io.WriteString(conn, t+"")
		if err != nil {
			return
		}
		time.Sleep(1 * time.Second)
	}

}
