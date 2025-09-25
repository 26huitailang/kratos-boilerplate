package middleware

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// Recovery 异常恢复中间件
func Recovery(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					logger.Log(log.LevelError,
						"msg", "panic recovered",
						"error", err,
						"stack", string(stack),
					)
				}
			}()
			return handler(ctx, req)
		}
	}
}

// RequestID 请求ID中间件
func RequestID() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 从传输层获取请求ID，如果没有则生成
			var requestID string
			if tr, ok := transport.FromServerContext(ctx); ok {
				if reqID := tr.RequestHeader().Get("X-Request-ID"); reqID != "" {
					requestID = reqID
				}
			}
			
			if requestID == "" {
				requestID = generateRequestID()
			}
			
			// 将请求ID注入到上下文
			ctx = context.WithValue(ctx, "request_id", requestID)
			
			return handler(ctx, req)
		}
	}
}

// Logging 日志中间件
func Logging(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
			)
			
			startTime := time.Now()
			
			// 获取传输信息
			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}
			
			// 获取请求ID
			requestID := getRequestIDFromContext(ctx)
			
			// 记录请求日志
			logger.Log(log.LevelInfo,
				"msg", "request started",
				"kind", kind,
				"operation", operation,
				"request_id", requestID,
				"timestamp", startTime.Format(time.RFC3339),
			)
			
			// 执行处理器
			reply, err := handler(ctx, req)
			
			// 处理错误
			if err != nil {
				if se := errors.FromError(err); se != nil {
					code = se.Code
					reason = se.Reason
				}
			}
			
			duration := time.Since(startTime)
			
			// 记录响应日志
			level := log.LevelInfo
			if err != nil {
				level = log.LevelError
			}
			
			logger.Log(level,
				"msg", "request completed",
				"kind", kind,
				"operation", operation,
				"request_id", requestID,
				"code", code,
				"reason", reason,
				"duration", duration.String(),
				"error", err,
			)
			
			return reply, err
		}
	}
}

// RateLimit 限流中间件配置
type RateLimitConfig struct {
	Rate  int           // 每秒请求数
	Burst int           // 突发请求数
	TTL   time.Duration // 限流窗口时间
}

// RateLimit 限流中间件
func RateLimit(config *RateLimitConfig) middleware.Middleware {
	limiter := NewTokenBucket(config.Rate, config.Burst)
	
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if !limiter.Allow() {
				return nil, errors.New(429, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded")
			}
			
			return handler(ctx, req)
		}
	}
}

// CORS 跨域中间件配置
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig 默认CORS配置
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		MaxAge:       86400,
	}
}

// CORS 跨域中间件（主要用于HTTP）
func CORS(config *CORSConfig) middleware.Middleware {
	if config == nil {
		config = DefaultCORSConfig()
	}
	
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 这里主要是示例，实际CORS处理通常在HTTP层处理
			// 在gRPC中通常不需要CORS
			return handler(ctx, req)
		}
	}
}

// Metrics 指标中间件
func Metrics(recorder MetricsRecorder) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var (
				kind      string
				operation string
			)
			
			startTime := time.Now()
			
			// 获取传输信息
			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}
			
			// 记录请求开始
			recorder.RequestStarted(kind, operation)
			
			// 执行处理器
			reply, err := handler(ctx, req)
			
			// 记录请求结束
			duration := time.Since(startTime)
			
			if err != nil {
				recorder.RequestFailed(kind, operation, duration)
			} else {
				recorder.RequestSucceeded(kind, operation, duration)
			}
			
			return reply, err
		}
	}
}

// MetricsRecorder 指标记录器接口
type MetricsRecorder interface {
	RequestStarted(kind, operation string)
	RequestSucceeded(kind, operation string, duration time.Duration)
	RequestFailed(kind, operation string, duration time.Duration)
}

// Validate 验证中间件
func Validate(validator Validator) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if err := validator.Validate(req); err != nil {
				return nil, errors.BadRequest("VALIDATION_ERROR", err.Error())
			}
			
			return handler(ctx, req)
		}
	}
}

// Validator 验证器接口
type Validator interface {
	Validate(req interface{}) error
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	rate      int
	burst     int
	tokens    int
	lastToken time.Time
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(rate, burst int) *TokenBucket {
	return &TokenBucket{
		rate:      rate,
		burst:     burst,
		tokens:    burst,
		lastToken: time.Now(),
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow() bool {
	now := time.Now()
	
	// 根据时间流逝补充令牌
	elapsed := now.Sub(tb.lastToken)
	tokensToAdd := int(elapsed.Seconds()) * tb.rate
	
	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.burst {
			tb.tokens = tb.burst
		}
		tb.lastToken = now
	}
	
	// 检查是否有可用令牌
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	
	return false
}

// Circuit 熔断器中间件
func Circuit(config *CircuitConfig) middleware.Middleware {
	breaker := NewCircuitBreaker(config)
	
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if !breaker.Allow() {
				return nil, errors.ServiceUnavailable("CIRCUIT_BREAKER_OPEN", "Circuit breaker is open")
			}
			
			reply, err := handler(ctx, req)
			
			if err != nil {
				breaker.RecordFailure()
			} else {
				breaker.RecordSuccess()
			}
			
			return reply, err
		}
	}
}

