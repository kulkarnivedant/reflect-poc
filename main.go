package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	server "github.com/vedantkulkarni/reflect-poc/server"
	service_proto "github.com/vedantkulkarni/reflect-poc/service-proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	v1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
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


	request:=&v1.ServerReflectionRequest{
		MessageRequest: &v1.ServerReflectionRequest_FileByFilename{
			FileByFilename: "reflect.proto",
		},
	}
	// Send the initial request to get the list of services
	if err := stream.Send(request); err != nil {
		log.Fatalf("failed to send reflection request: %v", err)
	}

	// Receive the response, json encode it, and print the list of services
	response, err := stream.Recv()
	if err != nil {
		log.Fatalf("failed to receive reflection response: %v", err)
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("failed to marshal reflection response to JSON: %v", err)
	}
	fmt.Println("JSON Response:", string(jsonResponse))

	fmt.Println("================================================")
	fmt.Println(response.GetFileDescriptorResponse().String())
	fmt.Println("================================================")
	
	// Get the file descriptor proto bytes
	fileDescBytes := response.GetFileDescriptorResponse().GetFileDescriptorProto()[0]
	
	// Create a new FileDescriptorProto
	fileDesc := &descriptorpb.FileDescriptorProto{}
	
	// Unmarshal the bytes into the FileDescriptorProto
	if err := proto.Unmarshal(fileDescBytes, fileDesc); err != nil {
		log.Fatalf("failed to unmarshal file descriptor: %v", err)
	}
	
	// Convert to JSON
	jsonBytes, err := protojson.Marshal(fileDesc)
	if err != nil {
		log.Fatalf("failed to marshal to JSON: %v", err)
	}
	
	// Print the JSON in a pretty format
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, jsonBytes, "", "  "); err != nil {
		log.Fatalf("failed to indent JSON: %v", err)
	}
	fmt.Println("File Descriptor as JSON:")
	fmt.Println(prettyJSON.String())

	fmt.Println("Services provided by the gRPC server:")

}
