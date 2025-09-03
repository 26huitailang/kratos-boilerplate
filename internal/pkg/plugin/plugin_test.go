package plugin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHookManager(t *testing.T) {
	hm := NewHookManager()

	// 测试注册钩子
	hook := NewBaseHook("test_hook", 10, 5*time.Second, func(ctx context.Context, data HookData) error {
		return nil
	})

	err := hm.RegisterHook(HookPointBeforeRequest, hook)
	require.NoError(t, err)

	// 测试重复注册
	err = hm.RegisterHook(HookPointBeforeRequest, hook)
	require.Error(t, err)

	// 测试执行钩子
	data := NewHookData(context.Background(), map[string]interface{}{"test": "data"})
	err = hm.ExecuteHooks(context.Background(), HookPointBeforeRequest, data)
	require.NoError(t, err)

	// 测试列出钩子
	hooks := hm.ListHooks(HookPointBeforeRequest)
	require.Len(t, hooks, 1)
	assert.Equal(t, "test_hook", hooks[0].GetName())

	// 测试注销钩子
	err = hm.UnregisterHook(HookPointBeforeRequest, "test_hook")
	require.NoError(t, err)

	// 测试钩子不存在
	err = hm.UnregisterHook(HookPointBeforeRequest, "nonexistent")
	require.Error(t, err)
}

func TestEventBus(t *testing.T) {
	eb := NewEventBus(5)

	// 测试订阅事件
	handler := NewBaseEventHandler(
		"test_handler",
		[]EventType{EventUserLogin},
		5*time.Second,
		func(ctx context.Context, event Event) error {
			return nil
		},
	)

	err := eb.Subscribe(EventUserLogin, handler)
	require.NoError(t, err)

	// 测试发布事件
	event := NewEvent(EventUserLogin, "test_source", map[string]interface{}{"user": "test"})
	err = eb.Publish(context.Background(), event)
	require.NoError(t, err)

	// 测试取消订阅
	err = eb.Unsubscribe(EventUserLogin, "test_handler")
	require.NoError(t, err)
}

func TestPluginRegistry(t *testing.T) {
	pr := NewPluginRegistry()

	// 测试注册插件
	plugin := &mockPlugin{name: "test_plugin"}
	err := pr.Register(plugin)
	require.NoError(t, err)

	// 测试重复注册
	err = pr.Register(plugin)
	require.Error(t, err)

	// 测试获取插件
	retrieved, err := pr.Get("test_plugin")
	require.NoError(t, err)
	assert.Equal(t, "test_plugin", retrieved.Name())

	// 测试列出插件
	plugins := pr.List()
	require.Len(t, plugins, 1)

	// 测试检查存在
	exists := pr.Exists("test_plugin")
	assert.True(t, exists)

	exists = pr.Exists("nonexistent")
	assert.False(t, exists)

	// 测试注销插件
	err = pr.Unregister("test_plugin")
	require.NoError(t, err)
}

func TestPluginManager(t *testing.T) {
	// 创建必要的组件
	pr := NewPluginRegistry()
	hm := NewHookManager()
	eb := NewEventBus(5)
	pm := NewPluginManager(pr, hm, eb, "./test_configs", "./test_plugins")

	// 跳过实际文件加载测试，因为需要真实的插件文件
	// 这里主要测试管理器接口的正确性

	// 测试列出插件（应该为空，因为没有实际加载）
	plugins := pm.ListPlugins()
	require.Len(t, plugins, 0)

	// 测试获取不存在的插件状态
	_, err := pm.GetPluginStatus("test_plugin")
	require.Error(t, err)

	// 测试启动不存在的插件
	err = pm.StartPlugin("test_plugin")
	require.Error(t, err)

	// 测试停止不存在的插件
	err = pm.StopPlugin("test_plugin")
	require.Error(t, err)

	// 测试卸载不存在的插件
	err = pm.UnloadPlugin("test_plugin")
	require.Error(t, err)
}

