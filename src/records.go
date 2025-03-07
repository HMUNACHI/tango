/*
Tango is a product of Cactus Compute, Inc.
This code is proprietary. Do not share the code.
*/
package tango

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
)

// AppendRecord appends a new transaction record to the local CSV file "transaction_cache.csv".
// It records the device ID, consumer ID, and the number of floating-point operations (flops) performed.
// If the file does not exist, it is created in the current working directory.
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

// UploadRecordsToGCS uploads the local "transaction_cache.csv" file to a Google Cloud Storage bucket.
// The bucket name is taken from the application configuration. The file is uploaded with a name based on the provided jobID.
// After a successful upload, the local CSV file is cleared.
// Returns an error if any step in the process fails.
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

	// Clear the local CSV file after successful upload.
	if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to clear transaction_cache.csv: %v", err)
	}
	return nil
}
