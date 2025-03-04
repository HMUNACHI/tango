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
	// Use global read lock to get the job pointer.
	s.jobsMu.RLock()
	job, exists := s.jobs[res.JobId]
	s.jobsMu.RUnlock()
	if !exists {
		return &pb.ResultResponse{
			Success: false,
			Message: "Job not found.",
		}, nil
	}
	// Lock the individual job.
	job.mu.Lock()
	defer job.mu.Unlock()

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
		finalResult, err := reassembleCShards(job.Results, int(job.ColSplits))
		if err != nil {
			log.Printf("Job %s complete, but failed to reassemble C_shards: %v", job.JobID, err)
		} else {
			job.FinalResult = finalResult
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
