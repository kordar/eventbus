package eventbus

type Driver interface {
	Name() string
	Publish(event Event)
}
