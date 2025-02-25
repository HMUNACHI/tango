package tango

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func validateToken(token string) bool {
	for _, t := range AppConfig.Tokens.Approved {
		if token == t {
			return true
		}
	}
	return false
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
	if !validateToken(tokens[0]) {
		return nil, errors.New("invalid CACTUS_TOKEN")
	}
	return handler(ctx, req)
}
