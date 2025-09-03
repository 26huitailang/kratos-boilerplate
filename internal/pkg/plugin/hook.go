package plugin

import (
	"context"
	"time"
)

// HookPoint 钩子点类型
type HookPoint string

// 预定义钩子点常量
const (
	// 请求处理钩子点
	HookPointBeforeRequest  HookPoint = "before_request"
	HookPointAfterRequest   HookPoint = "after_request"
	HookPointBeforeResponse HookPoint = "before_response"
	HookPointAfterResponse  HookPoint = "after_response"

	// 认证相关钩子点
	HookPointBeforeAuth HookPoint = "before_auth"
	HookPointAfterAuth  HookPoint = "after_auth"
	HookPointAuthFailed HookPoint = "auth_failed"

	// 业务逻辑钩子点
	HookPointBeforeBiz HookPoint = "before_biz"
	HookPointAfterBiz  HookPoint = "after_biz"
	HookPointBizError  HookPoint = "biz_error"

	// 数据访问钩子点
	HookPointBeforeData HookPoint = "before_data"
	HookPointAfterData  HookPoint = "after_data"
	HookPointDataError  HookPoint = "data_error"
)

// HookData 钩子数据接口
type HookData interface {
	// GetContext 获取上下文
	GetContext() context.Context
	// GetData 获取数据
	GetData() map[string]interface{}
	// SetData 设置数据
	SetData(key string, value interface{})
	// GetMetadata 获取元数据
	GetMetadata() map[string]string
}

// Hook 钩子接口
type Hook interface {
	// GetName 获取钩子名称
	GetName() string
	// GetPriority 获取优先级 (数值越小优先级越高)
	GetPriority() int
	// Execute 执行钩子
	Execute(ctx context.Context, data HookData) error
	// GetTimeout 获取执行超时时间
	GetTimeout() time.Duration
}

// HookManager 钩子管理器接口
type HookManager interface {
	// RegisterHook 注册钩子
	RegisterHook(point HookPoint, hook Hook) error
	// UnregisterHook 注销钩子
	UnregisterHook(point HookPoint, hookName string) error
	// ExecuteHooks 执行钩子点的所有钩子
	ExecuteHooks(ctx context.Context, point HookPoint, data HookData) error
	// ListHooks 列出钩子点的所有钩子
	ListHooks(point HookPoint) []Hook
	// GetHook 获取指定钩子
	GetHook(point HookPoint, hookName string) (Hook, error)
}
