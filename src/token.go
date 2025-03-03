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

	return payload, nil
}

func generateHmacSha256Signature(data, secretKey string) []byte {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(data))
	return h.Sum(nil)
}

func TokenInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("missing metadata")
	}
	tokens := md["cactus-token"]
	if len(tokens) == 0 {
		return nil, errors.New("missing CACTUS_TOKEN")
	}
	JWTSecret, err := getTangoJWTSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to get JWT secret: %v", err)
	}
	_, err = ValidateJWT(tokens[0], JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT: %v", err)
	}
	return handler(ctx, req)
}
