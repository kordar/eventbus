package eventbus

import (
	"sync"

	"github.com/kordar/gotask"
)

type (
	EventChan chan Event
	Listener  func(event Event)
)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[EventChan]struct{}
	listeners   map[string][]Listener
	taskHandle  *gotask.TaskHandle
	bufferSize  int
	closed      bool
}

type Option func(*EventBus)

func WithBuffer(size int) Option {
	return func(eb *EventBus) {
		if size > 0 {
			eb.bufferSize = size
		}
	}
}

func WithHandle(handle *gotask.TaskHandle) Option {
	return func(eb *EventBus) {
		eb.taskHandle = handle
	}
}

func NewEventBus(opts ...Option) *EventBus {
	eb := &EventBus{
		subscribers: make(map[string]map[EventChan]struct{}),
		listeners:   make(map[string][]Listener),
		bufferSize:  16,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(eb)
		}
	}

	return eb
}

// AddListener 一般topic使用事件id进行绑定listener
func (eb *EventBus) AddListener(topic string, listener Listener) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.listeners[topic] = append(eb.listeners[topic], listener)
}

func (eb *EventBus) Publish(topic string, event Event) {
	eb.mu.RLock()
	if eb.closed {
		eb.mu.RUnlock()
		return
	}

	listeners := append([]Listener{}, eb.listeners[topic]...)

	var subscribers []EventChan
	if subs, ok := eb.subscribers[topic]; ok {
		subscribers = make([]EventChan, 0, len(subs))
		for ch := range subs {
			subscribers = append(subscribers, ch)
		}
	}
	eb.mu.RUnlock()

	switch event.Async {

	case NoneAsync:
		for _, listener := range listeners {
			func() {
				defer recoverPanic()
				listener(event)
			}()
		}
		for _, ch := range subscribers {
			safeSend(ch, event)
		}

	case TaskAsync:
		if eb.taskHandle != nil {
			eb.taskHandle.SendToTaskQueue(&EventBody{
				Event:      event,
				Subscribes: subscribers,
				Listeners:  listeners,
			})
		}

	case GoroutineAsync:
		go func() {
			defer recoverPanic()
			for _, listener := range listeners {
				listener(event)
			}
			for _, ch := range subscribers {
				safeSend(ch, event)
			}
		}()
	}
}

func (eb *EventBus) Subscribe(topic string) EventChan {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return nil
	}

	ch := make(EventChan, eb.bufferSize)

	if _, ok := eb.subscribers[topic]; !ok {
		eb.subscribers[topic] = make(map[EventChan]struct{})
	}
	eb.subscribers[topic][ch] = struct{}{}

	return ch
}

func (eb *EventBus) Unsubscribe(topic string, ch EventChan) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if subs, ok := eb.subscribers[topic]; ok {
		if _, ok := subs[ch]; ok {
			delete(subs, ch)
			close(ch)
		}
	}
}

func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return
	}
	eb.closed = true

	for _, subs := range eb.subscribers {
		for ch := range subs {
			close(ch)
		}
	}

	eb.subscribers = nil
	eb.listeners = nil
}
