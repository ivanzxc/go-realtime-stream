package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"log"
	"math"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/ivanzxc/go-realtime-stream/internal/analysis"
	"github.com/ivanzxc/go-realtime-stream/internal/stream"
)

type ParamMsg struct {
	Subject string `json:"subject"`
	Ts      int64  `json:"ts"`
	HR      int    `json:"hr"`
}

func main() {

	var (
		natsURL = flag.String("nats", "nats://127.0.0.1:4222", "NATS url")
		in      = flag.String("in", "ecg.wave", "input subject")
		out     = flag.String("out", "ecg.params", "output subject")
	)
	flag.Parse()

	nc, err := stream.Connect(*natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	detector := analysis.NewHRDetector()

	_, err = nc.Subscribe(*in, func(msg *nats.Msg) {

		samples := len(msg.Data) / 4

		for i := 0; i < samples; i++ {

			bits := binary.LittleEndian.Uint32(msg.Data[i*4:])
			v := math.Float32frombits(bits)

			ts := time.Now()

			if bpm, ok := detector.Process(v, ts); ok {

				param := ParamMsg{
					Subject: *out,
					Ts:      time.Now().UnixMilli(),
					HR:      bpm,
				}

				b, _ := json.Marshal(param)
				nc.Publish(*out, b)

				log.Printf("HR detected: %d BPM", bpm)
			}
		}
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("processor running...")
	select {}
}
