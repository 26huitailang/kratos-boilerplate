package plugin

import (
	"context"
	"time"
)

// EventType 事件类型
type EventType string

// 预定义事件类型常量
const (
	// 插件生命周期事件
	EventPluginLoaded   EventType = "plugin.loaded"
	EventPluginUnloaded EventType = "plugin.unloaded"
	EventPluginStarted  EventType = "plugin.started"
	EventPluginStopped  EventType = "plugin.stopped"
	EventPluginError    EventType = "plugin.error"

	// 系统事件
	EventSystemStartup  EventType = "system.startup"
	EventSystemShutdown EventType = "system.shutdown"
	EventConfigChanged  EventType = "config.changed"

	// 业务事件
	EventUserLogin   EventType = "user.login"
	EventUserLogout  EventType = "user.logout"
	EventDataCreated EventType = "data.created"
	EventDataUpdated EventType = "data.updated"
	EventDataDeleted EventType = "data.deleted"
)

// Event 事件接口
type Event interface {
	// GetID 获取事件ID
	GetID() string
	// GetType 获取事件类型
	GetType() EventType
	// GetSource 获取事件源
	GetSource() string
	// GetTimestamp 获取事件时间戳
	GetTimestamp() time.Time
	// GetData 获取事件数据
	GetData() map[string]interface{}
	// GetMetadata 获取事件元数据
	GetMetadata() map[string]string
}

// EventHandler 事件处理器接口
type EventHandler interface {
	// GetName 获取处理器名称
	GetName() string
	// GetEventTypes 获取处理的事件类型
	GetEventTypes() []EventType
	// Handle 处理事件
	Handle(ctx context.Context, event Event) error
	// GetTimeout 获取处理超时时间
	GetTimeout() time.Duration
}

// EventFilter 事件过滤器接口
type EventFilter interface {
	// Match 检查事件是否匹配过滤条件
	Match(event Event) bool
}

// EventBus 事件总线接口
type EventBus interface {
	// Subscribe 订阅事件
	Subscribe(eventType EventType, handler EventHandler) error
	// Unsubscribe 取消订阅
	Unsubscribe(eventType EventType, handlerName string) error
	// Publish 发布事件
	Publish(ctx context.Context, event Event) error
	// PublishAsync 异步发布事件
	PublishAsync(ctx context.Context, event Event) error
	// AddFilter 添加事件过滤器
	AddFilter(filter EventFilter) error
	// RemoveFilter 移除事件过滤器
	RemoveFilter(filter EventFilter) error
}
