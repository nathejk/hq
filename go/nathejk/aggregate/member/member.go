package member

import (
	"log"
	"sync"
	"time"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type MemberAggregate struct {
	messages.NathejkMemberUpdated

	Status    types.MemberStatus `json:"status"`
	CreatedAt time.Time          `json:"createdAt"`
	deleted   bool
}

func (v *MemberAggregate) ID() string {
	return string(v.MemberID)
}

func (v *MemberAggregate) IsValid() bool {
	return v.MemberID != "" && v.TeamID != "" && !v.deleted
}

type memberModel struct {
	Publisher streaminterface.Publisher

	sync.Mutex
	ap *aggregate.Publisher
	//validator *validator.Validator
	members map[types.MemberID]*MemberAggregate
	live    bool
}

func NewMemberModel(publisher streaminterface.Publisher) *memberModel {
	m := memberModel{
		members: map[types.MemberID]*MemberAggregate{},
		//validator: &validator.Validator{IgnoreMissing: true},
	}
	//vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "member.aggregate")
	return &m
}

func (m *memberModel) get(ID types.MemberID) *MemberAggregate {
	if m.members[ID] == nil {
		m.members[ID] = &MemberAggregate{Status: types.MemberStatusActive}
		m.members[ID].MemberID = ID
	}
	return m.members[ID]
}

func (m *memberModel) CaughtUp() {
	aggregates := aggregate.MapToAggregates(m.members)
	m.ap.Flush(aggregates)
	m.live = true
}

func (m *memberModel) Consumes() []streaminterface.Subject {
	return []streaminterface.Subject{
		streaminterface.SubjectFromStr("monolith:nathejk_member"),
		streaminterface.SubjectFromStr("nathejk:member.status.changed"),
	}
	//return []string{"nathejk:member.updated", "nathejk:member.deleted"}
}

func (m *memberModel) Produces() []string {
	return []string{"member.aggregate:updated", "member.aggregate:removed", "member.aggregate:caughtup"}
}

func (m *memberModel) HandleMessage(msg streaminterface.Message) error {
	/*msg, ok := i.(eventstream.Message)
	if !ok {
		return
	}
	/*if err := m.validator.Validate(msg); err != nil {
		log.Println(err)
		return
	}*/
	switch msg.Subject().Subject() {
	case "monolith:nathejk_member":
		var body messages.MonolithNathejkMember
		msg.Body(&body)
		member := m.get(body.Entity.ID)
		member.NathejkMemberUpdated = messages.NathejkMemberUpdated{
			MemberID:    body.Entity.ID,
			TeamID:      body.Entity.TeamID,
			Name:        body.Entity.Title,
			Address:     body.Entity.Address,
			PostalCode:  body.Entity.PostalCode,
			Phone:       body.Entity.Phone,
			PhoneParent: body.Entity.ContactPhone,
			Email:       body.Entity.Mail,
			Birthday:    body.Entity.BirthDate,
			Returning:   body.Entity.Returning != "0",
		}
		if member.CreatedAt.IsZero() {
			member.CreatedAt = msg.Time()
		}
		member.deleted = body.Entity.DeletedUts.Time() != nil

		if m.live {
			m.ap.Publish(member)
		}

	case "nathejk:member.status.changed":
		var body messages.NathejkMemberStatusChanged
		msg.Body(&body)
		member := m.get(body.MemberID)
		member.Status = body.Status

		if m.live {
			m.ap.Publish(member)
		}

	case "nathejk:member.updated":
		var body messages.NathejkMemberUpdated
		msg.Body(&body)
		member := m.get(body.MemberID)
		member.CreatedAt = msg.Time()
		msg.Body(&member)

		if m.live {
			m.ap.Publish(member)
		}
	case "nathejk:member.deleted":
		var body messages.NathejkMemberDeleted
		msg.Body(&body)
		member := m.get(body.MemberID)
		member.TeamID = ""

		if m.live {
			m.ap.Publish(member)
		}
	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
