package sos

import (
	"log"
	"sync"
	"time"

	"nathejk.dk/nathejk/aggregate"
	"nathejk.dk/nathejk/messages"
	"nathejk.dk/nathejk/types"
	"nathejk.dk/pkg/streaminterface"
)

type SosComment struct {
	PlainText       string             `json:"plainText"`
	CommentID       types.SosCommentID `json:"commentId"`
	CreatedAt       time.Time          `json:"createdAt"`
	CreatedByUserID types.UserID       `json:"createdByUserId"`
	Edited          bool               `json:"edited"`
}

type SosActivity struct {
	Type            types.Enum         `json:"type"` // comment, severity
	CommentID       types.SosCommentID `json:"commentId"`
	Comment         *SosComment        `json:"comment,omitempty"`
	CreatedAt       time.Time          `json:"createdAt"`
	CreatedByUserID types.UserID       `json:"createdByUserId"`
	//CreatedByUser   *User              `json:"user,omitempty"`
	Value  string `json:"value"`
	Status string `json:"status"`
}

type SosAggregate struct {
	SosID           types.SosID           `json:"sosId"`
	Headline        string                `json:"headline"`
	Description     string                `json:"description"`
	CreatedByUserID types.UserID          `json:"createdByUserId"`
	Severity        types.Enum            `json:"severity"`
	Assignee        types.Enum            `json:"assignee"`
	Closed          bool                  `json:"closed"`
	Activities      []*SosActivity        `json:"activities"`
	TeamIDs         map[types.TeamID]bool `json:"teamIds"`

	CreatedAt      time.Time `json:"createdAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`

	deleted  bool
	comments map[types.SosCommentID]*SosComment
}

func (v *SosAggregate) ID() string {
	return string(v.SosID)
}

func (v *SosAggregate) IsValid() bool {
	return v.SosID != "" && !v.deleted
}

func (v *SosAggregate) compile() *SosAggregate {
	for _, a := range v.Activities {
		if c, found := v.comments[a.CommentID]; found {
			a.Comment = c
		}
		v.LastActivityAt = a.CreatedAt
	}
	return v
}

type sosModel struct {
	Publisher streaminterface.Publisher

	sync.Mutex
	ap *aggregate.Publisher
	//	validator *validator.Validator
	soses map[types.SosID]*SosAggregate
	live  bool
}

func NewSosModel(publisher streaminterface.Publisher) *sosModel {
	m := sosModel{
		soses: map[types.SosID]*SosAggregate{},
		//		validator: &validator.Validator{IgnoreMissing: true},
	}
	//	vPublisher := &validator.Publisher{Publisher: publisher, Validator: m.validator}
	//	mPublisher := model.Publisher(&m, vPublisher)
	m.ap = aggregate.NewPublisher(publisher, "sos.aggregate")
	return &m
}

func (m *sosModel) get(ID types.SosID) *SosAggregate {
	if m.soses[ID] == nil {
		m.soses[ID] = &SosAggregate{SosID: ID, Activities: []*SosActivity{}, comments: map[types.SosCommentID]*SosComment{}, TeamIDs: map[types.TeamID]bool{}}
	}
	return m.soses[ID]
}

func (m *sosModel) CaughtUp() {
	aggregates := aggregate.MapToAggregates(m.soses)
	m.ap.Flush(aggregates)
	m.live = true
}

func (m *sosModel) Consumes() []streaminterface.Subject {
	strings := []string{
		"nathejk:sos.created",
		"nathejk:sos.headline.updated",
		"nathejk:sos.description.updated",
		"nathejk:sos.commented",
		"nathejk:sos.comment.updated",
		"nathejk:sos.severity.specified",
		"nathejk:sos.assigned",
		"nathejk:sos.deleted",
		"nathejk:sos.closed",
		"nathejk:sos.reopened",
		"nathejk:sos.team.associated",
		"nathejk:sos.team.disassociated",
		"nathejk:member.status.changed",
		"nathejk:member.positionsms.sent",
		"nathejk:member.positionsms.failed",
	}
	subjects := []streaminterface.Subject{}
	for _, subject := range strings {
		subjects = append(subjects, streaminterface.SubjectFromStr(subject))
	}
	return subjects
}

func (m *sosModel) Produces() []string {
	return []string{"sos.aggregate:updated", "sos.aggregate:removed", "sos.aggregate:caughtup"}
}

