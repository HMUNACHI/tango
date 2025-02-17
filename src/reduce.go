package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
)

func (s *server) ReportResult(ctx context.Context, res *pb.TaskResult) (*pb.ResultResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Received result for task %s of job %s from device %s", res.TaskId, res.JobId, res.DeviceId)

	job, exists := s.jobs[res.JobId]
	if !exists {
		return &pb.ResultResponse{
			Success: false,
			Message: "Job not found.",
		}, nil
	}

	// Extract the shard index from the task ID (expected format: "jobID_shardIndex")
	shardIndex, err := extractShardIndex(res.TaskId)
	if err != nil {
		return &pb.ResultResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid task id format: %v", err),
		}, nil
	}

	// Initialize the Results map if needed.
	if job.Results == nil {
		job.Results = make(map[int][]byte)
	}
	// Store the binary C_shard result.
	job.Results[shardIndex] = []byte(res.ResultData)

	job.ReceivedUpdates++

	// When all shards have been received, reassemble the final result.
	if job.ReceivedUpdates == job.ExpectedSplits {
		finalResult, err := reassembleCShards(job.Results)
		if err != nil {
			log.Printf("Job %s complete, but failed to reassemble C_shards: %v", job.JobID, err)
		} else {
			log.Printf("Job %s complete. Final aggregated result: %v", job.JobID, finalResult)
			job.FinalResult = finalResult
		}
		// Note: We are not deleting the job immediately so the consumer can retrieve FinalResult.
	}

	return &pb.ResultResponse{
		Success: true,
		Message: "Result received and processed.",
	}, nil
}

// extractShardIndex extracts the shard index from the task ID.
// Expected format is "jobID_shardIndex".
func extractShardIndex(taskId string) (int, error) {
	parts := strings.Split(taskId, "_")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid task id format")
	}
	return strconv.Atoi(parts[1])
}

// reassembleCShards aggregates the binary C_shard results by concatenating them in order.
func reassembleCShards(results map[int][]byte) ([]byte, error) {
	var keys []int
	for k := range results {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var finalResult []byte
	for _, k := range keys {
		finalResult = append(finalResult, results[k]...)
	}
	return finalResult, nil
}
