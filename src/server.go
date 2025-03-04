package tango

import (
	pb "cactus/tango/src/protobuff"
	"sync"
	"time"
)

type server struct {
	pb.UnimplementedTangoServiceServer
	jobsMu   sync.RWMutex
	jobs     map[string]*Job
	jobQueue []string
}

func NewServer() *server {
	s := &server{
		jobs:     make(map[string]*Job),
		jobQueue: make([]string, 0),
	}
	go s.reapExpiredTasks()
	return s
}

func (s *server) reapExpiredTasks() {
	interval := time.Duration(AppConfig.Task.ReaperIntervalMilliseconds) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UnixNano()
		s.jobsMu.Lock()
		for _, job := range s.jobs {
			for shard, td := range job.PendingTasks {
				if now > td.Deadline {
					delete(job.PendingTasks, shard)
				}
			}
		}
		s.jobsMu.Unlock()
	}
}

func (s *server) RemoveDevicePendingTasks(deviceID string) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	for _, job := range s.jobs {
		for shard, td := range job.PendingTasks {
			if td.DeviceID == deviceID {
				delete(job.PendingTasks, shard)
			}
		}
	}
}
