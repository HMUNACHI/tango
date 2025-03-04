package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"
)

func (s *server) SubmitTask(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	// For full 2D splitting, ExpectedSplits becomes (num_splits)^2.
	totalTasks := int(req.NumSplits * req.NumSplits)
	log.Printf("Received new job submission: %s expecting %d tasks (2D splitting)", req.JobId, totalTasks)

	job := &Job{
		ConsumerID:      req.ConsumerId,
		JobID:           req.JobId,
		Operation:       req.Operation,
		AData:           req.AData,
		BData:           req.BData,
		m:               req.M,
		n:               req.N,
		d:               req.D,
		ExpectedSplits:  totalTasks, // update here
		AssignedSplits:  0,
		ReceivedUpdates: 0,
		Results:         make(map[int][]byte),
		ScaleBytes:      req.ScaleBytes,
		ScaleScalar: func() float32 {
			if req.ScaleScalar != nil {
				return *req.ScaleScalar
			}
			return 0
		}(),
		PendingTasks: make(map[int]TimeDeadline),
	}
	s.jobs[req.JobId] = job
	s.jobQueue = append(s.jobQueue, req.JobId)

	return &pb.TaskResponse{
		Accepted: true,
		Message:  "Job submitted successfully.",
	}, nil
}

func (s *server) FetchTask(ctx context.Context, req *pb.DeviceRequest) (*pb.TaskAssignment, error) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	log.Printf("Device %s requesting a task", req.DeviceId)
	now := time.Now().UnixNano()
	var gridSize int
	for _, jobID := range s.jobQueue {
		job, exists := s.jobs[jobID]
		if !exists {
			continue
		}
		// Set gridSize = sqrt(ExpectedSplits). (Assume perfect square.)
		gridSize = int(math.Sqrt(float64(job.ExpectedSplits)))
		var taskIndex int
		found := false
		// Iterate over all task indices from 1 to ExpectedSplits.
		for idx := 1; idx <= job.ExpectedSplits; idx++ {
			if _, done := job.Results[idx]; done {
				continue
			}
			if td, pending := job.PendingTasks[idx]; !pending || now > td.Deadline {
				taskIndex = idx
				found = true
				job.PendingTasks[idx] = TimeDeadline{
					Deadline: time.Now().Add(time.Second).UnixNano(),
					DeviceID: req.DeviceId,
				}
				if !pending {
					job.AssignedSplits++
				}
				break
			}
		}
		if found {
			// Determine block coordinates.
			rowBlock := (taskIndex - 1) / gridSize
			colBlock := (taskIndex - 1) % gridSize

			// Partition A (split by rows) and B (split by columns).
			var fullA, fullB [][]float32
			if err := json.Unmarshal(job.AData, &fullA); err != nil {
				return nil, fmt.Errorf("failed to unmarshal AData: %w", err)
			}
			if err := json.Unmarshal(job.BData, &fullB); err != nil {
				return nil, fmt.Errorf("failed to unmarshal BData: %w", err)
			}

			// Compute row boundaries for A.
			totalRows := len(fullA)
			rowsPerBlock := totalRows / gridSize
			extraRows := totalRows % gridSize
			startRow := rowBlock*rowsPerBlock + min(rowBlock, extraRows)
			endRow := startRow + rowsPerBlock
			if rowBlock < extraRows {
				endRow++
			}
			shardA := fullA[startRow:endRow]

			// Compute column boundaries for B.
			totalCols := len(fullB[0])
			colsPerBlock := totalCols / gridSize
			extraCols := totalCols % gridSize
			startCol := colBlock*colsPerBlock + min(colBlock, extraCols)
			endCol := startCol + colsPerBlock
			if colBlock < extraCols {
				endCol++
			}
			// For each row in B, take the slice.
			shardB := make([][]float32, len(fullB))
			for i := range fullB {
				shardB[i] = fullB[i][startCol:endCol]
			}

			shardABytes, err := json.Marshal(shardA)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal shardA: %w", err)
			}
			shardBBytes, err := json.Marshal(shardB)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal shardB: %w", err)
			}

			taskID := fmt.Sprintf("%s_%d", job.JobID, taskIndex)
			log.Printf("Assigning task %s (rowBlock=%d, colBlock=%d of job %s) to device %s",
				taskID, rowBlock, colBlock, job.JobID, req.DeviceId)

			assignment := &pb.TaskAssignment{
				JobId:     job.JobID,
				TaskId:    taskID,
				Operation: job.Operation,
				AData:     shardABytes,
				BData:     shardBBytes,
				// Pass original dimensions as needed.
				M:           job.m,
				N:           job.n,
				D:           job.d,
				NumSplits:   int32(job.ExpectedSplits),
				ScaleBytes:  job.ScaleBytes,
				ScaleScalar: &job.ScaleScalar,
			}
			return assignment, nil
		}
	}
	return nil, fmt.Errorf("no available tasks at this time")
}

// Helper: minimal function for min.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
