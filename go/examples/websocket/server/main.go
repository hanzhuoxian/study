package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader 用于将 HTTP 连接升级为 WebSocket 连接
var upgrader = websocket.Upgrader{
	// 为了简化示例，允许所有来源的连接。在生产环境中，请谨慎配置。
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	// 注册处理 WebSocket 连接的路由
	http.HandleFunc("/ws", handleWebSocket)

	log.Println("WebSocket 服务器已启动，监听端口 :8080")
	// 启动 HTTP 服务器
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}

// handleWebSocket 处理单个 WebSocket 连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 1. 升级 HTTP 连接为 WebSocket 连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("升级连接失败:", err)
		return
	}
	// 确保连接在函数结束时关闭
	defer conn.Close()

	log.Println("客户端已连接")

	// 2. 在一个循环中持续读取和写入消息
	for {
		// 读取来自客户端的消息
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			// 如果读取出错（例如客户端断开连接），则退出循环
			log.Println("读取消息失败:", err)
			break
		}
		log.Printf("收到消息: %s", message)

		// 将收到的消息原样写回给客户端（回声）
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			log.Println("写入消息失败:", err)
			break
		}
	}
}
