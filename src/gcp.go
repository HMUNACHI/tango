package tango

import (
	"context"
	"fmt"
	"os"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

func SetupGCP() error {
	if os.Getenv("ENV") != "production" && os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		defaultCredFile := "cactus-gcp-credentials.json"
		if _, err := os.Stat(defaultCredFile); err == nil {
			os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", defaultCredFile)
		} else {
			return fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS env variable not set and default (%s) not found", defaultCredFile)
		}
	}
	return nil
}

var (
	cachedTangoJWTSecret string
	cachedSecretErr      error
	jwtSecretOnce        sync.Once
	cachedTestToken      string
	cachedTestTokenErr   error
	testTokenOnce        sync.Once
)

func getTangoJWTSecret() (string, error) {
	jwtSecretOnce.Do(func() {
		ctx := context.Background()
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			cachedSecretErr = fmt.Errorf("failed to create secret manager client: %v", err)
			return
		}
		defer client.Close()

		secretName := "projects/263237337139/secrets/TangoJWTSecret/versions/latest"
		req := &secretmanagerpb.AccessSecretVersionRequest{
			Name: secretName,
		}
		result, err := client.AccessSecretVersion(ctx, req)
		if err != nil {
			cachedSecretErr = fmt.Errorf("failed to access secret version: %v", err)
			return
		}
		cachedTangoJWTSecret = string(result.Payload.Data)
	})
	return cachedTangoJWTSecret, cachedSecretErr
}

func GetTestToken() (string, error) {
	testTokenOnce.Do(func() {
		if err := SetupGCP(); err != nil {
			cachedTestTokenErr = fmt.Errorf("failed to setup GCP: %v", err)
			return
		}
		ctx := context.Background()
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			cachedTestTokenErr = fmt.Errorf("failed to create secret manager client: %v", err)
			return
		}
		defer client.Close()
		secretName := AppConfig.GCP.TestTokenSecretName
		req := &secretmanagerpb.AccessSecretVersionRequest{
			Name: secretName,
		}
		result, err := client.AccessSecretVersion(ctx, req)
		if err != nil {
			cachedTestTokenErr = fmt.Errorf("failed to access secret version: %v", err)
			return
		}
		cachedTestToken = string(result.Payload.Data)
	})
	return cachedTestToken, cachedTestTokenErr
}
