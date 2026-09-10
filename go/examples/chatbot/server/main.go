package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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
	defer c.conn.Close()

	for {
		message, ok := <-c.send
		if !ok {
			return
		}
		if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return
		}
		err := c.conn.WriteMessage(
			websocket.TextMessage,
			message,
		)
		if err != nil {
			return
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
