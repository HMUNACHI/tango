package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
)

// Job represents a distributed computation job.
// - Operation specifies the computation to perform (e.g., "matmul").
// - AData and BData hold the binary fp16 data for operands.
// - ExpectedSplits is the total number of tasks for this job.
// - AssignedSplits and ReceivedUpdates track task distribution and result progress.
// - Results stores individual C_shard outputs (binary data).
// - FinalResult holds the aggregated result after all shards have been processed.
type Job struct {
	JobID           string
	Operation       string
	AData           []byte
	BData           []byte
	ExpectedSplits  int
	AssignedSplits  int
	ReceivedUpdates int
	Results         map[int][]byte
	FinalResult     []byte // New: holds the final aggregated C_shards.
}

// GetJobStatus returns the current status of the job.
// When all shards have been received, it returns the FinalResult.
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
			IsComplete:  true,
			Message:     "Job is complete.",
			FinalResult: job.FinalResult, // Return the aggregated C_shards.
		}, nil
	}
	return &pb.JobStatusReply{
		IsComplete: false,
		Message:    "Job is still in progress.",
	}, nil
}
