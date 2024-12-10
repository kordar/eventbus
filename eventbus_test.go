package eventbus_test

import (
	"fmt"
	"github.com/kordar/eventbus"
	logger "github.com/kordar/gologger"
	"github.com/kordar/gotask"
	"testing"
	"time"
)

type Demo struct {
}

func (d *Demo) Reg(event eventbus.Event) {
	logger.Info("register event:", event.Id, event.Payload)
}

func TestEventBus_Publish(t *testing.T) {

	handle := gotask.NewTaskHandleWithName("ABC", 3, 10)
	handle.StartWorkerPool()
	handle.AddTask(eventbus.EventTask{})

	eventBus := eventbus.NewEventBus(handle)

	// 订阅 post 主题事件
	subscribe := eventBus.Subscribe("post")
	//defer eventBus.Unsubscribe("post", subscribe)

	go func() {
		for event := range subscribe {
			fmt.Println(event)
		}
	}()

	demo := &Demo{}

	eventBus.AddListener("post2", demo.Reg)

	eventBus.Publish("post", eventbus.Event{Payload: map[string]interface{}{
		"postId": 1,
		"title":  "Go 事件驱动编程：实现一个简单的事件总线",
		"author": "陈明勇",
	}})

	eventBus.AddListener("post2", func(event eventbus.Event) {
		logger.Info("3333333333333333333333")
	})

	time.Sleep(time.Second * 2)
	// 不存在订阅者的 topic
	eventBus.Publish("post", eventbus.Event{Payload: "pay"})

	time.Sleep(time.Second * 2)
	eventBus.Publish("post2", eventbus.Event{Payload: "pay222", Async: eventbus.TaskAsync})

	time.Sleep(time.Second * 2)
	eventBus.Publish("post", eventbus.Event{Payload: "pay"})

	time.Sleep(time.Second * 2)
	eventBus.Publish("post", eventbus.Event{Payload: "pay"})

	time.Sleep(time.Second * 2)
	// 取消订阅 post 主题事件
	//eventBus.Unsubscribe("post", subscribe)
}