func (m *sosModel) HandleMessage(msg streaminterface.Message) error {
	if msg.Time().Year() != time.Now().Year() {
		// only handle messages from this year
		return nil
	}
	/*msg, ok := i.(eventstream.Message)
	if !ok {
		return
	}
	/*if err := m.validator.Validate(msg); err != nil {
		log.Println(err)
		return
	}*/
	switch msg.Subject().Subject() {
	case "nathejk:sos.created":
		var body messages.NathejkSosCreated
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.deleted = false
		sos.CreatedAt = msg.Time()
		sos.LastActivityAt = msg.Time()
		sos.Headline = body.Headline
		sos.Description = body.Description
		sos.CreatedByUserID = body.UserID

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:sos.headline.updated":
		var body messages.NathejkSosHeadlineUpdated
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.Headline = body.Headline

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:sos.commented":
		var body messages.NathejkSosCommented
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.comments[body.CommentID] = &SosComment{
			PlainText:       body.Comment,
			CommentID:       body.CommentID,
			CreatedAt:       msg.Time(),
			CreatedByUserID: body.UserID,
		}
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:            "comment",
			CommentID:       body.CommentID,
			CreatedAt:       msg.Time(),
			CreatedByUserID: body.UserID,
		})
		sos.compile()

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:sos.closed":
		var body messages.NathejkSosClosed
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.Closed = true
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:            "close",
			CreatedAt:       msg.Time(),
			CreatedByUserID: body.UserID,
		})
		sos.compile()

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:sos.reopened":
		var body messages.NathejkSosReopened
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.Closed = false
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:            "reopen",
			CreatedAt:       msg.Time(),
			CreatedByUserID: body.UserID,
		})
		sos.compile()

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:sos.severity.specified":
		var body messages.NathejkSosSeveritySpecified
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.Severity = body.Severity
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:            "severity",
			CreatedAt:       msg.Time(),
			CreatedByUserID: body.UserID,
			Value:           string(body.Severity),
		})
		sos.compile()

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:sos.assigned":
		var body messages.NathejkSosAssigned
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.Assignee = body.Assignee
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:            "assign",
			CreatedAt:       msg.Time(),
			CreatedByUserID: body.UserID,
			Value:           string(body.Assignee),
		})
		sos.compile()

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:sos.team.associated":
		var body messages.NathejkSosTeamAssociated
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.TeamIDs[body.TeamID] = true
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:            "associate",
			CreatedAt:       msg.Time(),
			CreatedByUserID: body.UserID,
			Value:           string(body.TeamID),
		})
		sos.compile()

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:sos.team.disassociated":
		var body messages.NathejkSosTeamDisassociated
		msg.Body(&body)
		sos := m.get(body.SosID)
		delete(sos.TeamIDs, body.TeamID)
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:            "disassociate",
			CreatedAt:       msg.Time(),
			CreatedByUserID: body.UserID,
			Value:           string(body.TeamID),
		})
		sos.compile()

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:member.status.changed":
		var meta messages.NathejkSosMetadata
		msg.Meta(&meta)
		if meta.SosID == "" {
			return nil
		}
		var body messages.NathejkMemberStatusChanged
		msg.Body(&body)
		sos := m.get(meta.SosID)
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:            "memberstatus",
			CreatedAt:       msg.Time(),
			CreatedByUserID: meta.UserID,
			Value:           string(body.MemberID),
			Status:          string(body.Status),
		})

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:member.positionsms.sent":
		var body messages.NathejkMemberPositionSmsSent
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:      "positionsms.sent",
			CreatedAt: msg.Time(),
			//CreatedByUserID: body.UserID,
			Value: string(body.MemberID),
		})

		if m.live {
			m.ap.Publish(sos)
		}

	case "nathejk:member.positionsms.failed":
		var body messages.NathejkMemberPositionSmsSent
		msg.Body(&body)
		sos := m.get(body.SosID)
		sos.Activities = append(sos.Activities, &SosActivity{
			Type:      "positionsms.failed",
			CreatedAt: msg.Time(),
			//CreatedByUserID: body.UserID,
			Value: string(body.MemberID),
		})

		if m.live {
			m.ap.Publish(sos)
		}
		/*
				   "nathejk:sos.headline.updated",
				   "nathejk:sos.description.updated",
				   "nathejk:sos.commented",
				   "nathejk:sos.comment.updated",
				   "nathejk:sos.severity.specified",
				   "nathejk:sos.assigned",
				   "nathejk:sos.deleted",
				   "nathejk:sos.closed",
				   "nathejk:sos.reopened",
				*
			case "nathejk:user.updated":
				var body messages.NathejkUserUpdated
				msg.DecodeBody(&body)
				user := m.get(body.UserID)
				user.deleted = false
				user.CreatedAt = msg.Msg().Datetime
				msg.DecodeBody(&user)

				if m.live {
					m.ap.Publish(user)
				}
			case "nathejk:user.deleted":
				var body messages.NathejkUserDeleted
				msg.DecodeBody(&body)
				user := m.get(body.UserID)
				user.deleted = true

				if m.live {
					m.ap.Publish(user)
				}
		*/
	default:
		log.Printf("Unsupported message type %q", msg.Subject().Subject())
	}
	return nil
}
