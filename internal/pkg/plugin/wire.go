package plugin

import "github.com/google/wire"

// ProviderSet 插件系统依赖注入配置
var ProviderSet = wire.NewSet(
	NewPluginManager,
	NewPluginRegistry,
	NewHookManager,
	NewEventBus,
	wire.Bind(new(PluginManager), new(*pluginManagerImpl)),
	wire.Bind(new(PluginRegistry), new(*pluginRegistryImpl)),
	wire.Bind(new(HookManager), new(*hookManagerImpl)),
	wire.Bind(new(EventBus), new(*eventBusImpl)),
)