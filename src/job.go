package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
)

type Job struct {
	JobID           string
	Operation       string
	AData           []byte
	BData           []byte
	m               int32
	n               int32
	d               int32
	ExpectedSplits  int
	AssignedSplits  int
	ReceivedUpdates int
	Results         map[int][]byte
	FinalResult     []byte
	ScaleBytes      []byte
	ScaleScalar     float32
	// New: tracks pending tasks with deadlines keyed by shard index.
	PendingTasks map[int]TimeDeadline
}

// TimeDeadline encapsulates the deadline for a task.
type TimeDeadline struct {
	Deadline int64 // Unix nano timestamp
}

func (s *server) GetJobStatus(ctx context.Context, req *pb.JobStatusRequest) (*pb.JobStatusReply, error) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()

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
			FinalResult: job.FinalResult,
		}, nil
	}
	return &pb.JobStatusReply{
		IsComplete: false,
		Message:    "Job is still in progress.",
	}, nil
}
