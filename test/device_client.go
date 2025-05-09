package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	tango "tango/tango/src"
	pb "tango/tango/src/protobuff"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var serverAddr string

// multiplyMatrices multiplies two matrices A and B and scales the resulting matrix by the given scale factor.
// It returns the product matrix or an error if the matrix dimensions are incompatible.
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
	return C, nil
}

// matrixToString converts a 2D matrix of float32 values into a formatted string.
// Each element is printed with two decimal places and rows are separated by newlines.
func matrixToString(mat [][]float32) string {
	s := ""
	for _, row := range mat {
		for _, v := range row {
			s += fmt.Sprintf("%.8f ", v)
		}
		s += "\n"
	}
	return s
}

// initDeviceClient initializes a gRPC client for the device without using TLS.
func initDeviceClient(deviceID string) (pb.TangoServiceClient, *grpc.ClientConn) {
	var addr string
	if serverAddr != "" {
		addr = serverAddr
	} else {
		addr = "localhost:50051"
	}
	conn, err := grpc.Dial(addr,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("Device %s: failed to connect: %v", deviceID, err)
		return nil, nil
	}
	return pb.NewTangoServiceClient(conn), conn
}

// createAuthCtx creates an authenticated context for the device using a test token.
func createAuthCtx(deviceID string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	token, err := tango.GetTestToken()
	if err != nil {
		log.Printf("Device %s: failed to get test token: %v", deviceID, err)
		return ctx, cancel
	}
	md := metadata.New(map[string]string{"tango-token": token})
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx, cancel
}

// processTask fetches and processes a task for the specified device.
func processTask(deviceID string, client pb.TangoServiceClient) {
	ctx, cancel := createAuthCtx(deviceID)
	req := &pb.DeviceRequest{DeviceId: deviceID}
	task, err := client.FetchTask(ctx, req)
	cancel()
	if err != nil {
		//log.Printf("Device %s: FetchTask failed: %v", deviceID, err)
		return
	}
	var resultData []byte
	if task.Operation == "scaled_matmul" {
		var A, B [][]float32
		if err := json.Unmarshal(task.AData, &A); err != nil {
			log.Printf("Device %s: failed to unmarshal AData: %v", deviceID, err)
			return
		}
		if err := json.Unmarshal(task.BData, &B); err != nil {
			log.Printf("Device %s: failed to unmarshal BData: %v", deviceID, err)
			return
		}
		scale := float32(1.0)
		if task.ScaleScalar != nil {
			scale = *task.ScaleScalar
		}
		C, err := multiplyMatrices(A, B, scale)
		if err != nil {
			log.Printf("Device %s: matrix multiplication error: %v", deviceID, err)
			return
		}
		resultData = []byte(matrixToString(C))
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
	ctx, cancel = createAuthCtx(deviceID)
	report, err := client.ReportResult(ctx, taskRes)
	cancel()
	if err != nil {
		log.Printf("Device %s: ReportResult failed: %v", deviceID, err)
		return
	}
	if !report.Success {
		log.Printf("Device %s: ReportResult failed: %s", deviceID, report.Message)
	}
}

// processDevice continuously processes tasks for the given device.
func processDevice(deviceID string) {
	client, conn := initDeviceClient(deviceID)
	if client == nil || conn == nil {
		log.Printf("Device %s: Unable to initialize client", deviceID)
		return
	}
	defer conn.Close()
	for {
		processTask(deviceID, client)
		time.Sleep(100 * time.Millisecond)
	}
}

// main is the entry point of the program.
func main() {
	// Change default number of devices from 100 to runtime.NumCPU()
	numDevices := flag.Int("devices", runtime.NumCPU(), "number of devices to simulate")
	tangoAddressPointer := flag.String("tango-address", "", "the external IP for the Tango server")
	flag.Parse()

	log.Printf("Number of devices: %d", *numDevices)

	if *tangoAddressPointer == "" {
		log.Printf("Tango address must be explicitly provided via --tango-address flag")
		return
	}
	serverAddr = *tangoAddressPointer

	fmt.Printf("Using Tango server address: %s\n", serverAddr)

	var wg sync.WaitGroup
	for i := 0; i < *numDevices; i++ {
		wg.Add(1)
		go func(deviceID string) {
			defer wg.Done()
			processDevice(deviceID)
		}(fmt.Sprintf("TestDevice%d", i))
	}
	wg.Wait()
}
