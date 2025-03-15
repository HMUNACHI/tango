/*
Tango is a product of Cactus Compute, Inc.
This code is proprietary. Do not share the code.
*/
package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	tango "cactus/tango/src"
	pb "cactus/tango/src/protobuff"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	listenAddress := "0.0.0.0:" + port
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", listenAddress, err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tango.TokenInterceptor),
	)
	tangoServer := tango.NewServer()
	pb.RegisterTangoServiceServer(grpcServer, tangoServer)
	reflection.Register(grpcServer)

	log.Printf("Tango server is running on %s without TLS", listenAddress)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
