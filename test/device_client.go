package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	tango "cactus/tango/src"
	pb "cactus/tango/src/protobuff"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func multiplyMatrices(A, B [][]float32, scale float32) ([][]float32, error) {
	if len(A) == 0 || len(B) == 0 || len(A[0]) != len(B) {
		return nil, errors.New("incompatible matrix dimensions")
	}
	m, d, n := len(A), len(B), len(B[0])
	C := make([][]float32, m)
	for i := 0; i < m; i++ {
		C[i] = make([]float32, n)
		for j := 0; j < n; j++ {
			var sum float32
			for k := 0; k < d; k++ {
				sum += A[i][k] * B[k][j]
			}
			C[i][j] = sum * scale
		}
	}
	fmt.Println("Matrix A:")
	fmt.Println(matrixToString(A))
	fmt.Println("Matrix B:")
	return C, nil
}

func matrixToString(mat [][]float32) string {
	s := ""
	for _, row := range mat {
		for _, v := range row {
			s += fmt.Sprintf("%.2f ", v)
		}
		s += "\n"
	}
	return s
}

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(deviceID string) {
			defer wg.Done()
			conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
			if err != nil {
				log.Fatalf("Device %s: failed to connect: %v", deviceID, err)
			}
			defer conn.Close()
			client := pb.NewTangoServiceClient(conn)
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				token, err := tango.GetTestToken()
				if err != nil {
					log.Fatalf("Device %s: failed to get test token: %v", deviceID, err)
				}
				md := metadata.New(map[string]string{"cactus-token": token})
				ctx = metadata.NewOutgoingContext(ctx, md)
				req := &pb.DeviceRequest{DeviceId: deviceID}

				task, err := client.FetchTask(ctx, req)
				if err != nil {
					cancel()
					time.Sleep(2 * time.Second)
					continue
				}
				log.Printf("Device %s fetched task: JobId=%s TaskId=%s Operation=%s", deviceID, task.JobId, task.TaskId, task.Operation)
				var resultData []byte
				if task.Operation == "scaled_matmul" {
					var A, B [][]float32
					if err := json.Unmarshal(task.AData, &A); err != nil {
						log.Fatalf("Device %s: failed to unmarshal AData: %v", deviceID, err)
					}
					if err := json.Unmarshal(task.BData, &B); err != nil {
						log.Fatalf("Device %s: failed to unmarshal BData: %v", deviceID, err)
					}
					scale := float32(1.0)
					if task.ScaleScalar != nil {
						scale = *task.ScaleScalar
					}
					C, err := multiplyMatrices(A, B, scale)
					if err != nil {
						log.Fatalf("Device %s: matrix multiplication error: %v", deviceID, err)
					}
					resultData = []byte(matrixToString(C))
					fmt.Println(string(resultData))
				} else {
					resultData = []byte("unsupported operation")
				}

				taskRes := &pb.TaskResult{
					DeviceId:   deviceID,
					JobId:      task.JobId,
					TaskId:     task.TaskId,
					ResultData: resultData,
					Flops:      2 * task.M * task.N * task.D,
				}
				report, err := client.ReportResult(ctx, taskRes)
				if err != nil {
					log.Printf("Device %s: ReportResult failed: %v", deviceID, err)
					cancel()
					time.Sleep(2 * time.Second)
					continue
				}
				log.Printf("Device %s: ReportResult: %s", deviceID, report.Message)
				cancel()
				time.Sleep(1 * time.Second)
			}
		}(fmt.Sprintf("Device_%d", i))
	}
	wg.Wait()
}
