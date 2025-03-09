/*
Tango is a product of Cactus Compute, Inc.
This code is proprietary. Do not share the code.
*/
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	tango "cactus/tango/src"
	pb "cactus/tango/src/protobuff"
	"crypto/tls"
	"crypto/x509"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
			s += fmt.Sprintf("%.2f ", v)
		}
		s += "\n"
	}
	return s
}

// initDeviceClient initializes and returns a TangoService gRPC client and its connection for the specified deviceID.
// It sets up TLS using server secrets and returns the client and underlying connection.
func initDeviceClient(deviceID string) (pb.TangoServiceClient, *grpc.ClientConn) {
	crt, _, err := tango.GetServerSecrets()
	if err != nil {
		log.Fatalf("Device %s: failed to get server secrets: %v", deviceID, err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM([]byte(crt)) {
		log.Fatalf("Device %s: failed to append server cert", deviceID)
	}
	creds := credentials.NewTLS(&tls.Config{
		RootCAs:    certPool,
		ServerName: "tango",
	})
	var serverAddr string
	if serverAddr != "" {
		serverAddr = serverAddr
	} else {
		serverAddr = "localhost:50051"
	}
	conn, err := grpc.Dial(serverAddr,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("zstd")),
	)
	if err != nil {
		log.Fatalf("Device %s: failed to connect: %v", deviceID, err)
	}
	return pb.NewTangoServiceClient(conn), conn
}

// createAuthCtx creates an authenticated context for the given device using a test token.
// It returns the context with metadata and a cancel function to free resources.
func createAuthCtx(deviceID string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	token, err := tango.GetTestToken()
	if err != nil {
		log.Fatalf("Device %s: failed to get test token: %v", deviceID, err)
	}
	md := metadata.New(map[string]string{"cactus-token": token})
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx, cancel
}

// processTask fetches and processes a task for the specified device.
// If the task operation is "scaled_matmul", it performs matrix multiplication on the provided matrices,
// formats the result as a string, and reports the result back to the Tango service.
func processTask(deviceID string, client pb.TangoServiceClient) {
	ctx, cancel := createAuthCtx(deviceID)
	req := &pb.DeviceRequest{DeviceId: deviceID}
	task, err := client.FetchTask(ctx, req)
	cancel()
	if err != nil {
		return
	}
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
// It initializes a client for the device and repeatedly fetches and processes tasks at 1-second intervals.
func processDevice(deviceID string) {
	client, conn := initDeviceClient(deviceID)
	defer conn.Close()
	for {
		processTask(deviceID, client)
		time.Sleep(1 * time.Second)
	}
}

// main is the entry point of the program.
// It parses the number of devices to simulate from the command-line flag, spawns a goroutine for each device,
// and waits for all goroutines to complete.
func main() {
	numDevices := flag.Int("devices", 1000, "number of device to simulate")
	tangoAddressPointer := flag.String("tango-address", "", "custom server address (e.g. 'myserver:50051')")
	flag.Parse()
	serverAddr = *tangoAddressPointer

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
