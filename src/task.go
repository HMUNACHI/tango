package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
	"fmt"
	"log"
)

// SubmitTask creates a new Job using the provided TaskRequest.
// The request now includes the Operation, AData, and BData fields.
// AData and BData are expected to carry the binary (e.g. fp16) data for the respective operands.
func (s *server) SubmitTask(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Received new job submission: %s expecting %d splits", req.JobId, req.NumSplits)

	job := &Job{
		JobID:           req.JobId,
		Operation:       req.Operation,
		AData:           req.AData,
		BData:           req.BData,
		ExpectedSplits:  int(req.NumSplits),
		AssignedSplits:  0,
		ReceivedUpdates: 0,
		Results:         make(map[int][]byte),
	}
	s.jobs[req.JobId] = job
	s.jobQueue = append(s.jobQueue, req.JobId)

	return &pb.TaskResponse{
		Accepted: true,
		Message:  "Job submitted successfully.",
	}, nil
}

// FetchTask assigns a task to a requesting device.
// It returns a TaskAssignment that includes the operation to perform and the operands (AData and BData).
func (s *server) FetchTask(ctx context.Context, req *pb.DeviceRequest) (*pb.TaskAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Device %s requesting a task", req.DeviceId)
	// Iterate over the jobQueue in FIFO order.
	for _, jobID := range s.jobQueue {
		job, exists := s.jobs[jobID]
		if !exists {
			continue
		}

		if job.AssignedSplits < job.ExpectedSplits {
			job.AssignedSplits++

			taskID := fmt.Sprintf("%s_%d", job.JobID, job.AssignedSplits)

			log.Printf("Assigning task %s (split %d of job %s) to device %s",
				taskID, job.AssignedSplits, job.JobID, req.DeviceId)
			assignment := &pb.TaskAssignment{
				JobId:     job.JobID,
				TaskId:    taskID,
				Operation: job.Operation,
				AData:     job.AData,
				BData:     job.BData,
			}
			return assignment, nil
		}
	}
	return nil, fmt.Errorf("no available tasks at this time")
}
