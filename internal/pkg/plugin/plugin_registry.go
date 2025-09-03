package plugin

import (
	"sort"
	"sync"
)

// pluginRegistryImpl 插件注册表实现
type pluginRegistryImpl struct {
	mu       sync.RWMutex
	plugins  map[string]Plugin
	byStatus map[PluginStatus][]Plugin
}

// NewPluginRegistry 创建新的插件注册表
func NewPluginRegistry() PluginRegistry {
	return &pluginRegistryImpl{
		plugins:  make(map[string]Plugin),
		byStatus: make(map[PluginStatus][]Plugin),
	}
}

// Register 注册插件
func (pr *pluginRegistryImpl) Register(plugin Plugin) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	name := plugin.Name()
	if _, exists := pr.plugins[name]; exists {
		return NewPluginError(ErrCodePluginAlreadyExist, "plugin already registered", name, nil)
	}

	// 检查依赖关系
	if err := pr.checkDependencies(plugin); err != nil {
		return err
	}

	pr.plugins[name] = plugin
	pr.byStatus[PluginStatusLoaded] = append(pr.byStatus[PluginStatusLoaded], plugin)

	return nil
}

// Unregister 注销插件
func (pr *pluginRegistryImpl) Unregister(name string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	_, exists := pr.plugins[name]
	if !exists {
		return NewPluginError(ErrCodePluginNotFound, "plugin not found", name, nil)
	}

	// 检查是否有其他插件依赖此插件
	if err := pr.checkDependents(name); err != nil {
		return err
	}

	delete(pr.plugins, name)

	// 从状态分类中移除
	for status, plugins := range pr.byStatus {
		for i, p := range plugins {
			if p.Name() == name {
				pr.byStatus[status] = append(plugins[:i], plugins[i+1:]...)
				break
			}
		}
	}

	return nil
}

// Get 获取插件
func (pr *pluginRegistryImpl) Get(name string) (Plugin, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugin, exists := pr.plugins[name]
	if !exists {
		return nil, NewPluginError(ErrCodePluginNotFound, "plugin not found", name, nil)
	}

	return plugin, nil
}

// List 列出所有插件
func (pr *pluginRegistryImpl) List() []Plugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	result := make([]Plugin, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		result = append(result, plugin)
	}

	return result
}

// Exists 检查插件是否存在
func (pr *pluginRegistryImpl) Exists(name string) bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	_, exists := pr.plugins[name]
	return exists
}

// ListByStatus 按状态列出插件
func (pr *pluginRegistryImpl) ListByStatus(status PluginStatus) []Plugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return pr.byStatus[status]
}

// UpdateStatus 更新插件状态
func (pr *pluginRegistryImpl) UpdateStatus(plugin Plugin, oldStatus, newStatus PluginStatus) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// 从旧状态中移除
	if plugins, exists := pr.byStatus[oldStatus]; exists {
		for i, p := range plugins {
			if p.Name() == plugin.Name() {
				pr.byStatus[oldStatus] = append(plugins[:i], plugins[i+1:]...)
				break
			}
		}
	}

	// 添加到新状态
	pr.byStatus[newStatus] = append(pr.byStatus[newStatus], plugin)
}

// GetLoadOrder 获取插件加载顺序
func (pr *pluginRegistryImpl) GetLoadOrder() ([]Plugin, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	// 构建依赖图（反向关系：graph[node] = 依赖node的插件列表）
	dependencyGraph := make(map[string][]string)
	plugins := make([]Plugin, 0, len(pr.plugins))

	// 初始化图
	for name := range pr.plugins {
		dependencyGraph[name] = []string{}
	}

	// 构建依赖关系
	for name, plugin := range pr.plugins {
		plugins = append(plugins, plugin)
		for _, dep := range plugin.Dependencies() {
			dependencyGraph[dep] = append(dependencyGraph[dep], name)
		}
	}

	// 拓扑排序
	sorted, err := pr.topologicalSort(dependencyGraph)
	if err != nil {
		return nil, err
	}

	// 按排序顺序返回插件
	result := make([]Plugin, 0, len(sorted))
	for _, name := range sorted {
		result = append(result, pr.plugins[name])
	}

	return result, nil
}

// checkDependencies 检查插件依赖关系
func (pr *pluginRegistryImpl) checkDependencies(plugin Plugin) error {
	dependencies := plugin.Dependencies()
	if len(dependencies) == 0 {
		return nil
	}

	for _, dep := range dependencies {
		if _, exists := pr.plugins[dep]; !exists {
			return NewPluginError(ErrCodePluginDependency, "dependency not found: "+dep, plugin.Name(), nil)
		}
	}

	return nil
}

// checkDependents 检查是否有其他插件依赖此插件
func (pr *pluginRegistryImpl) checkDependents(pluginName string) error {
	for _, p := range pr.plugins {
		for _, dep := range p.Dependencies() {
			if dep == pluginName {
				return NewPluginError(ErrCodePluginDependency, "other plugins depend on this plugin", pluginName, nil)
			}
		}
	}

	return nil
}

// topologicalSort 拓扑排序
func (pr *pluginRegistryImpl) topologicalSort(graph map[string][]string) ([]string, error) {
	inDegree := make(map[string]int)
	for node := range graph {
		inDegree[node] = 0
	}

	// 计算入度
	for _, dependencies := range graph {
		for _, dep := range dependencies {
			inDegree[dep]++
		}
	}

	// 找到入度为0的节点
	queue := make([]string, 0)
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	result := make([]string, 0)
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, neighbor := range graph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// 检查是否有环
	if len(result) != len(graph) {
		return nil, NewPluginError(ErrCodePluginDependency, "circular dependency detected", "", nil)
	}

	return result, nil
}

// GetDependents 获取依赖此插件的插件列表
func (pr *pluginRegistryImpl) GetDependents(pluginName string) []Plugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	result := make([]Plugin, 0)
	for _, plugin := range pr.plugins {
		for _, dep := range plugin.Dependencies() {
			if dep == pluginName {
				result = append(result, plugin)
				break
			}
		}
	}

	return result
}

// GetDependencies 获取此插件依赖的插件列表
func (pr *pluginRegistryImpl) GetDependencies(pluginName string) ([]Plugin, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugin, exists := pr.plugins[pluginName]
	if !exists {
		return nil, NewPluginError(ErrCodePluginNotFound, "plugin not found", pluginName, nil)
	}

	result := make([]Plugin, 0)
	for _, depName := range plugin.Dependencies() {
		if depPlugin, exists := pr.plugins[depName]; exists {
			result = append(result, depPlugin)
		}
	}

	return result, nil
}

// FilterByMetadata 根据元数据过滤插件
func (pr *pluginRegistryImpl) FilterByMetadata(key, value string) []Plugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	result := make([]Plugin, 0)
	for _, p := range pr.plugins {
		// 简化实现：只检查名称包含
		if p.Name() == value {
			result = append(result, p)
		}
	}

	return result
}

// SortByPriority 按优先级排序插件
func (pr *pluginRegistryImpl) SortByPriority(plugins []Plugin) []Plugin {
	sorted := make([]Plugin, len(plugins))
	copy(sorted, plugins)

	sort.Slice(sorted, func(i, j int) bool {
		// 这里需要根据实际接口获取优先级
		// 简化实现：按名称排序
		return sorted[i].Name() < sorted[j].Name()
	})

	return sorted
}