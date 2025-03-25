/*
Tango is a product of Cactus Compute, Inc.
This code is proprietary. Do not share the code.
*/
package tango

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ValidateJWT validates a JSON Web Token (JWT) using the provided secret key.
// It checks the token format, decodes the header, payload, and signature,
// verifies the algorithm (HS256), compares the expected and provided signatures,
// and ensures the token has not expired.
// Returns the payload as a map if the token is valid, otherwise an error.
func ValidateJWT(token, secretKey string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	encodedHeader := parts[0]
	encodedPayload := parts[1]
	encodedSignature := parts[2]

	headerBytes, err := base64.RawURLEncoding.DecodeString(encodedHeader)
	if err != nil {
		return nil, errors.New("invalid header encoding")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}

	signature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, errors.New("invalid header JSON")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, errors.New("invalid payload JSON")
	}

	if header["alg"] != "HS256" {
		return nil, errors.New("invalid algorithm")
	}

	signatureInput := fmt.Sprintf("%s.%s", encodedHeader, encodedPayload)
	expectedSignature := generateHmacSha256Signature(signatureInput, secretKey)

	if subtle.ConstantTimeCompare(expectedSignature, signature) != 1 {
		return nil, errors.New("invalid signature")
	}

	if exp, ok := payload["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, errors.New("token expired")
		}
	}

	if consumerID, ok := payload["consumerId"].(string); ok {
		payload["consumerID"] = consumerID
	}

	return payload, nil
}

// generateHmacSha256Signature generates an HMAC-SHA256 signature for the given data
// using the provided secret key. It returns the resulting signature as a byte slice.
func generateHmacSha256Signature(data, secretKey string) []byte {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(data))
	return h.Sum(nil)
}

// TokenInterceptor is a gRPC unary interceptor that validates the JWT provided in the request metadata.
// It retrieves the "cactus-token" from the incoming metadata, fetches the expected JWT secret,
// validates the token using ValidateJWT, and only allows the request to proceed if the token is valid.
// Returns an error if the token is missing or invalid.
func TokenInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("missing metadata")
	}
	tokens := md["cactus-token"]
	if len(tokens) == 0 {
		return nil, errors.New("missing CACTUS_TOKEN")
	}
	JWTSecret, _ := getTangoJWTSecret()
	payload, err := ValidateJWT(tokens[0], JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT: %v", err)
	}

	// Extract consumerID from the payload
	if consumerID, ok := payload["consumerID"].(string); ok {
		// Create a new context with the consumerID
		ctx = context.WithValue(ctx, "consumerID", consumerID)
	}

	return handler(ctx, req)
}
