/*
Tango is a product of Cactus Compute, Inc.
This code is proprietary. Do not share the code.
*/
package tango

import (
	pb "cactus/tango/src/protobuff"
	"sync"
	"time"
)

// server implements the TangoServiceServer interface and manages job processing.
// It maintains a map of active jobs, a job queue, and a read-write mutex for safe concurrent access.
type server struct {
	pb.UnimplementedTangoServiceServer
	jobsMu   sync.RWMutex
	jobs     map[string]*Job
	jobQueue []string
}

// NewServer creates and initializes a new server instance.
// It sets up an empty jobs map and job queue, and starts a background goroutine to reap expired tasks.
func NewServer() *server {
	s := &server{
		jobs:     make(map[string]*Job),
		jobQueue: make([]string, 0),
	}
	go s.reapExpiredTasks()
	return s
}

// reapExpiredTasks periodically scans through all jobs to remove pending tasks that have exceeded their deadlines.
// The interval between scans is defined by the application's configuration.
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

// RemoveDevicePendingTasks removes all pending tasks associated with the specified deviceID from all jobs.
// It ensures thread-safe access by locking the jobs map during the operation.
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
