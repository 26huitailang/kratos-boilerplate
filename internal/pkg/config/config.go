package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
)

// ConfigManager 配置管理器接口
type ConfigManager interface {
	// Load 加载配置
	Load() error
	// Get 获取配置值
	Get(key string) config.Value
	// Watch 监听配置变化
	Watch(key string, observer config.Observer) error
	// Close 关闭配置管理器
	Close() error
	// Validate 验证配置
	Validate() error
}

// Config 配置结构
type Config struct {
	Server struct {
		HTTP struct {
			Addr    string        `yaml:"addr" validate:"required"`
			Timeout time.Duration `yaml:"timeout"`
		} `yaml:"http"`
		GRPC struct {
			Addr    string        `yaml:"addr" validate:"required"`
			Timeout time.Duration `yaml:"timeout"`
		} `yaml:"grpc"`
	} `yaml:"server"`
	
	Data struct {
		Database struct {
			Driver string `yaml:"driver" validate:"required,oneof=postgres mysql"`
			Source string `yaml:"source" validate:"required"`
		} `yaml:"database"`
		Redis struct {
			Addr         string        `yaml:"addr" validate:"required"`
			ReadTimeout  time.Duration `yaml:"read_timeout"`
			WriteTimeout time.Duration `yaml:"write_timeout"`
		} `yaml:"redis"`
	} `yaml:"data"`
	
	Auth struct {
		JWTSecretKey             string        `yaml:"jwt_secret_key" validate:"required,min=16"`
		AccessTokenExpiration    time.Duration `yaml:"access_token_expiration"`
		RefreshTokenExpiration   time.Duration `yaml:"refresh_token_expiration"`
		CaptchaEnabled           bool          `yaml:"captcha_enabled"`
		CaptchaExpiration        time.Duration `yaml:"captcha_expiration"`
		MaxLoginAttempts         int           `yaml:"max_login_attempts" validate:"min=1,max=10"`
		LockDuration            time.Duration `yaml:"lock_duration"`
		TOTPEnabled             bool          `yaml:"totp_enabled"`
	} `yaml:"auth"`
	
	Log struct {
		Level  string `yaml:"level" validate:"oneof=debug info warn error fatal"`
		Format string `yaml:"format" validate:"oneof=json text"`
		Output string `yaml:"output" validate:"oneof=stdout file"`
		File   struct {
			Path       string `yaml:"path"`
			MaxSize    int    `yaml:"max_size"`
			MaxBackups int    `yaml:"max_backups"`
			MaxAge     int    `yaml:"max_age"`
			Compress   bool   `yaml:"compress"`
		} `yaml:"file"`
	} `yaml:"log"`
	
	Tracing struct {
		Enabled bool   `yaml:"enabled"`
		Jaeger  struct {
			Endpoint string  `yaml:"endpoint"`
			SampleRate float64 `yaml:"sample_rate" validate:"min=0,max=1"`
		} `yaml:"jaeger"`
	} `yaml:"tracing"`
}

// Manager 配置管理器实现
type Manager struct {
	config      config.Config
	logger      log.Logger
	configPath  string
	environment string
	watchers    map[string][]config.Observer
	mu          sync.RWMutex
	validator   Validator
}

// NewManager 创建配置管理器
func NewManager(configPath, environment string, logger log.Logger) *Manager {
	return &Manager{
		configPath:  configPath,
		environment: environment,
		logger:      logger,
		watchers:    make(map[string][]config.Observer),
		validator:   NewValidator(),
	}
}

// Load 加载配置
func (m *Manager) Load() error {
	var sources []config.Source
	
	// 添加文件配置源
	if m.configPath != "" {
		sources = append(sources, file.NewSource(m.configPath))
		
		// 如果有环境特定配置文件，也加载它
		if m.environment != "" {
			envConfigPath := m.getEnvConfigPath(m.configPath, m.environment)
			if _, err := os.Stat(envConfigPath); err == nil {
				sources = append(sources, file.NewSource(envConfigPath))
			}
		}
	}
	
	// 添加环境变量配置源
	sources = append(sources, env.NewSource("KRATOS_"))
	
	// 创建配置
	c := config.New(
		config.WithSource(sources...),
	)
	
	if err := c.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	m.config = c
	
	// 启动配置监听（如果支持）
	m.startWatching()
	
	return nil
}

// Get 获取配置值
func (m *Manager) Get(key string) config.Value {
	if m.config == nil {
		return config.Value{}
	}
	return m.config.Value(key)
}

