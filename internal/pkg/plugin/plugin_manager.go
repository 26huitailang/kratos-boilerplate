package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// pluginManagerImpl 插件管理器实现
type pluginManagerImpl struct {
	mu              sync.RWMutex
	plugins         map[string]*pluginWrapper
	registry        PluginRegistry
	hookManager     HookManager
	eventBus        EventBus
	configDir       string
	pluginDir       string
	autoLoadEnabled bool
}

// pluginWrapper 插件包装器
type pluginWrapper struct {
	plugin      Plugin
	info        PluginInfo
	config      PluginConfig
	hooks       []Hook
	eventHandlers []EventHandler
}

// NewPluginManager 创建新的插件管理器
func NewPluginManager(registry PluginRegistry, hookManager HookManager, eventBus EventBus, configDir, pluginDir string) PluginManager {
	return &pluginManagerImpl{
		plugins:         make(map[string]*pluginWrapper),
		registry:        registry,
		hookManager:     hookManager,
		eventBus:        eventBus,
		configDir:       configDir,
		pluginDir:       pluginDir,
		autoLoadEnabled: true,
	}
}

// LoadPlugin 加载插件
func (pm *pluginManagerImpl) LoadPlugin(path string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查插件文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return NewPluginError(ErrCodePluginNotFound, "plugin file not found", filepath.Base(path), err)
	}

	// 加载插件配置
	config, err := pm.loadPluginConfig(filepath.Base(path))
	if err != nil {
		return err
	}

	if !config.Enabled {
		return NewPluginError(ErrCodePluginConfigError, "plugin is disabled", filepath.Base(path), nil)
	}

	// 这里应该是实际的插件加载逻辑
	// 由于这是示例，我们创建一个模拟插件
	pluginName := filepath.Base(path)
	plugin := &mockPlugin{
		name: pluginName,
	}

	wrapper := &pluginWrapper{
		plugin: plugin,
		info: PluginInfo{
			Metadata: PluginMetadata{
				Name:        pluginName,
				Version:     "1.0.0",
				Description: "Mock plugin for demonstration",
				Author:      "System",
			},
			Status:   PluginStatusLoaded,
			LoadTime: time.Now(),
			Path:     path,
		},
		config: config,
	}

	pm.plugins[pluginName] = wrapper

	// 注册到注册表
	if err := pm.registry.Register(plugin); err != nil {
		delete(pm.plugins, pluginName)
		return err
	}

	// 发布插件加载事件
	pm.eventBus.PublishAsync(context.Background(), NewEvent(
		EventPluginLoaded,
		"plugin_manager",
		map[string]interface{}{
			"plugin": pluginName,
			"path":   path,
		},
	))

	return nil
}

// UnloadPlugin 卸载插件
func (pm *pluginManagerImpl) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	wrapper, exists := pm.plugins[name]
	if !exists {
		return NewPluginError(ErrCodePluginNotFound, "plugin not found", name, nil)
	}

	// 如果插件正在运行，先停止
	if wrapper.info.Status == PluginStatusStarted {
		if err := pm.stopPluginInternal(wrapper); err != nil {
			return err
		}
	}

	// 清理钩子和事件处理器
	for _, hook := range wrapper.hooks {
		// 这里需要根据具体的钩子点来注销钩子
		// 简化实现：清理所有钩子点
		for _, point := range []HookPoint{
			HookPointBeforeRequest, HookPointAfterRequest,
			HookPointBeforeAuth, HookPointAfterAuth,
			HookPointBeforeBiz, HookPointAfterBiz,
			HookPointBeforeData, HookPointAfterData,
		} {
			pm.hookManager.UnregisterHook(point, hook.GetName())
		}
	}

	for _, handler := range wrapper.eventHandlers {
		for _, eventType := range handler.GetEventTypes() {
			pm.eventBus.Unsubscribe(eventType, handler.GetName())
		}
	}

	// 从注册表注销
	if err := pm.registry.Unregister(name); err != nil {
		return err
	}

	delete(pm.plugins, name)

	// 发布插件卸载事件
	pm.eventBus.PublishAsync(context.Background(), NewEvent(
		EventPluginUnloaded,
		"plugin_manager",
		map[string]interface{}{
			"plugin": name,
		},
	))

	return nil
}

