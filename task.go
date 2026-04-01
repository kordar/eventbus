package eventbus

import (
	"log/slog"

	"github.com/kordar/gotask"
)

// EventBody 承载事件、订阅者与监听器
type EventBody struct {
	Event      Event
	Subscribes []EventChan
	Listeners  []Listener
}

func (e EventBody) TaskId() string {
	return "@eventbus"
}

// EventTask 执行事件派发
type EventTask struct {
	Name string
}

func (e EventTask) Id() string {
	return "@eventbus"
}

func (e EventTask) Execute(body gotask.IBody) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[EventTask] panic recovered", "name", e.Name, "err", r)
		}
	}()

	eventBody, ok := body.(*EventBody)
	if !ok {
		slog.Error("[EventTask] invalid body type", "name", e.Name, "body", body)
		return
	}

	// 执行 listener
	for _, listener := range eventBody.Listeners {
		func(l Listener) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("[EventTask] listener panic recovered", "name", e.Name, "err", r)
				}
			}()
			l(eventBody.Event)
		}(listener)
	}

	// 推送给 subscriber
	for _, subscriber := range eventBody.Subscribes {
		safeSend(subscriber, eventBody.Event)
	}
}
