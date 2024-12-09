package eventbus

type Event struct {
	DriverName string
	Payload    interface{}
}
