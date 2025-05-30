/*
Tango is a product of tango Compute, Inc.
This code is proprietary. Do not share the code.
*/
package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	tango "tango/tango/src"
	pb "tango/tango/src/protobuff"
)

func main() {
	MESSAGE_LIMIT := 100 * 512 * 512
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	listenAddress := "0.0.0.0:" + port
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", listenAddress, err)
	}

	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(MESSAGE_LIMIT),
		grpc.MaxSendMsgSize(MESSAGE_LIMIT),
	}
	grpcServer := grpc.NewServer(
		append(opts, grpc.UnaryInterceptor(tango.TokenInterceptor))...,
	)
	tangoServer := tango.NewServer()
	pb.RegisterTangoServiceServer(grpcServer, tangoServer)
	reflection.Register(grpcServer)

	log.Printf("Tango server is running on %s without TLS", listenAddress)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
