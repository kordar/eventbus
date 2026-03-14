package eventbus

import (
	"sync"

	logger "github.com/kordar/gologger"
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
		logger.Fatalf("event bus %s not exist.", name)
	}
	return bus
}

func Provide(name string, bus *EventBus) {
	mu.Lock()
	defer mu.Unlock()
	buses[name] = bus
}
