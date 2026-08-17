package sos

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/types"
)

// Commands is the write surface, as the application sees it.
//
// Every method takes an Actor rather than reading the acting user out of the
// request context the way `year`, `checkgroup` and `checkpoint` do. That is not
// gratuitous inconsistency: reading the context here would mean importing
// nathejk.dk/internal/requestctx, and this package is meant to move to shared-go
// as a file copy (PRD 001 §8, task 055). The handler is already the layer that
// knows about HTTP, so it is the natural place to resolve who is calling.
type Commands interface {
	Create(ctx context.Context, actor Actor, year types.YearSlug, headline, description string) (types.SosID, error)
	Patch(ctx context.Context, actor Actor, id types.SosID, cmd PatchCommand) error
	Comment(ctx context.Context, actor Actor, id types.SosID, comment string) (CommentID, error)
	UpdateComment(ctx context.Context, actor Actor, id types.SosID, commentID CommentID, comment string) error
	Delete(ctx context.Context, actor Actor, id types.SosID) error
	AssociateTeam(ctx context.Context, actor Actor, id types.SosID, teamID types.TeamID) error
	DisassociateTeam(ctx context.Context, actor Actor, id types.SosID, teamID types.TeamID) error
	SetSectionAssignable(ctx context.Context, actor Actor, year types.YearSlug, slug types.Slug, assignable bool) error

	// The member lifecycle summaries (PRD 006). One per operation, published by the
	// handler *after* the per-member events it describes, so that anything reading
	// the summary is guaranteed the changes are already in the log.
	//
	// These take an assembled body rather than a list of arguments because the caller
	// is the only party that can fill it: member names live in the spejder entity and
	// the resulting strength comes from the member commands, and neither this package
	// nor the member package may import the other.
	RecordMemberStatusChanged(ctx context.Context, actor Actor, year types.YearSlug, body MemberStatusChanged) error
	RecordTeamCollected(ctx context.Context, actor Actor, year types.YearSlug, body TeamCollected) error
	RecordMembersMoved(ctx context.Context, actor Actor, year types.YearSlug, body MembersMoved) error
}

// ErrEmptyField is returned when a case is created without the two things that
// make it a case. A blank row nobody can interpret later is worse than a rejected
// form.
var ErrEmptyField = errors.New("headline and description are required")

// ErrEmptyComment is returned rather than silently publishing an empty comment,
// which would put a meaningless entry on a timeline used for handovers.
var ErrEmptyComment = errors.New("comment is required")

type commander struct {
	p stream.Publisher
	q Queries
}

