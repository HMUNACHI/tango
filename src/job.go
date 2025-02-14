package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
)

type Job struct {
	JobID            string
	ComputationGraph string
	Data             []byte
	ExpectedSplits   int
	AssignedSplits   int
	ReceivedUpdates  int
	SumWeights       []float32
	WeightLength     int
}

func (s *server) GetJobStatus(ctx context.Context, req *pb.JobStatusRequest) (*pb.JobStatusReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[req.JobId]
	if !exists {
		return &pb.JobStatusReply{
			IsComplete: true,
			Message:    "Job not found (possible completion).",
		}, nil
	}
	if job.ReceivedUpdates >= job.ExpectedSplits {
		return &pb.JobStatusReply{
			IsComplete:   true,
			Message:      "Job is complete.",
			FinalWeights: job.SumWeights,
		}, nil
	}
	return &pb.JobStatusReply{
		IsComplete: false,
		Message:    "Job is still in progress.",
	}, nil
}
