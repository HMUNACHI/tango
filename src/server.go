package tango

import (
	pb "cactus/tango/src/protobuff"
	"sync"
	"time"
)

type server struct {
	pb.UnimplementedTangoServiceServer
	devicesMu         sync.Mutex
	jobsMu            sync.Mutex
	registeredDevices map[string]*pb.DeviceInfo
	jobs              map[string]*Job
	jobQueue          []string
}

func NewServer() *server {
	s := &server{
		registeredDevices: make(map[string]*pb.DeviceInfo),
		jobs:              make(map[string]*Job),
		jobQueue:          make([]string, 0),
	}
	// Launch background reaper
	go s.reapExpiredTasks()
	return s
}

// reapExpiredTasks periodically scans jobs to remove tasks that timed out.
func (s *server) reapExpiredTasks() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UnixNano()
		s.jobsMu.Lock()
		for _, job := range s.jobs {
			for shard, td := range job.PendingTasks {
				if now > td.Deadline {
					// Timed-out task: remove pending record and allow re-assignment.
					delete(job.PendingTasks, shard)
					// Optionally log and adjust AssignedSplits if needed.
				}
			}
		}
		s.jobsMu.Unlock()
	}
}
