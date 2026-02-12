#!/bin/bash

set -e

echo "Starting NATS..."
nats-server -js > /dev/null 2>&1 &
NATS_PID=$!

sleep 1

echo "Starting producer..."
go run ./cmd/producer &
PRODUCER_PID=$!

echo "Starting processor..."
go run ./cmd/processor &
PROCESSOR_PID=$!

echo "Starting server..."
go run ./cmd/server &
SERVER_PID=$!

cleanup() {
    echo "Shutting down..."
    kill $PRODUCER_PID
    kill $PROCESSOR_PID
    kill $SERVER_PID
    kill $NATS_PID
    exit
}

trap cleanup SIGINT

wait
