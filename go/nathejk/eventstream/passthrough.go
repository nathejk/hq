package eventstream

type Passthrough struct {
	Channel   string
	Publisher Publisher
}

func (p *Passthrough) Handlers() map[string]EventHandler {
	return map[string]EventHandler{
		p.Channel: p,
	}
}

func (p *Passthrough) Handle(msg Message) {
	p.Publisher.Publish(p.Channel, msg)
}
