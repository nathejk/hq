// Package sos is the nødtelefon (emergency phone) domain: SOS cases, their
// activity timeline, and the patrols associated with them.
//
// The package is written to shared-go's guidelines so it can be lifted to
// shared-go/tables/sos unchanged (PRD 001 §8, roadmap task 055). The rule that
// makes that a file move rather than a rewrite is simple and unenforced by the
// compiler: nothing here may import nathejk.dk/... — see the guard in
// lift_test.go. In particular the acting user is passed in by the caller rather
// than read from the request context, which is how every other local table
// package gets it.
package sos

import (
	"github.com/google/uuid"
	"github.com/nathejk/shared-go/types"
)

// CommentID identifies a comment on a case.
//
// Comments are entries on the activity timeline, so a comment id is also an
// activity id: editing a comment appends a new activity that refers back to
// this id rather than overwriting anything. Lifts to shared-go as SosCommentID.
type CommentID types.ID

// NewCommentID mints an id for a new comment. The server mints these, so a
// client cannot collide with one it has not seen.
func NewCommentID() CommentID {
	return CommentID("soscomment-" + uuid.New().String())
}

// NewSosID mints an id for a new case. types.SosID already exists in shared-go,
// so only the constructor lives here.
func NewSosID() types.SosID {
	return types.SosID("sos-" + uuid.New().String())
}

// Severity is how urgent a case is, shown to the operator as a coloured badge.
//
// Three values, confirmed with organizers (PRD 001 §11 Decisions). Deliberately
// not a free string: the legacy API accepted anything and the values drifted.
type Severity string

const (
	SeverityGreen  Severity = "green"
	SeverityYellow Severity = "yellow"
	SeverityRed    Severity = "red"
)

// Valid reports whether s is one of the three known severities.
//
// The empty severity is *not* valid, but a case may legitimately have none:
// severity is set after creation, so callers check Valid only on input.
func (s Severity) Valid() bool {
	switch s {
	case SeverityGreen, SeverityYellow, SeverityRed:
		return true
	}
	return false
}

// Status is whether a case is still being handled.
//
// Closing and reopening are ordinary field assignments rather than distinct
// verbs, which is what makes them idempotent (PRD 001 §5).
type Status string

const (
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusOpen, StatusClosed:
		return true
	}
	return false
}

// ActivityType tags an entry on a case's timeline.
//
// The set grows: PRD 006 adds member transitions, team collection and
// understrength exceptions to the same timeline. Both the projection and the SPA
// must therefore tolerate a type they do not recognise rather than failing on
// it — which is why this is a string and not an enum with exhaustive switches.
type ActivityType string

const (
	ActivityCreated            ActivityType = "created"
	ActivityHeadlineUpdated    ActivityType = "headline.updated"
	ActivityDescriptionUpdated ActivityType = "description.updated"
	ActivityCommented          ActivityType = "commented"
	ActivityCommentUpdated     ActivityType = "comment.updated"
	ActivitySeveritySpecified  ActivityType = "severity.specified"
	ActivityAssigned           ActivityType = "assigned"
	ActivityClosed             ActivityType = "closed"
	ActivityReopened           ActivityType = "reopened"
	ActivityTeamAssociated     ActivityType = "team.associated"
	ActivityTeamDisassociated  ActivityType = "team.disassociated"
	ActivityDeleted            ActivityType = "deleted"
)

// Actor is who performed an action.
//
// Passed in by the caller — the HTTP handler resolves it from the request
// context and hands it over. That indirection is not ceremony: reading the
// request context in here would mean importing nathejk.dk/internal/requestctx
// and turning the eventual lift into a rewrite.
//
// Until the planned auth service exists the API authenticates nobody, so UserID
// is empty in practice and the timeline is attributable by time rather than by
// person (PRD 001 §6 Auth). The field is populated anyway so that nothing has to
// change when identity arrives.
type Actor struct {
	UserID types.UserID
	Name   string
}
