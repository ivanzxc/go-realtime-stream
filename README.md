# Go Real-Time Streaming Architecture

Production-oriented real-time streaming architecture built in Go using NATS and binary WebSocket delivery.

This project demonstrates how to design a low-latency event-driven pipeline using a high-frequency workload (250 Hz signal stream) as a realistic streaming scenario.

The signal is biomedical-inspired, but the architecture is domain-agnostic and applicable to:

- IoT telemetry
- Industrial sensors
- Financial tick streams
- Edge computing workloads
- Real-time analytics pipelines

---

## 🧠 What This Project Demonstrates

- High-frequency streaming (250 Hz)
- Binary Float32 pipeline (no JSON overhead in waves)
- Event-driven decoupled services
- Real-time signal processing
- Horizontal scalability
- WebSocket binary delivery
- Backpressure-safe design
- Dockerized reproducible environment
- Graceful shutdown
- Basic observability

---

## 🏗 Architecture Overview

Producer (Float32 batch)
↓
NATS
↓
Processor (Peak detection + HR calculation)
↓
NATS
↓
WebSocket Server (binary passthrough)
↓
Browser (Float32Array rendering @ ~60 FPS)


---

## ⚙️ Services

### Producer

- Generates continuous waveform at 250 Hz
- Batches samples (Float32)
- Publishes binary payloads to `ecg.wave`

### Processor

- Subscribes to `ecg.wave`
- Reads binary Float32 batches
- Detects R-peaks (with refractory control)
- Publishes HR metrics to `ecg.params`

### Server

- Subscribes to NATS
- Pass-through binary WebSocket delivery
- Drops slow clients
- Exposes `/metrics`
- Graceful shutdown

### Browser UI

- Uses `Float32Array`
- Incremental canvas rendering
- No JSON parsing for wave data
- Smooth scrolling visualization

---

## 📊 Benchmarks (Local M4 ARM)

| Metric | Result |
|--------|--------|
| Producer throughput | ~250 samples/sec |
| WebSocket batch size | 10 samples |
| CPU usage | ~2–4% |
| End-to-end latency | < 15 ms (local) |
| JSON overhead in waves | 0 |

---

## 🔄 Backpressure Strategy

- Producer batches samples before publishing
- Server does binary passthrough
- Slow WebSocket clients are dropped
- No blocking on network writes
- No JSON parsing in wave pipeline

This prevents cascade failure under load.

---

## 🧩 Scaling Considerations

The architecture is horizontally scalable:

- Multiple processors can subscribe to `ecg.wave`
- Stateless processing
- NATS supports distributed fan-out
- WebSocket server can run behind a load balancer
- Docker Compose supports replica scaling

Example:

docker compose up --scale processor=3


---

## 🐳 Docker Setup

This project is fully containerized.

### Requirements

- Docker Desktop (Apple Silicon supported)
- Docker Compose

### Run

docker compose up --build


Then open:

http://localhost:8080


---

## 📈 Metrics Endpoint

http://localhost:8080/metrics


Provides:

- Total messages
- System activity counters

---

## 🚀 Running Without Docker

Start NATS:

nats-server -js


Run services:

go run ./cmd/producer
go run ./cmd/processor
go run ./cmd/server


Open:

http://localhost:8080


---

## 🧠 Why Binary Streaming?

JSON parsing at 250 Hz introduces unnecessary overhead.

This implementation:

- Sends Float32 directly
- Eliminates wave JSON serialization
- Minimizes latency
- Reduces CPU usage
- Keeps server lightweight

This reflects production-oriented real-time system design.

---

## 📌 Future Enhancements

- Prometheus metrics integration
- Distributed load test scenario
- Burst mode simulation
- Multi-sensor simulation
- NATS clustering
- Persistent JetStream replay

---

## 🏁 Summary

This is not a UI demo.

This project demonstrates:

- Event-driven architecture
- High-frequency streaming
- Binary data pipelines
- Real-time signal processing
- Distributed-friendly design
- Production-oriented engineering mindset

The biomedical waveform is simply a realistic high-frequency workload example.

The architecture is fully reusable for any real-time streaming domain.