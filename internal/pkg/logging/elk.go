package logging

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ELKConfig ELK日志配置
type ELKConfig struct {
	// File配置
	File struct {
		Enabled   bool   `yaml:"enabled" json:"enabled"`
		Path      string `yaml:"path" json:"path"`
		MaxSize   int    `yaml:"max_size" json:"max_size"`
		MaxBackups int    `yaml:"max_backups" json:"max_backups"`
		MaxAge    int    `yaml:"max_age" json:"max_age"`
		Compress  bool   `yaml:"compress" json:"compress"`
	} `yaml:"file" json:"file"`

	// Elasticsearch配置
	Elasticsearch struct {
		Enabled  bool     `yaml:"enabled" json:"enabled"`
		Hosts    []string `yaml:"hosts" json:"hosts"`
		Index    string   `yaml:"index" json:"index"`
		Username string   `yaml:"username" json:"username"`
		Password string   `yaml:"password" json:"password"`
	} `yaml:"elasticsearch" json:"elasticsearch"`

	// Fluentd配置
	Fluentd struct {
		Enabled bool   `yaml:"enabled" json:"enabled"`
		Host    string `yaml:"host" json:"host"`
		Port    int    `yaml:"port" json:"port"`
		Tag     string `yaml:"tag" json:"tag"`
	} `yaml:"fluentd" json:"fluentd"`
}

// DefaultELKConfig 默认ELK配置
func DefaultELKConfig() *ELKConfig {
	return &ELKConfig{
		File: struct {
			Enabled   bool   `yaml:"enabled" json:"enabled"`
			Path      string `yaml:"path" json:"path"`
			MaxSize   int    `yaml:"max_size" json:"max_size"`
			MaxBackups int    `yaml:"max_backups" json:"max_backups"`
			MaxAge    int    `yaml:"max_age" json:"max_age"`
			Compress  bool   `yaml:"compress" json:"compress"`
		}{
			Enabled:   true,
			Path:      "logs/app.log",
			MaxSize:   100,
			MaxBackups: 10,
			MaxAge:    30,
			Compress:  true,
		},
		Elasticsearch: struct {
			Enabled  bool     `yaml:"enabled" json:"enabled"`
			Hosts    []string `yaml:"hosts" json:"hosts"`
			Index    string   `yaml:"index" json:"index"`
			Username string   `yaml:"username" json:"username"`
			Password string   `yaml:"password" json:"password"`
		}{
			Enabled: false,
			Hosts:   []string{"localhost:9200"},
			Index:   "kratos-boilerplate-%Y.%m.%d",
		},
		Fluentd: struct {
			Enabled bool   `yaml:"enabled" json:"enabled"`
			Host    string `yaml:"host" json:"host"`
			Port    int    `yaml:"port" json:"port"`
			Tag     string `yaml:"tag" json:"tag"`
		}{
			Enabled: false,
			Host:    "localhost",
			Port:    24224,
			Tag:     "kratos-boilerplate",
		},
	}
}

// ELKLogger ELK日志记录器
type ELKLogger struct {
	fileLogger        *lumberjack.Logger
	elasticsearchHook *ElasticsearchHook
	fluentdHook       *FluentdHook
	logger            log.Logger
}

// NewELKLogger 创建ELK日志记录器
func NewELKLogger(config *ELKConfig, logger log.Logger) (*ELKLogger, error) {
	if config == nil {
		config = DefaultELKConfig()
	}

	elk := &ELKLogger{
		logger: logger,
	}

	// 初始化文件日志
	if config.File.Enabled {
		// 确保日志目录存在
		logDir := filepath.Dir(config.File.Path)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		elk.fileLogger = &lumberjack.Logger{
			Filename:   config.File.Path,
			MaxSize:    config.File.MaxSize,
			MaxBackups: config.File.MaxBackups,
			MaxAge:     config.File.MaxAge,
			Compress:   config.File.Compress,
		}
	}

	// 初始化Elasticsearch钩子
	if config.Elasticsearch.Enabled {
		hook, err := NewElasticsearchHook(config.Elasticsearch.Hosts, config.Elasticsearch.Index, config.Elasticsearch.Username, config.Elasticsearch.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to create elasticsearch hook: %w", err)
		}
		elk.elasticsearchHook = hook
	}

	// 初始化Fluentd钩子
	if config.Fluentd.Enabled {
		hook, err := NewFluentdHook(config.Fluentd.Host, config.Fluentd.Port, config.Fluentd.Tag)
		if err != nil {
			return nil, fmt.Errorf("failed to create fluentd hook: %w", err)
		}
		elk.fluentdHook = hook
	}

	return elk, nil
}

