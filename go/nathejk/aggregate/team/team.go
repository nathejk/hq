package team

import (
	"fmt"
	"log"
	"sync"
	"time"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type TeamAggregate struct {
	messages.NathejkTeamUpdated

	ParentTeamID types.TeamID   `json:"parentTeamId,omitempty"`
	SubTeamIDs   []types.TeamID `json:"subTeamIds,omitempty"`
	TeamNumber   string         `json:"teamNumber"`
	LokSlug      string         `json:"lokSlug"`
	CreatedAt    time.Time      `json:"createdAt"`
}

func (v *TeamAggregate) ID() string {
	return string(v.TeamID)
}

func (v *TeamAggregate) IsValid() bool {
	return v.TeamID != "" && v.Type != ""
}

type teamModel struct {
	Publisher streaminterface.Publisher

	sync.Mutex
	ap *aggregate.Publisher
	//validator *validator.Validator
	teams map[types.TeamID]*TeamAggregate
	live  bool
}

func NewTeamModel(publisher streaminterface.Publisher) *teamModel {
	m := teamModel{
		teams: map[types.TeamID]*TeamAggregate{},
		//validator: &validator.Validator{IgnoreMissing: true},
	}
	//vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "team.aggregate")
	return &m
}

func (m *teamModel) get(ID types.TeamID) *TeamAggregate {
	if m.teams[ID] == nil {
		m.teams[ID] = &TeamAggregate{}
		m.teams[ID].TeamID = ID
	}
	return m.teams[ID]
}

func (m *teamModel) CaughtUp() {
	aggregates := aggregate.MapToAggregates(m.teams)
	m.ap.Flush(aggregates)
	m.live = true
}

func (m *teamModel) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("monolith:nathejk_team"),
		streaminterface.SubjectFromStr("nathejk:team.merged"),
		streaminterface.SubjectFromStr("nathejk:team.splited"),
	}
	//return []string{"nathejk:team.updated", "nathejk:team.deleted"}
}

func (m *teamModel) Produces() []string {
	return []string{"team.aggregate:updated", "team.aggregate:removed", "team.aggregate:caughtup"}
}

func (m *teamModel) HandleMessage(msg streaminterface.Message) error {
	//if msg.Time().Year() != time.Now().Year() {
	// only handle messages from this year
	//	return nil
	//}
	/*msg, ok := i.(eventstream.Message)
	if !ok {
		return
	}
	/*if err := m.validator.Validate(msg); err != nil {
		log.Println(err)
		return
	}*/
	switch msg.Subject().Subject() {
	case "monolith:nathejk_team":
		var body messages.MonolithNathejkTeam
		msg.Body(&body)
		if !map[string]bool{"PAY": true, "PAID": true}[body.Entity.SignupStatusTypeName] {
			return nil
		}
		if string(body.Entity.ID) < fmt.Sprintf("%d000", time.Now().Year()) {
			// skip teams from previous years
			return nil
		}
		team := m.get(body.Entity.ID)
		team.NathejkTeamUpdated = messages.NathejkTeamUpdated{
			TeamID:         body.Entity.ID,
			Type:           body.Entity.TypeName,
			Name:           body.Entity.Title,
			GroupName:      body.Entity.Gruppe,
			Korps:          body.Entity.Korps,
			AdvspejdNumber: body.Entity.LigaNumber,
			ContactName:    body.Entity.ContactTitle,
			ContactAddress: body.Entity.ContactAddress,
			ContactEmail:   body.Entity.ContactMail,
			ContactPhone:   body.Entity.ContactPhone,
			ContactRole:    body.Entity.ContactRole,
		}
		if body.Entity.TeamNumber != "0" {
			team.TeamNumber = body.Entity.TeamNumber + "-" + body.Entity.MemberCount
		}
		if body.Entity.LokNumber != "0" {
			team.LokSlug = "lok" + body.Entity.LokNumber
		}
		team.LokSlug = "lok" + body.Entity.LokNumber
		if team.CreatedAt.IsZero() {
			team.CreatedAt = msg.Time()
		}

		if m.live {
			m.ap.Publish(team)
		}
	case "nathejk:team.merged":
		var body messages.NathejkTeamMerged
		msg.Body(&body)
		team := m.get(body.TeamID)
		team.ParentTeamID = body.ParentTeamID
		if m.live {
			m.ap.Publish(team)
		}
		team = m.get(body.ParentTeamID)
		team.SubTeamIDs = append(team.SubTeamIDs, body.TeamID)
		if m.live {
			m.ap.Publish(team)
		}
	case "nathejk:team.splited":
		var body messages.NathejkTeamSplited
		msg.Body(&body)
		team := m.get(body.TeamID)
		parentTeamID := team.ParentTeamID
		team.ParentTeamID = ""
		if m.live {
			m.ap.Publish(team)
		}
		team = m.get(parentTeamID)
		subTeamIDs := []types.TeamID{}
		for _, teamID := range team.SubTeamIDs {
			if teamID != body.TeamID {
				subTeamIDs = append(subTeamIDs, teamID)
			}
		}
		team.SubTeamIDs = subTeamIDs
		if m.live {
			m.ap.Publish(team)
		}

	case "nathejk:team.updated":
		var body messages.NathejkTeamUpdated
		msg.Body(&body)
		team := m.get(body.TeamID)
		team.CreatedAt = msg.Time()
		msg.Body(&team)

		if m.live {
			m.ap.Publish(team)
		}
	case "nathejk:team.deleted":
		var body messages.NathejkTeamDeleted
		msg.Body(&body)
		team := m.get(body.TeamID)
		team.TeamID = ""

		if m.live {
			m.ap.Publish(team)
		}
	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
