package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
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

// WaveMsg: solo necesitamos V (sample) para empaquetar binario.
// msg.Data viene como JSON: {"subject":"ecg.wave","ts":...,"seq":...,"fs":...,"v":...}
type WaveMsg struct {
	V float32 `json:"v"`
}

func main() {
	var (
		natsURL      = flag.String("nats", "nats://127.0.0.1:4222", "NATS url")
		addr         = flag.String("addr", ":8080", "http address")
		batchEveryMs = flag.Int("batch_ms", 40, "wave batch interval in ms (~25Hz default)")
		maxBatch     = flag.Int("max_batch", 200, "max samples per batch before drop")
	)
	flag.Parse()

	nc, err := stream.Connect(*natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	hub := newHub()

	// -------------------------
	// METRICS (atomic)
	// -------------------------
	var batchesSent int64
	var samplesSent int64
	var samplesDropped int64
	var clientsNow int64

	// -------------------------
	// WAVES: batching + binary WS
	// -------------------------
	go func() {
		sub, err := nc.SubscribeSync("ecg.wave")
		if err != nil {
			log.Printf("subscribe ecg.wave error: %v", err)
			return
		}

		ticker := time.NewTicker(time.Duration(*batchEveryMs) * time.Millisecond)
		defer ticker.Stop()

		batch := make([][]byte, 0, 64)

		for {
			// drenamos lo que haya disponible (rápido)
			for {
				msg, err := sub.NextMsg(500 * time.Microsecond)
				if err != nil {
					break
				}
				batch = append(batch, msg.Data)

				// defensa: si se acumula demasiado, dropeamos para no bloquear
				if len(batch) >= *maxBatch {
					atomic.AddInt64(&samplesDropped, int64(len(batch)))
					batch = batch[:0]
					break
				}
			}

			<-ticker.C

			if len(batch) == 0 {
				continue
			}

			// Empaquetar como Float32 LE: len(batch) samples
			out := make([]byte, 4*len(batch))

			okCount := 0
			for i, b := range batch {
				var w WaveMsg
				if err := json.Unmarshal(b, &w); err != nil {
					continue
				}
				binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(w.V))
				okCount++
			}

			// Nota: si hubo errores de unmarshal, okCount puede ser < len(batch).
			// Para simplificar, enviamos igual el buffer (con ceros donde falló).
			// (En este demo, no debería fallar.)
			hub.broadcastBinary(out)

			atomic.AddInt64(&batchesSent, 1)
			atomic.AddInt64(&samplesSent, int64(len(batch)))

			batch = batch[:0]
		}
	}()

	// -------------------------
	// PARAMS: texto JSON directo
	// -------------------------
	_, err = nc.Subscribe("ecg.params", func(msg *nats.Msg) {
		hub.broadcastText(msg.Data)
	})
	if err != nil {
		log.Fatal(err)
	}

	// -------------------------
	// HTTP: UI + WS + metrics
	// -------------------------
	http.Handle("/", http.FileServer(http.Dir("./web")))

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "clients %d\n", atomic.LoadInt64(&clientsNow))
		fmt.Fprintf(w, "batches_sent %d\n", atomic.LoadInt64(&batchesSent))
		fmt.Fprintf(w, "samples_sent %d\n", atomic.LoadInt64(&samplesSent))
		fmt.Fprintf(w, "samples_dropped %d\n", atomic.LoadInt64(&samplesDropped))
		fmt.Fprintf(w, "batch_ms %d\n", *batchEveryMs)
		fmt.Fprintf(w, "max_batch %d\n", *maxBatch)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.add(conn)
		atomic.AddInt64(&clientsNow, 1)

		defer func() {
			hub.remove(conn)
			atomic.AddInt64(&clientsNow, -1)
			_ = conn.Close()
		}()

		// Mantenemos viva la conexión leyendo (aunque no usemos mensajes del cliente)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})

	// -------------------------
	// Graceful shutdown
	// -------------------------
	srv := &http.Server{Addr: *addr}

	go func() {
		log.Println("server running on", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	ch := make(chan os.Signal, 1)
	osSignal.Notify(ch, os.Interrupt)

	<-ch
	log.Println("server: shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctx)
	_ = nc.Drain()
	log.Println("server: bye")
}
