package eventbus

import (
	"github.com/kordar/gotask"
	"sync"
)

type (
	EventChan chan Event
	Listener  func(event Event)
)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]EventChan
	listeners   map[string][]Listener
	taskHandle  *gotask.TaskHandle
}

func NewEventBus(handle *gotask.TaskHandle) *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventChan),
		listeners:   make(map[string][]Listener),
		mu:          sync.RWMutex{},
		taskHandle:  handle,
	}
}

// AddListener 一般topic使用事件id进行绑定listener
func (eb *EventBus) AddListener(topic string, listener Listener) {
	eb.listeners[topic] = append(eb.listeners[topic], listener)
}

func (eb *EventBus) Publish(topic string, event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// 复制一个新的订阅者列表，避免在发布事件时修改订阅者列表
	subscribers := append([]EventChan{}, eb.subscribers[topic]...)
	listeners := append([]Listener{}, eb.listeners[topic]...)

	if event.Async == NoneAsync {
		defer recoverPanic()
		for _, listener := range listeners {
			listener(event)
		}
		for _, subscriber := range subscribers {
			subscriber <- event
		}
	}

	if event.Async == TaskAsync && eb.taskHandle != nil {
		eb.taskHandle.SendToTaskQueue(&EventBody{Event: event, Subscribes: subscribers, Listeners: listeners})
	}

	if event.Async == GoroutineAsync {
		go func() {
			defer recoverPanic()

			for _, listener := range listeners {
				listener(event)
			}

			for _, subscriber := range subscribers {
				subscriber <- event
			}
		}()
	}

}

func (eb *EventBus) Subscribe(topic string) EventChan {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(EventChan)
	eb.subscribers[topic] = append(eb.subscribers[topic], ch)
	return ch
}

func (eb *EventBus) Unsubscribe(topic string, ch EventChan) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	if subscribers, ok := eb.subscribers[topic]; ok {
		for i, subscriber := range subscribers {
			if ch == subscriber {
				eb.subscribers[topic] = append(subscribers[:i], subscribers[i+1:]...)
				close(ch)
				return
			}
		}
	}
}
