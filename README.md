# Go Real-Time Streaming Architecture

High-frequency event-driven streaming architecture built in Go using NATS / JetStream.

This project demonstrates how to design and implement a low-latency real-time processing pipeline, using a biomedical signal workload (250 Hz) as a realistic streaming example.

The biomedical signal is used purely as a high-frequency streaming scenario.  
The architecture itself is domain-agnostic.

---

## 🧠 What This Project Demonstrates

- 250 Hz continuous streaming
- Event-driven decoupled services
- Real-time peak detection & metric extraction
- Backpressure-aware batching
- WebSocket low-latency delivery
- Clean concurrency model in Go
- Graceful shutdown & reconnection handling

This simulates real-world high-frequency telemetry pipelines such as:

- IoT sensor networks
- Medical monitoring systems
- Industrial telemetry
- Edge streaming workloads
- Real-time analytics ingestion

---

## 🏗 Architecture Overview

Producer (250Hz signal)
↓
NATS
↓
Processor (Peak detection + HR calculation)
↓
NATS
↓
WebSocket Server (batched streaming)
↓
Browser (Canvas rendering @ ~60 FPS)


### Design Principles

- Fully decoupled services
- Stateless processors
- Horizontal scaling possible at processor layer
- Message-driven architecture
- Controlled UI batching (25Hz)
- Latency-aware processing

---

## ⚙️ Services

### Producer

- Generates 250 samples/sec
- Simulated ECG-like waveform
- Publishes JSON messages to `ecg.wave`

### Processor

- Subscribes to `ecg.wave`
- Detects R-peaks with refractory control
- Calculates RR interval
- Publishes BPM metrics to `ecg.params`

### Server

- Subscribes to NATS
- Batches wave samples (~40ms interval)
- Sends array payloads via WebSocket
- Handles client backpressure

### Browser UI

- Uses incremental Canvas rendering
- Decouples rendering from WebSocket events
- Maintains smooth visual output

---

## 📊 Benchmarks (Local M4)

| Metric | Result |
|--------|--------|
| Producer rate | ~250 msg/sec |
| Processor latency | ~2–5 ms |
| WebSocket batch rate | ~25 fps |
| CPU usage | ~2–4% |
| End-to-end latency | < 15 ms (local) |

---

## 🔄 Backpressure Strategy

To prevent UI congestion:

- Server batches samples before sending
- Max batch size enforced
- Slow WebSocket clients are dropped
- No service blocks the streaming pipeline

This prevents cascade failure under load.

---

## 🧩 Scaling Considerations

- Multiple processors can subscribe to the same subject
- Horizontal scaling at processing layer
- NATS JetStream enables durability if needed
- WebSocket server can be replicated behind a load balancer
- Binary streaming mode can reduce overhead further

---

## 🚀 Running Locally

Start NATS:

nats-server -js


Run producer:

go run ./cmd/producer


Run processor:

go run ./cmd/processor


Run server:

go run ./cmd/server


Open:

http://localhost:8080


---

## 🧠 Why Biomedical Signal?

Biomedical streaming (ECG-like waveform) is a realistic high-frequency workload:

- Continuous signal
- Strict timing requirements
- Peak detection
- Low-latency feedback

However, the architecture is completely generic and can be applied to:

- Industrial sensors
- Financial tick streams
- Telemetry ingestion pipelines
- Edge data processing systems

---

## 📌 Future Improvements

- Binary WebSocket streaming (Float32Array)
- Prometheus metrics endpoint
- Load testing scenario
- Configurable scaling parameters
- Distributed processor workers

---

## 🏁 Summary

This project is not a UI demo.  
It is a demonstration of designing and implementing a clean real-time streaming architecture in Go.

It emphasizes:

- Event-driven design
- Throughput control
- Latency awareness
- Proper service decoupling
- Production-oriented thinking