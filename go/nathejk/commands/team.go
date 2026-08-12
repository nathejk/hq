package commands

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
)

type teamQuerier interface {
	GetKlan(types.TeamID) (*data.Klan, error)
	RequestedSeniorCount() int
}
type team struct {
	p stream.Publisher
	q teamQuerier

	producerSlug string
	yearSlug     string
}

func NewTeam(p stream.Publisher, q teamQuerier) *team {
	return &team{
		p: p,
		q: q,

		producerSlug: "hq-api",
		yearSlug:     "2026",
	}
}

func (c *team) UpdatePatrulje(teamID types.TeamID, team Patrulje, contact Contact, members []Spejder) error {
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.patrulje.%s.updated", "2024", teamID)))
	msg.SetBody(&messages.NathejkTeamUpdated{
		TeamID:            teamID,
		Type:              types.TeamTypePatrulje,
		Name:              team.Name,
		GroupName:         team.Group,
		Korps:             team.Korps,
		AdvspejdNumber:    team.AdventureLigaID,
		ContactName:       contact.Name,
		ContactAddress:    contact.Address,
		ContactPostalCode: contact.PostalCode,
		ContactEmail:      contact.Email,
		ContactPhone:      contact.Phone,
		ContactRole:       contact.Role,
	})
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
	if err := c.p.Publish(msg); err != nil {
		return err
	}

	for _, m := range members {
		if m.Deleted {
			msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.spejder.%s.deleted", "2024", m.MemberID)))
			msg.SetBody(&messages.NathejkMemberDeleted{
				MemberID: m.MemberID,
				TeamID:   teamID,
			})
			msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
			if err := c.p.Publish(msg); err != nil {
				return err
			}
			continue
		}

		// TODO test if MemberID exits or not
		if m.MemberID == "" {
			m.MemberID = types.MemberID(uuid.New().String())
		}
		msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.spejder.%s.updated", "2024", m.MemberID)))
		msg.SetBody(&messages.NathejkScoutUpdated{
			MemberID:     m.MemberID,
			Name:         m.Name,
			Address:      m.Address,
			PostalCode:   m.PostalCode,
			Email:        m.Email,
			Phone:        m.Phone,
			PhoneContact: m.PhoneContact,
			BirthDate:    m.Birthday,
			TShirtSize:   m.TShirtSize,
		})
		msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
		if err := c.p.Publish(msg); err != nil {
			return err
		}
	}

	return nil
}

type StartPatruljeMember struct {
	MemberID    types.MemberID
	Phone       types.PhoneNumber
	PhoneParent types.PhoneNumber
	Starter     bool
}

func (c *team) StartPatrulje(teamID types.TeamID, members []StartPatruljeMember) error {
	body := &messages.NathejkTeamStarted{
		TeamID: teamID,
	}
	for _, m := range members {
		if m.Starter {
			body.Members = append(body.Members, messages.NathejkTeamStarted_Member{
				MemberID:      m.MemberID,
				Phone:         m.Phone,
				PhoneGuardian: m.PhoneParent,
			})
			continue
		}
		msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.spejder.%s.deleted", c.yearSlug, m.MemberID)))
		msg.SetBody(&messages.NathejkMemberDeleted{
			MemberID: m.MemberID,
			TeamID:   teamID,
		})
		msg.SetMeta(&messages.Metadata{Producer: c.producerSlug})
		if err := c.p.Publish(msg); err != nil {
			return err
		}
	}

	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.patrulje.%s.started", c.yearSlug, teamID)))
	msg.SetBody(body)
	msg.SetMeta(&messages.Metadata{Producer: c.producerSlug})
	if err := c.p.Publish(msg); err != nil {
		return err
	}
	/*
		for _, m := range members {
			if m.Deleted {
				msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.spejder.%s.deleted", c.yearSlug, m.MemberID)))
				msg.SetBody(&messages.NathejkMemberDeleted{
					MemberID: m.MemberID,
					TeamID:   teamID,
				})
				msg.SetMeta(&messages.Metadata{Producer: c.producerSlug})
				if err := c.p.Publish(msg); err != nil {
					return err
				}
				continue
			}

			// TODO test if MemberID exits or not
			if m.MemberID == "" {
				m.MemberID = types.MemberID(uuid.New().String())
			}
			msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.spejder.%s.updated", "2024", m.MemberID)))
			msg.SetBody(&messages.NathejkScoutUpdated{
				MemberID:     m.MemberID,
				Name:         m.Name,
				Address:      m.Address,
				PostalCode:   m.PostalCode,
				Email:        m.Email,
				Phone:        m.Phone,
				PhoneContact: m.PhoneContact,
				BirthDate:    m.Birthday,
				TShirtSize:   m.TShirtSize,
			})
			msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
			if err := c.p.Publish(msg); err != nil {
				return err
			}
		}
	*/
	return nil
}

