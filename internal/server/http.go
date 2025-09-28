package server

import (
	authv1 "kratos-boilerplate/api/auth/v1"
	v1 "kratos-boilerplate/api/helloworld/v1"
	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/pkg/security"
	"kratos-boilerplate/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, greeter *service.GreeterService, auth *service.AuthService, logger log.Logger) *http.Server {
	// Security configuration
	securityConfig := security.DefaultSecurityConfig()

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			// Security middleware will be added as filters
			// 暂时注释掉操作日志中间件，等实现了 repo 后再启用
			// middleware.OperationLogMiddleware(repo),
		),
		// Add security filters
		http.Filter(
			security.SecurityHeadersFilter(securityConfig),
			security.CORSFilter(securityConfig),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterGreeterHTTPServer(srv, greeter)
	authv1.RegisterAuthHTTPServer(srv, auth)
	return srv
}
