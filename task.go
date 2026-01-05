package eventbus

import (
	"log"

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
			log.Printf("[EventTask-%s] panic recovered: %+v", e.Name, r)
		}
	}()

	eventBody, ok := body.(*EventBody)
	if !ok {
		log.Printf("[EventTask-%s] invalid body type: %T", e.Name, body)
		return
	}

	// 执行 listener
	for _, listener := range eventBody.Listeners {
		func(l Listener) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[EventTask-%s] listener panic recovered: %+v", e.Name, r)
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
