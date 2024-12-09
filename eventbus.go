package eventbus

import "sync"

type (
	EventChan chan Event
)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]EventChan
	driver      map[string]Driver
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventChan),
		mu:          sync.RWMutex{},
		driver:      map[string]Driver{},
	}
}

func (eb *EventBus) RegDriver(driver Driver) {
	eb.driver[driver.Name()] = driver
}

func (eb *EventBus) Publish(topic string, event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	// 复制一个新的订阅者列表，避免在发布事件时修改订阅者列表
	subscribers := append([]EventChan{}, eb.subscribers[topic]...)
	go func() {
		for _, subscriber := range subscribers {
			if eb.driver[event.DriverName] != nil {
				eb.driver[event.DriverName].Publish(event)
			} else {
				subscriber <- event
			}
		}
	}()
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
