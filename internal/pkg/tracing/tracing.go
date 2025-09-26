package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Config 链路追踪配置
type Config struct {
	// 基础配置
	ServiceName    string `yaml:"service_name" json:"service_name"`
	ServiceVersion string `yaml:"service_version" json:"service_version"`
	Environment    string `yaml:"environment" json:"environment"`
	Enabled        bool   `yaml:"enabled" json:"enabled"`

	// Jaeger配置
	Jaeger JaegerConfig `yaml:"jaeger" json:"jaeger"`

	// 采样配置
	Sampling SamplingConfig `yaml:"sampling" json:"sampling"`

	// 资源配置
	Resource ResourceConfig `yaml:"resource" json:"resource"`
}

// JaegerConfig Jaeger配置
type JaegerConfig struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// SamplingConfig 采样配置
type SamplingConfig struct {
	Type  string  `yaml:"type" json:"type"`   // always_on, always_off, ratio
	Ratio float64 `yaml:"ratio" json:"ratio"` // 采样比例 0.0-1.0
}

// ResourceConfig 资源配置
type ResourceConfig struct {
	Attributes map[string]string `yaml:"attributes" json:"attributes"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "kratos-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Enabled:        true,
		Jaeger: JaegerConfig{
			Endpoint: "http://localhost:14268/api/traces",
		},
		Sampling: SamplingConfig{
			Type:  "ratio",
			Ratio: 0.1,
		},
		Resource: ResourceConfig{
			Attributes: map[string]string{
				"deployment.environment": "development",
			},
		},
	}
}

// TracingProvider 链路追踪提供者
type TracingProvider struct {
	tracer   oteltrace.Tracer
	provider *trace.TracerProvider
	config   *Config
	logger   log.Logger
}

// NewTracingProvider 创建链路追踪提供者
func NewTracingProvider(config *Config, logger log.Logger) (*TracingProvider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled {
		return &TracingProvider{
			tracer: oteltrace.NewNoopTracerProvider().Tracer("noop"),
			config: config,
			logger: logger,
		}, nil
	}

	// 创建资源
	res, err := createResource(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建导出器
	exporter, err := createJaegerExporter(config.Jaeger)
	if err != nil {
		return nil, fmt.Errorf("failed to create jaeger exporter: %w", err)
	}

	// 创建采样器
	sampler := createSampler(config.Sampling)

	// 创建TraceProvider
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	// 设置全局TraceProvider
	otel.SetTracerProvider(provider)

	// 设置全局传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 创建Tracer
	tracer := provider.Tracer(config.ServiceName)

	return &TracingProvider{
		tracer:   tracer,
		provider: provider,
		config:   config,
		logger:   logger,
	}, nil
}

// createResource 创建资源
func createResource(config *Config) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{
		semconv.ServiceName(config.ServiceName),
		semconv.ServiceVersion(config.ServiceVersion),
		semconv.DeploymentEnvironment(config.Environment),
	}

	// 添加自定义属性
	for key, value := range config.Resource.Attributes {
		attributes = append(attributes, attribute.String(key, value))
	}

	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			attributes...,
		),
	)
}

// createJaegerExporter 创建Jaeger导出器
func createJaegerExporter(config JaegerConfig) (trace.SpanExporter, error) {
	opts := []jaeger.CollectorEndpointOption{
		jaeger.WithEndpoint(config.Endpoint),
	}

	if config.Username != "" && config.Password != "" {
		opts = append(opts, jaeger.WithUsername(config.Username))
		opts = append(opts, jaeger.WithPassword(config.Password))
	}

	return jaeger.New(jaeger.WithCollectorEndpoint(opts...))
}

// createSampler 创建采样器
func createSampler(config SamplingConfig) trace.Sampler {
	switch config.Type {
	case "always_on":
		return trace.AlwaysSample()
	case "always_off":
		return trace.NeverSample()
	case "ratio":
		return trace.TraceIDRatioBased(config.Ratio)
	default:
		return trace.TraceIDRatioBased(0.1) // 默认10%采样率
	}
}

// GetTracer 获取Tracer
func (tp *TracingProvider) GetTracer() oteltrace.Tracer {
	return tp.tracer
}

// Shutdown 关闭链路追踪
func (tp *TracingProvider) Shutdown(ctx context.Context) error {
	if tp.provider != nil {
		return tp.provider.Shutdown(ctx)
	}
	return nil
}

// Middleware 链路追踪中间件
func (tp *TracingProvider) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var (
				operation string
				kind      string
			)

			// 获取传输信息
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
				kind = tr.Kind().String()
			}

			if operation == "" {
				operation = "unknown"
			}

			// 从请求头中提取跟踪上下文
			carrier := make(propagation.MapCarrier)
			if tr, ok := transport.FromServerContext(ctx); ok {
				headers := tr.RequestHeader()
				// 直接访问 header 值
				if traceParent := headers.Get("traceparent"); traceParent != "" {
					carrier["traceparent"] = traceParent
				}
				if traceState := headers.Get("tracestate"); traceState != "" {
					carrier["tracestate"] = traceState
				}
				if baggage := headers.Get("baggage"); baggage != "" {
					carrier["baggage"] = baggage
				}
			}

			ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

			// 创建Span
			ctx, span := tp.tracer.Start(ctx, operation,
				oteltrace.WithSpanKind(getSpanKind(kind)),
				oteltrace.WithAttributes(
					attribute.String("rpc.system", "kratos"),
					attribute.String("rpc.service", tp.config.ServiceName),
					attribute.String("rpc.method", operation),
				),
			)
			defer span.End()

			// 执行处理器
			reply, err := handler(ctx, req)

			// 记录错误
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetStatus(codes.Ok, "success")
			}

			return reply, err
		}
	}
}

// getSpanKind 获取Span类型
func getSpanKind(transportKind string) oteltrace.SpanKind {
	switch transportKind {
	case "HTTP":
		return oteltrace.SpanKindServer
	case "GRPC":
		return oteltrace.SpanKindServer
	default:
		return oteltrace.SpanKindInternal
	}
}

// SpanFromContext 从上下文获取Span
func SpanFromContext(ctx context.Context) oteltrace.Span {
	return oteltrace.SpanFromContext(ctx)
}

// TraceIDFromContext 从上下文获取TraceID
func TraceIDFromContext(ctx context.Context) string {
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// SpanIDFromContext 从上下文获取SpanID
func SpanIDFromContext(ctx context.Context) string {
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// StartSpan 创建新的Span
func StartSpan(ctx context.Context, tracer oteltrace.Tracer, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return tracer.Start(ctx, name, opts...)
}

// AddAttributes 添加属性到当前Span
func AddAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddEvent 添加事件到当前Span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, oteltrace.WithAttributes(attrs...))
	}
}

// RecordError 记录错误到当前Span
func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err, oteltrace.WithAttributes(attrs...))
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetStatus 设置Span状态
func SetStatus(ctx context.Context, code codes.Code, description string) {
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(code, description)
	}
}

// TracingLogger 带链路追踪的日志记录器
type TracingLogger struct {
	logger log.Logger
}

// NewTracingLogger 创建带链路追踪的日志记录器
func NewTracingLogger(logger log.Logger) *TracingLogger {
	return &TracingLogger{
		logger: logger,
	}
}

// Log 记录日志并添加追踪信息
func (tl *TracingLogger) Log(level log.Level, keyvals ...interface{}) error {
	// 这里需要从上下文获取，实际使用时需要传入上下文
	// 简化实现，实际项目中需要改进
	return tl.logger.Log(level, keyvals...)
}

// LogWithContext 带上下文的日志记录
func (tl *TracingLogger) LogWithContext(ctx context.Context, level log.Level, keyvals ...interface{}) error {
	// 添加追踪ID到日志
	traceID := TraceIDFromContext(ctx)
	spanID := SpanIDFromContext(ctx)

	if traceID != "" {
		keyvals = append(keyvals, "trace_id", traceID)
	}
	if spanID != "" {
		keyvals = append(keyvals, "span_id", spanID)
	}

	return tl.logger.Log(level, keyvals...)
}

// ClientTracing 客户端链路追踪
type ClientTracing struct {
	tracer oteltrace.Tracer
}

// NewClientTracing 创建客户端链路追踪
func NewClientTracing(tracer oteltrace.Tracer) *ClientTracing {
	return &ClientTracing{
		tracer: tracer,
	}
}

// WrapHTTPClient 包装HTTP客户端
func (ct *ClientTracing) WrapHTTPClient(ctx context.Context, method, url string, fn func(ctx context.Context) error) error {
	ctx, span := ct.tracer.Start(ctx, fmt.Sprintf("%s %s", method, url),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", url),
		),
	)
	defer span.End()

	// 注入追踪信息到HTTP头
	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	// 执行HTTP请求
	err := fn(ctx)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// WrapGRPCClient 包装gRPC客户端
func (ct *ClientTracing) WrapGRPCClient(ctx context.Context, method string, fn func(ctx context.Context) error) error {
	ctx, span := ct.tracer.Start(ctx, method,
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			attribute.String("rpc.system", "grpc"),
			attribute.String("rpc.method", method),
		),
	)
	defer span.End()

	// 执行gRPC请求
	err := fn(ctx)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// DatabaseTracing 数据库链路追踪
type DatabaseTracing struct {
	tracer oteltrace.Tracer
}

// NewDatabaseTracing 创建数据库链路追踪
func NewDatabaseTracing(tracer oteltrace.Tracer) *DatabaseTracing {
	return &DatabaseTracing{
		tracer: tracer,
	}
}

// WrapQuery 包装数据库查询
func (dt *DatabaseTracing) WrapQuery(ctx context.Context, operation, table, sql string, fn func(ctx context.Context) error) error {
	spanName := fmt.Sprintf("db.%s %s", operation, table)

	ctx, span := dt.tracer.Start(ctx, spanName,
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			attribute.String("db.system", "postgresql"), // 或其他数据库类型
			attribute.String("db.operation", operation),
			attribute.String("db.sql.table", table),
			attribute.String("db.statement", sql),
		),
	)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// CacheTracing 缓存链路追踪
type CacheTracing struct {
	tracer oteltrace.Tracer
}

// NewCacheTracing 创建缓存链路追踪
func NewCacheTracing(tracer oteltrace.Tracer) *CacheTracing {
	return &CacheTracing{
		tracer: tracer,
	}
}

// WrapOperation 包装缓存操作
func (ct *CacheTracing) WrapOperation(ctx context.Context, operation, key string, fn func(ctx context.Context) error) error {
	spanName := fmt.Sprintf("cache.%s", operation)

	ctx, span := ct.tracer.Start(ctx, spanName,
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			attribute.String("cache.system", "redis"),
			attribute.String("cache.operation", operation),
			attribute.String("cache.key", key),
		),
	)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("cache.duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// TracingHelper 链路追踪助手
type TracingHelper struct {
	tracer oteltrace.Tracer
	logger log.Logger
}

// NewTracingHelper 创建链路追踪助手
func NewTracingHelper(tracer oteltrace.Tracer, logger log.Logger) *TracingHelper {
	return &TracingHelper{
		tracer: tracer,
		logger: logger,
	}
}

// WithSpan 在新Span中执行函数
func (th *TracingHelper) WithSpan(ctx context.Context, name string, fn func(ctx context.Context) error, opts ...oteltrace.SpanStartOption) error {
	ctx, span := th.tracer.Start(ctx, name, opts...)
	defer span.End()

	err := fn(ctx)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		th.logger.Log(log.LevelError, "msg", "span error", "span", name, "error", err)
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// TraceAsync 异步操作追踪
func (th *TracingHelper) TraceAsync(ctx context.Context, name string, fn func(ctx context.Context)) {
	// 创建新的跟踪上下文，避免父Span过早结束
	ctx, span := th.tracer.Start(ctx, name,
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
	)

	go func() {
		defer span.End()

		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("async operation panic: %v", r)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				th.logger.Log(log.LevelError, "msg", "async span panic", "span", name, "panic", r)
			}
		}()

		fn(ctx)
		span.SetStatus(codes.Ok, "success")
	}()
}