// Watch 监听配置变化
func (m *Manager) Watch(key string, observer config.Observer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.watchers[key] == nil {
		m.watchers[key] = make([]config.Observer, 0)
	}
	m.watchers[key] = append(m.watchers[key], observer)
	
	if m.config != nil {
		return m.config.Watch(key, observer)
	}
	
	return nil
}

// Close 关闭配置管理器
func (m *Manager) Close() error {
	if m.config != nil {
		return m.config.Close()
	}
	return nil
}

// Validate 验证配置
func (m *Manager) Validate() error {
	var cfg Config
	if err := m.config.Scan(&cfg); err != nil {
		return fmt.Errorf("failed to scan config: %w", err)
	}
	
	return m.validator.Validate(&cfg)
}

// GetConfig 获取完整配置结构
func (m *Manager) GetConfig() (*Config, error) {
	var cfg Config
	if err := m.config.Scan(&cfg); err != nil {
		return nil, fmt.Errorf("failed to scan config: %w", err)
	}
	
	// 设置默认值
	m.setDefaults(&cfg)
	
	return &cfg, nil
}

// setDefaults 设置默认值
func (m *Manager) setDefaults(cfg *Config) {
	if cfg.Server.HTTP.Timeout == 0 {
		cfg.Server.HTTP.Timeout = 30 * time.Second
	}
	if cfg.Server.GRPC.Timeout == 0 {
		cfg.Server.GRPC.Timeout = 30 * time.Second
	}
	if cfg.Data.Redis.ReadTimeout == 0 {
		cfg.Data.Redis.ReadTimeout = 3 * time.Second
	}
	if cfg.Data.Redis.WriteTimeout == 0 {
		cfg.Data.Redis.WriteTimeout = 3 * time.Second
	}
	if cfg.Auth.AccessTokenExpiration == 0 {
		cfg.Auth.AccessTokenExpiration = 24 * time.Hour
	}
	if cfg.Auth.RefreshTokenExpiration == 0 {
		cfg.Auth.RefreshTokenExpiration = 7 * 24 * time.Hour
	}
	if cfg.Auth.CaptchaExpiration == 0 {
		cfg.Auth.CaptchaExpiration = 5 * time.Minute
	}
	if cfg.Auth.MaxLoginAttempts == 0 {
		cfg.Auth.MaxLoginAttempts = 5
	}
	if cfg.Auth.LockDuration == 0 {
		cfg.Auth.LockDuration = 30 * time.Minute
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "json"
	}
	if cfg.Log.Output == "" {
		cfg.Log.Output = "stdout"
	}
	if cfg.Tracing.Jaeger.SampleRate == 0 {
		cfg.Tracing.Jaeger.SampleRate = 0.1
	}
}

// getEnvConfigPath 获取环境特定配置文件路径
func (m *Manager) getEnvConfigPath(configPath, environment string) string {
	dir := filepath.Dir(configPath)
	ext := filepath.Ext(configPath)
	base := strings.TrimSuffix(filepath.Base(configPath), ext)
	return filepath.Join(dir, fmt.Sprintf("%s.%s%s", base, environment, ext))
}

// startWatching 启动配置文件监听
func (m *Manager) startWatching() {
	if m.configPath == "" {
		return
	}
	
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			m.logger.Log(log.LevelError, "msg", "Failed to create file watcher", "error", err)
			return
		}
		defer watcher.Close()
		
		err = watcher.Add(m.configPath)
		if err != nil {
			m.logger.Log(log.LevelError, "msg", "Failed to watch config file", "file", m.configPath, "error", err)
			return
		}
		
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					m.logger.Log(log.LevelInfo, "msg", "Config file changed", "file", event.Name)
					m.handleConfigChange()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				m.logger.Log(log.LevelError, "msg", "File watcher error", "error", err)
			}
		}
	}()
}

// handleConfigChange 处理配置变化
func (m *Manager) handleConfigChange() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 重新加载配置
	if err := m.config.Load(); err != nil {
		m.logger.Log(log.LevelError, "msg", "Failed to reload config", "error", err)
		return
	}
	
	m.logger.Log(log.LevelInfo, "msg", "Config reloaded successfully")
}

// GetString 获取字符串配置
func (m *Manager) GetString(key string, defaultValue ...string) string {
	val := m.Get(key)
	if val == nil || val.String() == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	return val.String()
}

// GetInt 获取整数配置
func (m *Manager) GetInt(key string, defaultValue ...int) int {
	val := m.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	i, _ := val.Int()
	return int(i)
}

// GetBool 获取布尔配置
func (m *Manager) GetBool(key string, defaultValue ...bool) bool {
	val := m.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	b, _ := val.Bool()
	return b
}

// GetDuration 获取时间间隔配置
func (m *Manager) GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	val := m.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	d, _ := val.Duration()
	return d
}