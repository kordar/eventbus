package eventbus

import (
	"github.com/kordar/gologger"
	"github.com/kordar/gotask"
	"github.com/spf13/cast"
)

type EventBusModule struct {
	name string
	load func(moduleName string, itemId string, item map[string]string, eventBus *EventBus)
}

func NewEventBusModule(name string, load func(moduleName string, itemId string, item map[string]string, eventBus *EventBus)) *EventBusModule {
	return &EventBusModule{name, load}
}

func (m EventBusModule) Name() string {
	return m.name
}

func (m EventBusModule) _load(id string, cfg map[string]string) {

	if id == "" {
		logger.Fatalf("[%s] the attribute id cannot be empty.", m.Name())
		return
	}

	var handle *gotask.TaskHandle = nil
	if cfg["async"] == "on" {
		workSize := cast.ToInt(cfg["work_size"])
		queueBuffLen := cast.ToInt(cfg["queue_buff_len"])
		if workSize == 0 {
			workSize = 3
		}
		if queueBuffLen == 0 {
			queueBuffLen = 20
		}
		handle = gotask.NewTaskHandleWithName(id, workSize, queueBuffLen)
		handle.StartWorkerPool()
		handle.AddTask(EventTask{})
	}

	eventBus := NewEventBus(handle)
	if m.load != nil {
		m.load(m.name, id, cfg, eventBus)
		logger.Debugf("[%s] triggering custom loader completion", m.Name())
	}

	logger.Infof("[%s] loading module '%s' successfully", m.Name(), id)
}

func (m EventBusModule) Load(value interface{}) {
	items := cast.ToStringMap(value)
	if items["id"] != nil {
		id := cast.ToString(items["id"])
		m._load(id, cast.ToStringMapString(value))
		return
	}

	for key, item := range items {
		m._load(key, cast.ToStringMapString(item))
	}
}

func (m EventBusModule) Close() {
}
