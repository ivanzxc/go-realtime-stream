package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	osSignal "os/signal"
	"time"

	"github.com/nats-io/nats.go"

	sim "github.com/ivanzxc/go-realtime-stream/internal/signal"
	"github.com/ivanzxc/go-realtime-stream/internal/stream"
)

type WaveMsg struct {
	Subject string  `json:"subject"`
	Ts      int64   `json:"ts"`     // unix ms
	Seq     uint64  `json:"seq"`
	Fs      int     `json:"fs"`     // Hz
	V       float32 `json:"v"`      // sample
}

func main() {
	var (
		natsURL = flag.String("nats", "nats://127.0.0.1:4222", "NATS url")
		subject = flag.String("subject", "ecg.wave", "subject")
		fs      = flag.Int("fs", 250, "sampling rate Hz")
		hr      = flag.Float64("hr", 72, "heart rate bpm")
		noise   = flag.Float64("noise", 0.02, "noise 0..")
	)
	flag.Parse()

	nc, err := stream.Connect(*natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	log.Printf("producer: connected %s subject=%s fs=%d hr=%.1f", nc.ConnectedUrl(), *subject, *fs, *hr)

	ecg := sim.NewECGSim(float64(*fs), *hr, *noise)

	ctx, stop := signalNotifyContext()
	defer stop()

	period := time.Second / time.Duration(*fs)
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	var seq uint64
	var sent uint64
	last := time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Println("producer: stopping")
			return
		case <-ticker.C:
			seq++
			msg := WaveMsg{
				Subject: *subject,
				Ts:      time.Now().UnixMilli(),
				Seq:     seq,
				Fs:      *fs,
				V:       ecg.Next(),
			}
			b, _ := json.Marshal(msg)
			if err := nc.Publish(*subject, b); err != nil {
				log.Printf("publish error: %v", err)
				continue
			}
			sent++

			// log throughput cada ~2s
			if time.Since(last) >= 2*time.Second {
				rate := float64(sent) / time.Since(last).Seconds()
				log.Printf("producer: ~%.1f msg/s", rate)
				sent = 0
				last = time.Now()
			}
		}
	}
}

func signalNotifyContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	osSignal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		cancel()
	}()
	return ctx, cancel
}

// (Opcional) ejemplo de publish sync si algún día querés ack/flush:
func flush(nc *nats.Conn) {
	_ = nc.FlushTimeout(500 * time.Millisecond)
}

