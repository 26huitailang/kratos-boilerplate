package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	v1 "kratos-boilerplate/api/helloworld/v1"
	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/pkg/errors"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	kratosErrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
)

var (
	flagconf string
	id, _    = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "./configs/test-config.yaml", "config path")
}

// SimpleGreeterService 简化的Greeter服务，用于测试错误处理
type SimpleGreeterService struct {
	v1.UnimplementedGreeterServer
}

// SayHello 实现错误测试端点
func (s *SimpleGreeterService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
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
		return &v1.HelloReply{Message: "Hello " + in.Name}, nil
	}
}

func main() {
	flag.Parse()

	// 创建日志器
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", "test-server",
	)

	// 加载配置
	c := config.New(config.WithSource(file.NewSource(flagconf)))
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// 创建错误处理中间件
	errorHandler := errors.NewErrorHandlerMiddleware(logger, true, false)
	validationErrorHandler := errors.NewValidationErrorMiddleware(logger)
	rateLimitErrorHandler := errors.NewRateLimitErrorMiddleware(logger)

	// 创建HTTP服务器
	var opts = []kratosHttp.ServerOption{
		kratosHttp.Middleware(
			recovery.Recovery(),
			errorHandler.Middleware(),
			validationErrorHandler.Middleware(),
			rateLimitErrorHandler.Middleware(),
		),
	}

	if bc.Server.Http.Addr != "" {
		opts = append(opts, kratosHttp.Address(bc.Server.Http.Addr))
	}
	if bc.Server.Http.Timeout != nil {
		opts = append(opts, kratosHttp.Timeout(bc.Server.Http.Timeout.AsDuration()))
	}

	httpServer := kratosHttp.NewServer(opts...)

	// 注册服务
	greeterService := &SimpleGreeterService{}
	v1.RegisterGreeterHTTPServer(httpServer, greeterService)

	// 添加健康检查端点
	httpServer.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"test-server"}`))
	})

	// 创建应用
	app := kratos.New(
		kratos.ID(id),
		kratos.Name("test-server"),
		kratos.Logger(logger),
		kratos.Server(httpServer),
	)

	log.NewHelper(logger).Info("Starting test server for error handling middleware...")

	// 启动应用
	if err := app.Run(); err != nil {
		panic(err)
	}
}
