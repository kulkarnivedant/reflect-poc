package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	server "github.com/vedantkulkarni/reflect-poc/server"
	service_proto "github.com/vedantkulkarni/reflect-poc/service-proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	v1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
)

func main() {
	//start grpc server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	testService := &server.MyTestService{}

	service_proto.RegisterTestServiceServer(grpcServer, testService)
	syncService := &server.MySyncService{}
	service_proto.RegisterSyncServiceServer(grpcServer, syncService)
	reflection.Register(grpcServer)

	go createClient()

	// Add error handling for Serve
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// create a new grpc client to fetch server methods based on grpc reflection
func createClient() {
	time.Sleep(2 * time.Second)

	conn, err := grpc.Dial(":50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	// Create a new reflection client
	reflectionClient := v1.NewServerReflectionClient(conn)

	// Call the ServerReflectionInfo RPC to get the list of services
	stream, err := reflectionClient.ServerReflectionInfo(context.Background())
	if err != nil {
		log.Fatalf("failed to get server reflection info: %v", err)
	}


	// Send the initial request to get the list of services
	if err := stream.Send(&v1.ServerReflectionRequest{
		MessageRequest: &v1.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: "reflect.TestService[Test]",
		},
	}); err != nil {
		log.Fatalf("failed to send reflection request: %v", err)
	}

	// Receive the response and print the list of services
	response, err := stream.Recv()
	if err != nil {
		log.Fatalf("failed to receive reflection response: %v", err)
	}
	fmt.Println(response.GetFileDescriptorResponse())

	

	fmt.Println("Services provided by the gRPC server:")

}
