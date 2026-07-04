package eventbus

import (
	"log/slog"
)

// Dispatcher 事件分发器接口。
// 实现此接口可以自定义事件如何派发给 Handler 集合，例如同步、goroutine、工作池、缓冲批处理等。
type Dispatcher interface {
	Dispatch(event Event, handlers []Handler)
}

// Middleware 分发器中间件，可对 Dispatcher 进行包装增强（如日志、指标、限流等）。
type Middleware func(next Dispatcher) Dispatcher

// DispatchFunc 将普通函数适配为 Dispatcher 接口。
type DispatchFunc func(event Event, handlers []Handler)

func (f DispatchFunc) Dispatch(event Event, handlers []Handler) {
	f(event, handlers)
}

// ---------- 内置 Dispatcher 实现 ----------

// SyncDispatcher 同步分发：按 handler 注册顺序依次执行，每个 handler 有 panic 隔离。
// 这是默认分发器（未设置任何其他分发策略时使用）。
type SyncDispatcher struct{}

func (d SyncDispatcher) Dispatch(event Event, handlers []Handler) {
	for _, h := range handlers {
		safeCall(h, event)
	}
}

// GoroutineDispatcher goroutine 异步分发：每个事件启动一个 goroutine。
type GoroutineDispatcher struct{}

func (d GoroutineDispatcher) Dispatch(event Event, handlers []Handler) {
	go func() {
		for _, h := range handlers {
			safeCall(h, event)
		}
	}()
}

// ---------- Middleware 示例 ----------

// RecoveryMiddleware 为 Dispatcher 添加 panic 恢复。
func RecoveryMiddleware() Middleware {
	return func(next Dispatcher) Dispatcher {
		return DispatchFunc(func(event Event, handlers []Handler) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("[EventBus] dispatcher panic recovered", "err", r)
				}
			}()
			next.Dispatch(event, handlers)
		})
	}
}

// LoggingMiddleware 为 Dispatcher 添加日志。
func LoggingMiddleware() Middleware {
	return func(next Dispatcher) Dispatcher {
		return DispatchFunc(func(event Event, handlers []Handler) {
			slog.Debug("[EventBus] dispatching", "id", event.Id, "dispatcher", event.Dispatcher, "handlers", len(handlers))
			next.Dispatch(event, handlers)
		})
	}
}
