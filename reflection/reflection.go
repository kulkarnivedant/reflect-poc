package reflection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc"
	v1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type GRPCReflectionHelper struct {
	conn   *grpc.ClientConn
	client v1.ServerReflectionClient
}

// NewGRPCReflectionHelper creates a new helper instance
func NewGRPCReflectionHelper(conn *grpc.ClientConn) *GRPCReflectionHelper {
	return &GRPCReflectionHelper{
		conn:   conn,
		client: v1.NewServerReflectionClient(conn),
	}
}

// GetFileDescriptor retrieves the full file descriptor from a gRPC connection
func (g *GRPCReflectionHelper) GetFileDescriptor(ctx context.Context, filename string) (*descriptorpb.FileDescriptorProto, error) {
	stream, err := g.client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reflection stream: %w", err)
	}

	request := &v1.ServerReflectionRequest{
		MessageRequest: &v1.ServerReflectionRequest_FileByFilename{
			FileByFilename: filename,
		},
	}

	if err := stream.Send(request); err != nil {
		return nil, fmt.Errorf("failed to send reflection request: %w", err)
	}

	response, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive reflection response: %w", err)
	}

	fileDescBytes := response.GetFileDescriptorResponse().GetFileDescriptorProto()[0]
	fileDesc := &descriptorpb.FileDescriptorProto{}
	
	if err := proto.Unmarshal(fileDescBytes, fileDesc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file descriptor: %w", err)
	}

	return fileDesc, nil
}

// formatJSON converts a protobuf message to a pretty-printed JSON string
func (g *GRPCReflectionHelper) formatJSON(msg proto.Message) (string, error) {
	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, jsonBytes, "", "  "); err != nil {
		return "", fmt.Errorf("failed to indent JSON: %w", err)
	}

	return prettyJSON.String(), nil
}

// GetFileDescriptorBySymbol retrieves file descriptor using service and method name
func (g *GRPCReflectionHelper) GetFileDescriptorBySymbol(ctx context.Context, serviceName, methodName string) (*descriptorpb.FileDescriptorProto, error) {
	stream, err := g.client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reflection stream: %w", err)
	}

	query := fmt.Sprintf("reflect.%s.%s", serviceName, methodName)
	request := &v1.ServerReflectionRequest{
		MessageRequest: &v1.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: query,
		},
	}

	if err := stream.Send(request); err != nil {
		return nil, fmt.Errorf("failed to send reflection request: %w", err)
	}

	response, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive reflection response: %w", err)
	}

	fileDescBytes := response.GetFileDescriptorResponse().GetFileDescriptorProto()[0]
	fileDesc := &descriptorpb.FileDescriptorProto{}
	
	if err := proto.Unmarshal(fileDescBytes, fileDesc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file descriptor: %w", err)
	}

	_, err = g.formatJSON(fileDesc.ProtoReflect().Interface())
	if err != nil {
		return nil, fmt.Errorf("failed to format JSON: %w", err)
	}


	return fileDesc, nil
}

// GetMethodDescriptor retrieves the method descriptor for a specific service and method
func (g *GRPCReflectionHelper) GetMethodDescriptor(ctx context.Context, serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	fileDesc, err := g.GetFileDescriptorReflect(ctx, serviceName, methodName)
	if err != nil {
		return nil, err
	}

	services := fileDesc.Services()
	service := services.ByName(protoreflect.Name(serviceName))
	if service == nil {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, fmt.Errorf("method %s not found in service %s", methodName, serviceName)
	}

	return method, nil
}

// GetInputOutputTypes retrieves the input and output message descriptors for a method
func (g *GRPCReflectionHelper) GetInputOutputTypes(ctx context.Context, serviceName, methodName string) (protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, error) {
	fileDesc, err := g.GetFileDescriptorReflect(ctx, serviceName, methodName)
	if err != nil {
		return nil, nil, err
	}

	methodDesc, err := g.GetMethodDescriptor(ctx, serviceName, methodName)
	if err != nil {
		return nil, nil, err
	}

	return methodDesc.Input(), methodDesc.Output(), nil


	inputDesc := fileDesc.Messages().ByName(protoreflect.Name("TestMessageRequest"))
	outputDesc := fileDesc.Messages().ByName(protoreflect.Name("TestMessageResponse"))

	fmt.Println("found inputDesc: ", inputDesc.Name())
	fmt.Println("found outputDesc: ", outputDesc.Name())

	if inputDesc == nil || outputDesc == nil {
		return nil, nil, fmt.Errorf("failed to find input or output message descriptors")
	}

	return inputDesc, outputDesc, nil
}

// PopulateMessageFromJSON populates a dynamic protobuf message with JSON data
func (g *GRPCReflectionHelper) PopulateMessageFromJSON(msg *dynamicpb.Message, jsonData []byte) error {
	// Log input JSON and validate it's properly formatted
	fmt.Printf("Input JSON: %s\n", string(jsonData))
	
	// Create UnmarshalOptions with more lenient settings
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
		AllowPartial:   true,
	}

	if err := unmarshaler.Unmarshal(jsonData, msg); err != nil {
		// Add more detailed error information
		fmt.Printf("Error details - Message type: %s\n", msg.Descriptor().FullName())
		fmt.Printf("Expected fields:\n")
		fields := msg.Descriptor().Fields()
		for i := 0; i < fields.Len(); i++ {
			fd := fields.Get(i)
			fmt.Printf("  - Name: %s, JSONName: %s, Number: %d, Type: %s\n",
				fd.Name(), fd.JSONName(), fd.Number(), fd.Kind())
		}
		return fmt.Errorf("failed to unmarshal JSON into dynamic message: %w", err)
	}

	// Log the successfully populated message
	marshaler := protojson.MarshalOptions{Indent: "  "}
	if debugJSON, err := marshaler.Marshal(msg); err == nil {
		fmt.Printf("Successfully populated message:\n%s\n", string(debugJSON))
	}

	return nil
}

// Helper function to print message fields recursively
func printMessageFields(desc protoreflect.MessageDescriptor, indent string) {
	fields := desc.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		fmt.Printf("%s- Field: %s (json: %s)\n", indent, fd.Name(), fd.JSONName())
		
		// If the field is a message type, recursively print its fields
		if fd.Kind() == protoreflect.MessageKind && !fd.IsMap() && !fd.IsList() {
			fmt.Printf("%s  Message fields:\n", indent)
			printMessageFields(fd.Message(), indent+"    ")
		}
	}
} 

// ConvertMessageToJSON converts a dynamic protobuf message to JSON
func (g *GRPCReflectionHelper) ConvertMessageToJSON(msg *dynamicpb.Message) (string, error) {
	marshaler := protojson.MarshalOptions{
		Indent: "  ",
	}
	jsonBytes, err := marshaler.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// GetFileDescriptorReflect retrieves the full file descriptor from a gRPC connection and returns it as protoreflect.FileDescriptor
func (g *GRPCReflectionHelper) GetFileDescriptorReflect(ctx context.Context, serviceName, methodName string) (protoreflect.FileDescriptor, error) {
	fileDescProto, err := g.GetFileDescriptorBySymbol(ctx, serviceName, methodName)
	if err != nil {
		return nil, fmt.Errorf("failed to get file descriptor proto: %w", err)
	}

	files := &protoregistry.Files{}
	fileDesc, err := protodesc.NewFile(fileDescProto, files)
	if err != nil {
		return nil, fmt.Errorf("failed to create file descriptor: %w", err)
	}

	return fileDesc, nil
}
