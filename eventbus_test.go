package eventbus_test

import (
	"testing"
	"time"

	"github.com/kordar/eventbus"
	"github.com/kordar/gotask"
)

func TestEventBus_Publish(t *testing.T) {
	bus := eventbus.NewEventBus(eventbus.WithBuffer(1))

	listenerCalled := make(chan struct{}, 1)
	bus.AddListener("post", func(event eventbus.Event) {
		listenerCalled <- struct{}{}
	})

	subscribe := bus.Subscribe("post")
	defer bus.Unsubscribe("post", subscribe)

	bus.Publish("post", eventbus.Event{Payload: "payload"})

	select {
	case <-listenerCalled:
	case <-time.After(2 * time.Second):
		t.Fatalf("listener not called")
	}

	select {
	case <-subscribe:
	case <-time.After(2 * time.Second):
		t.Fatalf("subscriber not received")
	}
}

func TestEventBus_Publish_GoroutineAsync_IsolatesPanics(t *testing.T) {
	bus := eventbus.NewEventBus(eventbus.WithBuffer(1))

	listenerCalled := make(chan struct{}, 1)
	bus.AddListener("t", func(event eventbus.Event) {
		panic("boom")
	})
	bus.AddListener("t", func(event eventbus.Event) {
		listenerCalled <- struct{}{}
	})

	subscribe := bus.Subscribe("t")
	defer bus.Unsubscribe("t", subscribe)

	bus.Publish("t", eventbus.Event{Payload: "payload", Async: eventbus.GoroutineAsync})

	select {
	case <-listenerCalled:
	case <-time.After(2 * time.Second):
		t.Fatalf("listener not called")
	}

	select {
	case <-subscribe:
	case <-time.After(2 * time.Second):
		t.Fatalf("subscriber not received")
	}
}

func TestEventBus_Publish_TaskAsync_FallbackWithoutHandle(t *testing.T) {
	bus := eventbus.NewEventBus(eventbus.WithBuffer(1))

	listenerCalled := make(chan struct{}, 1)
	bus.AddListener("t", func(event eventbus.Event) {
		listenerCalled <- struct{}{}
	})

	subscribe := bus.Subscribe("t")
	defer bus.Unsubscribe("t", subscribe)

	bus.Publish("t", eventbus.Event{Payload: "payload", Async: eventbus.TaskAsync})

	select {
	case <-listenerCalled:
	case <-time.After(2 * time.Second):
		t.Fatalf("listener not called")
	}

	select {
	case <-subscribe:
	case <-time.After(2 * time.Second):
		t.Fatalf("subscriber not received")
	}
}

func TestEventBus_Publish_TaskAsync_UsesHandle(t *testing.T) {
	handle := gotask.NewTaskHandleWithName("ABC", 1, 10)
	handle.StartWorkerPool()
	handle.AddTask(eventbus.EventTask{Name: "ABC"})

	bus := eventbus.NewEventBus(eventbus.WithHandle(handle), eventbus.WithBuffer(1))

	listenerCalled := make(chan struct{}, 1)
	bus.AddListener("t", func(event eventbus.Event) {
		listenerCalled <- struct{}{}
	})

	subscribe := bus.Subscribe("t")
	defer bus.Unsubscribe("t", subscribe)

	bus.Publish("t", eventbus.Event{Payload: "payload", Async: eventbus.TaskAsync})

	select {
	case <-listenerCalled:
	case <-time.After(2 * time.Second):
		t.Fatalf("listener not called")
	}

	select {
	case <-subscribe:
	case <-time.After(2 * time.Second):
		t.Fatalf("subscriber not received")
	}
}
