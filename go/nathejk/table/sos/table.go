package sos

import (
	"database/sql"
	"log"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/nathejk/shared-go/types"

	_ "embed"
)

// Sos is one case: a call to the emergency phone, and everything that has
// happened to it since.
type Sos struct {
	ID                  types.SosID    `json:"id"`
	YearSlug            types.YearSlug `json:"year"`
	Headline            string         `json:"headline"`
	Description         string         `json:"description"`
	Status              Status         `json:"status"`
	Severity            Severity       `json:"severity"`
	AssigneeSectionSlug types.Slug     `json:"assigneeSectionSlug"`
	CreatedAt           time.Time      `json:"createdAt"`
	CreatedBy           types.UserID   `json:"createdBy"`

	// LastActivityAt advances on every event for the case, including comments and
	// team associations. It is what the list sorts by, so an operator's eye lands
	// on the case that just moved rather than on the one opened first.
	LastActivityAt time.Time `json:"lastActivityAt"`

	// Timeline and Teams are filled in by GetByID only — the list view neither
	// needs them nor wants the joins.
	Timeline []Activity `json:"timeline,omitempty"`
	Teams    []Team     `json:"teams,omitempty"`
}

// Activity is one entry on a case's timeline.
//
// Seq is the stream sequence of the event that produced it, which gives the
// timeline a total order that agrees with the event log, and makes replay
// idempotent for free: the same event always lands on the same row.
type Activity struct {
	Seq   uint64       `json:"seq"`
	SosID types.SosID  `json:"sosId"`
	Type  ActivityType `json:"type"`
	Actor types.UserID `json:"actorUserId"`

	// ActivityID is set for entries a later event can refer back to — today only
	// comments, which can be edited.
	ActivityID string `json:"activityId,omitempty"`

	// RefActivityID points at the entry this one amends. A comment edit appends an
	// entry pointing at the original comment rather than overwriting it, so the
	// fact that the text changed survives in the log.
	RefActivityID string `json:"refActivityId,omitempty"`

	// Value is the entry's payload: comment text, the new severity, the assigned
	// section slug, the team id. A single loose column rather than one per type,
	// because PRD 006 adds entry types to this table and must not need a schema
	// change to do it.
	Value     string    `json:"value,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Team is a patrol associated with a case.
//
// Only the association is stored here; the patrol's name, number and contact are
// joined in by the handler from the patrulje read model, so this table never
// holds a stale copy of them.
type Team struct {
	TeamID    types.TeamID `json:"teamId"`
	CreatedAt time.Time    `json:"createdAt"`
}

type table struct {
	commander
	consumer
	querier
}

// New builds the SOS read model and command surface.
//
// The publisher is only used by the commander and the writer only by the
// consumer; they are separate arguments rather than one "db" because the write
// side never reads through the path the projection writes.
func New(p stream.Publisher, w cqrs.Writer, r *sql.DB) *table {
	q := querier{db: r, r: goqu.New("mysql", r)}
	t := &table{
		commander: commander{p: p, q: &q},
		consumer:  consumer{w: w},
		querier:   q,
	}
	if err := w.Consume(t.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	if err := w.Consume(assignableSchema); err != nil {
		log.Printf("Error creating table %q", err)
	}
	return t
}

//go:embed table.sql
var tableSchema string

// The assignable-section flag lives in its own file rather than in table.sql
// because it is a different kind of thing: the three tables in table.sql are the
// case and its history, this one is configuration of which organisation sections
// the nødtelefon may route to.
//
//go:embed assignable.sql
var assignableSchema string

func (t *table) CreateTableSql() string {
	return tableSchema
}
