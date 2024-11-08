# 事件总线

## 定义事件

- 事件

```go
type Event interface {
    GetId() string          // 事件id
    GetObject() interface{} // 事件data
}
```
- 事件消费

```go
// EventListener 事件监听者
type EventListener interface {
    Consumer(event Event) // 事件消费
    Binding() []Event     // 返回监听器订阅的事件类型，异步监听器在此处进行异步消费监听。
}
```

## 绑定、发布事件

```go
BindEvent(listener EventListener) // 绑定事件
PublishEvent(events ...Event) // 发布事件
```

> 可使用`starter`配合容器管理进行事件程序绑定。
```go
NewRabbitmqModule("eventbus", func(id string, value interface{}) {
	// 此处进行 BindEvent
})
```