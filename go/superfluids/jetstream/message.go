package jetstream

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/google/uuid"
	"nathejk.dk/superfluids/streaminterface"
)

// struct 'message' represents a message of a stream
type message struct {
	sequence      uint64
	eventID       EventID
	correlationID EventID
	causationID   EventID
	version       Version
	time          time.Time
	subject       streaminterface.Subject
	body          json.RawMessage
	meta          json.RawMessage
	defaultMeta   json.RawMessage
}

func NewMessage() *message {
	eventID := EventID("event-" + uuid.New().String())
	return &message{
		eventID:       eventID,
		correlationID: eventID,
		causationID:   eventID,
		time:          time.Now().UTC(),
	}
}

// Own message ID
func (m *message) EventID() EventID {
	return m.eventID
}
func (m *message) SetEventID(ID EventID) {
	m.eventID = ID
}

// Parent message ID
func (m *message) CausationID() EventID {
	return m.causationID
}
func (m *message) SetCausationID(ID EventID) {
	m.causationID = ID
}

// Ancestor message ID
func (m *message) CorrelationID() EventID {
	return m.correlationID
}
func (m *message) SetCorrelationID(ID EventID) {
	m.correlationID = ID
}

func (m *message) SetCausationCorrelationFromMessage(msg Identifiable) {
	m.SetCausationID(msg.EventID())
	m.SetCorrelationID(msg.CorrelationID())
}

func (m *message) Body(dst interface{}) error {
	return json.Unmarshal(m.body, dst)
}
func (m *message) Meta(dst interface{}) error {
	return json.Unmarshal(m.meta, dst)
}
func (m *message) RawBody() interface{} {
	return m.body
}
func (m *message) RawMeta() interface{} {
	return m.meta
}

func (m *message) Sequence() uint64 {
	return m.sequence
}

func (m *message) SetBody(v interface{}) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	m.body = body
	return nil
}
func (m *message) SetMeta(v interface{}) error {
	meta, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if isRangeable(v) {
		meta, err = mergeRawMessages(meta, m.defaultMeta)
		if err != nil {
			return err
		}
	}

	m.meta = meta
	return nil
}

func (m *message) SetDefaultMeta(v interface{}) error {
	meta, err := json.Marshal(v)
	if err != nil {
		return err
	}
	m.defaultMeta = meta

	// apply new defaults on existing metadata if any.
	var values map[string]interface{}
	m.Meta(&values)

	return m.SetMeta(values)
}

func (m *message) Subject() streaminterface.Subject {
	return m.subject
}
func (m *message) SetSubject(subj streaminterface.Subject) {
	m.subject = subj
}

func (m *message) Time() time.Time {
	return m.time
}
func (m *message) SetTime(t time.Time) error {
	m.time = t
	return nil
}

func isRangeable(v interface{}) bool {
	if v == nil {
		return false
	}
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}
func mergeRawMessages(a, b json.RawMessage) (json.RawMessage, error) {
	if a == nil {
		return b, nil
	}
	if b == nil {
		return a, nil
	}
	var am, bm map[string]interface{}
	if err := json.Unmarshal(a, &am); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &bm); err != nil {
		return nil, err
	}

	// Merge: new values override defaults on key conflicts
	for k, v := range am {
		bm[k] = v
	}

	return json.Marshal(bm)
}
