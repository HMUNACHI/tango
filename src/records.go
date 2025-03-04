package tango

import (
	"fmt"
	"os"
	"path/filepath"
)

func AppendRecord(deviceID string, consumerID string, numElements int32) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	filePath := filepath.Join(cwd, "transaction_cache.csv")

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	record := fmt.Sprintf("%s,%s,%d\n", deviceID, consumerID, numElements)
	_, err = f.WriteString(record)
	return err
}
