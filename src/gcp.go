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
	cachedTangoJWTSecret   string
	cachedSecretErr        error
	jwtSecretOnce          sync.Once
	cachedTestToken        string
	cachedTestTokenErr     error
	testTokenOnce          sync.Once
	cachedServerCrt        string
	cachedServerKey        string
	cachedServerSecretsErr error
	serverSecretsOnce      sync.Once
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

func GetServerSecrets() (string, string, error) {
	serverSecretsOnce.Do(func() {
		ctx := context.Background()
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			cachedServerSecretsErr = fmt.Errorf("failed to create secret manager client: %v", err)
			return
		}
		defer client.Close()

		crtReq := &secretmanagerpb.AccessSecretVersionRequest{
			Name: AppConfig.GCP.ServerCrt,
		}
		crtResp, err := client.AccessSecretVersion(ctx, crtReq)
		if err != nil {
			cachedServerSecretsErr = fmt.Errorf("failed to access server_crt: %v", err)
			return
		}
		cachedServerCrt = string(crtResp.Payload.Data)

		keyReq := &secretmanagerpb.AccessSecretVersionRequest{
			Name: AppConfig.GCP.ServerKey,
		}
		keyResp, err := client.AccessSecretVersion(ctx, keyReq)
		if err != nil {
			cachedServerSecretsErr = fmt.Errorf("failed to access server_key: %v", err)
			return
		}
		cachedServerKey = string(keyResp.Payload.Data)
	})
	return cachedServerCrt, cachedServerKey, cachedServerSecretsErr
}
