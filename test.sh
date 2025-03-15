#!/bin/bash
# filepath: ./deploy_test.sh
# This script builds the module, compiles protos, terminates any process using port 50051,
# runs the server and clients, and stops all processes after a fixed duration.

set -e

export GOOGLE_APPLICATION_CREDENTIALS=cactus-gcp-credentials.json
export PATH="$PATH:$(go env GOPATH)/bin"

go mod tidy

protoc -I. \
  --go_out=src/protobuff --go_opt=paths=source_relative \
  --go-grpc_out=src/protobuff --go-grpc_opt=paths=source_relative \
  protobuff.proto

cleanup() {
  echo "Stopping all processes..."
  kill $SERVER_PID $DEVICE_PID $JOB_PID 2>/dev/null || true
  exit
}

trap cleanup EXIT INT TERM

OLD_PID=$(lsof -t -i:50051 || true)
if [ -n "$OLD_PID" ]; then
  echo "Port 50051 is in use. Terminating process(es): $OLD_PID"
  kill -9 $OLD_PID
  sleep 2
fi

echo "Starting server..."
go run main.go &
SERVER_PID=$!
sleep 3

echo "Starting device simulator..."
go run test/device_client.go --devices 100 &
DEVICE_PID=$!
sleep 3

echo "Starting job submission client..."
go run test/job_client.go &
JOB_PID=$!

echo "Server PID: $SERVER_PID"
echo "Device Simulator PID: $DEVICE_PID"
echo "Job Submission PID: $JOB_PID"

SLEEP_DURATION=30  # increased from 5 to 30 seconds
echo "Test will run for $SLEEP_DURATION seconds..."
sleep $SLEEP_DURATION

cleanup