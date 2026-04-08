package commands

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/lok"
	"nathejk.dk/superfluids/streaminterface"
)

type lokQuerier interface {
	GetByID(context.Context, types.LokID) (*lok.Lok, error)
}
type lokcmd struct {
	p streaminterface.Publisher
	q lokQuerier

	producerSlug string
	yearSlug     string
}

func NewLok(p streaminterface.Publisher, q lokQuerier) *lokcmd {
	return &lokcmd{
		p: p,
		q: q,

		producerSlug: "hq-api",
		yearSlug:     "2025",
	}
}

func (c *lokcmd) UpdateLok(lokID types.LokID, name string, sortOrder int, userIDs []types.UserID, teamIDs []types.TeamID) error {
	lok, _ := c.q.GetByID(context.Background(), lokID)
	var dirty bool
	if lok == nil {
		dirty = true
		lokID = types.LokID(uuid.New().String())
	} else {
		dirty = (name != lok.Name) || (sortOrder != lok.SortOrder) || !slices.Equal(userIDs, lok.UserIDs) || !slices.Equal(teamIDs, lok.TeamIDs)
	}
	if !dirty {
		return nil
	}
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK.%s.lok.%s.updated", c.yearSlug, lokID)))
	msg.SetBody(&messages.NathejkLokUpdated{
		LokID:     lokID,
		Name:      name,
		SortOrder: sortOrder,
		UserIDs:   userIDs,
		TeamIDs:   teamIDs,
	})
	msg.SetMeta(&messages.Metadata{Producer: c.producerSlug})
	if err := c.p.Publish(msg); err != nil {
		return err
	}

	return nil
}
func (c *lokcmd) DeleteLok(lokID types.LokID) error {
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK.%s.lok.%s.deleted", c.yearSlug, lokID)))
	msg.SetBody(&messages.NathejkLokDeleted{
		LokID: lokID,
	})
	msg.SetMeta(&messages.Metadata{Producer: c.producerSlug})
	if err := c.p.Publish(msg); err != nil {
		return err
	}
	return nil
}
func (c *lokcmd) UpdateUser(userID types.UserID, armNumber string) error {
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK.%s.bandit.%s.armNumber.assigned", c.yearSlug, userID)))
	msg.SetBody(&messages.NathejkLokArmNumberAssigned{
		ArmNumber: armNumber,
		Type:      "lokchef",
	})
	msg.SetMeta(&messages.Metadata{Producer: c.producerSlug})
	return c.p.Publish(msg)
}
func (c *lokcmd) UpdateMember(memberID types.MemberID, armNumber string) error {
	msg := c.p.MessageFunc()(streaminterface.SubjectFromStr(fmt.Sprintf("NATHEJK.%s.bandit.%s.armNumber.assigned", c.yearSlug, memberID)))
	msg.SetBody(&messages.NathejkLokArmNumberAssigned{
		ArmNumber: armNumber,
		Type:      types.TeamTypeKlan,
	})
	msg.SetMeta(&messages.Metadata{Producer: c.producerSlug})
	return c.p.Publish(msg)
}
