package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	osSignal "os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"

	"github.com/ivanzxc/go-realtime-stream/internal/stream"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]bool
}

func newHub() *Hub {
	return &Hub{conns: make(map[*websocket.Conn]bool)}
}

func (h *Hub) add(c *websocket.Conn) {
	h.mu.Lock()
	h.conns[c] = true
	h.mu.Unlock()
}

func (h *Hub) remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.conns, c)
	h.mu.Unlock()
}

func (h *Hub) snapshot() []*websocket.Conn {
	h.mu.Lock()
	clients := make([]*websocket.Conn, 0, len(h.conns))
	for c := range h.conns {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	return clients
}

func (h *Hub) broadcastBinary(b []byte) {
	clients := h.snapshot()
	for _, c := range clients {
		_ = c.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
		if err := c.WriteMessage(websocket.BinaryMessage, b); err != nil {
			_ = c.Close()
			h.remove(c)
		}
	}
}

func (h *Hub) broadcastText(b []byte) {
	clients := h.snapshot()
	for _, c := range clients {
		_ = c.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
		if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
			_ = c.Close()
			h.remove(c)
		}
	}
}

func main() {

	var (
		natsURL = flag.String("nats", "nats://127.0.0.1:4222", "NATS url")
		addr    = flag.String("addr", ":8080", "http address")
	)
	flag.Parse()

	nc, err := stream.Connect(*natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	hub := newHub()

	var totalMessages int64

	// Waves (binario passthrough)
	go func() {
		sub, _ := nc.SubscribeSync("ecg.wave")

		for {
			msg, err := sub.NextMsg(time.Second)
			if err != nil {
				continue
			}
			atomic.AddInt64(&totalMessages, 1)
			hub.broadcastBinary(msg.Data)
		}
	}()

	// Params (JSON)
	nc.Subscribe("ecg.params", func(msg *nats.Msg) {
		hub.broadcastText(msg.Data)
	})

	http.Handle("/", http.FileServer(http.Dir("./web")))

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "messages %d\n", atomic.LoadInt64(&totalMessages))
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.add(conn)
		defer func() {
			hub.remove(conn)
			conn.Close()
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})

	server := &http.Server{Addr: *addr}

	go func() {
		log.Println("server running on", *addr)
		server.ListenAndServe()
	}()

	ch := make(chan os.Signal, 1)
	osSignal.Notify(ch, os.Interrupt)
	<-ch

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Shutdown(ctx)
	log.Println("server stopped")
}
