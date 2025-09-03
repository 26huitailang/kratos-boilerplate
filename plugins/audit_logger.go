//go:build plugin

package main

import (
	"context"
	"fmt"
	"time"

	"kratos-boilerplate/internal/pkg/plugin"
)

// AuditLoggerPlugin 审计日志插件
type AuditLoggerPlugin struct {
	name    string
	config  plugin.PluginConfig
	started bool
}

// NewAuditLoggerPlugin 创建审计日志插件
func NewAuditLoggerPlugin() *AuditLoggerPlugin {
	return &AuditLoggerPlugin{
		name: "audit_logger",
	}
}

func (p *AuditLoggerPlugin) Name() string {
	return p.name
}

func (p *AuditLoggerPlugin) Version() string {
	return "1.0.0"
}

func (p *AuditLoggerPlugin) Description() string {
	return "Audit logging plugin for security and compliance"
}

func (p *AuditLoggerPlugin) Dependencies() []string {
	return nil
}

func (p *AuditLoggerPlugin) Initialize(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config
	fmt.Printf("AuditLoggerPlugin initialized with config: %+v\n", config)
	return nil
}

func (p *AuditLoggerPlugin) Start(ctx context.Context) error {
	p.started = true
	fmt.Println("AuditLoggerPlugin started")
	return nil
}

func (p *AuditLoggerPlugin) Stop(ctx context.Context) error {
	p.started = false
	fmt.Println("AuditLoggerPlugin stopped")
	return nil
}

func (p *AuditLoggerPlugin) Cleanup(ctx context.Context) error {
	fmt.Println("AuditLoggerPlugin cleaned up")
	return nil
}

func (p *AuditLoggerPlugin) HealthCheck(ctx context.Context) error {
	if !p.started {
		return fmt.Errorf("plugin not started")
	}
	return nil
}

func (p *AuditLoggerPlugin) RegisterHooks(manager plugin.HookManager) error {
	// 注册请求前置钩子
	requestHook := plugin.NewBaseHook(
		"audit_logger_pre_request",
		5, // 高优先级
		3*time.Second,
		func(ctx context.Context, data plugin.HookData) error {
			fmt.Println("AuditLoggerPlugin: Pre-request hook executed")
			// 在这里可以记录请求开始时间、用户信息等
			return nil
		},
	)

	if err := manager.RegisterHook(plugin.HookPointBeforeRequest, requestHook); err != nil {
		return err
	}

	// 注册请求后置钩子
	responseHook := plugin.NewBaseHook(
		"audit_logger_post_request",
		15, // 较低优先级
		3*time.Second,
		func(ctx context.Context, data plugin.HookData) error {
			fmt.Println("AuditLoggerPlugin: Post-request hook executed")
			// 在这里可以记录请求完成时间、响应状态等
			return nil
		},
	)

	return manager.RegisterHook(plugin.HookPointAfterRequest, responseHook)
}

func (p *AuditLoggerPlugin) RegisterEventHandlers(bus plugin.EventBus) error {
	// 注册多种事件处理器
	events := []plugin.EventType{
		plugin.EventUserLogin,
		plugin.EventUserLogout,
		plugin.EventDataCreated,
		plugin.EventDataUpdated,
		plugin.EventDataDeleted,
		plugin.EventPluginLoaded,
		plugin.EventPluginStarted,
	}

	eventHandler := plugin.NewBaseEventHandler(
		"audit_logger_event_handler",
		events,
		15*time.Second,
		func(ctx context.Context, event plugin.Event) error {
			fmt.Printf("AuditLoggerPlugin: Event %s received: %+v\n", event.GetType(), event.GetData())
			// 在这里可以记录详细的审计日志
			return nil
		},
	)

	return bus.Subscribe(plugin.EventUserLogin, eventHandler)
}

// 导出插件实例
var AuditLoggerPluginInstance = NewAuditLoggerPlugin()