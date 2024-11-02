package eventbus

import "sync"

var (
	container map[string][]EventListener = map[string][]EventListener{}
	locker    sync.Mutex                 = sync.Mutex{}
)

type TriggerType string

type Event interface {
	GetId() string          // 事件id
	GetObject() interface{} // 事件data
}

type DefaultEvent struct {
	Id     string      `json:"id"`
	Object interface{} `json:"object"`
}

func NewEvent(id string, object interface{}) Event {
	return &DefaultEvent{Id: id, Object: object}
}

func (e DefaultEvent) GetId() string {
	return e.Id
}

func (e DefaultEvent) GetObject() interface{} {
	return e.Object
}

// EventListener 事件监听者
type EventListener interface {
	Consumer(event Event) // 事件消费
	Binding() []Event     // 返回监听器订阅的事件类型，异步监听器在此处进行异步消费监听。
}

// PublishEvent 发布事件
func PublishEvent(events ...Event) {
	if events == nil || len(events) == 0 {
		return
	}
	for _, event := range events {
		subscribes := container[event.GetId()]
		if subscribes == nil {
			continue
		}

		for _, subscribe := range subscribes {
			subscribe.Consumer(event)
		}
	}
}

// BindEvent 订阅事件
func BindEvent(listener EventListener) {
	events := listener.Binding()
	if events == nil || len(events) == 0 {
		return
	}

	locker.Lock()
	defer locker.Unlock()
	for _, event := range events {
		id := event.GetId()
		if container[id] == nil {
			container[id] = []EventListener{}
		}
		container[id] = append(container[id], listener)
	}
}