// StartPlugin 启动插件
func (pm *pluginManagerImpl) StartPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	wrapper, exists := pm.plugins[name]
	if !exists {
		return NewPluginError(ErrCodePluginNotFound, "plugin not found", name, nil)
	}

	if wrapper.info.Status == PluginStatusStarted {
		return NewPluginError(ErrCodePluginAlreadyExist, "plugin already started", name, nil)
	}

	// 初始化插件
	if err := wrapper.plugin.Initialize(context.Background(), wrapper.config); err != nil {
		wrapper.info.Status = PluginStatusError
		wrapper.info.ErrorMsg = err.Error()
		return NewPluginError(ErrCodePluginStartFailed, "plugin initialization failed", name, err)
	}

	// 注册钩子
	if hookPlugin, ok := wrapper.plugin.(HookPlugin); ok {
		if err := hookPlugin.RegisterHooks(pm.hookManager); err != nil {
			wrapper.info.Status = PluginStatusError
			wrapper.info.ErrorMsg = err.Error()
			return NewPluginError(ErrCodePluginStartFailed, "hook registration failed", name, err)
		}
	}

	// 注册事件处理器
	if eventPlugin, ok := wrapper.plugin.(EventPlugin); ok {
		if err := eventPlugin.RegisterEventHandlers(pm.eventBus); err != nil {
			wrapper.info.Status = PluginStatusError
			wrapper.info.ErrorMsg = err.Error()
			return NewPluginError(ErrCodePluginStartFailed, "event handler registration failed", name, err)
		}
	}

	// 启动插件
	if err := wrapper.plugin.Start(context.Background()); err != nil {
		wrapper.info.Status = PluginStatusError
		wrapper.info.ErrorMsg = err.Error()
		return NewPluginError(ErrCodePluginStartFailed, "plugin start failed", name, err)
	}

	wrapper.info.Status = PluginStatusStarted
	now := time.Now()
	wrapper.info.StartTime = &now

	// 发布插件启动事件
	pm.eventBus.PublishAsync(context.Background(), NewEvent(
		EventPluginStarted,
		"plugin_manager",
		map[string]interface{}{
			"plugin": name,
		},
	))

	return nil
}

// StopPlugin 停止插件
func (pm *pluginManagerImpl) StopPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	wrapper, exists := pm.plugins[name]
	if !exists {
		return NewPluginError(ErrCodePluginNotFound, "plugin not found", name, nil)
	}

	if wrapper.info.Status != PluginStatusStarted {
		return NewPluginError(ErrCodePluginStopFailed, "plugin not running", name, nil)
	}

	return pm.stopPluginInternal(wrapper)
}

// stopPluginInternal 内部停止插件实现
func (pm *pluginManagerImpl) stopPluginInternal(wrapper *pluginWrapper) error {
	if err := wrapper.plugin.Stop(context.Background()); err != nil {
		wrapper.info.Status = PluginStatusError
		wrapper.info.ErrorMsg = err.Error()
		return NewPluginError(ErrCodePluginStopFailed, "plugin stop failed", wrapper.info.Metadata.Name, err)
	}

	wrapper.info.Status = PluginStatusStopped
	now := time.Now()
	wrapper.info.StopTime = &now

	// 发布插件停止事件
	pm.eventBus.PublishAsync(context.Background(), NewEvent(
		EventPluginStopped,
		"plugin_manager",
		map[string]interface{}{
			"plugin": wrapper.info.Metadata.Name,
		},
	))

	return nil
}

// GetPlugin 获取插件
func (pm *pluginManagerImpl) GetPlugin(name string) (Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	wrapper, exists := pm.plugins[name]
	if !exists {
		return nil, NewPluginError(ErrCodePluginNotFound, "plugin not found", name, nil)
	}

	return wrapper.plugin, nil
}