// Log 记录日志
func (e *ELKLogger) Log(level log.Level, keyvals ...interface{}) error {
	// 写入文件日志
	if e.fileLogger != nil {
		// 格式化日志消息
		msg := formatLogMessage(level, keyvals...)
		if _, err := e.fileLogger.Write([]byte(msg + "\n")); err != nil {
			log.NewHelper(e.logger).Errorf("Failed to write to file logger: %v", err)
		}
	}

	// 发送到Elasticsearch
	if e.elasticsearchHook != nil {
		if err := e.elasticsearchHook.Send(level, keyvals...); err != nil {
			log.NewHelper(e.logger).Errorf("Failed to send to elasticsearch: %v", err)
		}
	}

	// 发送到Fluentd
	if e.fluentdHook != nil {
		if err := e.fluentdHook.Send(level, keyvals...); err != nil {
			log.NewHelper(e.logger).Errorf("Failed to send to fluentd: %v", err)
		}
	}

	return nil
}

// Close 关闭日志记录器
func (e *ELKLogger) Close() error {
	if e.fileLogger != nil {
		if err := e.fileLogger.Close(); err != nil {
			log.NewHelper(e.logger).Errorf("Failed to close file logger: %v", err)
		}
	}

	if e.elasticsearchHook != nil {
		e.elasticsearchHook.Close()
	}

	if e.fluentdHook != nil {
		e.fluentdHook.Close()
	}

	return nil
}

// formatLogMessage 格式化日志消息
func formatLogMessage(level log.Level, keyvals ...interface{}) string {
	// 创建时间戳
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")

	// 格式化键值对
	var msg string
	for i := 0; i < len(keyvals); i += 2 {
		if i > 0 {
			msg += " "
		}
		if i+1 < len(keyvals) {
			msg += fmt.Sprintf("%v=%v", keyvals[i], keyvals[i+1])
		} else {
			msg += fmt.Sprintf("%v=null", keyvals[i])
		}
	}

	return fmt.Sprintf("%s level=%s %s", timestamp, level.String(), msg)
}

// ElasticsearchHook Elasticsearch钩子
type ElasticsearchHook struct {
	hosts    []string
	index    string
	username string
	password string
	client   *http.Client
}

// NewElasticsearchHook 创建Elasticsearch钩子
func NewElasticsearchHook(hosts []string, index, username, password string) (*ElasticsearchHook, error) {
	hook := &ElasticsearchHook{
		hosts:    hosts,
		index:    index,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return hook, nil
}

// Send 发送日志到Elasticsearch
func (e *ElasticsearchHook) Send(level log.Level, keyvals ...interface{}) error {
	// 这里应该实现发送到Elasticsearch的逻辑
	// 为简化起见，我们只记录一条消息
	fmt.Printf("Sending log to Elasticsearch: level=%s, data=%v\n", level, keyvals)
	return nil
}

// Close 关闭Elasticsearch钩子
func (e *ElasticsearchHook) Close() {
	// 清理资源
}

// FluentdHook Fluentd钩子
type FluentdHook struct {
	host string
	port int
	tag  string
	conn net.Conn
}

// NewFluentdHook 创建Fluentd钩子
func NewFluentdHook(host string, port int, tag string) (*FluentdHook, error) {
	hook := &FluentdHook{
		host: host,
		port: port,
		tag:  tag,
	}

	return hook, nil
}

// Send 发送日志到Fluentd
func (f *FluentdHook) Send(level log.Level, keyvals ...interface{}) error {
	// 这里应该实现发送到Fluentd的逻辑
	// 为简化起见，我们只记录一条消息
	fmt.Printf("Sending log to Fluentd: level=%s, data=%v\n", level, keyvals)
	return nil
}

// Close 关闭Fluentd钩子
func (f *FluentdHook) Close() {
	// 清理资源
	if f.conn != nil {
		f.conn.Close()
	}
}