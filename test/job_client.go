/*
Tango is a product of Cactus Compute, Inc.
This code is proprietary. Do not share the code.
*/
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	tango "cactus/tango/src"
	pb "cactus/tango/src/protobuff"

	"crypto/tls"
	"crypto/x509"

	"net"

	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

var tangoAddress string

func init() {
	flag.StringVar(&tangoAddress, "tango-address", "localhost:50051", " address of the Tango service")
}

// matrixToString converts a 2D float32 matrix into a formatted string.
// Each element is formatted to two decimal places and rows are separated by newlines.
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

// multiplyFull multiplies matrix A by matrix B and scales the result by the given scale factor.
// A should be of dimensions m x d and B of dimensions d x n.
// It returns the resulting m x n matrix.
func multiplyFull(A, B [][]float32, scale float32) [][]float32 {
	m := len(A)
	d := len(B)
	n := len(B[0])
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
	return C
}

// generateMatrix creates a matrix with the given number of rows and columns.
// Each element is randomly generated.
func generateMatrix(rows, cols int, factor float32) [][]float32 {
	m := make([][]float32, rows)
	for i := 0; i < rows; i++ {
		m[i] = make([]float32, cols)
		for j := 0; j < cols; j++ {
			m[i][j] = factor * rand.Float32()
		}
	}
	return m
}

func newFloat32(val float32) *float32 {
	return &val
}

// parseMatrix converts a string representation of a matrix into a 2D slice of float32.
// The string should have rows separated by newlines and values separated by spaces.
func parseMatrix(s string) ([][]float32, error) {
	var result [][]float32
	rows := strings.Split(strings.TrimSpace(s), "\n")
	for _, row := range rows {
		if strings.TrimSpace(row) == "" {
			continue
		}
		fields := strings.Fields(row)
		var numericRow []float32
		for _, valStr := range fields {
			var v float32
			_, err := fmt.Sscanf(valStr, "%f", &v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value %q: %v", valStr, err)
			}
			numericRow = append(numericRow, v)
		}
		result = append(result, numericRow)
	}
	return result, nil
}

