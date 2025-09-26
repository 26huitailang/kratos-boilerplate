package config

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	logger := log.NewStdLogger(nil)
	manager := NewManager("test.yaml", "dev", logger)

	assert.NotNil(t, manager)
	assert.Equal(t, "test.yaml", manager.configPath)
	assert.Equal(t, "dev", manager.environment)
	assert.NotNil(t, manager.logger)
	assert.NotNil(t, manager.watchers)
	assert.NotNil(t, manager.validator)
}

func TestManagerWithInMemoryConfig(t *testing.T) {
	// 创建一个内存中的配置管理器用于测试
	logger := log.NewStdLogger(nil)
	manager := NewManager("", "", logger)

	// 测试默认值方法
	assert.Equal(t, "", manager.GetString("nonexistent"))
	assert.Equal(t, "default", manager.GetString("nonexistent", "default"))
	assert.Equal(t, 0, manager.GetInt("nonexistent"))
	assert.Equal(t, 42, manager.GetInt("nonexistent", 42))
	assert.Equal(t, false, manager.GetBool("nonexistent"))
	assert.Equal(t, true, manager.GetBool("nonexistent", true))
	assert.Equal(t, time.Duration(0), manager.GetDuration("nonexistent"))
	assert.Equal(t, time.Hour, manager.GetDuration("nonexistent", time.Hour))
}

func TestConfigDefaults(t *testing.T) {
	logger := log.NewStdLogger(nil)
	manager := NewManager("", "", logger)

	// 创建一个基本配置用于测试默认值设置
	cfg := &Config{}

	// 设置必需的字段以通过验证
	cfg.Server.HTTP.Addr = "0.0.0.0:8000"
	cfg.Server.GRPC.Addr = "0.0.0.0:9000"
	cfg.Data.Database.Driver = "postgres"
	cfg.Data.Database.Source = "test://test"
	cfg.Data.Redis.Addr = "127.0.0.1:6379"
	cfg.Auth.JWTSecretKey = "test-secret-key-16-chars"

	manager.setDefaults(cfg)

	// 验证默认值
	assert.Equal(t, 30*time.Second, cfg.Server.HTTP.Timeout)
	assert.Equal(t, 30*time.Second, cfg.Server.GRPC.Timeout)
	assert.Equal(t, 3*time.Second, cfg.Data.Redis.ReadTimeout)
	assert.Equal(t, 3*time.Second, cfg.Data.Redis.WriteTimeout)
	assert.Equal(t, 24*time.Hour, cfg.Auth.AccessTokenExpiration)
	assert.Equal(t, 7*24*time.Hour, cfg.Auth.RefreshTokenExpiration)
	assert.Equal(t, 5*time.Minute, cfg.Auth.CaptchaExpiration)
	assert.Equal(t, 5, cfg.Auth.MaxLoginAttempts)
	assert.Equal(t, 30*time.Minute, cfg.Auth.LockDuration)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, "stdout", cfg.Log.Output)
	assert.Equal(t, 0.1, cfg.Tracing.Jaeger.SampleRate)
}

func TestGetEnvConfigPath(t *testing.T) {
	logger := log.NewStdLogger(nil)
	manager := NewManager("", "", logger)

	tests := []struct {
		name        string
		configPath  string
		environment string
		expected    string
	}{
		{
			name:        "yaml config",
			configPath:  "/path/to/config.yaml",
			environment: "dev",
			expected:    "/path/to/config.dev.yaml",
		},
		{
			name:        "json config",
			configPath:  "/path/to/config.json",
			environment: "prod",
			expected:    "/path/to/config.prod.json",
		},
		{
			name:        "no extension",
			configPath:  "/path/to/config",
			environment: "test",
			expected:    "/path/to/config.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.getEnvConfigPath(tt.configPath, tt.environment)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManagerClose(t *testing.T) {
	logger := log.NewStdLogger(nil)
	manager := NewManager("", "", logger)

	// 测试没有配置时的关闭
	err := manager.Close()
	assert.NoError(t, err)
}

func TestConfigWatch(t *testing.T) {
	logger := log.NewStdLogger(nil)
	manager := NewManager("", "", logger)

	// 创建一个简单的观察者
	observer := config.Observer(func(key string, value config.Value) {
		// 观察者回调函数
	})

	// 测试在没有配置的情况下添加观察者
	err := manager.Watch("test.key", observer)
	assert.NoError(t, err)

	// 验证观察者被添加到内部映射
	manager.mu.RLock()
	watchers := manager.watchers["test.key"]
	manager.mu.RUnlock()

	assert.Len(t, watchers, 1)
}

// MockConfig 用于测试的模拟配置
type MockConfig struct {
	TestString   string        `yaml:"test_string" validate:"required"`
	TestInt      int           `yaml:"test_int" validate:"min=1,max=100"`
	TestDuration time.Duration `yaml:"test_duration"`
	TestBool     bool          `yaml:"test_bool"`
}

func TestConfigValidation(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		config      MockConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: MockConfig{
				TestString:   "test",
				TestInt:      50,
				TestDuration: time.Hour,
				TestBool:     true,
			},
			expectError: false,
		},
		{
			name: "missing required field",
			config: MockConfig{
				TestInt:      50,
				TestDuration: time.Hour,
				TestBool:     true,
			},
			expectError: true,
		},
		{
			name: "int out of range",
			config: MockConfig{
				TestString:   "test",
				TestInt:      150,
				TestDuration: time.Hour,
				TestBool:     true,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(&tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManagerHelperMethods(t *testing.T) {
	logger := log.NewStdLogger(nil)
	manager := NewManager("", "", logger)

	// 由于没有实际的配置源，这些方法应该返回默认值
	assert.Equal(t, "", manager.GetString("test.key"))
	assert.Equal(t, "default", manager.GetString("test.key", "default"))

	assert.Equal(t, 0, manager.GetInt("test.key"))
	assert.Equal(t, 42, manager.GetInt("test.key", 42))

	assert.Equal(t, false, manager.GetBool("test.key"))
	assert.Equal(t, true, manager.GetBool("test.key", true))

	assert.Equal(t, time.Duration(0), manager.GetDuration("test.key"))
	assert.Equal(t, time.Hour, manager.GetDuration("test.key", time.Hour))
}

// BenchmarkManagerHelperMethods 基准测试
func BenchmarkManagerHelperMethods(b *testing.B) {
	logger := log.NewStdLogger(nil)
	manager := NewManager("", "", logger)

	b.ResetTimer()

	b.Run("GetString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager.GetString("test.key", "default")
		}
	})

	b.Run("GetInt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager.GetInt("test.key", 42)
		}
	})

	b.Run("GetBool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager.GetBool("test.key", true)
		}
	})

	b.Run("GetDuration", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager.GetDuration("test.key", time.Hour)
		}
	})
}

// TestConfigConcurrency 并发测试
func TestConfigConcurrency(t *testing.T) {
	logger := log.NewStdLogger(nil)
	manager := NewManager("", "", logger)

	// 并发测试观察者添加
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan bool, 100)

	// 启动多个goroutine同时添加观察者
	for i := 0; i < 100; i++ {
		go func(id int) {
			observer := config.Observer(func(key string, value config.Value) {
				// 简单的观察者函数
			})
			err := manager.Watch("test.key", observer)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 100; i++ {
		select {
		case <-done:
			// 成功
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}

	// 验证观察者数量
	manager.mu.RLock()
	watchers := manager.watchers["test.key"]
	manager.mu.RUnlock()

	assert.Len(t, watchers, 100)
}
