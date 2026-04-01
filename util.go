package eventbus

import "log/slog"

func recoverPanic() {
	if r := recover(); r != nil {
		slog.Error("catch the exception execution", "err", r)
	}
}

func safeCall(listener Listener, event Event) {
	if listener == nil {
		return
	}
	defer recoverPanic()
	listener(event)
}

func safeSend(ch EventChan, event Event) {
	defer func() { recover() }()
	select {
	case ch <- event:
	default:
		// channel 满了，丢弃事件
	}
}
