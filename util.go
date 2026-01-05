package eventbus

import logger "github.com/kordar/gologger"

func recoverPanic() {
	if r := recover(); r != nil {
		logger.Errorf("catch the exception execution, err = %v", r)
	}
}

func safeSend(ch EventChan, event Event) {
	defer func() { recover() }()
	select {
	case ch <- event:
	default:
		// channel 满了，丢弃事件
	}
}
