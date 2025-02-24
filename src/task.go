package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
	"fmt"
	"log"
	"time"
)

func (s *server) SubmitTask(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

	log.Printf("Received new job submission: %s expecting %d splits", req.JobId, req.NumSplits)

	job := &Job{
		JobID:           req.JobId,
		Operation:       req.Operation,
		AData:           req.AData,
		BData:           req.BData,
		m:               req.M,
		n:               req.N,
		d:               req.D,
		ExpectedSplits:  int(req.NumSplits),
		AssignedSplits:  0,
		ReceivedUpdates: 0,
		Results:         make(map[int][]byte),
		ScaleBytes:      req.ScaleBytes,
		ScaleScalar: func() float32 {
			if req.ScaleScalar != nil {
				return *req.ScaleScalar
			} else {
				return 0
			}
		}(),
		// Initialize pending tasks map.
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
	for _, jobID := range s.jobQueue {
		job, exists := s.jobs[jobID]
		if !exists {
			continue
		}
		// Try to find a shard index that is not yet completed.
		var shardIndex int
		found := false
		for i := 1; i <= job.ExpectedSplits; i++ {
			if _, done := job.Results[i]; done {
				continue // already completed
			}
			// If task is not pending or its deadline has expired, assign it.
			if td, pending := job.PendingTasks[i]; !pending || now > td.Deadline {
				shardIndex = i
				found = true
				// Mark as pending: deadline = now + 1 second.
				job.PendingTasks[i] = TimeDeadline{Deadline: time.Now().Add(time.Second).UnixNano()}
				// Increase AssignedSplits only if first time assignment.
				if !pending {
					job.AssignedSplits++
				}
				break
			}
		}
		if found {
			taskID := fmt.Sprintf("%s_%d", job.JobID, shardIndex)
			log.Printf("Assigning task %s (shard %d of job %s) to device %s",
				taskID, shardIndex, job.JobID, req.DeviceId)
			assignment := &pb.TaskAssignment{
				JobId:       job.JobID,
				TaskId:      taskID,
				Operation:   job.Operation,
				AData:       job.AData,
				BData:       job.BData,
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
