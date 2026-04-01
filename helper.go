package eventbus

import (
	"log/slog"
	"sync"
)

var (
	buses = make(map[string]*EventBus)
	mu    sync.RWMutex
)

func Get(name string) *EventBus {
	mu.RLock()
	defer mu.RUnlock()
	bus, ok := buses[name]
	if !ok {
		slog.Error("event bus not exist", "name", name)
	}
	return bus
}

func Provide(name string, bus *EventBus) {
	mu.Lock()
	defer mu.Unlock()
	buses[name] = bus
}
