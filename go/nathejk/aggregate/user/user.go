package user

import (
	"log"
	"sync"
	"time"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type UserAggregate struct {
	messages.NathejkPersonnelUpdated

	CreatedAt time.Time `json:"createdAt"`
	deleted   bool
}

func (v *UserAggregate) ID() string {
	return string(v.UserID)
}

func (v *UserAggregate) IsValid() bool {
	return v.UserID != "" && !v.deleted
}

type userModel struct {
	Publisher streaminterface.Publisher

	sync.Mutex
	ap *aggregate.Publisher
	//validator *validator.Validator
	users map[types.UserID]*UserAggregate
	live  bool
}

func NewUserModel(publisher streaminterface.Publisher) *userModel {
	m := userModel{
		users: map[types.UserID]*UserAggregate{},
		//validator: &validator.Validator{IgnoreMissing: true},
	}
	//vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "user.aggregate")
	return &m
}

func (m *userModel) get(ID types.UserID) *UserAggregate {
	if m.users[ID] == nil {
		m.users[ID] = &UserAggregate{}
		m.users[ID].UserID = ID
	}
	return m.users[ID]
}

func (m *userModel) CaughtUp() {
	aggregates := aggregate.MapToAggregates(m.users)
	m.ap.Flush(aggregates)
	m.live = true
}

func (m *userModel) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("nathejk:personnel.updated"),
		streaminterface.SubjectFromStr("nathejk:personnel.deleted"),
	}
}

func (m *userModel) Produces() []string {
	return []string{"user.aggregate:updated", "user.aggregate:removed", "user.aggregate:caughtup"}
}

func (m *userModel) HandleMessage(msg streaminterface.Message) error {
	/*msg, ok := i.(eventstream.Message)
	if !ok {
		return
	}
	/*if err := m.validator.Validate(msg); err != nil {
		log.Println(err)
		return
	}*/
	switch msg.Subject().Subject() {
	case "nathejk:personnel.updated":
		var body messages.NathejkPersonnelUpdated
		msg.Body(&body)
		user := m.get(body.UserID)
		user.deleted = false
		user.CreatedAt = msg.Time()
		msg.Body(&user)

		if m.live {
			m.ap.Publish(user)
		}
	case "nathejk:personnel.deleted":
		var body messages.NathejkPersonnelDeleted
		msg.Body(&body)
		user := m.get(body.UserID)
		user.deleted = true

		if m.live {
			m.ap.Publish(user)
		}
	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
