package plugin

import (
	"context"
)

// Plugin 插件基础接口
type Plugin interface {
	// 插件元数据
	Name() string
	Version() string
	Description() string
	Dependencies() []string

	// 生命周期方法
	Initialize(ctx context.Context, config PluginConfig) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Cleanup(ctx context.Context) error

	// 健康检查
	HealthCheck(ctx context.Context) error
}

// HookPlugin 支持钩子的插件接口
type HookPlugin interface {
	Plugin
	RegisterHooks(manager HookManager) error
}

// EventPlugin 支持事件的插件接口
type EventPlugin interface {
	Plugin
	RegisterEventHandlers(bus EventBus) error
}

// PluginManager 插件管理器接口
type PluginManager interface {
	// 插件生命周期管理
	LoadPlugin(path string) error
	UnloadPlugin(name string) error
	StartPlugin(name string) error
	StopPlugin(name string) error

	// 插件查询
	GetPlugin(name string) (Plugin, error)
	ListPlugins() []PluginInfo
	GetPluginStatus(name string) (PluginStatus, error)

	// 配置管理
	UpdatePluginConfig(name string, config PluginConfig) error
	GetPluginConfig(name string) (PluginConfig, error)
}

// PluginRegistry 插件注册表接口
type PluginRegistry interface {
	Register(plugin Plugin) error
	Unregister(name string) error
	Get(name string) (Plugin, error)
	List() []Plugin
	Exists(name string) bool
	ListByStatus(status PluginStatus) []Plugin
	UpdateStatus(plugin Plugin, oldStatus, newStatus PluginStatus)
	GetLoadOrder() ([]Plugin, error)
	GetDependents(pluginName string) []Plugin
	GetDependencies(pluginName string) ([]Plugin, error)
	FilterByMetadata(key, value string) []Plugin
	SortByPriority(plugins []Plugin) []Plugin
}
