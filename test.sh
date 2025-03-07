#!/bin/bash

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

wait
