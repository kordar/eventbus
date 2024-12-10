package eventbus

type EventAsyncType int

const (
	NoneAsync EventAsyncType = iota
	TaskAsync
	GoroutineAsync
)

type Event struct {
	Id      string
	Payload interface{}
	Async   EventAsyncType
}
