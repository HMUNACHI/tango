package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	tango "cactus/tango/grpc_proto/go"
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

type server struct {
	tango.UnimplementedTangoServiceServer
	mu                sync.Mutex
	registeredDevices map[string]*tango.DeviceInfo
	jobs              map[string]*Job
	jobQueue          []string
}

func newServer() *server {
	return &server{
		registeredDevices: make(map[string]*tango.DeviceInfo),
		jobs:              make(map[string]*Job),
		jobQueue:          make([]string, 0),
	}
}

func (s *server) SubmitTask(ctx context.Context, req *tango.TaskRequest) (*tango.TaskResponse, error) {
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

	return &tango.TaskResponse{
		Accepted: true,
		Message:  "Job submitted successfully.",
	}, nil
}

func (s *server) RegisterDevice(ctx context.Context, info *tango.DeviceInfo) (*tango.DeviceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Registering device %s", info.DeviceId)

	s.registeredDevices[info.DeviceId] = info
	return &tango.DeviceResponse{
		Registered: true,
		Message:    "Device registered successfully.",
	}, nil
}

func (s *server) UpdateDeviceStatus(ctx context.Context, status *tango.DeviceStatus) (*tango.DeviceStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Updating status for device %s", status.DeviceId)
	if device, exists := s.registeredDevices[status.DeviceId]; exists {
		device.AvailableRam = status.AvailableRam
		device.CpuUsage = int32(status.CpuUsage)
		device.InternetSpeed = status.InternetSpeed
		device.IsCharging = status.IsCharging
		return &tango.DeviceStatusResponse{
			Updated: true,
			Message: "Status updated.",
		}, nil
	}
	return &tango.DeviceStatusResponse{
		Updated: false,
		Message: "Device not registered.",
	}, nil
}

func (s *server) FetchTask(ctx context.Context, req *tango.DeviceRequest) (*tango.TaskAssignment, error) {
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
			assignment := &tango.TaskAssignment{
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

func (s *server) ReportResult(ctx context.Context, res *tango.TaskResult) (*tango.ResultResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Received result for task %s of job %s from device %s: %s",
		res.TaskId, res.JobId, res.DeviceId, res.ResultData)

	job, exists := s.jobs[res.JobId]
	if !exists {
		return &tango.ResultResponse{
			Success: false,
			Message: "Job not found.",
		}, nil
	}

	update, err := parseWeights(res.ResultData)
	if err != nil {
		return &tango.ResultResponse{
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
			return &tango.ResultResponse{
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

	return &tango.ResultResponse{
		Success: true,
		Message: "Result received and aggregated if complete.",
	}, nil
}

func (s *server) removeJobFromQueue(jobID string) {
	index := -1
	for i, id := range s.jobQueue {
		if id == jobID {
			index = i
			break
		}
	}
	if index != -1 {
		s.jobQueue = append(s.jobQueue[:index], s.jobQueue[index+1:]...)
	}
}

func parseWeights(data string) ([]float32, error) {
	parts := strings.Split(data, ",")
	weights := make([]float32, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		val, err := strconv.ParseFloat(trimmed, 32)
		if err != nil {
			return nil, err
		}
		weights = append(weights, float32(val))
	}
	return weights, nil
}

func (s *server) GetJobStatus(ctx context.Context, req *tango.JobStatusRequest) (*tango.JobStatusReply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[req.JobId]
	if !exists {
		return &tango.JobStatusReply{
			IsComplete: true,
			Message:    "Job not found (possible completion).",
		}, nil
	}
	if job.ReceivedUpdates >= job.ExpectedSplits {
		return &tango.JobStatusReply{
			IsComplete:   true,
			Message:      "Job is complete.",
			FinalWeights: job.SumWeights,
		}, nil
	}
	return &tango.JobStatusReply{
		IsComplete: false,
		Message:    "Job is still in progress.",
	}, nil
}

func main() {
	// Start listening on TCP port 50051.
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	tangoServer := newServer()
	tango.RegisterTangoServiceServer(grpcServer, tangoServer)

	// Enable reflection for debugging.
	reflection.Register(grpcServer)

	log.Println("Tango server is running on port :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
