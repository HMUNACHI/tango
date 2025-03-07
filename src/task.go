package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

func createJob(req *pb.TaskRequest) *Job {
	return &Job{
		ConsumerID:      req.ConsumerId,
		JobID:           req.JobId,
		Operation:       req.Operation,
		AData:           req.AData,
		BData:           req.BData,
		m:               req.M,
		n:               req.N,
		d:               req.D,
		ExpectedSplits:  int(req.RowSplits * req.ColSplits),
		RowSplits:       req.RowSplits,
		ColSplits:       req.ColSplits,
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
		mu:           sync.Mutex{},
	}
}

func (s *server) SubmitTask(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
	job := createJob(req)
	s.jobsMu.Lock()
	s.jobs[req.JobId] = job
	s.jobQueue = append(s.jobQueue, req.JobId)
	s.jobsMu.Unlock()

	return &pb.TaskResponse{
		Accepted: true,
		Message:  "Job submitted successfully.",
	}, nil
}

func getAvailableTaskIndex(job *Job, now int64, deviceID string) (int, bool) {
	job.mu.Lock()
	defer job.mu.Unlock()
	var taskIndex int
	found := false
	for idx := 1; idx <= job.ExpectedSplits; idx++ {
		if _, done := job.Results[idx]; done {
			continue
		}
		if td, pending := job.PendingTasks[idx]; !pending || now > td.Deadline {
			taskIndex = idx
			found = true
			job.PendingTasks[idx] = TimeDeadline{
				Deadline: time.Now().Add(time.Second).UnixNano(),
				DeviceID: deviceID,
			}
			if !pending {
				job.AssignedSplits++
			}
			break
		}
	}
	return taskIndex, found
}

func prepareTaskAssignment(job *Job, taskIndex, gridRows, gridCols int) (*pb.TaskAssignment, error) {
	var fullA, fullB [][]float32
	if err := json.Unmarshal(job.AData, &fullA); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AData: %w", err)
	}
	if err := json.Unmarshal(job.BData, &fullB); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BData: %w", err)
	}

	rowBlock := (taskIndex - 1) / gridCols
	colBlock := (taskIndex - 1) % gridCols

	totalRows := len(fullA)
	rowsPerBlock := totalRows / gridRows
	extraRows := totalRows % gridRows
	startRow := rowBlock*rowsPerBlock + min(rowBlock, extraRows)
	endRow := startRow + rowsPerBlock
	if rowBlock < extraRows {
		endRow++
	}
	shardA := fullA[startRow:endRow]

	totalCols := len(fullB[0])
	colsPerBlock := totalCols / gridCols
	extraCols := totalCols % gridCols
	startCol := colBlock*colsPerBlock + min(colBlock, extraCols)
	endCol := startCol + colsPerBlock
	if colBlock < extraCols {
		endCol++
	}
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

	assignment := &pb.TaskAssignment{
		JobId:       job.JobID,
		TaskId:      taskID,
		Operation:   job.Operation,
		AData:       shardABytes,
		BData:       shardBBytes,
		M:           int32(rowBlock),
		N:           int32(colBlock),
		D:           int32(gridRows),
		ScaleBytes:  job.ScaleBytes,
		ScaleScalar: &job.ScaleScalar,
	}
	return assignment, nil
}

func (s *server) FetchTask(ctx context.Context, req *pb.DeviceRequest) (*pb.TaskAssignment, error) {
	now := time.Now().UnixNano()

	s.jobsMu.RLock()
	jobIDs := make([]string, len(s.jobQueue))
	copy(jobIDs, s.jobQueue)
	s.jobsMu.RUnlock()

	for _, jobID := range jobIDs {
		s.jobsMu.RLock()
		job, exists := s.jobs[jobID]
		s.jobsMu.RUnlock()
		if !exists {
			continue
		}

		taskIndex, found := getAvailableTaskIndex(job, now, req.DeviceId)
		if !found {
			continue
		}

		gridRows := int(job.RowSplits)
		gridCols := int(job.ColSplits)
		return prepareTaskAssignment(job, taskIndex, gridRows, gridCols)
	}
	return nil, fmt.Errorf("no available tasks")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
