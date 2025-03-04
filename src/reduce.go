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
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	job, exists := s.jobs[res.JobId]
	if !exists {
		return &pb.ResultResponse{
			Success: false,
			Message: "Job not found.",
		}, nil
	}

	shardIndex, err := extractShardIndex(res.TaskId)
	if err != nil {
		return &pb.ResultResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid task id format: %v", err),
		}, nil
	}

	if job.Results == nil {
		job.Results = make(map[int][]byte)
	}

	delete(job.PendingTasks, shardIndex)

	job.Results[shardIndex] = []byte(res.ResultData)

	job.ReceivedUpdates++

	if job.ReceivedUpdates == job.ExpectedSplits {
		finalResult, err := reassembleCShards(job.Results)
		if err != nil {
			log.Printf("Job %s complete, but failed to reassemble C_shards: %v", job.JobID, err)
		} else {
			job.FinalResult = finalResult
			if err := AppendRecord(res.DeviceId, job.ConsumerID, res.NumElements); err != nil {
				log.Printf("Failed to write record for job %s: %v", job.JobID, err)
			}
		}
	}

	return &pb.ResultResponse{
		Success: true,
		Message: "Result received and processed.",
	}, nil
}

func extractShardIndex(taskId string) (int, error) {
	parts := strings.Split(taskId, "_")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid task id format")
	}
	return strconv.Atoi(parts[1])
}

// Updated reassembleCShards to vertically concatenate shards.
// It assumes each shard result is a multi-line string representing a 4x16 block.
func reassembleCShards(results map[int][]byte) ([]byte, error) {
	var keys []int
	for k := range results {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var allLines []string
	for _, k := range keys {
		shardStr := string(results[k])
		lines := strings.Split(shardStr, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				allLines = append(allLines, line)
			}
		}
	}
	finalResult := strings.Join(allLines, "\n")
	return []byte(finalResult), nil
}
