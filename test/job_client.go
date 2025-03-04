package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	tango "cactus/tango/src"
	pb "cactus/tango/src/protobuff"

	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

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

func generateMatrix(rows, cols int, factor float32) [][]float32 {
	m := make([][]float32, rows)
	for i := 0; i < rows; i++ {
		m[i] = make([]float32, cols)
		for j := 0; j < cols; j++ {
			m[i][j] = factor * float32(i*cols+j+1)
		}
	}
	return m
}

func newFloat32(val float32) *float32 {
	return &val
}

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

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewTangoServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := tango.GetTestToken()
	if err != nil {
		log.Fatalf("failed to get test token: %v", err)
	}

	md := metadata.New(map[string]string{"cactus-token": token})
	ctx = metadata.NewOutgoingContext(ctx, md)

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

	for {
		status, err := client.GetJobStatus(ctx, &pb.JobStatusRequest{JobId: jobID})
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
					tolerance := float32(0.001)
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
		time.Sleep(2 * time.Second)
	}
}
