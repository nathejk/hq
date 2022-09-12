package team

import (
	"log"
	"sync"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/pkg/streaminterface"
)

type spejderModel struct {
	Publisher streaminterface.Publisher

	sync.Mutex
	ap *aggregate.Publisher
	// validator *validator.Validator
}

func NewSpejderModel(publisher streaminterface.Publisher) *spejderModel {
	m := spejderModel{
		//validator: &validator.Validator{IgnoreMissing: true},
	}
	//vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "spejder.aggregate")
	return &m
}

func (m *spejderModel) CaughtUp() {
	m.ap.SendCaughtUp()
}

func (m *spejderModel) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("team-patrulje.aggregate:updated"),
		streaminterface.SubjectFromStr("team-patrulje.aggregate:removed"),
	}
}

func (m *spejderModel) Produces() []string {
	return []string{"spejder.aggregate:updated", "spejder.aggregate:removed", "spejder.aggregate:caughtup"}
}

func (m *spejderModel) HandleMessage(msg streaminterface.Message) error {
	/*msg, ok := i.(eventstream.Message)
	if !ok {
		return
	}
	/*if err := m.validator.Validate(msg); err != nil {
		log.Println(err)
		return
	}*/
	switch msg.Subject().Subject() {
	case "team-patrulje.aggregate:updated":
		var body TeamMembersAggregate
		msg.Body(&body)
		for _, member := range body.Members {
			m.ap.Publish(member)
		}

	case "team-patrulje.aggregate:removed":
		var body TeamMembersAggregate
		msg.Body(&body)
		for _, member := range body.Members {
			member.TeamID = ""
			m.ap.Publish(member)
		}
	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
