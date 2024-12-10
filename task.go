package eventbus

import (
	"github.com/kordar/gotask"
)

type EventBody struct {
	Event      Event
	Subscribes []EventChan
	Listeners  []Listener
}

func (e EventBody) TaskId() string {
	return "@eventbus"
}

type EventTask struct {
}

func (e EventTask) Id() string {
	return "@eventbus"
}

func (e EventTask) Execute(body gotask.IBody) {
	defer recoverPanic()
	eventBody := body.(*EventBody)
	defer recoverPanic()

	for _, listener := range eventBody.Listeners {
		listener(eventBody.Event)
	}

	for _, subscriber := range eventBody.Subscribes {
		subscriber <- eventBody.Event
	}
}
