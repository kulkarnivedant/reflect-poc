package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	// bridge "github.com/golain-io/mqtt-bridge"
	client "github.com/vedantkulkarni/reflect-poc/client"
	server "github.com/vedantkulkarni/reflect-poc/server"
	service_proto "github.com/vedantkulkarni/reflect-poc/service-proto"
	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	// Create MQTT client
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("echo-net-service")

	mqttClient := mqtt.NewClient(opts)
	token := mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatal("Failed to connect to MQTT broker:", token.Error())
	}
	defer mqttClient.Disconnect(0)

	logger, _ := zap.NewProduction()
	// netBridge := bridge.NewMQTTNetBridge(mqttClient, logger, "echo-service1")
	listener, err := net.Listen("tcp", ":1884")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	testService := &server.MyTestService{}
	service_proto.RegisterTestServiceServer(grpcServer, testService)
	syncService := &server.MySyncService{}
	service_proto.RegisterSyncServiceServer(grpcServer, syncService)

	reflection.RegisterV1(grpcServer)

	go createClient(os.Args)
	

	// Add error handling for Serve
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down server...")
	listener.Close()
}

// create a new grpc client to fetch server methods based on grpc reflection
func createClient(args []string) {
	time.Sleep(5 * time.Second)

	fmt.Println("Creating client")

	// GetNewGRPCMQTTClient returns a new grpc client connection and a new mqtt bridge
	conn, _ := client.GetNewGRPCMQTTConnection("echo-service1")

	reflectionClient := client.NewReflectionClient(conn)
	switch args[1] {
	case "unary":
		reflectionClient.TestUnaryRPC(context.Background())
	case "server_stream":
		reflectionClient.TestServerStreamRPC(context.Background())
	case "client_stream":
		reflectionClient.TestClientStreamRPC(context.Background())
	case "bidi_stream":
		reflectionClient.TestBidiStreamRPC(context.Background())
	}	

}
