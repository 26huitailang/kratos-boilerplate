package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, "info", config.Level)
	assert.Equal(t, "json", config.Format)
	assert.Equal(t, "stdout", config.Output)
	assert.True(t, config.EnableCaller)
	assert.True(t, config.EnableTrace)
	assert.Equal(t, "logs/app.log", config.File.Path)
	assert.Equal(t, 100, config.File.MaxSize)
	assert.Equal(t, 10, config.File.MaxBackups)
	assert.Equal(t, 30, config.File.MaxAge)
	assert.True(t, config.File.Compress)
	assert.NotNil(t, config.SampleConfig)
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "default config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "custom config",
			config: &Config{
				Level:        "debug",
				Format:       "text",
				Output:       "stdout",
				EnableCaller: true,
				EnableTrace:  false,
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: &Config{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zapcore.Level
		wantErr  bool
	}{
		{"debug", "debug", zapcore.DebugLevel, false},
		{"info", "info", zapcore.InfoLevel, false},
		{"warn", "warn", zapcore.WarnLevel, false},
		{"warning", "warning", zapcore.WarnLevel, false},
		{"error", "error", zapcore.ErrorLevel, false},
		{"fatal", "fatal", zapcore.FatalLevel, false},
		{"Debug", "Debug", zapcore.DebugLevel, false},
		{"INFO", "INFO", zapcore.InfoLevel, false},
		{"invalid", "invalid", zapcore.InfoLevel, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := parseLevel(tt.level)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, level)
			}
		})
	}
}

func TestFields(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected interface{}
		keyCheck string
		typeCheck FieldType
	}{
		{
			name:      "string field",
			field:     String("test", "value"),
			expected:  "value",
			keyCheck:  "test",
			typeCheck: StringType,
		},
		{
			name:      "int field",
			field:     Int("count", 42),
			expected:  42,
			keyCheck:  "count",
			typeCheck: IntType,
		},
		{
			name:      "bool field",
			field:     Bool("enabled", true),
			expected:  true,
			keyCheck:  "enabled",
			typeCheck: BoolType,
		},
		{
			name:      "error field",
			field:     Error(fmt.Errorf("test error")),
			expected:  fmt.Errorf("test error"),
			keyCheck:  "error",
			typeCheck: ErrorType,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.keyCheck, tt.field.Key())
			assert.Equal(t, tt.typeCheck, tt.field.Type())
			
			// 对于错误类型，需要特殊处理比较
			if tt.typeCheck == ErrorType {
				expectedErr := tt.expected.(error)
				actualErr := tt.field.Value().(error)
				assert.Equal(t, expectedErr.Error(), actualErr.Error())
			} else {
				assert.Equal(t, tt.expected, tt.field.Value())
			}
		})
	}
}

func TestLoggerWithObserver(t *testing.T) {
	// 创建观察者记录器用于测试
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	observedLogger := zap.New(observedZapCore)
	
	logger := &zapLogger{
		zap:   observedLogger,
		level: zapcore.DebugLevel,
		config: &Config{
			Level:        "debug",
			Format:       "json",
			Output:       "stdout",
			EnableCaller: false,
		},
	}
	
	// 测试不同级别的日志
	logger.Debug("debug message", String("key", "value"))
	logger.Info("info message", Int("count", 1))
	logger.Warn("warn message", Bool("flag", true))
	logger.Error("error message", Error(fmt.Errorf("test error")))
	
	// 验证日志记录
	logs := observedLogs.All()
	assert.Len(t, logs, 4)
	
	// 验证debug日志
	assert.Equal(t, zapcore.DebugLevel, logs[0].Level)
	assert.Equal(t, "debug message", logs[0].Message)
	assert.Equal(t, "value", logs[0].Context[0].String)
	
	// 验证info日志
	assert.Equal(t, zapcore.InfoLevel, logs[1].Level)
	assert.Equal(t, "info message", logs[1].Message)
	assert.Equal(t, int64(1), logs[1].Context[0].Integer)
	
	// 验证warn日志
	assert.Equal(t, zapcore.WarnLevel, logs[2].Level)
	assert.Equal(t, "warn message", logs[2].Message)
	assert.True(t, logs[2].Context[0].Bool)
	
	// 验证error日志
	assert.Equal(t, zapcore.ErrorLevel, logs[3].Level)
	assert.Equal(t, "error message", logs[3].Message)
}

