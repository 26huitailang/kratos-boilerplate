package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 日志接口
type Logger interface {
	kratoslog.Logger
	
	// 结构化日志方法
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	
	// 格式化日志方法
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Fatalf(template string, args ...interface{})
	
	// 上下文方法
	WithContext(ctx context.Context) Logger
	WithFields(fields ...Field) Logger
	
	// 控制方法
	Sync() error
	Close() error
}

// Field 日志字段
type Field interface {
	Key() string
	Value() interface{}
	Type() FieldType
}

// FieldType 字段类型
type FieldType int

const (
	StringType FieldType = iota
	IntType
	Int64Type
	Float64Type
	BoolType
	TimeType
	DurationType
	ErrorType
	AnyType
)

// field 字段实现
type field struct {
	key       string
	value     interface{}
	fieldType FieldType
}

func (f *field) Key() string        { return f.key }
func (f *field) Value() interface{} { return f.value }
func (f *field) Type() FieldType    { return f.fieldType }

// 字段构造函数
func String(key string, val string) Field {
	return &field{key: key, value: val, fieldType: StringType}
}

func Int(key string, val int) Field {
	return &field{key: key, value: val, fieldType: IntType}
}

func Int64(key string, val int64) Field {
	return &field{key: key, value: val, fieldType: Int64Type}
}

func Float64(key string, val float64) Field {
	return &field{key: key, value: val, fieldType: Float64Type}
}

func Bool(key string, val bool) Field {
	return &field{key: key, value: val, fieldType: BoolType}
}

func Time(key string, val time.Time) Field {
	return &field{key: key, value: val, fieldType: TimeType}
}

func Duration(key string, val time.Duration) Field {
	return &field{key: key, value: val, fieldType: DurationType}
}

func Error(err error) Field {
	return &field{key: "error", value: err, fieldType: ErrorType}
}

func Any(key string, val interface{}) Field {
	return &field{key: key, value: val, fieldType: AnyType}
}

// Config 日志配置
type Config struct {
	Level         string      `yaml:"level" json:"level"`                   // 日志级别
	Format        string      `yaml:"format" json:"format"`                 // 输出格式 (json/text)
	Output        string      `yaml:"output" json:"output"`                 // 输出目标 (stdout/file)
	File          FileConfig  `yaml:"file" json:"file"`                     // 文件配置
	EnableCaller  bool        `yaml:"enable_caller" json:"enable_caller"`   // 是否显示调用位置
	EnableTrace   bool        `yaml:"enable_trace" json:"enable_trace"`     // 是否启用链路追踪
	SampleConfig  *SampleConfig `yaml:"sample" json:"sample"`               // 采样配置
}

// FileConfig 文件日志配置
type FileConfig struct {
	Path       string `yaml:"path" json:"path"`               // 文件路径
	MaxSize    int    `yaml:"max_size" json:"max_size"`       // 最大文件大小(MB)
	MaxBackups int    `yaml:"max_backups" json:"max_backups"` // 最大备份数
	MaxAge     int    `yaml:"max_age" json:"max_age"`         // 最大保存天数
	Compress   bool   `yaml:"compress" json:"compress"`       // 是否压缩
}

// SampleConfig 采样配置
type SampleConfig struct {
	Initial    int           `yaml:"initial" json:"initial"`       // 初始采样数
	Thereafter int           `yaml:"thereafter" json:"thereafter"` // 后续采样间隔
	Tick       time.Duration `yaml:"tick" json:"tick"`             // 采样时间间隔
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		EnableCaller: true,
		EnableTrace:  true,
		File: FileConfig{
			Path:       "logs/app.log",
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
		},
		SampleConfig: &SampleConfig{
			Initial:    100,
			Thereafter: 100,
			Tick:       time.Second,
		},
	}
}

// zapLogger Zap日志器实现
type zapLogger struct {
	zap    *zap.Logger
	level  zapcore.Level
	config *Config
	mu     sync.RWMutex
}

// NewLogger 创建新的日志器
func NewLogger(config *Config) (Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	// 解析日志级别
	level, err := parseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	
	// 创建编码器配置
	encoderConfig := getEncoderConfig(config.Format)
	
	// 创建编码器
	var encoder zapcore.Encoder
	if config.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
	
	// 创建输出
	writeSyncer := getWriteSyncer(config)
	
	// 创建核心
	core := zapcore.NewCore(encoder, writeSyncer, level)
	
	// 应用采样（如果配置了）
	if config.SampleConfig != nil {
		core = zapcore.NewSamplerWithOptions(
			core,
			config.SampleConfig.Tick,
			config.SampleConfig.Initial,
			config.SampleConfig.Thereafter,
		)
	}
	
	// 创建zap logger
	zapOpts := []zap.Option{
		zap.ErrorOutput(zapcore.AddSync(os.Stderr)),
	}
	
	if config.EnableCaller {
		zapOpts = append(zapOpts, zap.AddCaller(), zap.AddCallerSkip(1))
	}
	
	zapLogger := zap.New(core, zapOpts...)
	
	return &zapLogger{
		zap:    zapLogger,
		level:  level,
		config: config,
	}, nil
}