func (c *team) UpdateKlan(teamID types.TeamID, team Klan, members []Senior) error {
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.updated", "2026", teamID)))
	msg.SetBody(&messages.NathejkKlanUpdated{
		TeamID:    teamID,
		Name:      team.Name,
		GroupName: team.Group,
		Korps:     team.Korps,
	})
	msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
	if err := c.p.Publish(msg); err != nil {
		return err
	}
	klan, _ := c.q.GetKlan(teamID)
	if klan.Status == types.SignupStatusOnHold {
		// The team is on waiting list, do not do anything
		return nil
	}
	if c.q.RequestedSeniorCount() > 115 {
		msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.status.changed", "2026", teamID)))
		msg.SetBody(&messages.NathejkKlanStatusChanged{TeamID: teamID, Status: types.SignupStatusOnHold})
		msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
		if (klan.Status != types.SignupStatusPay) && (klan.Status != types.SignupStatusPaid) {
			if err := c.p.Publish(msg); err != nil {
				return err
			}
		}
	}
	if klan.Status == "" {
		msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.status.changed", "2026", teamID)))
		msg.SetBody(&messages.NathejkKlanStatusChanged{TeamID: teamID, Status: types.SignupStatusPay})
		msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
		if err := c.p.Publish(msg); err != nil {
			return err
		}
	}
	if len(members) == 0 {
		for i := 0; i < team.MemberCount; i++ {
			members = append(members, Senior{})
		}
	}

	for _, m := range members {
		if m.Deleted {
			msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.senior.%s.deleted", "2026", m.MemberID)))
			msg.SetBody(&messages.NathejkMemberDeleted{
				MemberID: m.MemberID,
				TeamID:   teamID,
			})
			msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
			if err := c.p.Publish(msg); err != nil {
				return err
			}
			continue
		}

		// TODO test if MemberID exits or not
		if m.MemberID == "" {
			m.MemberID = types.MemberID(uuid.New().String())
		}
		msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.senior.%s.updated", "2026", m.MemberID)))
		msg.SetBody(&messages.NathejkSeniorUpdated{
			MemberID:   m.MemberID,
			Name:       m.Name,
			Address:    m.Address,
			PostalCode: m.PostalCode,
			Email:      m.Email,
			Phone:      m.Phone,
			BirthDate:  m.Birthday,
			TShirtSize: m.TShirtSize,
			Diet:       m.Diet,
		})
		msg.SetMeta(&messages.Metadata{Producer: "tilmelding-api"})
		if err := c.p.Publish(msg); err != nil {
			return err
		}
	}

	return nil
}

func (c *team) AssignToLok(teamID types.TeamID, lok string) error {
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.klan.%s.assigned", c.yearSlug, teamID)))
	msg.SetBody(&messages.NathejkKlanAssigned{
		TeamID: teamID,
		Lok:    lok,
	})
	msg.SetMeta(&messages.Metadata{Producer: c.producerSlug})
	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
	//klan, _ := c.q.GetKlan(teamID)
}
