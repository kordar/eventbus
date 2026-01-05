# EventBus 使用说明

## 1. 初始化 EventBus + TaskHandle

生产环境建议配合 **`gotask.TaskHandle`** 使用，以便异步任务队列安全执行事件。

```go
handle := gotask.NewTaskHandleWithName("ABC", 3, 10)
handle.StartWorkerPool()
handle.AddTask(eventbus.EventTask{})  // 绑定 EventBus 任务
eventBus := eventbus.NewEventBus(eventbus.WithHandle(handle))
```

说明：

- `"ABC"`：TaskHandle 名称
- `3`：Worker 数量
- `10`：任务队列长度
- `EventTask{}`：EventBus 的任务执行器，负责异步事件分发

------

## 2. 订阅事件（Subscriber 模式）

Subscriber 是通过 channel 获取事件，适合长时间任务或 fan-out 模型。

```go
subscribe := eventBus.Subscribe("post")
defer eventBus.Unsubscribe("post", subscribe) // 程序结束前取消订阅

// 监听事件
go func() {
    for event := range subscribe {
        logger.Info(event)
    }
}()
```

说明：

- `Subscribe("post")`：订阅 `post` 主题
- `Unsubscribe`：取消订阅，关闭 channel
- channel 消费可用 goroutine 独立处理，异步安全

------

## 3. 添加监听器（Listener 模式）

Listener 是函数回调模式，适合处理日志、缓存刷新等轻量逻辑。

```go
type Demo struct{}

func (d *Demo) Reg(event eventbus.Event) {
    logger.Info("register event:", event.Id, event.Payload)
}

demo := &Demo{}
eventBus.AddListener("post2", demo.Reg)

// 也可以直接添加匿名函数
eventBus.AddListener("post2", func(event eventbus.Event) {
    logger.Info("匿名监听器触发")
})
```

说明：

- Listener 会在 **同步或 TaskAsync 模式**下触发
- panic 会被隔离，不影响其他 listener 或 subscriber

------

## 4. 发布事件（Publish）

### 4.1 普通发布（同步 / 默认模式）

```go
eventBus.Publish("post", eventbus.Event{
    Payload: map[string]interface{}{
        "postId": 1,
        "title":  "Go 事件驱动编程：实现一个简单的事件总线",
        "author": "陈明勇",
    },
})
```

说明：

- `Publish(topic, event)` 发布事件
- 事件会被 **所有 listener + subscriber** 接收
- 不存在订阅者的 topic 事件会被忽略

------

### 4.2 异步任务模式发布（TaskAsync）

```go
eventBus.Publish("post2", eventbus.Event{
    Payload: "pay222",
    Async:   eventbus.TaskAsync,
})
```

说明：

- 事件通过 `gotask.TaskHandle` 异步执行
- 避免 goroutine 泄漏和阻塞
- 推荐生产环境使用

------

## 5. 示例总结

完整示例流程：

1. 创建 TaskHandle 并绑定 EventTask
2. 初始化 EventBus
3. Subscriber 订阅主题
4. Listener 注册回调函数
5. Publish 发布事件
6. 可通过 TaskAsync 异步执行
7. 程序结束前取消订阅 / 调用 EventBus.Close()

------

## 6. 注意事项

- Listener 执行尽量保持轻量
- Subscriber channel 建议设置缓冲区，避免阻塞
- 发布事件时 channel 满可能丢弃事件
- 异步模式建议配合 TaskHandle 使用，防止 goroutine 爆炸
- 发布不存在的 topic 不会报错