// Log 实现 kratoslog.Logger 接口
func (l *zapLogger) Log(level kratoslog.Level, keyvals ...interface{}) error {
	var zapLevel zapcore.Level
	switch level {
	case kratoslog.LevelDebug:
		zapLevel = zapcore.DebugLevel
	case kratoslog.LevelInfo:
		zapLevel = zapcore.InfoLevel
	case kratoslog.LevelWarn:
		zapLevel = zapcore.WarnLevel
	case kratoslog.LevelError:
		zapLevel = zapcore.ErrorLevel
	case kratoslog.LevelFatal:
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	
	if !l.zap.Core().Enabled(zapLevel) {
		return nil
	}
	
	fields := make([]zap.Field, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			key := fmt.Sprintf("%v", keyvals[i])
			value := keyvals[i+1]
			fields = append(fields, zap.Any(key, value))
		}
	}
	
	switch zapLevel {
	case zapcore.DebugLevel:
		l.zap.Debug("", fields...)
	case zapcore.InfoLevel:
		l.zap.Info("", fields...)
	case zapcore.WarnLevel:
		l.zap.Warn("", fields...)
	case zapcore.ErrorLevel:
		l.zap.Error("", fields...)
	case zapcore.FatalLevel:
		l.zap.Fatal("", fields...)
	}
	
	return nil
}

// Debug 调试级别日志
func (l *zapLogger) Debug(msg string, fields ...Field) {
	if l.zap.Core().Enabled(zapcore.DebugLevel) {
		l.zap.Debug(msg, l.fieldsToZapFields(fields...)...)
	}
}

// Info 信息级别日志
func (l *zapLogger) Info(msg string, fields ...Field) {
	if l.zap.Core().Enabled(zapcore.InfoLevel) {
		l.zap.Info(msg, l.fieldsToZapFields(fields...)...)
	}
}

// Warn 警告级别日志
func (l *zapLogger) Warn(msg string, fields ...Field) {
	if l.zap.Core().Enabled(zapcore.WarnLevel) {
		l.zap.Warn(msg, l.fieldsToZapFields(fields...)...)
	}
}

// Error 错误级别日志
func (l *zapLogger) Error(msg string, fields ...Field) {
	if l.zap.Core().Enabled(zapcore.ErrorLevel) {
		l.zap.Error(msg, l.fieldsToZapFields(fields...)...)
	}
}

// Fatal 致命级别日志
func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.zap.Fatal(msg, l.fieldsToZapFields(fields...)...)
}

// Debugf 格式化调试日志
func (l *zapLogger) Debugf(template string, args ...interface{}) {
	if l.zap.Core().Enabled(zapcore.DebugLevel) {
		l.zap.Debug(fmt.Sprintf(template, args...))
	}
}

// Infof 格式化信息日志
func (l *zapLogger) Infof(template string, args ...interface{}) {
	if l.zap.Core().Enabled(zapcore.InfoLevel) {
		l.zap.Info(fmt.Sprintf(template, args...))
	}
}

// Warnf 格式化警告日志
func (l *zapLogger) Warnf(template string, args ...interface{}) {
	if l.zap.Core().Enabled(zapcore.WarnLevel) {
		l.zap.Warn(fmt.Sprintf(template, args...))
	}
}

// Errorf 格式化错误日志
func (l *zapLogger) Errorf(template string, args ...interface{}) {
	if l.zap.Core().Enabled(zapcore.ErrorLevel) {
		l.zap.Error(fmt.Sprintf(template, args...))
	}
}

// Fatalf 格式化致命日志
func (l *zapLogger) Fatalf(template string, args ...interface{}) {
	l.zap.Fatal(fmt.Sprintf(template, args...))
}

// WithContext 带上下文的日志器
func (l *zapLogger) WithContext(ctx context.Context) Logger {
	fields := extractContextFields(ctx)
	if len(fields) == 0 {
		return l
	}
	
	newZap := l.zap.With(l.fieldsToZapFields(fields...)...)
	return &zapLogger{
		zap:    newZap,
		level:  l.level,
		config: l.config,
	}
}

// WithFields 带字段的日志器
func (l *zapLogger) WithFields(fields ...Field) Logger {
	if len(fields) == 0 {
		return l
	}
	
	newZap := l.zap.With(l.fieldsToZapFields(fields...)...)
	return &zapLogger{
		zap:    newZap,
		level:  l.level,
		config: l.config,
	}
}

