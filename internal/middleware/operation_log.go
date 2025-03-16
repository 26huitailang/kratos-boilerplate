package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// OperationLogRepo 定义操作日志接口
type OperationLogRepo interface {
	Create(ctx context.Context, log *OperationLog) error
}

// OperationLog 操作日志结构
type OperationLog struct {
	Path   string
	Method string
	// 可以根据需要添加更多字段
}

// OperationLogMiddleware 创建操作日志中间件
func OperationLogMiddleware(repo OperationLogRepo) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				// 记录操作日志
				log := &OperationLog{
					Path:   tr.Operation(),
					Method: tr.Kind().String(),
				}
				_ = repo.Create(ctx, log) // 异步记录日志，忽略错误
			}
			return handler(ctx, req)
		}
	}
}
