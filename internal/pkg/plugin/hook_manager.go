package plugin

import (
	"context"
	"sort"
	"sync"
	"time"
)

// hookEntry 钩子条目
type hookEntry struct {
	hook     Hook
	priority int
}

// hookManagerImpl 钩子管理器实现
type hookManagerImpl struct {
	mu     sync.RWMutex
	hooks  map[HookPoint][]hookEntry
	timers map[string]*time.Timer
}

// NewHookManager 创建新的钩子管理器
func NewHookManager() HookManager {
	return &hookManagerImpl{
		hooks:  make(map[HookPoint][]hookEntry),
		timers: make(map[string]*time.Timer),
	}
}

// RegisterHook 注册钩子
func (hm *hookManagerImpl) RegisterHook(point HookPoint, hook Hook) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// 检查钩子是否已存在
	for _, entry := range hm.hooks[point] {
		if entry.hook.GetName() == hook.GetName() {
			return NewPluginError(ErrCodePluginAlreadyExist, "hook already exists", hook.GetName(), nil)
		}
	}

	// 添加钩子并按优先级排序
	hm.hooks[point] = append(hm.hooks[point], hookEntry{
		hook:     hook,
		priority: hook.GetPriority(),
	})

	// 按优先级排序（数值越小优先级越高）
	sort.Slice(hm.hooks[point], func(i, j int) bool {
		return hm.hooks[point][i].priority < hm.hooks[point][j].priority
	})

	return nil
}

// UnregisterHook 注销钩子
func (hm *hookManagerImpl) UnregisterHook(point HookPoint, hookName string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hooks, exists := hm.hooks[point]
	if !exists {
		return NewPluginError(ErrCodePluginNotFound, "hook point not found", hookName, nil)
	}

	for i, entry := range hooks {
		if entry.hook.GetName() == hookName {
			hm.hooks[point] = append(hooks[:i], hooks[i+1:]...)
			return nil
		}
	}

	return NewPluginError(ErrCodePluginNotFound, "hook not found", hookName, nil)
}

// ExecuteHooks 执行钩子点的所有钩子
func (hm *hookManagerImpl) ExecuteHooks(ctx context.Context, point HookPoint, data HookData) error {
	hm.mu.RLock()
	hooks, exists := hm.hooks[point]
	hm.mu.RUnlock()

	if !exists || len(hooks) == 0 {
		return nil
	}

	var lastError error

	for _, entry := range hooks {
		hook := entry.hook
		timeout := hook.GetTimeout()
		if timeout == 0 {
			timeout = 30 * time.Second // 默认超时时间
		}

		hookCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// 执行钩子
		if err := hm.executeSingleHook(hookCtx, hook, data); err != nil {
			lastError = err
			// 继续执行其他钩子
		}
	}

	return lastError
}

// executeSingleHook 执行单个钩子
func (hm *hookManagerImpl) executeSingleHook(ctx context.Context, hook Hook, data HookData) error {
	hookName := hook.GetName()
	requestID, _ := data.GetContext().Value("request_id").(string)
	hookID := requestID + "_" + hookName

	// 设置超时定时器
	timer := time.AfterFunc(hook.GetTimeout(), func() {
		// 超时处理
	})
	hm.timers[hookID] = timer
	defer delete(hm.timers, hookID)

	// 执行钩子
	if err := hook.Execute(ctx, data); err != nil {
		return NewPluginError(ErrCodePluginInternal, "hook execution failed", hookName, err)
	}

	timer.Stop()
	return nil
}

// ListHooks 列出钩子点的所有钩子
func (hm *hookManagerImpl) ListHooks(point HookPoint) []Hook {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	hooks, exists := hm.hooks[point]
	if !exists {
		return nil
	}

	result := make([]Hook, len(hooks))
	for i, entry := range hooks {
		result[i] = entry.hook
	}

	return result
}

// GetHook 获取指定钩子
func (hm *hookManagerImpl) GetHook(point HookPoint, hookName string) (Hook, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	hooks, exists := hm.hooks[point]
	if !exists {
		return nil, NewPluginError(ErrCodePluginNotFound, "hook point not found", hookName, nil)
	}

	for _, entry := range hooks {
		if entry.hook.GetName() == hookName {
			return entry.hook, nil
		}
	}

	return nil, NewPluginError(ErrCodePluginNotFound, "hook not found", hookName, nil)
}

// hookDataImpl 钩子数据实现
type hookDataImpl struct {
	ctx      context.Context
	data     map[string]interface{}
	metadata map[string]string
}

// NewHookData 创建新的钩子数据
func NewHookData(ctx context.Context, initialData map[string]interface{}) HookData {
	return &hookDataImpl{
		ctx:      ctx,
		data:     initialData,
		metadata: make(map[string]string),
	}
}

func (hd *hookDataImpl) GetContext() context.Context {
	return hd.ctx
}

func (hd *hookDataImpl) GetData() map[string]interface{} {
	return hd.data
}

func (hd *hookDataImpl) SetData(key string, value interface{}) {
	if hd.data == nil {
		hd.data = make(map[string]interface{})
	}
	hd.data[key] = value
}

func (hd *hookDataImpl) GetMetadata() map[string]string {
	return hd.metadata
}

// baseHook 基础钩子实现
type baseHook struct {
	name     string
	priority int
	timeout  time.Duration
	handler  func(context.Context, HookData) error
}

// NewBaseHook 创建基础钩子
func NewBaseHook(name string, priority int, timeout time.Duration, handler func(context.Context, HookData) error) Hook {
	return &baseHook{
		name:     name,
		priority: priority,
		timeout:  timeout,
		handler:  handler,
	}
}

func (h *baseHook) GetName() string {
	return h.name
}

func (h *baseHook) GetPriority() int {
	return h.priority
}

func (h *baseHook) GetTimeout() time.Duration {
	return h.timeout
}

func (h *baseHook) Execute(ctx context.Context, data HookData) error {
	return h.handler(ctx, data)
}