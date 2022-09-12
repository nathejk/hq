package team

import (
	"sync"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/pkg/streaminterface"
)

type teamTypeModel struct {
	Publisher streaminterface.Publisher

	sync.Mutex
	ap *aggregate.Publisher
	//validator *validator.Validator
	teamType string
}

func NewTeamTypeModel(publisher streaminterface.Publisher, teamType string) *teamTypeModel {
	m := teamTypeModel{
		teamType: teamType,
		//validator: &validator.Validator{IgnoreMissing: true},
	}
	//vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "team-"+m.teamType+".aggregate")
	return &m
}

func (m *teamTypeModel) CaughtUp() {
	m.ap.SendCaughtUp()
}

func (m *teamTypeModel) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("teammembers.aggregate:updated"),
		streaminterface.SubjectFromStr("teammembers.aggregate:removed"),
	}
}

func (m *teamTypeModel) Produces() []string {
	return []string{"team-" + m.teamType + ".aggregate:updated", "team-" + m.teamType + ".aggregate:removed", "team-" + m.teamType + ".aggregate:caughtup"}
}

func (m *teamTypeModel) HandleMessage(msg streaminterface.Message) error {
	/*msg, ok := i.(eventstream.Message)
	if !ok {
		return
	}
	/*if err := m.validator.Validate(msg); err != nil {
		log.Println(err)
		return
	}*/

	var body TeamMembersAggregate
	msg.Body(&body)
	if body.Type == m.teamType {
		m.ap.Publish(&body)
	}
	return nil
}
