package main

import (
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/ivanzxc/go-realtime-stream/internal/analysis"
	"github.com/ivanzxc/go-realtime-stream/internal/stream"
)

type WaveMsg struct {
	Ts int64   `json:"ts"`
	V  float32 `json:"v"`
}

type ParamMsg struct {
	Subject string `json:"subject"`
	Ts      int64  `json:"ts"`
	HR      int    `json:"hr"`
}

func main() {

	var (
		natsURL = flag.String("nats", "nats://127.0.0.1:4222", "NATS url")
		subject = flag.String("subject", "ecg.wave", "input subject")
		out     = flag.String("out", "ecg.params", "output subject")
	)
	flag.Parse()

	nc, err := stream.Connect(*natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	detector := analysis.NewHRDetector()

	_, err = nc.Subscribe(*subject, func(msg *nats.Msg) {

		var wave WaveMsg
		if err := json.Unmarshal(msg.Data, &wave); err != nil {
			return
		}

		ts := time.UnixMilli(wave.Ts)

		if bpm, ok := detector.Process(wave.V, ts); ok {

			param := ParamMsg{
				Subject: *out,
				Ts:      time.Now().UnixMilli(),
				HR:      bpm,
			}

			b, _ := json.Marshal(param)
			nc.Publish(*out, b)

			log.Printf("HR detected: %d BPM", bpm)
		}

	})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("processor running...")
	select {}
}

