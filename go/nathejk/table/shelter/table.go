// Package shelter is where scouts in Hønsegården physically are: the placering of every
// member the shelter crew has taken into its care (PRD 007).
//
// It owns one fact and deliberately no more. Status, team membership and the in-our-care
// count belong to spejderstatus; this package answers "which tent is she in?", which is the
// question asked at 3am by a parent standing at the door and by whoever takes over the shift.
//
// # Why a separate projection
//
// The placering could have been a column on spejderstatus, and that would have been wrong
// three times over: a bed is not a lifecycle fact; spejderstatus is queued for lifting to
// shared-go verbatim (task 083) and an hq-specific column makes that a rewrite; and the
// placering *vocabulary* is derived from these rows, which is what lets the zones define
// themselves at race start with nothing configured.
//
// The two projections consume the same events and neither knows about the other. Both write
// on shelter.accepted, and the writes are independent: spejderstatus records that the member
// is sheltered, this records where. Either can be replayed without the other.
package shelter

import (
	"database/sql"
	"log"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/jrgensen/cqrs"
	"github.com/nathejk/shared-go/types"

	_ "embed"
)

// Placement is one scout's whereabouts inside the shelter.
//
// PlacedAt is nil for somebody accepted but not yet bedded down. That distinction is on
// screen, so it is kept here rather than collapsed into a zero time the SPA would render as
// 1970: "arrived 00:42, not yet placed" is a job for the crew, and "arrived 00:42, in Telt 4
// since 00:51" is a job done.
type Placement struct {
	MemberID   types.MemberID `json:"memberId"`
	YearSlug   types.YearSlug `json:"year"`
	TeamID     types.TeamID   `json:"teamId"`
	Placement  string         `json:"placement"`
	AcceptedAt time.Time      `json:"acceptedAt"`
	PlacedAt   *time.Time     `json:"placedAt"`
}

// Zone is a placering in use tonight, with how many scouts are in it.
//
// This is the whole zone model: there is no zone entity, no configuration and no admin
// screen, because the zones are not known until race start (PRD 007 §6). The vocabulary is
// whatever the crew has already typed, and the count is what orders the suggestions so the
// real tent sits above a typo.
type Zone struct {
	Placement string `json:"placement"`
	Count     int    `json:"count"`
}

type table struct {
	consumer
	querier
}

// New builds the projection. The writer is the consumer's, the *sql.DB the querier's; this
// package publishes nothing, so it takes no publisher — the shelter's events are published
// by the spejderstatus commander, which owns the lifecycle.
func New(w cqrs.Writer, r *sql.DB) *table {
	t := &table{
		consumer: consumer{w: w},
		querier:  querier{db: r, r: goqu.New("mysql", r)},
	}
	if err := w.Consume(t.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	for _, stmt := range schemaMigrations {
		if err := w.Consume(stmt); err != nil {
			log.Printf("Error migrating shelter table %q", err)
		}
	}
	return t
}

// schemaMigrations brings an existing database up to the current shape.
//
// Empty today, and present anyway, because the alternative is the trap spejderstatus already
// fell into: CREATE TABLE IF NOT EXISTS is a no-op wherever the table exists, so a column
// added to table.sql later is silently absent from every database that has already booted
// once — including every developer's. Discovering that in November costs an afternoon;
// keeping the hook here costs nothing.
//
// Entries must be idempotent: they run on every boot, against both the old shape and the new.
var schemaMigrations = []string{}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string { return tableSchema }

// Assert the consumer contract at compile time. The mux accepts anything shaped vaguely
// right, so a signature drifting out of step would otherwise surface as a projection that
// silently never runs — and an empty shelter table looks exactly like an empty shelter.
var _ cqrs.Consumer = (*table)(nil)
