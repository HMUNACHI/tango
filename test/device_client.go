package main

import (
	"context"
	"fmt"
	"log"
	"time"

	tango "cactus/tango/src"
	pb "cactus/tango/src/protobuff"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func matMul(m, d, n int, scale float32) []byte {
	A := make([][]float32, m)
	for i := 0; i < m; i++ {
		A[i] = make([]float32, d)
		for j := 0; j < d; j++ {
			A[i][j] = float32(i + j + 1)
		}
	}
	B := make([][]float32, d)
	for i := 0; i < d; i++ {
		B[i] = make([]float32, n)
		for j := 0; j < n; j++ {
			B[i][j] = float32(i*j + 1)
		}
	}
	C := make([][]float32, m)
	for i := 0; i < m; i++ {
		C[i] = make([]float32, n)
		for j := 0; j < n; j++ {
			sum := float32(0)
			for k := 0; k < d; k++ {
				sum += A[i][k] * B[k][j]
			}
			C[i][j] = sum * scale
		}
	}
	s := ""
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			s += fmt.Sprintf("%.2f ", C[i][j])
		}
		s += "\n"
	}
	return []byte(s)
}

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewTangoServiceClient(conn)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		md := metadata.New(map[string]string{"cactus-token": tango.AppConfig.Tokens.TestToken})
		ctx = metadata.NewOutgoingContext(ctx, md)
		req := &pb.DeviceRequest{DeviceId: "device1"}

		task, err := client.FetchTask(ctx, req)
		if err != nil {
			cancel()
			time.Sleep(2 * time.Second)
			continue
		}
		log.Printf("Fetched task: JobId=%s TaskId=%s Operation=%s", task.JobId, task.TaskId, task.Operation)

		var resultData []byte
		if task.Operation == "matmul" {
			scale := float32(1.0)
			if task.ScaleScalar != nil {
				scale = *task.ScaleScalar
			}
			resultData = matMul(int(task.M), int(task.D), int(task.N), scale)
		} else {
			resultData = []byte("unsupported operation")
		}

		taskRes := &pb.TaskResult{
			DeviceId:   "device1",
			JobId:      task.JobId,
			TaskId:     task.TaskId,
			ResultData: resultData,
		}
		report, err := client.ReportResult(ctx, taskRes)
		if err != nil {
			log.Printf("ReportResult failed: %v", err)
			cancel()
			time.Sleep(2 * time.Second)
			continue
		}
		log.Printf("ReportResult: %s", report.Message)
		cancel()
		time.Sleep(1 * time.Second)
	}
}
