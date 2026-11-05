// Package main exposes the WebSocket gateway for the role-play chatbot.
// A separate gRPC backend streams model tokens; Redis Pub/Sub distributes events
// between gateway instances.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

const (
	writeWait     = 10 * time.Second
	pongWait      = 60 * time.Second
	pingPeriod    = 30 * time.Second
	roomQueueSize = 128
)

type ClientMessage struct {
	Type        string `json:"type"` // chat | join
	RoomID      string `json:"room_id,omitempty"`
	CharacterID string `json:"character_id,omitempty"`
	Content     string `json:"content,omitempty"`
}

type Event struct {
	Type        string    `json:"type"` // token | done | error | chat
	RoomID      string    `json:"room_id"`
	CharacterID string    `json:"character_id,omitempty"`
	Sender      string    `json:"sender,omitempty"`
	Content     string    `json:"content,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

type ModelRequest struct{ RoomID, CharacterID, UserID, Content string }
type ModelToken struct{ Content string }

// ModelClient is implemented by the generated gRPC client adapter.
type ModelClient interface {
	Stream(context.Context, ModelRequest, func(ModelToken) error) error
}

type Publisher interface {
	Publish(context.Context, string, Event) error
}

type RedisBus struct{ client *redis.Client }

func NewRedisBus(addr, password string, db int) *RedisBus {
	return &RedisBus{client: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})}
}
func (b *RedisBus) Publish(ctx context.Context, channel string, e Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, channel, data).Err()
}
func (b *RedisBus) Subscribe(ctx context.Context, channel string, handle func(Event)) error {
	pubsub := b.client.Subscribe(ctx, channel)
	defer pubsub.Close()
	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}
	for message := range pubsub.Channel() {
		var e Event
		if err := json.Unmarshal([]byte(message.Payload), &e); err != nil {
			log.Printf("invalid Redis event: %v", err)
			continue
		}
		handle(e)
	}
	return ctx.Err()
}
func (b *RedisBus) Close() error { return b.client.Close() }

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
	model ModelClient
	bus   Publisher
}

func NewHub(model ModelClient, bus Publisher) *Hub {
	return &Hub{rooms: make(map[string]map[*Client]struct{}), model: model, bus: bus}
}
func (h *Hub) Add(c *Client, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.room != "" {
		delete(h.rooms[c.room], c)
		if len(h.rooms[c.room]) == 0 {
			delete(h.rooms, c.room)
		}
	}
	c.room = room
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]struct{})
	}
	h.rooms[room][c] = struct{}{}
}
func (h *Hub) Remove(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room := c.room; room != "" {
		delete(h.rooms[room], c)
		if len(h.rooms[room]) == 0 {
			delete(h.rooms, room)
		}
		c.room = ""
	}
}
func (h *Hub) Broadcast(e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[e.RoomID] {
		select {
		case c.send <- e:
		default:
			go c.close()
		}
	}
}

type Client struct {
	id          string
	userID      string
	room        string
	characterID string
	conn        *websocket.Conn
	hub         *Hub
	send        chan Event
	closed      sync.Once
}

func (c *Client) close() { c.closed.Do(func() { c.hub.Remove(c); close(c.send); _ = c.conn.Close() }) }
func (c *Client) readPump() {
	defer c.close()
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { return c.conn.SetReadDeadline(time.Now().Add(pongWait)) })
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var req ClientMessage
		if err := json.Unmarshal(data, &req); err != nil {
			c.sendError("invalid JSON")
			continue
		}
		switch req.Type {
		case "join":
			if strings.TrimSpace(req.RoomID) == "" || strings.TrimSpace(req.CharacterID) == "" {
				c.sendError("room_id and character_id are required")
				continue
			}
			c.hub.Add(c, strings.TrimSpace(req.RoomID))
			c.characterID = strings.TrimSpace(req.CharacterID)
			c.emit(Event{Type: "joined", RoomID: c.room, CharacterID: c.characterID, Sender: c.id})
		case "chat":
			if c.room == "" {
				c.sendError("join a room before chatting")
				continue
			}
			if strings.TrimSpace(req.Content) == "" {
				c.sendError("content is required")
				continue
			}
			c.chat(req.Content)
		default:
			c.sendError("type must be join or chat")
		}
	}
}
func (c *Client) chat(content string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	req := ModelRequest{RoomID: c.room, CharacterID: c.characterID, UserID: c.userID, Content: content}
	err := c.hub.model.Stream(ctx, req, func(token ModelToken) error {
		if token.Content == "" {
			return nil
		}
		return c.emit(Event{Type: "token", RoomID: c.room, CharacterID: c.characterID, Sender: "assistant", Content: token.Content, Timestamp: time.Now().UTC()})
	})
	if err != nil {
		log.Printf("model stream failed room=%s: %v", c.room, err)
		c.sendError("model stream failed")
		return
	}
	_ = c.emit(Event{Type: "done", RoomID: c.room, CharacterID: c.characterID, Sender: "assistant", Timestamp: time.Now().UTC()})
}
func (c *Client) emit(e Event) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	if c.hub.bus != nil {
		if err := c.hub.bus.Publish(context.Background(), "chat:room:"+e.RoomID, e); err != nil {
			log.Printf("redis publish failed: %v", err)
			c.hub.Broadcast(e)
		}
		return nil
	}
	c.hub.Broadcast(e)
	return nil
}
func (c *Client) sendError(message string) {
	_ = c.emit(Event{Type: "error", RoomID: c.room, Content: message, Sender: "system"})
}
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() { ticker.Stop(); c.close() }()
	for {
		select {
		case e, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteJSON(e); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type MockModel struct{}

func (MockModel) Stream(ctx context.Context, req ModelRequest, send func(ModelToken) error) error {
	name := req.CharacterID
	answer := fmt.Sprintf("（%s）我听到了：%s", name, req.Content)
	for _, r := range []rune(answer) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := send(ModelToken{Content: string(r)}); err != nil {
			return err
		}
		time.Sleep(35 * time.Millisecond)
	}
	return nil
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func main() {
	addr := env("CHATBOT_ADDR", ":8080")
	var bus *RedisBus
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		bus = NewRedisBus(redisAddr, os.Getenv("REDIS_PASSWORD"), 0)
		defer bus.Close()
		ctx := context.Background()
		if err := bus.client.Ping(ctx).Err(); err != nil {
			log.Fatalf("redis unavailable: %v", err)
		}
	}
	hub := NewHub(MockModel{}, bus)
	if bus != nil {
		go func() {
			for {
				err := bus.Subscribe(context.Background(), "chat:room:*", hub.Broadcast)
				if err != nil && !errors.Is(err, context.Canceled) {
					log.Printf("redis subscribe: %v", err)
					time.Sleep(time.Second)
				}
			}
		}()
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade: %v", err)
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = "guest"
		}
		c := &Client{id: fmt.Sprintf("%d", time.Now().UnixNano()), userID: userID, conn: conn, hub: hub, send: make(chan Event, roomQueueSize)}
		go c.writePump()
		c.readPump()
	})
	log.Printf("chatbot gateway listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
