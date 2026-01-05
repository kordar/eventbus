package eventbus

import (
	"fmt"
	"strings"

	logger "github.com/kordar/gologger"
	"github.com/kordar/gotask"
	"github.com/spf13/cast"
)

type EventBusModule struct {
	name   string
	loadFn func(moduleName, itemId string, cfg map[string]string, bus *EventBus)
}

func NewEventBusModule(
	name string,
	loadFn func(moduleName, itemId string, cfg map[string]string, bus *EventBus),
) *EventBusModule {
	return &EventBusModule{
		name:   name,
		loadFn: loadFn,
	}
}

func (m *EventBusModule) Name() string {
	return m.name
}

func (m *EventBusModule) loadOne(id string, cfg map[string]string) error {
	if id == "" {
		return fmt.Errorf("[%s] attribute 'id' cannot be empty", m.Name())
	}

	var handle *gotask.TaskHandle

	if strings.EqualFold(cfg["async_task"], "on") {
		workSize := cast.ToInt(cfg["async_task_work_size"])
		queueSize := cast.ToInt(cfg["async_task_queue_buff_len"])

		if workSize <= 0 {
			workSize = 3
		}
		if queueSize <= 0 {
			queueSize = 20
		}

		handle = gotask.NewTaskHandleWithName(id, workSize, queueSize)
		handle.StartWorkerPool()
		handle.AddTask(EventTask{Name: id})
	}

	bus := NewEventBus(
		WithHandle(handle),
	)

	if m.loadFn != nil {
		m.loadFn(m.name, id, cfg, bus)
		logger.Debugf("[%s] custom loader executed for '%s'", m.Name(), id)
	}

	logger.Infof("[%s] module '%s' loaded successfully", m.Name(), id)
	return nil
}

func (m *EventBusModule) Load(value interface{}) {
	items := cast.ToStringMap(value)

	// 单实例
	if items["id"] != nil {
		id := cast.ToString(items["id"])
		if err := m.loadOne(id, cast.ToStringMapString(items)); err != nil {
			logger.Errorf(err.Error())
		}
		return
	}

	// 多实例
	for id, raw := range items {
		cfg := cast.ToStringMapString(raw)
		if err := m.loadOne(id, cfg); err != nil {
			logger.Errorf(err.Error())
		}
	}
}

func (m *EventBusModule) Close() {
}