func TestFormattedLogging(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	observedLogger := zap.New(observedZapCore)
	
	logger := &zapLogger{
		zap:   observedLogger,
		level: zapcore.DebugLevel,
		config: &Config{
			Level:        "debug",
			Format:       "json",
			Output:       "stdout",
			EnableCaller: false,
		},
	}
	
	// 测试格式化日志
	logger.Debugf("debug: %s = %d", "count", 42)
	logger.Infof("info: %s", "test message")
	logger.Warnf("warn: %v", true)
	logger.Errorf("error: %s", "something went wrong")
	
	// 验证日志记录
	logs := observedLogs.All()
	assert.Len(t, logs, 4)
	
	assert.Equal(t, "debug: count = 42", logs[0].Message)
	assert.Equal(t, "info: test message", logs[1].Message)
	assert.Equal(t, "warn: true", logs[2].Message)
	assert.Equal(t, "error: something went wrong", logs[3].Message)
}

func TestLoggerWithContext(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	observedLogger := zap.New(observedZapCore)
	
	logger := &zapLogger{
		zap:   observedLogger,
		level: zapcore.DebugLevel,
		config: &Config{
			Level:        "debug",
			Format:       "json",
			Output:       "stdout",
			EnableCaller: false,
		},
	}
	
	// 创建带上下文的日志器
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	ctx = WithUserID(ctx, "user-456")
	
	contextLogger := logger.WithContext(ctx)
	contextLogger.Info("test message with context")
	
	// 验证上下文字段被添加
	logs := observedLogs.All()
	require.Len(t, logs, 1)
	
	contextFields := logs[0].Context
	assert.Len(t, contextFields, 2)
	
	// 验证trace_id字段
	traceIDFound := false
	userIDFound := false
	for _, field := range contextFields {
		if field.Key == "trace_id" && field.String == "trace-123" {
			traceIDFound = true
		}
		if field.Key == "user_id" && field.String == "user-456" {
			userIDFound = true
		}
	}
	assert.True(t, traceIDFound, "trace_id field not found")
	assert.True(t, userIDFound, "user_id field not found")
}

func TestLoggerWithFields(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	observedLogger := zap.New(observedZapCore)
	
	logger := &zapLogger{
		zap:   observedLogger,
		level: zapcore.DebugLevel,
		config: &Config{
			Level:        "debug",
			Format:       "json",
			Output:       "stdout",
			EnableCaller: false,
		},
	}
	
	// 创建带字段的日志器
	fieldsLogger := logger.WithFields(
		String("service", "test-service"),
		String("version", "1.0.0"),
	)
	
	fieldsLogger.Info("test message with fields")
	
	// 验证字段被添加
	logs := observedLogs.All()
	require.Len(t, logs, 1)
	
	contextFields := logs[0].Context
	assert.Len(t, contextFields, 2)
	
	serviceFound := false
	versionFound := false
	for _, field := range contextFields {
		if field.Key == "service" && field.String == "test-service" {
			serviceFound = true
		}
		if field.Key == "version" && field.String == "1.0.0" {
			versionFound = true
		}
	}
	assert.True(t, serviceFound, "service field not found")
	assert.True(t, versionFound, "version field not found")
}

func TestKratosLoggerInterface(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zapcore.DebugLevel)
	observedLogger := zap.New(observedZapCore)
	
	logger := &zapLogger{
		zap:   observedLogger,
		level: zapcore.DebugLevel,
		config: &Config{
			Level:        "debug",
			Format:       "json",
			Output:       "stdout",
			EnableCaller: false,
		},
	}
	
	// 测试Kratos Log接口
	err := logger.Log(1, "msg", "test message", "key", "value")
	assert.NoError(t, err)
	
	// 验证日志记录
	logs := observedLogs.All()
	require.Len(t, logs, 1)
	
	assert.Equal(t, zapcore.InfoLevel, logs[0].Level) // Level 1 maps to Info
	
	// 查找字段
	msgFound := false
	keyFound := false
	for _, field := range logs[0].Context {
		if field.Key == "msg" && field.String == "test message" {
			msgFound = true
		}
		if field.Key == "key" && field.String == "value" {
			keyFound = true
		}
	}
	assert.True(t, msgFound, "msg field not found")
	assert.True(t, keyFound, "key field not found")
}

