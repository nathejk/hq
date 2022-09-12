package streamtable

import "time"

type StreamSubject string

type MessageReader interface {
	Subject() StreamSubject
	DecodeBody(interface{}) error
	DecodeMeta(interface{}) error
	Timestamp() time.Time
}
type MessageWriter interface {
	Subject(StreamSubject)
	EncodeBody(interface{}) error
	EncodeMeta(interface{}) error
	Timestamp(time.Time)
}

type MessageHandler interface {
	HandleMessage(*MessageReader) error
	CaughtUp() error
}

type StreamConsumer interface {
	Consumes() []StreamSubject
}

type StreamSubscriber interface {
	Subscribe(StreamSubject, *MessageHandler) error
	Unsubscribe(StreamSubject, *MessageHandler) error
}

type StreamPublisher interface {
	Message(StreamSubject) *MessageWriter
	Publish(*MessageWriter) error
}

type EntityName string
type Query string

type Entity interface {
	ID() string
	//Name() EntityName
	//WriteQuery(Query)
}

type StateWriter interface {
	Init(Entity) error
	Write(Entity) error
	Query(string) error
}

type StateSubscriber interface {
	Subscribe(EntityName, *StateChangeHandler) error
}

type StateChangeHandler interface {
	HandleStateChange(*Entity) error
}
