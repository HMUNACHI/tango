/*
Tango is a product of Cactus Compute, Inc.
This code is proprietary. Do not share the code.
*/
package main

import (
	"crypto/tls"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	tango "cactus/tango/src"
	pb "cactus/tango/src/protobuff"
)

// main initializes the Tango gRPC server with TLS encryption,
// sets up required GCP configurations and secrets, and begins
// listening on port :50051 for incoming connections.
func main() {
	if err := tango.SetupGCP(); err != nil {
		log.Fatalf("failed to setup GCP: %v", err)
	}

	crtStr, keyStr, err := tango.GetServerSecrets()
	if err != nil {
		log.Fatalf("failed to get server secrets: %v", err)
	}

	cert, err := tls.X509KeyPair([]byte(crtStr), []byte(keyStr))
	if err != nil {
		log.Fatalf("failed to load server key pair: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	creds := credentials.NewTLS(tlsConfig)

	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(tango.TokenInterceptor),
	)
	tangoServer := tango.NewServer()
	pb.RegisterTangoServiceServer(grpcServer, tangoServer)
	reflection.Register(grpcServer)

	log.Printf("Tango server is running on port :%s with TLS", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