// Sync 同步日志
func (l *zapLogger) Sync() error {
	return l.zap.Sync()
}

// Close 关闭日志器
func (l *zapLogger) Close() error {
	return l.Sync()
}

// fieldsToZapFields 转换字段到zap字段
func (l *zapLogger) fieldsToZapFields(fields ...Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	
	for _, field := range fields {
		var zapField zap.Field
		
		switch field.Type() {
		case StringType:
			zapField = zap.String(field.Key(), field.Value().(string))
		case IntType:
			zapField = zap.Int(field.Key(), field.Value().(int))
		case Int64Type:
			zapField = zap.Int64(field.Key(), field.Value().(int64))
		case Float64Type:
			zapField = zap.Float64(field.Key(), field.Value().(float64))
		case BoolType:
			zapField = zap.Bool(field.Key(), field.Value().(bool))
		case TimeType:
			zapField = zap.Time(field.Key(), field.Value().(time.Time))
		case DurationType:
			zapField = zap.Duration(field.Key(), field.Value().(time.Duration))
		case ErrorType:
			if err, ok := field.Value().(error); ok {
				zapField = zap.Error(err)
			} else {
				zapField = zap.Any(field.Key(), field.Value())
			}
		case AnyType:
			zapField = zap.Any(field.Key(), field.Value())
		default:
			zapField = zap.Any(field.Key(), field.Value())
		}
		
		zapFields = append(zapFields, zapField)
	}
	
	return zapFields
}

// parseLevel 解析日志级别
func parseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown level: %s", level)
	}
}

// getEncoderConfig 获取编码器配置
func getEncoderConfig(format string) zapcore.EncoderConfig {
	config := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	if format == "text" {
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	}
	
	return config
}

// getWriteSyncer 获取写入器
func getWriteSyncer(config *Config) zapcore.WriteSyncer {
	var writers []io.Writer
	
	// 添加标准输出
	if config.Output == "stdout" || config.Output == "both" {
		writers = append(writers, os.Stdout)
	}
	
	// 添加文件输出
	if config.Output == "file" || config.Output == "both" {
		// 确保日志目录存在
		logDir := filepath.Dir(config.File.Path)
		if err := os.MkdirAll(logDir, 0755); err == nil {
			fileWriter := &lumberjack.Logger{
				Filename:   config.File.Path,
				MaxSize:    config.File.MaxSize,
				MaxBackups: config.File.MaxBackups,
				MaxAge:     config.File.MaxAge,
				Compress:   config.File.Compress,
			}
			writers = append(writers, fileWriter)
		}
	}
	
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}
	
	return zapcore.AddSync(io.MultiWriter(writers...))
}

// extractContextFields 从上下文提取字段
func extractContextFields(ctx context.Context) []Field {
	var fields []Field
	
	// 提取trace_id
	if traceID := getTraceIDFromContext(ctx); traceID != "" {
		fields = append(fields, String("trace_id", traceID))
	}
	
	// 提取span_id
	if spanID := getSpanIDFromContext(ctx); spanID != "" {
		fields = append(fields, String("span_id", spanID))
	}
	
	// 提取user_id
	if userID := getUserIDFromContext(ctx); userID != "" {
		fields = append(fields, String("user_id", userID))
	}
	
	// 提取request_id
	if requestID := getRequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, String("request_id", requestID))
	}
	
	return fields
}

// 上下文键常量
type contextKey string

const (
	TraceIDKey   contextKey = "trace_id"
	SpanIDKey    contextKey = "span_id"
	UserIDKey    contextKey = "user_id"
	RequestIDKey contextKey = "request_id"
)

// getTraceIDFromContext 从上下文获取trace_id
func getTraceIDFromContext(ctx context.Context) string {
	if val := ctx.Value(TraceIDKey); val != nil {
		if traceID, ok := val.(string); ok {
			return traceID
		}
	}
	return ""
}

// getSpanIDFromContext 从上下文获取span_id
func getSpanIDFromContext(ctx context.Context) string {
	if val := ctx.Value(SpanIDKey); val != nil {
		if spanID, ok := val.(string); ok {
			return spanID
		}
	}
	return ""
}

// getUserIDFromContext 从上下文获取user_id
func getUserIDFromContext(ctx context.Context) string {
	if val := ctx.Value(UserIDKey); val != nil {
		if userID, ok := val.(string); ok {
			return userID
		}
	}
	return ""
}

// getRequestIDFromContext 从上下文获取request_id
func getRequestIDFromContext(ctx context.Context) string {
	if val := ctx.Value(RequestIDKey); val != nil {
		if requestID, ok := val.(string); ok {
			return requestID
		}
	}
	return ""
}

// WithTraceID 设置trace_id到上下文
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID 设置span_id到上下文
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// WithUserID 设置user_id到上下文
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithRequestID 设置request_id到上下文
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}