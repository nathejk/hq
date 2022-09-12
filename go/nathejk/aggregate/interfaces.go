package aggregate

type Message interface {
	DecodeBody(body interface{}) error
	EncodeBody(body interface{}) error
	DecodeMeta(meta interface{}) error
	EncodeMeta(meta interface{}) error

	ChannelType() string
	//	Type() string
}

type EventPublisher interface {
	Publish(string, interface{}) error
}
