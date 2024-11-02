package eventbus

type RabbitmqModule struct {
	name string
	load func(moduleName string, items interface{})
}

func NewRabbitmqModule(name string, load func(moduleName string, items interface{})) *RabbitmqModule {
	return &RabbitmqModule{name, load}
}

func (m RabbitmqModule) Name() string {
	return m.name
}

func (m RabbitmqModule) Load(value interface{}) {
	m.load(m.name, value)
}

func (m RabbitmqModule) Close() {
}
