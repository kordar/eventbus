package eventbus

import (
	"log/slog"
	"sync"
)

// Handler 事件处理函数
type Handler func(event Event)

// Selector 动态选择 Dispatcher。
// 在每个 Emit 调用时，根据 topic 和 event 决定使用哪个 Dispatcher。
// 返回 nil 时使用 EventBus 的默认 Dispatcher。
type Selector func(topic string, event Event) Dispatcher

// wildcardEntry 通配符注册项
type wildcardEntry struct {
	pattern topicPattern
	handler Handler
}

// EventBus 事件总线
type EventBus struct {
	mu        sync.RWMutex
	handlers  map[string][]Handler // 精确匹配
	wildcards []wildcardEntry      // 通配符匹配
	closed    bool

	// 可插拔的分发机制
	dispatchers map[string]Dispatcher // 命名分发器注册表
	dispatcher  Dispatcher            // 默认分发器（未指定名称时使用）
	selector    Selector              // 动态分发器选择器（可选）
	middleware  []Middleware
}

// Option EventBus 配置选项
type Option func(*EventBus)

// WithDispatcher 设置自定义 Dispatcher（替代默认的同步分发器）。
func WithDispatcher(d Dispatcher) Option {
	return func(eb *EventBus) {
		eb.dispatcher = d
	}
}

// WithDispatchers 批量注册命名分发器。
// 命名分发器可通过 Event.Dispatcher 字段在 Emit 时动态指定。
func WithDispatchers(m map[string]Dispatcher) Option {
	return func(eb *EventBus) {
		for name, d := range m {
			eb.dispatchers[name] = d
		}
	}
}

// WithSelector 设置动态 Dispatcher 选择器，可按 topic/event 决定分发策略。
func WithSelector(s Selector) Option {
	return func(eb *EventBus) {
		eb.selector = s
	}
}

// WithMiddleware 添加 Dispatcher 中间件，按添加顺序包裹。
func WithMiddleware(mw ...Middleware) Option {
	return func(eb *EventBus) {
		eb.middleware = append(eb.middleware, mw...)
	}
}

// NewEventBus 创建事件总线。
// 默认使用 SyncDispatcher（同步分发，每个 handler 有 panic 隔离），
// 可通过 WithDispatcher / WithDispatchers / WithSelector 覆盖。
func NewEventBus(opts ...Option) *EventBus {
	eb := &EventBus{
		handlers:    make(map[string][]Handler),
		dispatchers: make(map[string]Dispatcher),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(eb)
		}
	}

	return eb
}

// On 为 topic 注册事件处理函数。
// topic 支持通配符：
//   - "*"    匹配所有 topic
//   - "order.*" 匹配以 "order." 开头的 topic
//   - "*.created" 匹配以 ".created" 结尾的 topic
func (eb *EventBus) On(topic string, handler Handler) {
	if handler == nil {
		return
	}
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if containsWildcard(topic) {
		eb.wildcards = append(eb.wildcards, wildcardEntry{
			pattern: parseTopicPattern(topic),
			handler: handler,
		})
	} else {
		eb.handlers[topic] = append(eb.handlers[topic], handler)
	}
}

// Off 移除 topic 下所有事件处理函数（包括通配符）。
func (eb *EventBus) Off(topic string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.handlers, topic)

	if containsWildcard(topic) {
		pattern := parseTopicPattern(topic)
		filtered := eb.wildcards[:0]
		for _, w := range eb.wildcards {
			if w.pattern.raw != pattern.raw {
				filtered = append(filtered, w)
			}
		}
		eb.wildcards = filtered
	}
}

// RegisterDispatcher 运行时注册一个命名分发器。
func (eb *EventBus) RegisterDispatcher(name string, d Dispatcher) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.dispatchers[name] = d
}

// UnregisterDispatcher 运行时移除一个命名分发器。
func (eb *EventBus) UnregisterDispatcher(name string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	delete(eb.dispatchers, name)
}

// Emit 向 topic 发送事件。
// handler 收集规则：精确匹配的 handler + 所有匹配 topic 的通配符 handler。
// 分发策略优先级：Event.Dispatcher（命名分发器）→ Selector → WithDispatcher → 默认 SyncDispatcher。
func (eb *EventBus) Emit(topic string, event Event) {
	eb.mu.RLock()
	if eb.closed {
		eb.mu.RUnlock()
		return
	}

	handlers := eb.collectHandlersLocked(topic)
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	d := eb.resolveDispatcher(topic, event)
	if d == nil {
		return
	}
	d.Dispatch(event, handlers)
}

// collectHandlersLocked 收集所有匹配的 handler（需持有 RLock）。
func (eb *EventBus) collectHandlersLocked(topic string) []Handler {
	var result []Handler

	// 1. 精确匹配
	if hs := eb.handlers[topic]; len(hs) > 0 {
		result = append([]Handler(nil), hs...)
	}

	// 2. 通配符匹配
	for _, w := range eb.wildcards {
		if w.pattern.matches(topic) {
			result = append(result, w.handler)
		}
	}

	return result
}

// resolveDispatcher 按优先级选择 Dispatcher：
// 1. Event.Dispatcher 指定的命名分发器
// 2. 全局 Selector
// 3. 自定义默认 Dispatcher（WithDispatcher）
// 4. SyncDispatcher（默认同步分发）
func (eb *EventBus) resolveDispatcher(topic string, event Event) Dispatcher {
	if event.Dispatcher != "" {
		eb.mu.RLock()
		d, ok := eb.dispatchers[event.Dispatcher]
		eb.mu.RUnlock()
		if ok {
			return eb.applyMiddleware(d)
		}
		slog.Warn("[EventBus] named dispatcher not found, falling back", "dispatcher", event.Dispatcher, "topic", topic)
	}

	if eb.selector != nil {
		if d := eb.selector(topic, event); d != nil {
			return eb.applyMiddleware(d)
		}
	}

	if eb.dispatcher != nil {
		return eb.applyMiddleware(eb.dispatcher)
	}

	return eb.applyMiddleware(SyncDispatcher{})
}

func (eb *EventBus) applyMiddleware(d Dispatcher) Dispatcher {
	for i := len(eb.middleware) - 1; i >= 0; i-- {
		d = eb.middleware[i](d)
	}
	return d
}

// Close 关闭事件总线，释放所有 handler
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return
	}
	eb.closed = true

	eb.handlers = nil
	eb.wildcards = nil
}

func containsWildcard(topic string) bool {
	for i := 0; i < len(topic); i++ {
		if topic[i] == '*' {
			return true
		}
	}
	return false
}