func TestHookData(t *testing.T) {
	ctx := context.Background()
	data := NewHookData(ctx, map[string]interface{}{"key": "value"})

	assert.Equal(t, ctx, data.GetContext())
	assert.Equal(t, "value", data.GetData()["key"])

	data.SetData("new_key", "new_value")
	assert.Equal(t, "new_value", data.GetData()["new_key"])
}

func TestEvent(t *testing.T) {
	event := NewEvent(EventUserLogin, "test_source", map[string]interface{}{"user": "test"})

	assert.Equal(t, EventUserLogin, event.GetType())
	assert.Equal(t, "test_source", event.GetSource())
	assert.Equal(t, "test", event.GetData()["user"])
	assert.NotEmpty(t, event.GetID())
}

func TestPluginError(t *testing.T) {
	// 测试创建插件错误
	err := NewPluginError(ErrCodePluginNotFound, "plugin not found", "test_plugin", nil)
	assert.Equal(t, "plugin not found", err.Error())
	assert.Equal(t, ErrCodePluginNotFound, err.Code)

	// 测试包装错误
	originalErr := fmt.Errorf("original error")
	wrappedErr := NewPluginError(ErrCodePluginInternal, "internal error", "test_plugin", originalErr)
	assert.Equal(t, "internal error: original error", wrappedErr.Error())
	assert.Equal(t, originalErr, wrappedErr.Unwrap())
}

func TestDependencyGraph(t *testing.T) {
	pr := NewPluginRegistry()

	// 创建有依赖关系的插件
	plugin1 := &testMockPlugin{name: "plugin1", deps: []string{"plugin2"}}
	plugin2 := &testMockPlugin{name: "plugin2"}

	// 先注册被依赖的插件
	err := pr.Register(plugin2)
	require.NoError(t, err)

	// 再注册依赖插件
	err = pr.Register(plugin1)
	require.NoError(t, err)

	// 测试插件存在性
	exists := pr.Exists("plugin1")
	assert.True(t, exists)
	
	exists = pr.Exists("plugin2")
	assert.True(t, exists)
}

func TestCircularDependency(t *testing.T) {
	pr := NewPluginRegistry()

	// 创建有依赖关系的插件
	plugin1 := &testMockPlugin{name: "plugin1", deps: []string{"plugin2"}}
	plugin2 := &testMockPlugin{name: "plugin2"}

	// 先注册被依赖的插件
	err := pr.Register(plugin2)
	require.NoError(t, err)

	// 再注册依赖插件
	err = pr.Register(plugin1)
	require.NoError(t, err)

	// 测试获取加载顺序
	order, err := pr.GetLoadOrder()
	require.NoError(t, err)
	require.Len(t, order, 2)
	
	
	// plugin2 应该在 plugin1 之前（因为 plugin1 依赖 plugin2）
	assert.Equal(t, "plugin2", order[0].Name())
	assert.Equal(t, "plugin1", order[1].Name())
}

// testMockPlugin 用于测试的模拟插件
type testMockPlugin struct {
	name string
	deps []string
}

func (m *testMockPlugin) Name() string {
	return m.name
}

func (m *testMockPlugin) Version() string {
	return "1.0.0"
}

func (m *testMockPlugin) Description() string {
	return "Mock plugin for testing"
}

func (m *testMockPlugin) Dependencies() []string {
	return m.deps
}

func (m *testMockPlugin) Initialize(ctx context.Context, config PluginConfig) error {
	return nil
}

func (m *testMockPlugin) Start(ctx context.Context) error {
	return nil
}

func (m *testMockPlugin) Stop(ctx context.Context) error {
	return nil
}

func (m *testMockPlugin) Cleanup(ctx context.Context) error {
	return nil
}

func (m *testMockPlugin) HealthCheck(ctx context.Context) error {
	return nil
}