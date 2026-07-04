# eventbus

轻量级 Go 事件总线，基于 `On` / `Emit` 模式。支持通配符 Topic 匹配、可插拔 `Dispatcher`、命名分发器、Selector 动态路由、Middleware 链以及 OTel 指标/追踪埋点。

## 安装

```bash
go get github.com/kordar/eventbus
```

## 快速开始

```go
bus := eventbus.NewEventBus()

// 注册 handler
bus.On("order.created", func(e eventbus.Event) {
    fmt.Println("order created:", e.Payload)
})

// 发送事件
bus.Emit("order.created", eventbus.Event{
    Id:      "order.created",
    Payload: map[string]interface{}{"orderId": 1},
})
```

## Event

```go
type Event struct {
    Id         string      // 事件标识
    Payload    interface{} // 事件载荷
    Dispatcher string      // 指定使用哪个命名分发器；为空时走默认优先级
}
```

## Topic 通配符

`On` 注册时支持通配符 pattern：

| Pattern | 匹配 |
|---------|------|
| `"*"` | 所有 topic |
| `"order.*"` | 以 `order.` 开头，如 `order.created`、`order.paid.done` |
| `"*.created"` | 以 `.created` 结尾，如 `order.created`、`user.account.created` |

```go
bus.On("order.*", func(e eventbus.Event) {
    slog.Info("order event", "id", e.Id)
})
```

Emit 时，精确匹配的 handler 和所有通配符匹配的 handler 都会被调用。

---

## 可插拔分发机制

### Dispatcher 接口

```go
type Dispatcher interface {
    Dispatch(event Event, handlers []Handler)
}
```

### 内置 Dispatcher

| 实现 | 说明 |
|------|------|
| `SyncDispatcher` | 同步顺序执行（默认），每个 handler 有 panic 隔离 |
| `GoroutineDispatcher` | goroutine 异步派发，不阻塞调用方 |

### 设置默认 Dispatcher

```go
bus := eventbus.NewEventBus(
    eventbus.WithDispatcher(eventbus.GoroutineDispatcher{}),
)
```

---

## 命名分发器

注册多个命名分发器，Emit 时通过 `Event.Dispatcher` 动态选择：

```go
bus := eventbus.NewEventBus(
    eventbus.WithDispatchers(map[string]eventbus.Dispatcher{
        "sync":  eventbus.SyncDispatcher{},
        "async": eventbus.GoroutineDispatcher{},
    }),
)

bus.Emit("order.created", eventbus.Event{Dispatcher: "sync"})
bus.Emit("email.send",    eventbus.Event{Dispatcher: "async"})
```

运行时注册/移除：

```go
bus.RegisterDispatcher("custom", myDispatcher)
bus.UnregisterDispatcher("custom")
```

指定名称不存在时自动回退到默认优先级并输出 warn 日志。

---

## Selector — 动态路由

按 topic 或事件属性动态选择 Dispatcher：

```go
bus := eventbus.NewEventBus(
    eventbus.WithSelector(func(topic string, e eventbus.Event) eventbus.Dispatcher {
        if strings.HasPrefix(topic, "critical.") {
            return eventbus.SyncDispatcher{} // 关键事件同步执行
        }
        return eventbus.GoroutineDispatcher{} // 普通事件异步
    }),
)
```

Selector 返回 `nil` 时走默认 Dispatcher。

---

## 分发器选择优先级

```
Event.Dispatcher（命名分发器）→ Selector → WithDispatcher → SyncDispatcher（默认）
```

---

## Middleware 中间件

对 Dispatcher 进行包装增强，支持链式组合：

```go
type Middleware func(next Dispatcher) Dispatcher
```

### 内置中间件

| 中间件 | 说明 |
|--------|------|
| `RecoveryMiddleware()` | Dispatcher 级别 panic 恢复 |
| `LoggingMiddleware()` | 分发日志（debug 级别） |
| `MetricsMiddleware()` | OTel 指标：`eventbus_dispatch_total`(Counter) + `eventbus_handler_duration_seconds`(Histogram) |
| `TracingMiddleware()` | OTel 追踪：`eventbus.dispatch` Span + 每个 handler 创建 `eventbus.handler` 子 Span |

### 使用

```go
bus := eventbus.NewEventBus(
    eventbus.WithMiddleware(
        eventbus.RecoveryMiddleware(),
        eventbus.LoggingMiddleware(),
        eventbus.MetricsMiddleware(),
        eventbus.TracingMiddleware(),
    ),
)
```

中间件按添加顺序从外到内包裹，即 `logging → recovery → dispatcher`。

### 自定义中间件

```go
func RateLimitMiddleware(rate int) eventbus.Middleware {
    limiter := rate.NewLimiter(rate.Limit(rate), rate)
    return func(next eventbus.Dispatcher) eventbus.Dispatcher {
        return eventbus.DispatchFunc(func(e eventbus.Event, hs []eventbus.Handler) {
            if limiter.Allow() {
                next.Dispatch(e, hs)
            }
        })
    }
}
```

---

## 全局总线注册表

```go
// 注册
bus := eventbus.NewEventBus()
eventbus.Provide("default", bus)

// 获取
bus := eventbus.Get("default")
```

---

## API 总览

| 方法 | 说明 |
|------|------|
| `NewEventBus(opts ...Option)` | 创建事件总线 |
| `bus.On(topic, handler)` | 注册 handler（支持通配符） |
| `bus.Off(topic)` | 移除 topic 下所有 handler |
| `bus.Emit(topic, event)` | 发送事件 |
| `bus.RegisterDispatcher(name, d)` | 运行时注册命名分发器 |
| `bus.UnregisterDispatcher(name)` | 运行时移除命名分发器 |
| `bus.Close()` | 关闭总线，释放所有 handler |
| `eventbus.Provide(name, bus)` | 注册到全局总线表 |
| `eventbus.Get(name)` | 从全局总线表获取 |

### Option

| 选项 | 说明 |
|------|------|
| `WithDispatcher(d)` | 设置默认 Dispatcher |
| `WithDispatchers(m)` | 批量注册命名分发器 |
| `WithSelector(s)` | 设置动态 Dispatcher 选择器 |
| `WithMiddleware(mw ...)` | 添加中间件链 |
