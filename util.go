package eventbus

import "log/slog"

func recoverPanic() {
	if r := recover(); r != nil {
		slog.Error("catch the exception execution", "err", r)
	}
}

func safeCall(handler Handler, event Event) {
	if handler == nil {
		return
	}
	defer recoverPanic()
	handler(event)
}
