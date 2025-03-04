package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
)

func (s *server) ReportResult(ctx context.Context, res *pb.TaskResult) (*pb.ResultResponse, error) {
	s.jobsMu.RLock()
	job, exists := s.jobs[res.JobId]
	s.jobsMu.RUnlock()
	if !exists {
		return &pb.ResultResponse{
			Success: false,
			Message: "Job not found.",
		}, nil
	}
	job.mu.Lock()

	shardIndex, err := extractShardIndex(res.TaskId)
	if err != nil {
		job.mu.Unlock()
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

	if res.Flops > 0 && len(res.ResultData) > 0 {
		if err := AppendRecord(res.DeviceId, job.ConsumerID, res.Flops); err != nil {
			log.Printf("Failed to append record for job %s: %v", job.JobID, err)
		}
	}

	var completedJobID string
	if job.ReceivedUpdates == job.ExpectedSplits {
		finalResult, err := reassembleCShards(job.Results, int(job.ColSplits))
		if err != nil {
			log.Printf("Job %s complete, but failed to reassemble C_shards: %v", job.JobID, err)
		} else {
			job.FinalResult = finalResult
			completedJobID = job.JobID
		}
	}
	job.mu.Unlock()

	if completedJobID != "" {
		if err := UploadRecordsToGCS(completedJobID); err != nil {
			log.Printf("Failed to upload records to GCS for job %s: %v", completedJobID, err)
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

func reassembleCShards(results map[int][]byte, gridCols int) ([]byte, error) {
	total := len(results)
	if total == 0 {
		return nil, fmt.Errorf("no shard results")
	}
	gridRows := total / gridCols

	shards := make([][]([]string), gridRows)
	for i := range shards {
		shards[i] = make([][]string, gridCols)
	}

	for key, data := range results {
		rowBlock := (key - 1) / gridCols
		colBlock := (key - 1) % gridCols
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		shards[rowBlock][colBlock] = lines
	}

	var fullRows []string
	for r := 0; r < gridRows; r++ {
		if len(shards[r]) == 0 || len(shards[r][0]) == 0 {
			return nil, fmt.Errorf("empty shard in row block %d", r)
		}
		blockRows := len(shards[r][0])
		for i := 0; i < blockRows; i++ {
			var rowParts []string
			for c := 0; c < gridCols; c++ {
				rowParts = append(rowParts, strings.TrimSpace(shards[r][c][i]))
			}
			fullRows = append(fullRows, strings.Join(rowParts, " "))
		}
	}
	finalResult := strings.Join(fullRows, "\n")
	finalResult = strings.TrimSpace(finalResult)
	return []byte(finalResult), nil
}
