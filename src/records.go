package tango

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
)

func AppendRecord(deviceID string, consumerID string, flops int32) error {
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

	record := fmt.Sprintf("%s,%s,%d\n", deviceID, consumerID, flops)
	_, err = f.WriteString(record)
	return err
}

func UploadRecordsToGCS(jobID string) error {
	bucketName := AppConfig.GCP.RecordsBucket
	if bucketName == "" {
		return fmt.Errorf("record bucket not set in config")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	filePath := filepath.Join(cwd, "transaction_cache.csv")

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	objectName := fmt.Sprintf("%s.csv", jobID)
	writer := client.Bucket(bucketName).Object(objectName).NewWriter(ctx)
	if _, err := io.Copy(writer, f); err != nil {
		writer.Close()
		return fmt.Errorf("failed to copy file contents to GCS: %v", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close GCS writer: %v", err)
	}

	if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to clear transaction_cache.csv: %v", err)
	}
	return nil
}
