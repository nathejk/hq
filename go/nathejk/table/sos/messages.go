package sos

import "github.com/nathejk/shared-go/types"

// The bodies of the SOS domain events, published on
// NATHEJK.{year}.sos.{sosId}.{event} — see commands.go for the subjects.
//
// Each field of a case gets its own event rather than one fat "case updated",
// because the timeline is built from these: a granular event stream is what lets
// the UI say "Prioritet sat til rød" instead of "sagen blev ændret". The HTTP
// transport is still a single PATCH; the handler diffs and publishes only what
// changed.
//
// The event's own id lives in the subject, so bodies repeat SosID only where a
// consumer would otherwise have to parse it back out of the subject. They do
// carry it, deliberately: a body that is self-describing survives being read from
// a log or a dead-letter queue by a human.

// Created opens a case. Headline and description are both required, so a case is
// never a blank row somebody meant to fill in later.
type Created struct {
	SosID       types.SosID `json:"sosId"`
	Headline    string      `json:"headline"`
	Description string      `json:"description"`
}

type HeadlineUpdated struct {
	SosID    types.SosID `json:"sosId"`
	Headline string      `json:"headline"`
}

type DescriptionUpdated struct {
	SosID       types.SosID `json:"sosId"`
	Description string      `json:"description"`
}

// Commented adds a plain-text comment. CommentID is minted by the server so the
// comment has a stable target for a later edit.
type Commented struct {
	SosID     types.SosID `json:"sosId"`
	CommentID CommentID   `json:"commentId"`
	Comment   string      `json:"comment"`
}

// CommentUpdated amends a comment's text.
//
// It does not replace the original: the projection appends a new timeline entry
// pointing at CommentID, so the fact that an edit happened survives even though
// the current text is what the UI shows. An append-only log is the only reason
// editing is safe at all while every operator shares one identity.
type CommentUpdated struct {
	SosID     types.SosID `json:"sosId"`
	CommentID CommentID   `json:"commentId"`
	Comment   string      `json:"comment"`
}

type SeveritySpecified struct {
	SosID    types.SosID `json:"sosId"`
	Severity Severity    `json:"severity"`
}

// Assigned points a case at the organisation section responsible for it.
//
// The section is referenced by slug, not by label: a section can be renamed on
// the Organisation page and the case must follow it rather than keep a stale
// name. If the section is deleted the slug is retained and shown as such, rather
// than the assignment silently disappearing.
type Assigned struct {
	SosID       types.SosID `json:"sosId"`
	SectionSlug types.Slug  `json:"sectionSlug"`
}

type Closed struct {
	SosID types.SosID `json:"sosId"`
}

type Reopened struct {
	SosID types.SosID `json:"sosId"`
}

// Deleted soft-deletes a case created in error. Nothing is destroyed: the row and
// its timeline stay, marked deleted, and every read path filters them out.
type Deleted struct {
	SosID types.SosID `json:"sosId"`
}

// TeamAssociated ties a patrol to a case. Only patrols, never klaner.
type TeamAssociated struct {
	SosID  types.SosID  `json:"sosId"`
	TeamID types.TeamID `json:"teamId"`
}

type TeamDisassociated struct {
	SosID  types.SosID  `json:"sosId"`
	TeamID types.TeamID `json:"teamId"`
}

// SectionAssignableSet marks an organisation section as able (or no longer able)
// to be assigned SOS cases.
//
// This is a fact about the nødtelefon rather than about the section, which is why
// it is an SOS event and an SOS table instead of a column on shared-go's section
// (PRD 001 §8, amended 2026-08-11). Published on
// NATHEJK.{year}.sos.section.{slug}.assignable.
type SectionAssignableSet struct {
	SectionSlug types.Slug `json:"sectionSlug"`
	Assignable  bool       `json:"assignable"`
}

// --- The member lifecycle summaries (PRD 006) ---
//
// A member-changing operation publishes one event on the *member's* subject per
// affected member, and then one of these on the case's subject summarising the
// whole operation. The member events drive the spejderstatus projection; these
// drive the timeline. Neither is redundant: they say different things to different
// readers, and only the member events can be published by the future car and
// shelter interfaces, which know nothing about cases.
//
// # Why one entry per operation rather than per member
//
// sos_activity is keyed by stream sequence, so N member events would produce N
// rows. Collecting a patrol of three is one thing an operator did, and a handover
// log that renders it as three lines is a log somebody has to reassemble in their
// head at 3am.
//
// # Why these payloads are fat
//
// Every field needed to render the line is on the event, including names and the
// resulting team strength. The alternative — store ids, join to current state when
// rendering — would show *today's* truth on yesterday's line: a member moved twice
// would have their first move described using their second team. A timeline whose
// entries change meaning after the fact is worse than no timeline, so these carry
// what they mean and never look it up again.

// MemberChange is one member's transition inside an operation.
type MemberChange struct {
	MemberID types.MemberID `json:"memberId"`
	Name     string         `json:"name"`

	// From is the status the member was in before. Recorded rather than derived so
	// the line can read "fra racing til waiting" forever, whatever happens later.
	From types.MemberStatus `json:"from"`
	To   types.MemberStatus `json:"to"`
}

// MemberStatusChanged summarises an operation that changed one or more members'
// statuses on this case: a withdrawal request, a resume, or a correction.
//
// TeamStrength is the team's racing count *after* the operation, which is what makes
// a below-strength breach legible on the timeline rather than only in the live view.
type MemberStatusChanged struct {
	SosID        types.SosID    `json:"sosId"`
	TeamID       types.TeamID   `json:"teamId"`
	TeamName     string         `json:"teamName,omitempty"`
	Members      []MemberChange `json:"members"`
	TeamStrength int            `json:"teamStrength"`
}

// TeamCollected is every remaining racing member of a patrol going to waiting in one
// action — the team leaves together.
//
// A distinct type from MemberStatusChanged despite the identical shape, because it is
// a distinct act: the operator decided the patrol is done, not that these three
// individuals each wanted to stop. The timeline says "hele patruljen hentes" rather
// than listing three withdrawals, and after it TeamStrength is zero, which is what
// makes the patrol discontinued.
type TeamCollected struct {
	SosID        types.SosID    `json:"sosId"`
	TeamID       types.TeamID   `json:"teamId"`
	TeamName     string         `json:"teamName,omitempty"`
	Members      []MemberChange `json:"members"`
	TeamStrength int            `json:"teamStrength"`
}

// MemberMove is one member's move to another patrol.
type MemberMove struct {
	MemberID   types.MemberID `json:"memberId"`
	Name       string         `json:"name"`
	ToTeamID   types.TeamID   `json:"toTeamId"`
	ToTeamName string         `json:"toTeamName,omitempty"`
}

// MembersMoved summarises moving one or more members off a patrol.
//
// Members may go to *different* destinations in one operation — uncommon, but the
// field is per member for exactly that reason, so the flow is not forced to pretend
// a group has one destination.
//
// FromTeamStrength is the origin team's racing count afterwards. Zero means the
// patrol has been emptied and is now discontinued, which is what the legacy
// patrulje.merged event was clumsily expressing.
type MembersMoved struct {
	SosID            types.SosID  `json:"sosId"`
	FromTeamID       types.TeamID `json:"fromTeamId"`
	FromTeamName     string       `json:"fromTeamName,omitempty"`
	Members          []MemberMove `json:"members"`
	FromTeamStrength int          `json:"fromTeamStrength"`
}
