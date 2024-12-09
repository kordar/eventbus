package eventbus_test

import (
	"fmt"
	"github.com/kordar/eventbus"
	"testing"
	"time"
)

func TestEventBus_Publish(t *testing.T) {
	eventBus := eventbus.NewEventBus()

	// 订阅 post 主题事件
	subscribe := eventBus.Subscribe("post")
	defer eventBus.Unsubscribe("post", subscribe)

	go func() {
		for event := range subscribe {
			fmt.Println(event.Payload)

		}
	}()

	eventBus.Publish("post", eventbus.Event{Payload: map[string]interface{}{
		"postId": 1,
		"title":  "Go 事件驱动编程：实现一个简单的事件总线",
		"author": "陈明勇",
	}})
	// 不存在订阅者的 topic
	eventBus.Publish("pay", eventbus.Event{Payload: "pay"})

	time.Sleep(time.Second * 2)
	// 取消订阅 post 主题事件
	eventBus.Unsubscribe("post", subscribe)
}
