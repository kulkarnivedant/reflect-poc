package client

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	bridge "github.com/golain-io/mqtt-bridge"
	reflection "github.com/vedantkulkarni/reflect-poc/reflection"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ReflectionClient struct {
	conn   *grpc.ClientConn
	helper *reflection.GRPCReflectionHelper
}

func NewReflectionClient(conn *grpc.ClientConn) *ReflectionClient {
	conn, _ = GetNewGRPCMQTTConnection("echo-service1")
	return &ReflectionClient{
		conn:   conn,
		helper: reflection.NewGRPCReflectionHelper(conn),
	}
}

// Test unary RPC
func (drs *ReflectionClient) TestUnaryRPC(ctx context.Context) {
	serviceName := "TestService"
	methodName := "Test"

	// Get the method descriptor
	_, err := drs.helper.GetMethodDescriptor(ctx, serviceName, methodName)
	if err != nil {
		fmt.Println("Error getting method descriptor:", err)
		return
	}

	inputDesc, outputDesc, err := drs.helper.GetInputOutputTypes(ctx, serviceName, methodName)
	if err != nil {
		fmt.Println("Error getting input/output types:", err)
		return
	}

	// Create a new message using the input descriptor
	message := dynamicpb.NewMessage(inputDesc)

	// Create the input JSON with the correct message structure
	dummyJson := `{
		"message": "This is a Unary RPC test message!"
	}`

	err = protojson.Unmarshal([]byte(dummyJson), message)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}

	fullMethodName := fmt.Sprintf("/%s.%s/%s", "reflect", serviceName, methodName)

	// Create response message using output descriptor
	response := dynamicpb.NewMessage(outputDesc)

	// Invoke the RPC
	err = drs.conn.Invoke(ctx, fullMethodName, message, response)
	if err != nil {
		fmt.Println("Error invoking RPC:", err)
		return
	}

	// Format and print the response
	responseJson, err := protojson.Marshal(response)
	if err != nil {
		fmt.Println("Error marshaling response:", err)
		return
	}
	fmt.Println("Response:", string(responseJson))
}

// Test server streaming RPC
func (drs *ReflectionClient) TestServerStreamRPC(ctx context.Context) {
	serviceName := "TestService"
	methodName := "TestServerStream"

	inputDesc, outputDesc, err := drs.helper.GetInputOutputTypes(ctx, serviceName, methodName)
	if err != nil {
		fmt.Println("Error getting input/output types:", err)
		return
	}

	// Create input message
	message := dynamicpb.NewMessage(inputDesc)
	dummyJson := `{
        "message": "Start server streaming!"
    }`

	err = protojson.Unmarshal([]byte(dummyJson), message)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}

	fullMethodName := fmt.Sprintf("/%s.%s/%s", "reflect", serviceName, methodName)

	// Create server stream
	stream, err := drs.conn.NewStream(ctx, &grpc.StreamDesc{
		StreamName:    methodName,
		ServerStreams: true,
	}, fullMethodName)
	if err != nil {
		fmt.Println("Error creating stream:", err)
		return
	}

	// Send the initial message
	if err := stream.SendMsg(message); err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	// Receive messages
	for {
		response := dynamicpb.NewMessage(outputDesc)
		err := stream.RecvMsg(response)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error receiving message:", err)
			return
		}

		responseJson, err := protojson.Marshal(response)
		if err != nil {
			fmt.Println("Error marshaling response:", err)
			continue
		}
		fmt.Println("Received:", string(responseJson))
	}
}

// Test client streaming RPC
func (drs *ReflectionClient) TestClientStreamRPC(ctx context.Context) {
	serviceName := "TestService"
	methodName := "TestClientStream"

	inputDesc, outputDesc, err := drs.helper.GetInputOutputTypes(ctx, serviceName, methodName)
	if err != nil {
		fmt.Println("Error getting input/output types:", err)
		return
	}

	fullMethodName := fmt.Sprintf("/%s.%s/%s", "reflect", serviceName, methodName)

	// Create client stream
	stream, err := drs.conn.NewStream(ctx, &grpc.StreamDesc{
		StreamName:    methodName,
		ClientStreams: true,
	}, fullMethodName)
	if err != nil {
		fmt.Println("Error creating stream:", err)
		return
	}

	// Send multiple messages
	for i := 1; i <= 3; i++ {
		message := dynamicpb.NewMessage(inputDesc)
		dummyJson := fmt.Sprintf(`{
            "message": "Client stream message %d"
        }`, i)

		err = protojson.Unmarshal([]byte(dummyJson), message)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			return
		}

		if err := stream.SendMsg(message); err != nil {
			fmt.Println("Error sending message:", err)
			return
		}
	}

	// send a final message to close the stream
	stream.CloseSend()

	// Receive final response
	response := dynamicpb.NewMessage(outputDesc)
	if err := stream.RecvMsg(response); err != nil {
		fmt.Println("Error receiving response:", err)
		return
	}

	responseJson, err := protojson.Marshal(response)
	if err != nil {
		fmt.Println("Error marshaling response:", err)
		return
	}
	fmt.Println("Final response:", string(responseJson))
}