// Create opens a case. The server mints the id, so a client cannot choose one.
func (c commander) Create(ctx context.Context, actor Actor, year types.YearSlug, headline, description string) (types.SosID, error) {
	headline, description = strings.TrimSpace(headline), strings.TrimSpace(description)
	if headline == "" || description == "" {
		return "", ErrEmptyField
	}
	id := NewSosID()
	err := c.publish(actor, year, id, "created", &Created{
		SosID:       id,
		Headline:    headline,
		Description: description,
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

// PatchCommand is a partial update: a nil field is absent, a non-nil field is a
// new value — including an intentionally empty one.
//
// Pointers rather than a map or a set of "changed" flags, matching
// updateYearHandler and patchKlanHandler, because the distinction between "not
// mentioned" and "cleared" is the entire point of PATCH.
//
// The json tags exist because the handler echoes the accepted patch back to the
// client; without them the response would carry Go field names.
type PatchCommand struct {
	Headline            *string     `json:"headline,omitempty"`
	Description         *string     `json:"description,omitempty"`
	Severity            *Severity   `json:"severity,omitempty"`
	AssigneeSectionSlug *types.Slug `json:"assigneeSectionSlug,omitempty"`
	Status              *Status     `json:"status,omitempty"`
}

// Patch publishes one event per field that actually changed.
//
// The transport is a single PATCH, but the event stream stays granular, because
// the timeline is built from these events: "Prioritet sat til rød" is only
// possible if severity has its own event. A patch that changes nothing publishes
// nothing — which also means it emits no live signal, so other operators' screens
// do not flicker for a no-op.
func (c commander) Patch(ctx context.Context, actor Actor, id types.SosID, cmd PatchCommand) error {
	current, err := c.q.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if cmd.Headline != nil {
		headline := strings.TrimSpace(*cmd.Headline)
		if headline == "" {
			return ErrEmptyField
		}
		if headline != current.Headline {
			if err := c.publish(actor, current.YearSlug, id, "headline.updated",
				&HeadlineUpdated{SosID: id, Headline: headline}); err != nil {
				return err
			}
		}
	}
	if cmd.Description != nil {
		description := strings.TrimSpace(*cmd.Description)
		if description == "" {
			return ErrEmptyField
		}
		if description != current.Description {
			if err := c.publish(actor, current.YearSlug, id, "description.updated",
				&DescriptionUpdated{SosID: id, Description: description}); err != nil {
				return err
			}
		}
	}
	if cmd.Severity != nil && *cmd.Severity != current.Severity {
		if !cmd.Severity.Valid() {
			return fmt.Errorf("unknown severity %q", *cmd.Severity)
		}
		if err := c.publish(actor, current.YearSlug, id, "severity.specified",
			&SeveritySpecified{SosID: id, Severity: *cmd.Severity}); err != nil {
			return err
		}
	}
	if cmd.AssigneeSectionSlug != nil && *cmd.AssigneeSectionSlug != current.AssigneeSectionSlug {
		if err := c.publish(actor, current.YearSlug, id, "assigned",
			&Assigned{SosID: id, SectionSlug: *cmd.AssigneeSectionSlug}); err != nil {
			return err
		}
	}
	if cmd.Status != nil && *cmd.Status != current.Status {
		if !cmd.Status.Valid() {
			return fmt.Errorf("unknown status %q", *cmd.Status)
		}
		// Close and reopen are the same field assignment, which is what makes them
		// idempotent: closing a closed case is caught by the dirty-check above and
		// publishes nothing, rather than needing its own guard.
		event, body := "closed", any(&Closed{SosID: id})
		if *cmd.Status == StatusOpen {
			event, body = "reopened", &Reopened{SosID: id}
		}
		if err := c.publish(actor, current.YearSlug, id, event, body); err != nil {
			return err
		}
	}
	return nil
}

// Comment adds a plain-text comment and returns its id, which the client needs in
// order to edit it later.
func (c commander) Comment(ctx context.Context, actor Actor, id types.SosID, comment string) (CommentID, error) {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return "", ErrEmptyComment
	}
	current, err := c.q.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	commentID := NewCommentID()
	if err := c.publish(actor, current.YearSlug, id, "commented",
		&Commented{SosID: id, CommentID: commentID, Comment: comment}); err != nil {
		return "", err
	}
	return commentID, nil
}

// UpdateComment amends a comment's text.
//
// No check that the caller wrote it: there is no per-user identity to check
// against (PRD 001 §6 Auth), and the decision recorded in §11 is that every
// operator may edit. The append-only timeline is what makes that acceptable —
// the original text stays in the log.
func (c commander) UpdateComment(ctx context.Context, actor Actor, id types.SosID, commentID CommentID, comment string) error {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return ErrEmptyComment
	}
	if commentID == "" {
		return tables.ErrRecordNotFound
	}
	current, err := c.q.GetByID(ctx, id)
	if err != nil {
		return err
	}
	// The comment must belong to this case. Without this an operator could amend
	// another case's comment through a mistyped id, and the timeline would record
	// the edit somewhere nobody is looking.
	if !c.hasComment(current, commentID) {
		return tables.ErrRecordNotFound
	}
	return c.publish(actor, current.YearSlug, id, "comment.updated",
		&CommentUpdated{SosID: id, CommentID: commentID, Comment: comment})
}

// Delete soft-deletes a case created in error. Deleting an already-deleted case
// is not possible: GetByID stops resolving it.
func (c commander) Delete(ctx context.Context, actor Actor, id types.SosID) error {
	current, err := c.q.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return c.publish(actor, current.YearSlug, id, "deleted", &Deleted{SosID: id})
}

// AssociateTeam ties a patrol to the case.
//
// Publishing again for an already-associated patrol is harmless — the projection
// upserts — but it is still dirty-checked, so the timeline does not collect
// duplicate "patrulje tilknyttet" entries when two operators on the same call both
// reach for the same team.
func (c commander) AssociateTeam(ctx context.Context, actor Actor, id types.SosID, teamID types.TeamID) error {
	if teamID == "" {
		return tables.ErrRecordNotFound
	}
	current, err := c.q.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c.hasTeam(current, teamID) {
		return nil
	}
	return c.publish(actor, current.YearSlug, id, "team.associated",
		&TeamAssociated{SosID: id, TeamID: teamID})
}

// DisassociateTeam removes a patrol from the case, and is likewise a no-op when
// the patrol is not associated.
func (c commander) DisassociateTeam(ctx context.Context, actor Actor, id types.SosID, teamID types.TeamID) error {
	current, err := c.q.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !c.hasTeam(current, teamID) {
		return nil
	}
	return c.publish(actor, current.YearSlug, id, "team.disassociated",
		&TeamDisassociated{SosID: id, TeamID: teamID})
}

// SetSectionAssignable marks an organisation section as able (or no longer able)
// to be assigned SOS cases.
//
// Dirty-checked against the current list, so a toggle that changes nothing
// publishes nothing — the Organisation page can send the state it wants without
// first working out whether that is already the case.
func (c commander) SetSectionAssignable(ctx context.Context, actor Actor, year types.YearSlug, slug types.Slug, assignable bool) error {
	if !slug.Valid() {
		return fmt.Errorf("invalid section slug %q", slug)
	}
	current, err := c.q.AssignableSections(ctx, year)
	if err != nil {
		return err
	}
	already := false
	for _, s := range current {
		if s == slug {
			already = true
			break
		}
	}
	if already == assignable {
		return nil
	}

	// Subject is NATHEJK.{year}.sos.section.{slug}.assignable with the new state in
	// the body, rather than separate .set and .unset subjects: it is one fact with
	// two values, and a consumer that must match two subjects can handle one and
	// silently miss the other.
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.sos.section.%s.assignable", year, slug)))
	if err := msg.SetBody(&SectionAssignableSet{SectionSlug: slug, Assignable: assignable}); err != nil {
		return err
	}
	if err := msg.SetMeta(&messages.Metadata{UserID: actor.UserID}); err != nil {
		return err
	}
	return c.p.Publish(msg)
}

func (c commander) hasTeam(s *Sos, teamID types.TeamID) bool {
	for _, t := range s.Teams {
		if t.TeamID == teamID {
			return true
		}
	}
	return false
}

func (c commander) hasComment(s *Sos, commentID CommentID) bool {
	for _, a := range s.Timeline {
		if a.Type == ActivityCommented && a.ActivityID == string(commentID) {
			return true
		}
	}
	return false
}

// publish sends one domain event on NATHEJK.{year}.sos.{sosId}.{event}.
//
// The acting user goes in the metadata, where every other producer in the
// platform puts it, so the projection reads it the same way regardless of which
// service published. It is empty until the platform authenticates anybody.
func (c commander) publish(actor Actor, year types.YearSlug, id types.SosID, event string, body any) error {
	msg := c.p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK.%s.sos.%s.%s", year, id, event)))
	if err := msg.SetBody(body); err != nil {
		return err
	}
	if err := msg.SetMeta(&messages.Metadata{UserID: actor.UserID}); err != nil {
		return err
	}
	return c.p.Publish(msg)
}

// --- the member lifecycle summaries (PRD 006) ---
//
// Nothing is validated here beyond the case id, deliberately. These record something
// that has *already happened* — the member events are published before them — so
// refusing one would not undo the operation, it would only lose the record of it.
// The place to reject a member operation is the member command, before anything has
// been published.

// RecordMemberStatusChanged puts a status-changing operation on a case's timeline.
func (c commander) RecordMemberStatusChanged(ctx context.Context, actor Actor, year types.YearSlug, body MemberStatusChanged) error {
	if body.SosID == "" {
		return tables.ErrRecordNotFound
	}
	return c.publish(actor, year, body.SosID, "member.status.changed", &body)
}

// RecordTeamCollected puts a whole-team collection on a case's timeline as one line.
func (c commander) RecordTeamCollected(ctx context.Context, actor Actor, year types.YearSlug, body TeamCollected) error {
	if body.SosID == "" {
		return tables.ErrRecordNotFound
	}
	return c.publish(actor, year, body.SosID, "team.collected", &body)
}

// RecordMembersMoved puts a move of one or more members on a case's timeline.
func (c commander) RecordMembersMoved(ctx context.Context, actor Actor, year types.YearSlug, body MembersMoved) error {
	if body.SosID == "" {
		return tables.ErrRecordNotFound
	}
	return c.publish(actor, year, body.SosID, "member.moved", &body)
}
