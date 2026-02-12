package stream

import (
	"time"

	"github.com/nats-io/nats.go"
)

func Connect(url string) (*nats.Conn, error) {
	return nats.Connect(
		url,
		nats.Name("go-realtime-stream"),
		nats.Timeout(3*time.Second),
		nats.ReconnectWait(500*time.Millisecond),
		nats.MaxReconnects(-1),
	)
}

