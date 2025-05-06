package tango

import (
	"context"
	"fmt"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// Global variables used for caching secrets retrieved from GCP Secret Manager.
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

// getTangoJWTSecret retrieves and caches the Tango JWT secret from GCP Secret Manager.
// It uses sync.Once to ensure that the secret is fetched only once.
// Returns the JWT secret as a string or an error if retrieval fails.
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

// GetTestToken retrieves and caches a test token from GCP Secret Manager.
// It ensures the GCP environment is properly set up and uses sync.Once to fetch the token only once.
// Returns the test token as a string or an error if retrieval fails.
func GetTestToken() (string, error) {
	testTokenOnce.Do(func() {
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

// GetServerSecrets retrieves and caches the server certificate and key from GCP Secret Manager.
// It uses sync.Once to ensure the secrets are fetched only once.
// Returns the server certificate and key as strings, or an error if retrieval fails.
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
