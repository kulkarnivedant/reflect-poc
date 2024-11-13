package server

import (
	"context"
	"fmt"
	"io"
	"strings"

	service_proto "github.com/vedantkulkarni/reflect-poc/service-proto"
)

type MyTestService struct {
	service_proto.UnimplementedTestServiceServer
}

func (s *MyTestService) Test(ctx context.Context, req *service_proto.TestMessageRequest) (*service_proto.TestMessageResponse, error) {
	return &service_proto.TestMessageResponse{
		Message: "Response from Test method",
	}, nil
}

func (s *MyTestService) TestServerStream(req *service_proto.TestMessageRequest, stream service_proto.TestService_TestServerStreamServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&service_proto.TestMessageResponse{
			Message: fmt.Sprintf("Server streaming response %d", i),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *MyTestService) TestClientStream(stream service_proto.TestService_TestClientStreamServer) error {
	var messages []string
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&service_proto.TestMessageResponse{
				Message: fmt.Sprintf("Received messages: %v", strings.Join(messages, ", ")),
			})
		}
		if err != nil {
			return err
		}
		messages = append(messages, req.Message)
	}
}

func (s *MyTestService) TestBidiStream(stream service_proto.TestService_TestBidiStreamServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if err := stream.Send(&service_proto.TestMessageResponse{
			Message: fmt.Sprintf("Received: %s", req.Message),
		}); err != nil {
			return err
		}
	}
}

func (s *MyTestService) Run(ctx context.Context, req *service_proto.RunMessageRequest) (*service_proto.RunMessageResponse, error) {
	return &service_proto.RunMessageResponse{
		Message: "Response from Run method",
	}, nil
}
