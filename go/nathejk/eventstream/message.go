package eventstream

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID            string          `json:"-"`
	Sequence      int64           `json:"-"`
	Channel       string          `json:"-"`
	Type          string          `json:"type"`
	Body          json.RawMessage `json:"body"`
	Meta          json.RawMessage `json:"meta"`
	EventId       string          `json:"eventId"`
	CorrelationId string          `json:"correlationId"`
	CausationId   string          `json:"causationId"`
	Datetime      string          `json:"datetime"`
}

func NewMessage() Message {
	eventId := uuid.New().String()
	return Message{
		EventId:       eventId,
		CausationId:   eventId,
		CorrelationId: eventId,
		Datetime:      time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func (m *Message) SetBody(body interface{}) error {
	body, err := json.Marshal(body)
	if err != nil {
		return err
	}
	m.Body = (json.RawMessage)(body.([]byte))
	return nil
}

func (m *Message) SetMeta(meta interface{}) error {
	meta, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	m.Meta = (json.RawMessage)(meta.([]byte))
	return nil
}
