package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second
	// 连接建立或收到 Pong 后，最多等待 60 秒。
	pongWait = 3 * time.Second
	// Ping 间隔必须小于 pongWait，给客户端留出回复时间。
	pingPeriod = 1 * time.Second
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				fmt.Printf("Client %p discontected\n", client)
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// 发送队列已满，移除慢客户端，避免阻塞整个 Hub。
					delete(h.clients, client)
					close(client.send)
				}
			}

		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
		c.hub.unregister <- c
	}()

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	// ReadMessage 会处理 Pong 控制帧，并调用此回调续期读超时。
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		message = []byte(fmt.Sprintf("Client %p : %s\n", c, string(message)))
		c.hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			// Ping 和聊天消息由同一个 goroutine 写入，避免并发写连接。
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// upgrader 用于将 HTTP 连接升级为 WebSocket 连接
var upgrader = websocket.Upgrader{
	// 为了简化示例，允许所有来源的连接。在生产环境中，请谨慎配置。
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {

	hub := &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}

	go hub.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Println("server start: 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("server error: ", err)
	}
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	hub.register <- client

	fmt.Printf("Client %p connected\n", client)

	go client.readPump()
	go client.writePump()
}
