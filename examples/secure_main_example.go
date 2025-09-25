package main

import (
	"context"
	"flag"
	"os"

	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/server"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

// 增强版的newApp函数，集成安全的Swagger UI
func newAppWithSecureSwagger(
	logger log.Logger, 
	gs *grpc.Server, 
	hs *http.Server,
	authService *auth.Service,
	featureToggle feature.FeatureToggle,
	bc *conf.Bootstrap,
) *kratos.App {
	
	// 根据环境配置Swagger UI安全策略
	var swaggerConfig *server.SwaggerUISecurityConfig
	
	switch os.Getenv("ENVIRONMENT") {
	case "development":
		swaggerConfig = &server.SwaggerUISecurityConfig{
			EnableAuth:       false, // 开发环境不需要认证
			AllowedIPs:       []string{}, // 不限制IP
			AllowedUserTypes: []string{},
			EnableAuditLog:   false,
			PathPrefix:       "/q/",
		}
	case "staging":
		swaggerConfig = &server.SwaggerUISecurityConfig{
			EnableAuth:       true,
			AllowedIPs:       []string{"10.0.0.0/8"}, // 内网IP段
			AllowedUserTypes: []string{"admin", "developer", "tester"},
			EnableAuditLog:   true,
			PathPrefix:       "/q/",
		}
	case "production":
		swaggerConfig = &server.SwaggerUISecurityConfig{
			EnableAuth:       true,
			AllowedIPs:       []string{"172.16.0.100", "172.16.0.101"}, // 指定运维机器
			AllowedUserTypes: []string{"admin"},
			EnableAuditLog:   true,
			PathPrefix:       "/q/",
		}
	default:
		// 默认安全配置
		swaggerConfig = &server.SwaggerUISecurityConfig{
			EnableAuth:       true,
			AllowedIPs:       []string{"127.0.0.1"},
			AllowedUserTypes: []string{"admin"},
			EnableAuditLog:   true,
			PathPrefix:       "/q/",
		}
	}
	
	// 注册安全的Swagger UI
	server.RegisterSecureSwaggerUI(hs, swaggerConfig, authService, featureToggle, logger)
	
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// 使用增强版的app创建函数
	app, cleanup, err := wireAppWithSecureSwagger(bc.Server, bc.Data, bc.Auth, &bc, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}