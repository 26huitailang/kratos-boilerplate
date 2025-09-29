package service

import (
	"context"

	v1 "kratos-boilerplate/api/helloworld/v1"
	"kratos-boilerplate/internal/biz"

	kratosErrors "github.com/go-kratos/kratos/v2/errors"
)

// GreeterService is a greeter service.
type GreeterService struct {
	v1.UnimplementedGreeterServer

	uc *biz.GreeterUsecase
}

// NewGreeterService new a greeter service.
func NewGreeterService(uc *biz.GreeterUsecase) *GreeterService {
	return &GreeterService{uc: uc}
}

// SayHello implements helloworld.GreeterServer.
func (s *GreeterService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
	// Test error middleware with different error scenarios
	switch in.Name {
	case "error":
		return nil, kratosErrors.InternalServer("INTERNAL_ERROR", "Internal server error for testing")
	case "badrequest":
		return nil, kratosErrors.BadRequest("BAD_REQUEST", "Bad request error for testing")
	case "notfound":
		return nil, kratosErrors.NotFound("NOT_FOUND", "Resource not found for testing")
	case "forbidden":
		return nil, kratosErrors.Forbidden("FORBIDDEN", "Access forbidden for testing")
	case "timeout":
		return nil, kratosErrors.New(408, "TIMEOUT", "Request timeout for testing")
	default:
		g, err := s.uc.CreateGreeter(ctx, &biz.Greeter{Hello: in.Name})
		if err != nil {
			return nil, err
		}
		return &v1.HelloReply{Message: "Hello " + g.Hello}, nil
	}
}
