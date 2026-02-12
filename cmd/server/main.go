package main

import (
	//"encoding/json"
	"flag"
	"log"
	"net/http"
	"sync"
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

func (h *Hub) broadcast(b []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.conns {
		_ = c.WriteMessage(websocket.TextMessage, b)
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

	// batching waves
	go func() {
		sub, _ := nc.SubscribeSync("ecg.wave")

		ticker := time.NewTicker(40 * time.Millisecond) // ~25 FPS
		defer ticker.Stop()

		var batch [][]byte

		for {
			msg, err := sub.NextMsg(1 * time.Millisecond)
			if err == nil {
				batch = append(batch, msg.Data)
			}

			select {
			case <-ticker.C:
				if len(batch) > 0 {
					// enviar array JSON
					combined := []byte("[")
					for i, b := range batch {
						combined = append(combined, b...)
						if i != len(batch)-1 {
							combined = append(combined, ',')
						}
					}
					combined = append(combined, ']')

					hub.broadcast(combined)
					batch = nil
				}
			default:
			}
		}
	}()

	// Subscribe params
	nc.Subscribe("ecg.params", func(msg *nats.Msg) {
		hub.broadcast(msg.Data)
	})

	http.Handle("/", http.FileServer(http.Dir("./web")))

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.add(conn)
		defer hub.remove(conn)

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})

	log.Println("server running on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
