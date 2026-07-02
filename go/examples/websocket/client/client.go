package main

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// 1. 连接到 WebSocket 服务器
	log.Println("正在连接至 ws://localhost:8080/ws")
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer conn.Close()
	log.Println("连接成功！")

	// 2. 启动一个 goroutine 来接收服务器发来的消息
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("接收消息失败:", err)
				return
			}
			log.Printf("收到回声: %s", message)
		}
	}()

	// 3. 每隔一秒向服务器发送一条消息
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		msg := "Hello, WebSocket! " + time.Now().Format("15:04:05")
		log.Printf("发送消息: %s", msg)
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			log.Println("发送消息失败:", err)
			return
		}
	}
}