// initClient initializes the gRPC client for the Tango service using TLS.
// It retrieves server secrets, sets up the TLS configuration, and creates a context with a timeout
// that includes the necessary authentication token. It returns the TangoServiceClient, context,
// cancel function, and the gRPC connection.
func initClient() (pb.TangoServiceClient, context.Context, context.CancelFunc, *grpc.ClientConn) {
	crt, _, err := tango.GetServerSecrets()
	if err != nil {
		log.Fatalf("failed to get server secrets: %v", err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM([]byte(crt)) {
		log.Fatalf("failed to append server cert")
	}
	if !strings.Contains(tangoAddress, ":") {
		tangoAddress = tangoAddress + ":50051"
	}
	_, _, err = net.SplitHostPort(tangoAddress)
	if err != nil {
		log.Fatalf("invalid tango address: %v", err)
	}
	creds := credentials.NewTLS(&tls.Config{
		RootCAs:    certPool,
		ServerName: "tango",
	})

	conn, err := grpc.Dial(tangoAddress,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("zstd")),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	client := pb.NewTangoServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	token, err := tango.GetTestToken()
	if err != nil {
		log.Fatalf("failed to get test token: %v", err)
	}
	md := metadata.New(map[string]string{"cactus-token": token})
	ctx = metadata.NewOutgoingContext(ctx, md)
	return client, ctx, cancel, conn
}

// submitJob creates and submits a matrix multiplication job to the Tango service.
// It generates a random job ID, creates two matrices A and B, marshals them into JSON,
// and constructs a TaskRequest. If the task is accepted, it returns the job ID and the matrices.
func submitJob(client pb.TangoServiceClient, ctx context.Context) (string, [][]float32, [][]float32) {
	rand.Seed(uint64(time.Now().UnixNano()))
	taskID := rand.Intn(10000)
	jobID := fmt.Sprintf("Job%d", taskID)

	aMatrix := generateMatrix(16, 8, 0.1)
	bMatrix := generateMatrix(8, 16, 0.1)

	aBytes, err := json.Marshal(aMatrix)
	if err != nil {
		log.Fatalf("failed to marshal A matrix: %v", err)
	}
	bBytes, err := json.Marshal(bMatrix)
	if err != nil {
		log.Fatalf("failed to marshal B matrix: %v", err)
	}

	if err := tango.PrintCompressionStats(aBytes); err != nil {
		log.Printf("Failed to print compression stats for A matrix: %v", err)
	}
	if err := tango.PrintCompressionStats(bBytes); err != nil {
		log.Printf("Failed to print compression stats for B matrix: %v", err)
	}

	jobReq := &pb.TaskRequest{
		ConsumerId:  "234s5c2",
		JobId:       jobID,
		Operation:   "scaled_matmul",
		AData:       aBytes,
		BData:       bBytes,
		RowSplits:   8,
		ColSplits:   4,
		M:           16,
		N:           16,
		D:           8,
		ScaleScalar: newFloat32(1.0),
	}
	res, err := client.SubmitTask(ctx, jobReq)
	if err != nil {
		log.Fatalf("SubmitTask for %s failed: %v", jobID, err)
	}
	if !res.Accepted {
		log.Fatalf("SubmitTask for %s rejected: %s", jobID, res.Message)
	}
	return jobID, aMatrix, bMatrix
}

// pollJobStatus continuously polls the Tango service for the status of the submitted job.
// It stops polling when the job is complete or a timeout is reached.
// Once complete, it parses and verifies the final result against the expected matrix multiplication result.
func pollJobStatus(client pb.TangoServiceClient, ctx context.Context, jobID string, aMatrix, bMatrix [][]float32) {
	waitTime := 100 * time.Second
	for {
		pollCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		status, err := client.GetJobStatus(pollCtx, &pb.JobStatusRequest{JobId: jobID})
		cancel()
		if err != nil {
			log.Fatalf("GetJobStatus failed: %v", err)
		}
		if status.IsComplete {
			expectedMatrix := multiplyFull(aMatrix, bMatrix, 1.0)
			finalMatrix, err := parseMatrix(string(status.FinalResult))
			if err != nil {
				log.Printf("Failed to parse final result matrix: %v", err)
			} else {
				if len(expectedMatrix) != len(finalMatrix) {
					log.Printf("Verification failed: expected %d rows, got %d", len(expectedMatrix), len(finalMatrix))
				} else {
					tolerance := float32(0.005)
					pass := true
					for i, expRow := range expectedMatrix {
						if len(expRow) != len(finalMatrix[i]) {
							log.Printf("Verification failed: row %d column count mismatch", i)
							pass = false
							break
						}
						for j, expVal := range expRow {
							diff := expVal - finalMatrix[i][j]
							if diff < 0 {
								diff = -diff
							}
							if diff > tolerance {
								log.Printf("Mismatch at [%d][%d]: expected %.2f, got %.2f", i, j, expVal, finalMatrix[i][j])
								pass = false
							}
						}
					}
					if pass {
						log.Printf("Verification passed: final result matches expected matrix.")
					} else {
						log.Printf("Verification failed: matrices differ.")
					}
				}
			}
			break
		}
		time.Sleep(100 * time.Millisecond)
		waitTime -= 100 * time.Millisecond
		if waitTime <= 0 {
			log.Fatalf("Job %s did not complete within expected time.", jobID)
		}
	}
}

// main is the entry point of the program.
// It initializes the gRPC client, submits a matrix multiplication job,
// and polls for the job's status until the computation is complete.
func main() {
	flag.Parse()
	log.Printf("Using tango address: %s", tangoAddress)

	client, ctx, cancel, conn := initClient()
	defer cancel()
	defer conn.Close()

	jobID, aMatrix, bMatrix := submitJob(client, ctx)
	pollJobStatus(client, ctx, jobID, aMatrix, bMatrix)
}
