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
