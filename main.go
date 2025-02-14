package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// Changed dot-import to explicit alias for clarity.
	tango "cactus/tango/src"
	pb "cactus/tango/src/protobuff"
)

func main() {
	// Start listening on TCP port 50051.
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	// Updated to use the 'tango' alias.
	tangoServer := tango.NewServer()
	pb.RegisterTangoServiceServer(grpcServer, tangoServer)

	// Enable reflection for debugging.
	reflection.Register(grpcServer)

	log.Println("Tango server is running on port :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
