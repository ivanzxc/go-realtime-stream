# ---------- build stage ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o producer ./cmd/producer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o processor ./cmd/processor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

# ---------- runtime stage ----------
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/producer .
COPY --from=builder /app/processor .
COPY --from=builder /app/server .
COPY web ./web

EXPOSE 8080

CMD ["./server"]
