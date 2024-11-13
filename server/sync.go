package server

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	service_proto "github.com/vedantkulkarni/reflect-poc/service-proto"
)


type MySyncService struct {
	service_proto.UnimplementedSyncServiceServer
}

// Sync handles unary RPC calls
func (s *MySyncService) Sync(ctx context.Context, req *service_proto.SyncMessageRequest) (*service_proto.SyncMessageResponse, error) {
	return &service_proto.SyncMessageResponse{
		Message: "Received: " + req.GetMessage(),
	}, nil
}

// SyncServerStream handles server-side streaming RPC
func (s *MySyncService) SyncServerStream(req *service_proto.SyncMessageRequest, stream service_proto.SyncService_SyncServerStreamServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&service_proto.SyncMessageResponse{
			Message: fmt.Sprintf("Stream response %d for: %s", i, req.GetMessage()),
		}); err != nil {
			return err
		}
		time.Sleep(time.Second) // Simulate some work
	}
	return nil
}

// SyncClientStream handles client-side streaming RPC
func (s *MySyncService) SyncClientStream(stream service_proto.SyncService_SyncClientStreamServer) error {
	var messages []string
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// Client has finished sending
			return stream.SendAndClose(&service_proto.SyncMessageResponse{
				Message: "Received messages: " + strings.Join(messages, ", "),
			})
		}
		if err != nil {
			return err
		}
		messages = append(messages, req.GetMessage())
	}
}

// SyncBidiStream handles bidirectional streaming RPC
func (s *MySyncService) SyncBidiStream(stream service_proto.SyncService_SyncBidiStreamServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		
		if err := stream.Send(&service_proto.SyncMessageResponse{
			Message: "Echoing: " + req.GetMessage(),
		}); err != nil {
			return err
		}
	}
}
