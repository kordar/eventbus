package eventbus

import (
	"github.com/kordar/gotask"
)

type EventBody struct {
	Event     Event
	EventChan EventChan
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
	eventBody.EventChan <- eventBody.Event
}