// CircuitConfig 熔断器配置
type CircuitConfig struct {
	FailureThreshold int           // 失败阈值
	RecoveryTimeout  time.Duration // 恢复超时
	RequestThreshold int           // 请求阈值
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config        *CircuitConfig
	state         CircuitState
	failures      int
	requests      int
	lastFailTime  time.Time
}

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config *CircuitConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Allow 检查是否允许请求
func (cb *CircuitBreaker) Allow() bool {
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// 检查是否可以进入半开状态
		if time.Since(cb.lastFailTime) > cb.config.RecoveryTimeout {
			cb.state = StateHalfOpen
			cb.requests = 0
			return true
		}
		return false
	case StateHalfOpen:
		return cb.requests < cb.config.RequestThreshold
	}
	return false
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.requests++
	
	if cb.state == StateHalfOpen {
		if cb.requests >= cb.config.RequestThreshold {
			cb.state = StateClosed
			cb.failures = 0
		}
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()
	
	if cb.failures >= cb.config.FailureThreshold {
		cb.state = StateOpen
	}
}

// Timeout 超时中间件
func Timeout(timeout time.Duration) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			
			return handler(ctx, req)
		}
	}
}

// 工具函数

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// getRequestIDFromContext 从上下文获取请求ID
func getRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// MiddlewareChain 中间件链
type MiddlewareChain struct {
	middlewares []middleware.Middleware
}

// NewMiddlewareChain 创建中间件链
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]middleware.Middleware, 0),
	}
}

// Use 添加中间件
func (mc *MiddlewareChain) Use(m middleware.Middleware) *MiddlewareChain {
	mc.middlewares = append(mc.middlewares, m)
	return mc
}

// Build 构建中间件链
func (mc *MiddlewareChain) Build() []middleware.Middleware {
	return mc.middlewares
}

// DefaultMiddlewares 默认中间件链
func DefaultMiddlewares(logger log.Logger) []middleware.Middleware {
	return NewMiddlewareChain().
		Use(Recovery(logger)).
		Use(RequestID()).
		Use(Logging(logger)).
		Use(Timeout(30*time.Second)).
		Build()
}

// Auth 认证中间件
func Auth(authenticator Authenticator) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取认证信息
			var token string
			if tr, ok := transport.FromServerContext(ctx); ok {
				auth := tr.RequestHeader().Get("Authorization")
				if auth != "" && strings.HasPrefix(auth, "Bearer ") {
					token = strings.TrimPrefix(auth, "Bearer ")
				}
			}
			
			if token == "" {
				return nil, errors.Unauthorized("MISSING_TOKEN", "Missing authentication token")
			}
			
			// 验证令牌
			userInfo, err := authenticator.Authenticate(ctx, token)
			if err != nil {
				return nil, errors.Unauthorized("INVALID_TOKEN", "Invalid authentication token")
			}
			
			// 将用户信息注入上下文
			ctx = context.WithValue(ctx, "user_info", userInfo)
			
			return handler(ctx, req)
		}
	}
}

// Authenticator 认证器接口
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (interface{}, error)
}

// Permission 权限中间件
func Permission(authorizer Authorizer) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取用户信息
			userInfo := ctx.Value("user_info")
			if userInfo == nil {
				return nil, errors.Unauthorized("NO_USER_INFO", "No user information found")
			}
			
			// 获取操作信息
			var operation string
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
			}
			
			// 检查权限
			allowed, err := authorizer.Authorize(ctx, userInfo, operation, req)
			if err != nil {
				return nil, errors.InternalServer("AUTHORIZATION_ERROR", "Authorization check failed")
			}
			
			if !allowed {
				return nil, errors.Forbidden("PERMISSION_DENIED", "Permission denied")
			}
			
			return handler(ctx, req)
		}
	}
}

// Authorizer 授权器接口
type Authorizer interface {
	Authorize(ctx context.Context, userInfo interface{}, operation string, req interface{}) (bool, error)
}

// HealthCheck 健康检查中间件
func HealthCheck(healthChecker HealthChecker) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 检查服务健康状态
			if !healthChecker.IsHealthy(ctx) {
				return nil, errors.ServiceUnavailable("SERVICE_UNHEALTHY", "Service is currently unhealthy")
			}
			
			return handler(ctx, req)
		}
	}
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	IsHealthy(ctx context.Context) bool
}

// Retry 重试中间件
func Retry(config *RetryConfig) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var lastErr error
			
			for i := 0; i <= config.MaxRetries; i++ {
				reply, err := handler(ctx, req)
				if err == nil {
					return reply, nil
				}
				
				lastErr = err
				
				// 检查是否为可重试错误
				if !config.ShouldRetry(err) {
					break
				}
				
				// 如果不是最后一次重试，等待后重试
				if i < config.MaxRetries {
					select {
					case <-time.After(config.BackoffStrategy(i)):
						continue
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
			}
			
			return nil, lastErr
		}
	}
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int
	ShouldRetry     func(error) bool
	BackoffStrategy func(int) time.Duration
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		ShouldRetry: func(err error) bool {
			// 默认只重试5xx错误
			if se := errors.FromError(err); se != nil {
				return se.Code >= 500 && se.Code < 600
			}
			return false
		},
		BackoffStrategy: func(attempt int) time.Duration {
			// 指数退避
			return time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
		},
	}
}