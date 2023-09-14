package team

import (
	"sync"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type teamTypeModel struct {
	Publisher streaminterface.Publisher

	sync.Mutex
	ap *aggregate.Publisher
	//validator *validator.Validator
	teamType types.TeamType
}

func NewTeamTypeModel(publisher streaminterface.Publisher, teamType types.TeamType) *teamTypeModel {
	m := teamTypeModel{
		teamType: teamType,
		//validator: &validator.Validator{IgnoreMissing: true},
	}
	//vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "team-"+string(m.teamType)+".aggregate")
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
	return []string{"team-" + string(m.teamType) + ".aggregate:updated", "team-" + string(m.teamType) + ".aggregate:removed", "team-" + string(m.teamType) + ".aggregate:caughtup"}
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
