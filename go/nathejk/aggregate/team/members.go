package team

import (
	"log"
	"sync"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/nathejk/aggregate/member"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type TeamMembersAggregate struct {
	TeamAggregate
	Members       []*member.MemberAggregate `json:"members"`
	ActiveMembers []*member.MemberAggregate `json:"activeMembers"`
}

func (v *TeamMembersAggregate) ID() string {
	return string(v.TeamID)
}

func (v *TeamMembersAggregate) IsValid() bool {
	return v.TeamID != ""
}

type teamMembersModel struct {
	Publisher streaminterface.Publisher

	sync.Mutex
	ap *aggregate.Publisher
	//validator *validator.Validator
	teams map[types.TeamID]*TeamMembersAggregate
	live  bool
}

func NewTeamMembersModel(publisher streaminterface.Publisher) *teamMembersModel {
	m := teamMembersModel{
		teams: map[types.TeamID]*TeamMembersAggregate{},
		//	validator: &validator.Validator{IgnoreMissing: true},
	}
	//vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "teammembers.aggregate")
	return &m
}

func (m *teamMembersModel) get(ID types.TeamID) *TeamMembersAggregate {
	if m.teams[ID] == nil {
		m.teams[ID] = &TeamMembersAggregate{}
		m.teams[ID].TeamID = ID
		m.teams[ID].Members = []*member.MemberAggregate{}
	}
	return m.teams[ID]
}

func (m *teamMembersModel) CaughtUp() {
	aggregates := aggregate.MapToAggregates(m.teams)
	m.ap.Flush(aggregates)
	m.live = true
}

func (m *teamMembersModel) Consumes() []streaminterface.Subject {
	subjs := []string{"team.aggregate:updated", "team.aggregate:removed", "member.aggregate:updated", "member.aggregate:removed"}
	//return []string{"nathejk:team.updated", "nathejk:team.deleted"}
	subjects := []streaminterface.Subject{}
	for _, subject := range subjs {
		subjects = append(subjects, streaminterface.SubjectFromStr(subject))
	}
	return subjects
}

func (m *teamMembersModel) Produces() []string {
	return []string{"teammembers.aggregate:updated", "teammembers.aggregate:removed", "teammembers.aggregate:caughtup"}
}

func (m *teamMembersModel) HandleMessage(msg streaminterface.Message) error {
	/*msg, ok := i.(eventstream.Message)
	if !ok {
		return
	}
	/*if err := m.validator.Validate(msg); err != nil {
		log.Println(err)
		return
	}*/
	switch msg.Subject().Subject() {
	case "team.aggregate:updated":
		var body TeamAggregate
		msg.Body(&body)
		team := m.get(body.TeamID)
		msg.Body(&team)

		team.ActiveMembers = []*member.MemberAggregate{}
		for _, member := range team.Members {
			if member.Status == types.MemberStatusActive {
				team.ActiveMembers = append(team.ActiveMembers, member)
			}
		}
		for _, teamId := range team.SubTeamIDs {
			for _, member := range m.get(teamId).Members {
				if member.Status == types.MemberStatusActive {
					team.ActiveMembers = append(team.ActiveMembers, member)
				}
			}
		}

		if m.live {
			m.ap.Publish(team)
		}
	case "team.aggregate:removed":
		var body TeamAggregate
		msg.Body(&body)
		team := m.get(body.TeamID)
		team.TeamID = ""

		if m.live {
			m.ap.Publish(team)
		}
	case "member.aggregate:updated":
		var body member.MemberAggregate
		msg.Body(&body)
		team := m.get(body.TeamID)
		members := map[types.MemberID]*member.MemberAggregate{}
		for _, member := range team.Members {
			members[member.MemberID] = member
		}
		members[body.MemberID] = &body
		team.Members = []*member.MemberAggregate{}
		team.ActiveMembers = []*member.MemberAggregate{}
		for _, member := range members {
			team.Members = append(team.Members, member)
			if member.Status == types.MemberStatusActive {
				team.ActiveMembers = append(team.ActiveMembers, member)
			}
		}
		for _, teamId := range team.SubTeamIDs {
			for _, member := range m.get(teamId).Members {
				if member.Status == types.MemberStatusActive {
					team.ActiveMembers = append(team.ActiveMembers, member)
				}
			}
		}

		if team.ParentTeamID != "" {
			parent := m.get(team.ParentTeamID)

			parent.ActiveMembers = []*member.MemberAggregate{}
			for _, member := range parent.Members {
				if member.Status == types.MemberStatusActive {
					parent.ActiveMembers = append(parent.ActiveMembers, member)
				}
			}
			for _, teamId := range parent.SubTeamIDs {
				for _, member := range m.get(teamId).Members {
					if member.Status == types.MemberStatusActive {
						parent.ActiveMembers = append(parent.ActiveMembers, member)
					}
				}
			}
			if m.live {
				m.ap.Publish(parent)
			}
		}
		if m.live {
			m.ap.Publish(team)
		}
	case "member.aggregate:removed":
		var body member.MemberAggregate
		msg.Body(&body)
		team := m.get(body.TeamID)
		members := map[types.MemberID]*member.MemberAggregate{}
		for _, member := range team.Members {
			members[member.MemberID] = member
		}
		delete(members, body.MemberID)
		team.Members = []*member.MemberAggregate{}
		for _, member := range members {
			team.Members = append(team.Members, member)
		}
		if m.live {
			m.ap.Publish(team)
		}
	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
