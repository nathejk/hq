package crewmember

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

// Commands is the write-side interface for crew members.
type Commands interface {
	Register(ctx context.Context, year types.YearSlug, name string, phone types.PhoneNumber, email types.EmailAddress) (types.UserID, error)
	AssignSection(ctx context.Context, year types.YearSlug, userID types.UserID, section types.Slug) error
	Delete(ctx context.Context, year types.YearSlug, userID types.UserID) error
}

type commander struct {
	p stream.Publisher
	q Queries
}

// Register creates a new crew member with a generated UserID and publishes
// NathejkCrewMemberRegistered.
func (c commander) Register(ctx context.Context, year types.YearSlug, name string, phone types.PhoneNumber, email types.EmailAddress) (types.UserID, error) {
	if !year.Valid() {
		return "", fmt.Errorf("invalid year slug %q", year)
	}
	if name == "" {
		return "", errors.New("crew member name is required")
	}
	userID := types.UserID(uuid.New().String())
	body := messages.NathejkCrewMemberRegistered{
		UserID: userID,
		Name:   name,
		Phone:  phone,
		Email:  email,
	}
	msg := c.p.MessageFunc()(subject.FromStr(
		fmt.Sprintf("NATHEJK.%s.crewmember.%s.registered", year, userID),
	))
	msg.SetBody(&body)
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})
	if err := c.p.Publish(msg); err != nil {
		return "", err
	}
	return userID, nil
}

// AssignSection publishes NathejkCrewMemberSectionAssigned. Passing an empty
// slug is the way to unassign a crew member. Assigning a different section
// implicitly unassigns the current one.
func (c commander) AssignSection(ctx context.Context, year types.YearSlug, userID types.UserID, section types.Slug) error {
	if !year.Valid() {
		return fmt.Errorf("invalid year slug %q", year)
	}
	if userID == "" {
		return errors.New("userId is required")
	}
	if section != "" && !section.Valid() {
		return fmt.Errorf("invalid section slug %q", section)
	}
	existing, err := c.q.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing.SectionSlug == section {
		return nil
	}

	body := messages.NathejkCrewMemberSectionAssigned{
		UserID:      userID,
		SectionSlug: section,
	}
	msg := c.p.MessageFunc()(subject.FromStr(
		fmt.Sprintf("NATHEJK.%s.crewmember.%s.section.assigned", year, userID),
	))
	msg.SetBody(&body)
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})
	return c.p.Publish(msg)
}

// Delete publishes NathejkCrewMemberDeleted (soft delete in the read model).
func (c commander) Delete(ctx context.Context, year types.YearSlug, userID types.UserID) error {
	body := messages.NathejkCrewMemberDeleted{UserID: userID}
	msg := c.p.MessageFunc()(subject.FromStr(
		fmt.Sprintf("NATHEJK.%s.crewmember.%s.deleted", year, userID),
	))
	msg.SetBody(&body)
	msg.SetMeta(&messages.Metadata{Producer: "hq-api"})
	return c.p.Publish(msg)
}
