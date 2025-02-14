package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
	"fmt"
	"log"
)

func (s *server) SubmitTask(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Received new job submission: %s expecting %d splits", req.JobId, req.NumSplits)

	job := &Job{
		JobID:            req.JobId,
		ComputationGraph: req.ComputationGraph,
		Data:             req.Data,
		ExpectedSplits:   int(req.NumSplits),
		AssignedSplits:   0,
		ReceivedUpdates:  0,
	}
	s.jobs[req.JobId] = job
	s.jobQueue = append(s.jobQueue, req.JobId)

	return &pb.TaskResponse{
		Accepted: true,
		Message:  "Job submitted successfully.",
	}, nil
}

func (s *server) FetchTask(ctx context.Context, req *pb.DeviceRequest) (*pb.TaskAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Device %s requesting a task", req.DeviceId)
	// To-Do: This iterate over the jobQueue in FIFO order, has to use device priority
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
				JobId:            job.JobID,
				TaskId:           taskID,
				ComputationGraph: job.ComputationGraph,
				Data:             job.Data,
			}
			return assignment, nil
		}
	}
	return nil, fmt.Errorf("no available tasks at this time")
}
