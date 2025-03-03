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

	numTasks := 10
	for i := 1; i <= numTasks; i++ {
		jobID := fmt.Sprintf("job%d", i)
		jobReq := &pb.TaskRequest{
			JobId:       jobID,
			Operation:   "scaled_matmul",
			AData:       []byte(fmt.Sprintf("AData_for_%s", jobID)),
			BData:       []byte(fmt.Sprintf("BData_for_%s", jobID)),
			NumSplits:   2,
			M:           4,
			N:           4,
			D:           4,
			ScaleScalar: newFloat32(1.0),
		}
		res, err := client.SubmitTask(ctx, jobReq)
		if err != nil {
			log.Fatalf("SubmitTask for %s failed: %v", jobID, err)
		}
		log.Printf("SubmitTask response for %s: %s", jobID, res.Message)
	}
}
