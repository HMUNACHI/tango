package tango

import (
	"context"
	"sync"
	pb "tango/tango/src/protobuff"
)

// Job represents a computation job in the Tango system.
// It holds all relevant information for processing a matrix multiplication task,
// including input data, expected processing splits, results, and synchronization primitives.
type Job struct {
	JobID           string               // Unique identifier for the job.
	Operation       string               // The operation to be performed (e.g., "scaled_matmul").
	AData           []byte               // Serialized data for matrix A.
	BData           []byte               // Serialized data for matrix B.
	m               int32                // Number of rows in matrix A.
	n               int32                // Number of columns in matrix B.
	d               int32                // Shared dimension for matrices A and B.
	ExpectedSplits  int                  // Total number of expected splits/tasks.
	RowSplits       int32                // Number of row splits.
	ColSplits       int32                // Number of column splits.
	AssignedSplits  int                  // Number of splits assigned for processing.
	ReceivedUpdates int                  // Number of task updates received.
	Results         map[int][]byte       // Map storing partial results keyed by task index.
	FinalResult     []byte               // Serialized final result after aggregation.
	ScaleBytes      []byte               // Serialized scale factor, if provided.
	ScaleScalar     float32              // Numeric scale factor applied to the result.
	PendingTasks    map[int]TimeDeadline // Map of pending tasks with their deadlines.
	mu              sync.Mutex           // Mutex to protect concurrent access to the job.
}

// TimeDeadline represents the deadline information for a pending task.
// It includes the deadline timestamp and the associated device ID.
type TimeDeadline struct {
	Deadline int64  // Unix timestamp representing the task deadline.
	DeviceID string // Identifier of the device responsible for the task.
}

// GetJobStatus returns the current status of the job identified by req.JobId.
// It locks the jobs map for thread-safe access, and checks whether the job exists.
// If the job is not found, it assumes completion (possibly already aggregated).
// If the number of received updates meets or exceeds the expected splits,
// it returns a reply indicating that the job is complete, along with the final result.
// Otherwise, it indicates that the job is still in progress.
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
