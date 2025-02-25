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
	// use configured reaper interval from AppConfig.Task.ReaperIntervalMilliseconds
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
