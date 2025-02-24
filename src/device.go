package tango

import (
	pb "cactus/tango/src/protobuff"
	"context"
	"log"
)

func (s *server) RegisterDevice(ctx context.Context, info *pb.DeviceInfo) (*pb.DeviceResponse, error) {
	s.devicesMu.Lock()
	defer s.devicesMu.Unlock()

	log.Printf("Registering device %s", info.DeviceId)

	s.registeredDevices[info.DeviceId] = info
	return &pb.DeviceResponse{
		Registered: true,
		Message:    "Device registered successfully.",
	}, nil
}

func (s *server) UpdateDeviceStatus(ctx context.Context, status *pb.DeviceStatus) (*pb.DeviceStatusResponse, error) {
	s.devicesMu.Lock()
	defer s.devicesMu.Unlock()

	log.Printf("Updating status for device %s", status.DeviceId)
	if device, exists := s.registeredDevices[status.DeviceId]; exists {
		device.AvailableRam = status.AvailableRam
		device.CpuUsage = int32(status.CpuUsage)
		device.InternetSpeed = status.InternetSpeed
		device.IsCharging = status.IsCharging
		return &pb.DeviceStatusResponse{
			Updated: true,
			Message: "Status updated.",
		}, nil
	}
	return &pb.DeviceStatusResponse{
		Updated: false,
		Message: "Device not registered.",
	}, nil
}