// Test bidirectional streaming RPC
func (drs *ReflectionClient) TestBidiStreamRPC(ctx context.Context) {
	serviceName := "TestService"
	methodName := "TestBidiStream"

	inputDesc, outputDesc, err := drs.helper.GetInputOutputTypes(ctx, serviceName, methodName)
	if err != nil {
		fmt.Println("Error getting input/output types:", err)
		return
	}

	fullMethodName := fmt.Sprintf("/%s.%s/%s", "reflect", serviceName, methodName)

	// Create bidirectional stream
	stream, err := drs.conn.NewStream(ctx, &grpc.StreamDesc{
		StreamName:    methodName,
		ServerStreams: true,
		ClientStreams: true,
	}, fullMethodName)
	if err != nil {
		fmt.Println("Error creating stream:", err)
		return
	}

	// Start a goroutine to receive messages
	go func() {
		for {
			response := dynamicpb.NewMessage(outputDesc)
			err := stream.RecvMsg(response)
			if err != nil {
				if err.Error() == "EOF" {
					return
				}
				fmt.Println("Error receiving message:", err)
				return
			}

			responseJson, err := protojson.Marshal(response)
			if err != nil {
				fmt.Println("Error marshaling response:", err)
				continue
			}
			fmt.Println("Received:", string(responseJson))
		}
	}()

	// Send messages
	for i := 1; i <= 3; i++ {
		message := dynamicpb.NewMessage(inputDesc)
		dummyJson := fmt.Sprintf(`{
            "message": "Bidi stream message %d"
        }`, i)

		err = protojson.Unmarshal([]byte(dummyJson), message)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			return
		}

		if err := stream.SendMsg(message); err != nil {
			fmt.Println("Error sending message:", err)
			return
		}
		time.Sleep(time.Second) // Add delay between messages
	}
}

// Test Run RPC (another unary RPC)
func (drs *ReflectionClient) TestRunRPC(ctx context.Context) {
	serviceName := "TestService"
	methodName := "Run"

	inputDesc, outputDesc, err := drs.helper.GetInputOutputTypes(ctx, serviceName, methodName)
	if err != nil {
		fmt.Println("Error getting input/output types:", err)
		return
	}

	message := dynamicpb.NewMessage(inputDesc)
	dummyJson := `{
        "message": "Run command test",
        "id": 12345
    }`

	err = protojson.Unmarshal([]byte(dummyJson), message)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}

	fullMethodName := fmt.Sprintf("/%s.%s/%s", "reflect", serviceName, methodName)

	response := dynamicpb.NewMessage(outputDesc)

	err = drs.conn.Invoke(ctx, fullMethodName, message, response)
	if err != nil {
		fmt.Println("Error invoking RPC:", err)
		return
	}

	responseJson, err := protojson.Marshal(response)
	if err != nil {
		fmt.Println("Error marshaling response:", err)
		return
	}
	fmt.Println("Response:", string(responseJson))
}

func GetNewGRPCMQTTConnection(bridgID string) (*grpc.ClientConn, *bridge.MQTTNetBridge) {
	// reflectionBridge := bridge.NewMQTTNetBridge(DRS.mqttClient, otelzap.L().Logger, bridgID)
	// resolver.Register(reflectionBridge)

	conn, err := grpc.NewClient(
		"localhost:1884",
		// "mqtt://"+bridgID,
		// bridgID,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return net.Dial("tcp", addr)
		}),
		grpc.WithBlock(),
		grpc.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatal("Failed to dial server:", err)
	}
	fmt.Printf("Connected to %s\n", conn.GetState())
	return conn, nil
}
