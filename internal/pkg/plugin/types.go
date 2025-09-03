package plugin

import (
	"time"
)

// PluginStatus 插件状态枚举
type PluginStatus string

const (
	PluginStatusUnloaded    PluginStatus = "unloaded"    // 未加载
	PluginStatusLoaded      PluginStatus = "loaded"      // 已加载
	PluginStatusInitialized PluginStatus = "initialized" // 已初始化
	PluginStatusStarted     PluginStatus = "started"     // 已启动
	PluginStatusStopped     PluginStatus = "stopped"     // 已停止
	PluginStatusError       PluginStatus = "error"       // 错误状态
)

// PluginMetadata 插件元数据
type PluginMetadata struct {
	Name         string            `json:"name" yaml:"name"`
	Version      string            `json:"version" yaml:"version"`
	Description  string            `json:"description" yaml:"description"`
	Author       string            `json:"author" yaml:"author"`
	License      string            `json:"license" yaml:"license"`
	Homepage     string            `json:"homepage" yaml:"homepage"`
	Dependencies []string          `json:"dependencies" yaml:"dependencies"`
	Tags         []string          `json:"tags" yaml:"tags"`
	Metadata     map[string]string `json:"metadata" yaml:"metadata"`
}

// PluginInfo 插件信息
type PluginInfo struct {
	Metadata   PluginMetadata `json:"metadata"`
	Status     PluginStatus   `json:"status"`
	LoadTime   time.Time      `json:"load_time"`
	StartTime  *time.Time     `json:"start_time,omitempty"`
	StopTime   *time.Time     `json:"stop_time,omitempty"`
	ErrorMsg   string         `json:"error_msg,omitempty"`
	Path       string         `json:"path"`
	ConfigPath string         `json:"config_path,omitempty"`
}

// PluginConfig 插件配置
type PluginConfig struct {
	Enabled    bool                   `json:"enabled" yaml:"enabled"`
	Priority   int                    `json:"priority" yaml:"priority"`
	Settings   map[string]interface{} `json:"settings" yaml:"settings"`
	Timeout    time.Duration          `json:"timeout" yaml:"timeout"`
	RetryCount int                    `json:"retry_count" yaml:"retry_count"`
	Metadata   map[string]string      `json:"metadata" yaml:"metadata"`
}

// PluginError 插件错误类型
type PluginError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Plugin  string `json:"plugin"`
	Cause   error  `json:"-"`
}

func (e *PluginError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *PluginError) Unwrap() error {
	return e.Cause
}

// 预定义错误代码
const (
	ErrCodePluginNotFound     = "PLUGIN_NOT_FOUND"
	ErrCodePluginAlreadyExist = "PLUGIN_ALREADY_EXIST"
	ErrCodePluginLoadFailed   = "PLUGIN_LOAD_FAILED"
	ErrCodePluginStartFailed  = "PLUGIN_START_FAILED"
	ErrCodePluginStopFailed   = "PLUGIN_STOP_FAILED"
	ErrCodePluginConfigError  = "PLUGIN_CONFIG_ERROR"
	ErrCodePluginTimeout      = "PLUGIN_TIMEOUT"
	ErrCodePluginDependency   = "PLUGIN_DEPENDENCY_ERROR"
	ErrCodePluginPermission   = "PLUGIN_PERMISSION_ERROR"
	ErrCodePluginInternal     = "PLUGIN_INTERNAL_ERROR"
)

// NewPluginError 创建插件错误
func NewPluginError(code, message, plugin string, cause error) *PluginError {
	return &PluginError{
		Code:    code,
		Message: message,
		Plugin:  plugin,
		Cause:   cause,
	}
}
