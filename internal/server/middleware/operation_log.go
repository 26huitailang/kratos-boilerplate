package middleware

import (
	"context"
	"time"

	"trae-demo/internal/biz"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// OperationLogMiddleware 创建操作日志中间件
func OperationLogMiddleware(repo biz.OperationLogRepo) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			// 获取用户信息，这里假设用户信息已经在认证中间件中设置到了 context 中
			userID := getUserIDFromContext(ctx)
			username := getUsernameFromContext(ctx)

			// 获取操作相关信息
			operation := getOperation(ctx)
			target := getTarget(ctx)

			// 执行实际的处理
			reply, err = handler(ctx, req)

			// 记录操作日志
			log := &biz.OperationLog{
				UserID:    userID,
				Username:  username,
				Operation: operation,
				Target:    target,
				Content:   formatContent(req),
				Result:    formatResult(reply, err),
				CreatedAt: time.Now(),
			}

			// 异步记录日志
			go func() {
				if err := repo.CreateLog(context.Background(), log); err != nil {
					// 这里可以添加日志记录失败的处理逻辑
				}
			}()

			return reply, err
		}
	}
}

// 从 context 中获取用户 ID
func getUserIDFromContext(ctx context.Context) int64 {
	// 实际项目中需要根据认证中间件的实现来获取用户 ID
	return 0
}

// 从 context 中获取用户名
func getUsernameFromContext(ctx context.Context) string {
	// 实际项目中需要根据认证中间件的实现来获取用户名
	return ""
}

// 获取操作类型
func getOperation(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "unknown"
	}

	switch tr.Kind() {
	case transport.KindHTTP:
		if info, ok := tr.(*http.Transport); ok {
			return info.Request().Method + " " + info.Request().URL.Path
		}
	case transport.KindGRPC:
		if info, ok := tr.(*grpc.Transport); ok {
			return info.Operation()
		}
	}
	return "unknown"
}

// 获取操作对象
func getTarget(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "unknown"
	}

	switch tr.Kind() {
	case transport.KindHTTP:
		if info, ok := tr.(*http.Transport); ok {
			return info.Request().URL.Path
		}
	case transport.KindGRPC:
		if info, ok := tr.(*grpc.Transport); ok {
			return info.Operation()
		}
	}
	return "unknown"
}

// 格式化请求内容
func formatContent(req interface{}) string {
	// 实际项目中可以根据需要实现请求内容的格式化
	return ""
}

// 格式化响应结果
func formatResult(reply interface{}, err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	// 实际项目中可以根据需要实现响应结果的格式化
	return "success"
}