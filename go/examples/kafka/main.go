package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {
	// topic
	topic := "new-user"
	partition := 0

	conn, err := kafka.DialLeader(context.Background(), "tcp", "127.0.0.1:9092", topic, partition)
	if err != nil {
		log.Fatal("failed to dial leader:", err)
	}
	producer(conn)
	consumer(conn)
}

func consumer(conn *kafka.Conn) {
	// make a new reader that consumes from topic-A, partition 0, at offset 42
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"127.0.0.1:9092"},
		Topic:     "new-user",
		Partition: 0,
		GroupID:   "consumer",
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
	})

	// r.SetOffset(0)

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}
		fmt.Printf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))
	}

	if err := r.Close(); err != nil {
		log.Fatal("failed to close reader:", err)
	}
}

func producer(conn *kafka.Conn) {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := conn.WriteMessages(
		kafka.Message{Value: []byte("Liming")},
		kafka.Message{Value: []byte("Hanmeimei")},
		kafka.Message{Value: []byte("Lihua")},
	)
	if err != nil {
		log.Fatal("failed to write messages:", err)
	}

	if err := conn.Close(); err != nil {
		log.Fatal("failed to close writer:", err)
	}
}
