package server

import (
	featurev1 "kratos-boilerplate/api/feature/v1"
	v1 "kratos-boilerplate/api/helloworld/v1"
	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, greeter *service.GreeterService, feature *service.FeatureToggleService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			// 暂时注释掉操作日志中间件，等实现了 repo 后再启用
			// middleware.OperationLogMiddleware(repo),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterGreeterServer(srv, greeter)
	featurev1.RegisterFeatureToggleServer(srv, feature)
	return srv
}
