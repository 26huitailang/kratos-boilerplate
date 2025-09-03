//go:build plugin

package main

import (
	"context"
	"fmt"
	"time"

	"kratos-boilerplate/internal/pkg/plugin"
)

// AuthEnhancerPlugin 认证增强插件
type AuthEnhancerPlugin struct {
	name    string
	config  plugin.PluginConfig
	started bool
}

// NewAuthEnhancerPlugin 创建认证增强插件
func NewAuthEnhancerPlugin() *AuthEnhancerPlugin {
	return &AuthEnhancerPlugin{
		name: "auth_enhancer",
	}
}

func (p *AuthEnhancerPlugin) Name() string {
	return p.name
}

func (p *AuthEnhancerPlugin) Version() string {
	return "1.0.0"
}

func (p *AuthEnhancerPlugin) Description() string {
	return "Authentication enhancement plugin with JWT validation and rate limiting"
}

func (p *AuthEnhancerPlugin) Dependencies() []string {
	return nil
}

func (p *AuthEnhancerPlugin) Initialize(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config
	fmt.Printf("AuthEnhancerPlugin initialized with config: %+v\n", config)
	return nil
}

func (p *AuthEnhancerPlugin) Start(ctx context.Context) error {
	p.started = true
	fmt.Println("AuthEnhancerPlugin started")
	return nil
}

func (p *AuthEnhancerPlugin) Stop(ctx context.Context) error {
	p.started = false
	fmt.Println("AuthEnhancerPlugin stopped")
	return nil
}

func (p *AuthEnhancerPlugin) Cleanup(ctx context.Context) error {
	fmt.Println("AuthEnhancerPlugin cleaned up")
	return nil
}

func (p *AuthEnhancerPlugin) HealthCheck(ctx context.Context) error {
	if !p.started {
		return fmt.Errorf("plugin not started")
	}
	return nil
}

func (p *AuthEnhancerPlugin) RegisterHooks(manager plugin.HookManager) error {
	// 注册认证前置钩子
	authHook := plugin.NewBaseHook(
		"auth_enhancer_pre_auth",
		10, // 高优先级
		5*time.Second,
		func(ctx context.Context, data plugin.HookData) error {
			fmt.Println("AuthEnhancerPlugin: Pre-authentication hook executed")
			// 在这里可以添加JWT验证、速率限制等逻辑
			return nil
		},
	)

	if err := manager.RegisterHook(plugin.HookPointBeforeAuth, authHook); err != nil {
		return err
	}

	// 注册认证后置钩子
	authPostHook := plugin.NewBaseHook(
		"auth_enhancer_post_auth",
		20, // 较低优先级
		5*time.Second,
		func(ctx context.Context, data plugin.HookData) error {
			fmt.Println("AuthEnhancerPlugin: Post-authentication hook executed")
			// 在这里可以添加认证成功后的日志记录、审计等逻辑
			return nil
		},
	)

	return manager.RegisterHook(plugin.HookPointAfterAuth, authPostHook)
}

func (p *AuthEnhancerPlugin) RegisterEventHandlers(bus plugin.EventBus) error {
	// 注册用户登录事件处理器
	loginHandler := plugin.NewBaseEventHandler(
		"auth_enhancer_login_handler",
		[]plugin.EventType{plugin.EventUserLogin},
		10*time.Second,
		func(ctx context.Context, event plugin.Event) error {
			fmt.Printf("AuthEnhancerPlugin: User login event received: %+v\n", event.GetData())
			// 在这里可以添加登录审计、会话管理等逻辑
			return nil
		},
	)

	return bus.Subscribe(plugin.EventUserLogin, loginHandler)
}

// 导出插件实例
var AuthEnhancerPluginInstance = NewAuthEnhancerPlugin()