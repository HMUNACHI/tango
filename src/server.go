package tango

import (
	pb "cactus/tango/src/protobuff"
	"sync"
)

type server struct {
	pb.UnimplementedTangoServiceServer
	mu                sync.Mutex
	registeredDevices map[string]*pb.DeviceInfo
	jobs              map[string]*Job
	jobQueue          []string
}

func NewServer() *server {
	return &server{
		registeredDevices: make(map[string]*pb.DeviceInfo),
		jobs:              make(map[string]*Job),
		jobQueue:          make([]string, 0),
	}
}
