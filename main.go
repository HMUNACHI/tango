package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	tango "cactus/tango/src"
	pb "cactus/tango/src/protobuff"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(tango.TokenInterceptor))
	tangoServer := tango.NewServer()
	pb.RegisterTangoServiceServer(grpcServer, tangoServer)
	reflection.Register(grpcServer)

	log.Println("Tango server is running on port :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