func TestGetEncoderConfig(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"json format", "json"},
		{"text format", "text"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := getEncoderConfig(tt.format)
			
			assert.Equal(t, "timestamp", config.TimeKey)
			assert.Equal(t, "level", config.LevelKey)
			assert.Equal(t, "message", config.MessageKey)
			assert.Equal(t, "caller", config.CallerKey)
			
			if tt.format == "text" {
				assert.Equal(t, zapcore.CapitalColorLevelEncoder, config.EncodeLevel)
			} else {
				assert.Equal(t, zapcore.LowercaseLevelEncoder, config.EncodeLevel)
			}
		})
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	
	// 测试设置和获取trace_id
	ctx = WithTraceID(ctx, "trace-123")
	assert.Equal(t, "trace-123", getTraceIDFromContext(ctx))
	
	// 测试设置和获取span_id
	ctx = WithSpanID(ctx, "span-456")
	assert.Equal(t, "span-456", getSpanIDFromContext(ctx))
	
	// 测试设置和获取user_id
	ctx = WithUserID(ctx, "user-789")
	assert.Equal(t, "user-789", getUserIDFromContext(ctx))
	
	// 测试设置和获取request_id
	ctx = WithRequestID(ctx, "req-abc")
	assert.Equal(t, "req-abc", getRequestIDFromContext(ctx))
	
	// 测试空上下文
	emptyCtx := context.Background()
	assert.Equal(t, "", getTraceIDFromContext(emptyCtx))
	assert.Equal(t, "", getSpanIDFromContext(emptyCtx))
	assert.Equal(t, "", getUserIDFromContext(emptyCtx))
	assert.Equal(t, "", getRequestIDFromContext(emptyCtx))
}

func TestExtractContextFields(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	ctx = WithUserID(ctx, "user-456")
	
	fields := extractContextFields(ctx)
	
	// 应该有2个字段：trace_id和user_id
	assert.Len(t, fields, 2)
	
	traceIDFound := false
	userIDFound := false
	for _, field := range fields {
		if field.Key() == "trace_id" && field.Value() == "trace-123" {
			traceIDFound = true
		}
		if field.Key() == "user_id" && field.Value() == "user-456" {
			userIDFound = true
		}
	}
	assert.True(t, traceIDFound, "trace_id field not found")
	assert.True(t, userIDFound, "user_id field not found")
}

// BenchmarkLogger 性能基准测试
func BenchmarkLogger(b *testing.B) {
	logger, err := NewLogger(&Config{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		EnableCaller: false,
		EnableTrace:  false,
	})
	require.NoError(b, err)
	
	b.ResetTimer()
	
	b.Run("Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("test message", String("key", "value"))
		}
	})
	
	b.Run("InfoWithFields", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("test message",
				String("service", "test"),
				Int("count", i),
				Bool("enabled", true),
			)
		}
	})
	
	b.Run("Infof", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Infof("test message: %d", i)
		}
	})
}

func TestLoggerSyncAndClose(t *testing.T) {
	logger, err := NewLogger(nil)
	require.NoError(t, err)
	
	// 测试Sync
	err = logger.Sync()
	assert.NoError(t, err)
	
	// 测试Close
	err = logger.Close()
	assert.NoError(t, err)
}

func TestFieldsToZapFields(t *testing.T) {
	observedZapCore, _ := observer.New(zapcore.DebugLevel)
	observedLogger := zap.New(observedZapCore)
	
	logger := &zapLogger{
		zap:    observedLogger,
		level:  zapcore.DebugLevel,
		config: DefaultConfig(),
	}
	
	fields := []Field{
		String("str", "value"),
		Int("int", 42),
		Int64("int64", 123456789),
		Float64("float", 3.14),
		Bool("bool", true),
		Time("time", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
		Duration("duration", time.Hour),
		Error(fmt.Errorf("test error")),
		Any("any", map[string]interface{}{"key": "value"}),
	}
	
	zapFields := logger.fieldsToZapFields(fields...)
	
	assert.Len(t, zapFields, len(fields))
	
	// 验证每个字段类型
	assert.Equal(t, "str", zapFields[0].Key)
	assert.Equal(t, "int", zapFields[1].Key)
	assert.Equal(t, "int64", zapFields[2].Key)
	assert.Equal(t, "float", zapFields[3].Key)
	assert.Equal(t, "bool", zapFields[4].Key)
	assert.Equal(t, "time", zapFields[5].Key)
	assert.Equal(t, "duration", zapFields[6].Key)
	assert.Equal(t, "error", zapFields[7].Key)
	assert.Equal(t, "any", zapFields[8].Key)
}

func TestLoggerLevelFiltering(t *testing.T) {
	// 创建info级别的日志器
	observedZapCore, observedLogs := observer.New(zapcore.InfoLevel)
	observedLogger := zap.New(observedZapCore)
	
	logger := &zapLogger{
		zap:   observedLogger,
		level: zapcore.InfoLevel,
		config: &Config{
			Level:        "info",
			Format:       "json",
			Output:       "stdout",
			EnableCaller: false,
		},
	}
	
	// debug级别的日志应该被过滤掉
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	
	logs := observedLogs.All()
	assert.Len(t, logs, 2) // 只有info和warn被记录
	
	assert.Equal(t, "info message", logs[0].Message)
	assert.Equal(t, "warn message", logs[1].Message)
}