package eventbus

type Event struct {
	Id         string
	Payload    interface{}
	Dispatcher string // 指定该事件使用哪个命名分发器；为空时走默认优先级
}
