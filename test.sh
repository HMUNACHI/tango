#!/bin/bash
# filepath: ./deploy_test.sh
# This script builds the module, compiles protos, terminates any process using port 50051,
# runs the server and clients, and stops all processes after a fixed duration.
# New flags:
#   --production      If present, the test will use the deployed IP.
#   --project <ID>    Set the project ID. Defaults to "cactus-v1-452518" if not provided.

set -e

export GOOGLE_APPLICATION_CREDENTIALS=cactus-gcp-credentials.json
export PATH="$PATH:$(go env GOPATH)/bin"

DEFAULT_PROJECT_ID="cactus-v1-452518"
PROJECT_ID="$DEFAULT_PROJECT_ID"
PRODUCTION_FLAG=0
SLEEP_DURATION=10

while [[ $# -gt 0 ]]; do
  case "$1" in
    --production)
      PRODUCTION_FLAG=1
      shift
      ;;
    --project)
      PROJECT_ID="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

DEPLOYED_IP=$(gcloud compute addresses describe tango-static-ip \
  --project=${PROJECT_ID} \
  --region=us-central1 \
  --format="value(address)")

if [ $PRODUCTION_FLAG -eq 1 ]; then
  TANGO_ADDRESS="${DEPLOYED_IP}:50051"
else
  TANGO_ADDRESS="localhost:50051"
fi

echo "Using Tango address: ${TANGO_ADDRESS}"
echo "Using Project ID: ${PROJECT_ID}"

go mod tidy

protoc -I. \
  --go_out=src/protobuff --go_opt=paths=source_relative \
  --go-grpc_out=src/protobuff --go-grpc_opt=paths=source_relative \
  protobuff.proto

protoc -I. --cpp_out=cpp protobuff.proto

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

if [ $PRODUCTION_FLAG -eq 0 ]; then
  echo "Starting server..."
  go run main.go &
  SERVER_PID=$!
  sleep 2

  echo "Starting device simulator..."
  go run test/device_client.go --tango-address ${TANGO_ADDRESS} &
  DEVICE_PID=$!
  sleep 2
fi

echo "Starting job submission client..."
# go run test/job_client.go --tango-address ${TANGO_ADDRESS} &
python3 job_client.py --tango-address ${TANGO_ADDRESS} &
JOB_PID=$!

echo "Server PID: $SERVER_PID"
echo "Device Simulator PID: $DEVICE_PID"
echo "Job Submission PID: $JOB_PID"

echo "Test will run for $SLEEP_DURATION seconds..."
sleep $SLEEP_DURATION

cleanup