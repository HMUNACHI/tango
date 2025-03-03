package tango

func isAllowedDevice(deviceID string) bool {
	allowedDevices := map[string]bool{
		"device1": true,
		"device2": true,
	}
	return allowedDevices[deviceID]
}
