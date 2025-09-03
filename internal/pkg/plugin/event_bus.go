package plugin

import (
	"context"
	"sync"
	"time"
	"github.com/google/uuid"
)

// eventBusImpl 事件总线实现
type eventBusImpl struct {
	mu           sync.RWMutex
	subscribers  map[EventType][]*subscription
	filters      []EventFilter
	asyncWorkers int
}

// subscription 订阅信息
type subscription struct {
	handler EventHandler
	id      string
}

// NewEventBus 创建新的事件总线
func NewEventBus(asyncWorkers int) EventBus {
	if asyncWorkers <= 0 {
		asyncWorkers = 10
	}

	return &eventBusImpl{
		subscribers:  make(map[EventType][]*subscription),
		filters:      make([]EventFilter, 0),
		asyncWorkers: asyncWorkers,
	}
}

// Subscribe 订阅事件
func (eb *eventBusImpl) Subscribe(eventType EventType, handler EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	sub := &subscription{
		handler: handler,
		id:      uuid.New().String(),
	}

	eb.subscribers[eventType] = append(eb.subscribers[eventType], sub)
	return nil
}

// Unsubscribe 取消订阅
func (eb *eventBusImpl) Unsubscribe(eventType EventType, handlerName string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs, exists := eb.subscribers[eventType]
	if !exists {
		return NewPluginError(ErrCodePluginNotFound, "event type not found", handlerName, nil)
	}

	for i, sub := range subs {
		if sub.handler.GetName() == handlerName {
			eb.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
			return nil
		}
	}

	return NewPluginError(ErrCodePluginNotFound, "handler not found", handlerName, nil)
}

// Publish 发布事件
func (eb *eventBusImpl) Publish(ctx context.Context, event Event) error {
	return eb.publish(ctx, event, false)
}

// PublishAsync 异步发布事件
func (eb *eventBusImpl) PublishAsync(ctx context.Context, event Event) error {
	return eb.publish(ctx, event, true)
}

// publish 发布事件核心实现
func (eb *eventBusImpl) publish(ctx context.Context, event Event, async bool) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// 检查事件过滤器
	if !eb.passFilters(event) {
		return nil
	}

	subs, exists := eb.subscribers[event.GetType()]
	if !exists {
		return nil
	}

	var wg sync.WaitGroup
	var errors []error
	var mu sync.Mutex

	for _, sub := range subs {
		if async {
			wg.Add(1)
			go func(s *subscription) {
				defer wg.Done()
				if err := eb.handleEvent(ctx, s.handler, event); err != nil {
					mu.Lock()
					errors = append(errors, err)
					mu.Unlock()
				}
			}(sub)
		} else {
			if err := eb.handleEvent(ctx, sub.handler, event); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if async {
		wg.Wait()
	}

	if len(errors) > 0 {
		return NewPluginError(ErrCodePluginInternal, "event handling failed", "event_bus", errors[0])
	}

	return nil
}

// handleEvent 处理单个事件
func (eb *eventBusImpl) handleEvent(ctx context.Context, handler EventHandler, event Event) error {
	timeout := handler.GetTimeout()
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	handlerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return handler.Handle(handlerCtx, event)
}

// passFilters 检查事件是否通过所有过滤器
func (eb *eventBusImpl) passFilters(event Event) bool {
	for _, filter := range eb.filters {
		if !filter.Match(event) {
			return false
		}
	}
	return true
}

// AddFilter 添加事件过滤器
func (eb *eventBusImpl) AddFilter(filter EventFilter) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.filters = append(eb.filters, filter)
	return nil
}

// RemoveFilter 移除事件过滤器
func (eb *eventBusImpl) RemoveFilter(filter EventFilter) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for i, f := range eb.filters {
		if f == filter {
			eb.filters = append(eb.filters[:i], eb.filters[i+1:]...)
			return nil
		}
	}

	return NewPluginError(ErrCodePluginNotFound, "filter not found", "event_bus", nil)
}

// eventImpl 事件实现
type eventImpl struct {
	id        string
	type_     EventType
	source    string
	timestamp time.Time
	data      map[string]interface{}
	metadata  map[string]string
}

// NewEvent 创建新事件
func NewEvent(eventType EventType, source string, data map[string]interface{}) Event {
	return &eventImpl{
		id:        uuid.New().String(),
		type_:     eventType,
		source:    source,
		timestamp: time.Now(),
		data:      data,
		metadata:  make(map[string]string),
	}
}

func (e *eventImpl) GetID() string {
	return e.id
}

func (e *eventImpl) GetType() EventType {
	return e.type_
}

func (e *eventImpl) GetSource() string {
	return e.source
}

func (e *eventImpl) GetTimestamp() time.Time {
	return e.timestamp
}

func (e *eventImpl) GetData() map[string]interface{} {
	return e.data
}

func (e *eventImpl) GetMetadata() map[string]string {
	return e.metadata
}

// baseEventHandler 基础事件处理器
type baseEventHandler struct {
	name       string
	eventTypes []EventType
	timeout    time.Duration
	handler    func(context.Context, Event) error
}

// NewBaseEventHandler 创建基础事件处理器
func NewBaseEventHandler(name string, eventTypes []EventType, timeout time.Duration, handler func(context.Context, Event) error) EventHandler {
	return &baseEventHandler{
		name:       name,
		eventTypes: eventTypes,
		timeout:    timeout,
		handler:    handler,
	}
}

func (h *baseEventHandler) GetName() string {
	return h.name
}

func (h *baseEventHandler) GetEventTypes() []EventType {
	return h.eventTypes
}

func (h *baseEventHandler) GetTimeout() time.Duration {
	return h.timeout
}

func (h *baseEventHandler) Handle(ctx context.Context, event Event) error {
	return h.handler(ctx, event)
}

// typeFilter 类型事件过滤器
type typeFilter struct {
	allowedTypes map[EventType]bool
}

// NewTypeFilter 创建类型过滤器
func NewTypeFilter(allowedTypes []EventType) EventFilter {
	filter := &typeFilter{
		allowedTypes: make(map[EventType]bool),
	}
	for _, t := range allowedTypes {
		filter.allowedTypes[t] = true
	}
	return filter
}

func (f *typeFilter) Match(event Event) bool {
	return f.allowedTypes[event.GetType()]
}