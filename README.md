# EventBus

一个轻量的 Go 事件总线，支持：

- Listener 回调模式
- Subscriber channel 模式
- 同步、goroutine 异步、任务队列异步

## 安装

```bash
go get github.com/kordar/eventbus
```

## 初始化

```go
bus := eventbus.NewEventBus(
    eventbus.WithBuffer(16),
)
```

## 订阅事件（Subscriber）

```go
ch := bus.Subscribe("post")
defer bus.Unsubscribe("post", ch)

go func() {
    for e := range ch {
        _ = e
    }
}()
```

## 添加监听器（Listener）

```go
bus.AddListener("post", func(e eventbus.Event) {
    _ = e
})
```

## 发布事件

```go
bus.Publish("post", eventbus.Event{
    Id:      "post.created",
    Payload: map[string]interface{}{"postId": 1},
})
```

## 异步策略

`Event.Async` 支持三种策略：

- `NoneAsync`：同步执行
- `GoroutineAsync`：在 goroutine 中派发
- `TaskAsync`：提交到 `gotask.TaskHandle` 执行；未设置 handle 时会自动降级为 goroutine 派发

## TaskAsync（推荐生产使用）

```go
handle := gotask.NewTaskHandleWithName("ABC", 3, 10)
handle.StartWorkerPool()
handle.AddTask(eventbus.EventTask{Name: "ABC"})

bus := eventbus.NewEventBus(
    eventbus.WithHandle(handle),
    eventbus.WithBuffer(16),
)

bus.Publish("post", eventbus.Event{
    Payload: "payload",
    Async:   eventbus.TaskAsync,
})
```

## 行为说明

- 发布时会把事件同时派发给 Listener 与 Subscriber
- Listener 的 panic 会被隔离，不影响其他 Listener 或 Subscriber
- Subscriber channel 满时会丢弃事件（非阻塞）
- `Close()` 会关闭所有订阅 channel，并拒绝后续发布
