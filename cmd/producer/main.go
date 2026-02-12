package main

import (
	"context"
	"encoding/binary"
	"flag"
	"log"
	"math"
	"os"
	osSignal "os/signal"
	"time"

	"github.com/ivanzxc/go-realtime-stream/internal/signal"
	"github.com/ivanzxc/go-realtime-stream/internal/stream"
)

func main() {

	var (
		natsURL = flag.String("nats", "nats://127.0.0.1:4222", "NATS url")
		subject = flag.String("subject", "ecg.wave", "subject")
		fs      = flag.Int("fs", 250, "sampling rate Hz")
		hr      = flag.Float64("hr", 72, "heart rate bpm")
		batch   = flag.Int("batch", 10, "samples per message")
	)
	flag.Parse()

	nc, err := stream.Connect(*natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	sim := signal.NewECGSim(float64(*fs), *hr, 0.02)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	osSignal.Notify(ch, os.Interrupt)

	go func() {
		<-ch
		cancel()
	}()

	period := time.Second / time.Duration(*fs)
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	buffer := make([]float32, 0, *batch)

	for {
		select {
		case <-ctx.Done():
			log.Println("producer: stopping")
			return

		case <-ticker.C:
			buffer = append(buffer, sim.Next())

			if len(buffer) >= *batch {

				out := make([]byte, 4*len(buffer))

				for i, v := range buffer {
					binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
				}

				nc.Publish(*subject, out)
				buffer = buffer[:0]
			}
		}
	}
}
