run:
	nats-server -js & \
	sleep 1 && \
	go run ./cmd/producer & \
	go run ./cmd/processor & \
	go run ./cmd/server

producer:
	go run ./cmd/producer

processor:
	go run ./cmd/processor

server:
	go run ./cmd/server

clean:
	pkill -f producer || true
	pkill -f processor || true
	pkill -f server || true
	pkill -f nats-server || true
