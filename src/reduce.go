package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
	"log"
)

func (s *server) ReportResult(ctx context.Context, res *pb.TaskResult) (*pb.ResultResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Received result for task %s of job %s from device %s: %s",
		res.TaskId, res.JobId, res.DeviceId, res.ResultData)

	job, exists := s.jobs[res.JobId]
	if !exists {
		return &pb.ResultResponse{
			Success: false,
			Message: "Job not found.",
		}, nil
	}

	update, err := parseWeights(res.ResultData)
	if err != nil {
		return &pb.ResultResponse{
			Success: false,
			Message: "Failed to parse weight update.",
		}, nil
	}

	if job.SumWeights == nil {
		job.SumWeights = make([]float32, len(update))
		job.WeightLength = len(update)
		copy(job.SumWeights, update)
	} else {
		if len(update) != job.WeightLength {
			return &pb.ResultResponse{
				Success: false,
				Message: "Weight update length mismatch.",
			}, nil
		}
		// Aggregate: element-wise addition.
		for i, w := range update {
			job.SumWeights[i] += w
		}
	}

	job.ReceivedUpdates++

	if job.ReceivedUpdates == job.ExpectedSplits {
		aggregated := make([]float32, job.WeightLength)
		for i, sum := range job.SumWeights {
			aggregated[i] = sum / float32(job.ExpectedSplits)
		}
		log.Printf("Job %s complete. Aggregated weights: %v", job.JobID, aggregated)
		delete(s.jobs, job.JobID)
		s.removeJobFromQueue(job.JobID)
	}

	return &pb.ResultResponse{
		Success: true,
		Message: "Result received and aggregated if complete.",
	}, nil
}
