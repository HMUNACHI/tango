package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	log.Printf("SubmitTask response for %s: %s", jobID, res.Message)

	for {
		status, err := client.GetJobStatus(ctx, &pb.JobStatusRequest{JobId: jobID})
		if err != nil {
			log.Fatalf("GetJobStatus failed: %v", err)
		}
		if status.IsComplete {
			log.Printf("Job %s complete, final result:\n%s", jobID, string(status.FinalResult))
			expectedMatrix := multiplyFull(aMatrix, bMatrix, 1.0)
			expectedStr := matrixToString(expectedMatrix)
			if expectedStr == string(status.FinalResult) {
				log.Printf("Verification passed: final result matches expected matrix.")
			} else {
				log.Printf("Verification failed:\nExpected:\n%s\nBut got:\n%s", expectedStr, string(status.FinalResult))
			}
			break
		}
		log.Printf("Job %s still in progress...", jobID)
		time.Sleep(2 * time.Second)
	}
}
