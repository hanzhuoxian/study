package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second
	// 连接建立或收到 Pong 后，最多等待 60 秒。
	pongWait = 60 * time.Second
	// Ping 间隔必须小于 pongWait，给客户端留出回复时间。
	pingPeriod = 30 * time.Second
)

// ChatRequest 是客户端发送的消息，发送者和时间由服务端补充。
type ChatRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Message 是服务端广播的消息。当前只支持 type="chat"。
type Message struct {
	Type      string    `json:"type"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Client struct {
	ID   string
	hub  *Hub
	conn *websocket.Conn
	send chan Message
}

type Hub struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
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
		fmt.Printf("Client %p -> Pong\n", c)
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		if messageType != websocket.TextMessage {
			continue
		}
		var request ChatRequest
		if err := json.Unmarshal(data, &request); err != nil {
			log.Printf("Client %s 消息不是有效 JSON: %v", c.ID, err)
			continue
		}
		if request.Type != "chat" || strings.TrimSpace(request.Content) == "" {
			log.Printf("Client %s 消息无效: type 必须为 chat，content 不能为空", c.ID)
			continue
		}

		c.hub.broadcast <- Message{
			Type:      "chat",
			Sender:    c.ID,
			Content:   fmt.Sprintf("server say: %s", request.Content),
			Timestamp: time.Now().UTC(),
		}
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
			if err := c.conn.WriteJSON(message); err != nil {
				return
			}
		case <-ticker.C:
			fmt.Printf("ping -> Client %p\n", c)
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
		broadcast:  make(chan Message, 256),
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
		// 每条连接分配一个 ID；它不代表已认证的用户身份。
		ID:   rand.Text(),
		hub:  hub,
		conn: conn,
		send: make(chan Message, 256),
	}

	hub.register <- client

	fmt.Printf("Client %p connected\n", client)

	go client.readPump()
	go client.writePump()
}