// ListPlugins 列出所有插件
func (pm *pluginManagerImpl) ListPlugins() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]PluginInfo, 0, len(pm.plugins))
	for _, wrapper := range pm.plugins {
		result = append(result, wrapper.info)
	}

	return result
}

// GetPluginStatus 获取插件状态
func (pm *pluginManagerImpl) GetPluginStatus(name string) (PluginStatus, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	wrapper, exists := pm.plugins[name]
	if !exists {
		return "", NewPluginError(ErrCodePluginNotFound, "plugin not found", name, nil)
	}

	return wrapper.info.Status, nil
}

// UpdatePluginConfig 更新插件配置
func (pm *pluginManagerImpl) UpdatePluginConfig(name string, config PluginConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	wrapper, exists := pm.plugins[name]
	if !exists {
		return NewPluginError(ErrCodePluginNotFound, "plugin not found", name, nil)
	}

	// 保存配置到文件
	if err := pm.savePluginConfig(name, config); err != nil {
		return err
	}

	wrapper.config = config

	// 发布配置变更事件
	pm.eventBus.PublishAsync(context.Background(), NewEvent(
		EventConfigChanged,
		"plugin_manager",
		map[string]interface{}{
			"plugin": name,
			"config": config,
		},
	))

	return nil
}

// GetPluginConfig 获取插件配置
func (pm *pluginManagerImpl) GetPluginConfig(name string) (PluginConfig, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	wrapper, exists := pm.plugins[name]
	if !exists {
		return PluginConfig{}, NewPluginError(ErrCodePluginNotFound, "plugin not found", name, nil)
	}

	return wrapper.config, nil
}

// loadPluginConfig 加载插件配置
func (pm *pluginManagerImpl) loadPluginConfig(pluginName string) (PluginConfig, error) {
	configPath := filepath.Join(pm.configDir, pluginName+".yaml")

	config := PluginConfig{
		Enabled:    true,
		Priority:   100,
		Settings:   make(map[string]interface{}),
		Timeout:    30 * time.Second,
		RetryCount: 3,
		Metadata:   make(map[string]string),
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 配置文件不存在，使用默认配置
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, NewPluginError(ErrCodePluginConfigError, "failed to read config file", pluginName, err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, NewPluginError(ErrCodePluginConfigError, "failed to parse config file", pluginName, err)
	}

	return config, nil
}

// savePluginConfig 保存插件配置
func (pm *pluginManagerImpl) savePluginConfig(pluginName string, config PluginConfig) error {
	configPath := filepath.Join(pm.configDir, pluginName+".yaml")

	data, err := yaml.Marshal(config)
	if err != nil {
		return NewPluginError(ErrCodePluginConfigError, "failed to marshal config", pluginName, err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return NewPluginError(ErrCodePluginConfigError, "failed to write config file", pluginName, err)
	}

	return nil
}

// mockPlugin 模拟插件实现
type mockPlugin struct {
	name string
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) Version() string {
	return "1.0.0"
}

func (m *mockPlugin) Description() string {
	return "Mock plugin for demonstration"
}

func (m *mockPlugin) Dependencies() []string {
	return nil
}

func (m *mockPlugin) Initialize(ctx context.Context, config PluginConfig) error {
	fmt.Printf("Plugin %s initialized with config: %+v\n", m.name, config)
	return nil
}

func (m *mockPlugin) Start(ctx context.Context) error {
	fmt.Printf("Plugin %s started\n", m.name)
	return nil
}

func (m *mockPlugin) Stop(ctx context.Context) error {
	fmt.Printf("Plugin %s stopped\n", m.name)
	return nil
}

func (m *mockPlugin) Cleanup(ctx context.Context) error {
	fmt.Printf("Plugin %s cleaned up\n", m.name)
	return nil
}

func (m *mockPlugin) HealthCheck(ctx context.Context) error {
	return nil
}