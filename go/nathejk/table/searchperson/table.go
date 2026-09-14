// Package searchperson is the index behind HQ's person search: one row per findable
// person, keyed so that a phone number typed by an operator becomes an index seek.
//
// # Why a projection and not a query
//
// The people an operator might be looking for are spread over six sources — spejder,
// senior, personnel, crewmember, the contact columns on patrulje, and signup — and two
// facts about that data make a query-time UNION over them a poor answer:
//
//   - A contact person is not a row anywhere. For a patrulje they are three columns on
//     the team; for a klan they exist only on signup, because klan has no contact
//     columns at all. So the adult most likely to ring during signup season is the
//     hardest person in the platform to look up.
//   - A phone number is free text. `+45 12 34 56 78`, `12 34 56 78` and `12345678` are
//     all in there, so matching has to be against a normalized form — and no index can
//     serve REPLACE(REPLACE(phone,' ',”),'+45',”), which means every lookup would
//     scan all six tables.
//
// At today's row counts the scan would in fact be fast enough. This exists anyway
// because search sits on the critical path of an inbound emergency call, where a
// predictable index seek is worth the duplication (PRD 014 §8).
//
// # What it deliberately does not do
//
// It reads no other projection's table. Every field comes from the event payload. A
// consumer that read `spejder` to build its row would depend on the order in which two
// consumers happen to see the same event, which is not guaranteed — and the failure
// would be a row that is silently missing a name.
//
// Team names and current status are therefore *not* stored here. They are joined at
// read time, which is safe because the tables are consistent by then, and which keeps
// this consumer subscribed only to the events that carry identity. In particular
// nothing here consumes a status subject: PRD 006's state machine stays in one place
// instead of being half-reimplemented where it could disagree with itself.
//
// # No publisher
//
// Search is read-only. There is no command side, so New takes no Publisher — the only
// way into this table is the event log, and the only way out is a query.
package searchperson

import (
	_ "embed"
	"fmt"

	"github.com/jrgensen/cqrs"
)

//go:embed table.sql
var tableSchema string

// Kind is what sort of person a row describes.
//
// The value is the *event subject's* entity token rather than the source table's name,
// so it lines up with the live entity tokens a client declares a dependency on. That
// distinction has bitten this codebase before: there is no `personnel` token, only
// `gøgler` and `friend`.
type Kind string

const (
	KindSpejder Kind = "spejder"
)

// Table is the projection.
type Table struct {
	w cqrs.Writer
	r cqrs.Reader
}

// New creates the schema and returns the projection.
//
// It returns an error rather than log.Fatalf'ing the way shared-go's entities do: this
// runs inside an API whose caller can report a failure usefully, and a search index that
// failed to build should be a startup error rather than a table that is silently empty —
// an empty search_person looks exactly like an event nobody signed up for.
func New(w cqrs.Writer, r cqrs.Reader) (*Table, error) {
	if w == nil {
		return nil, fmt.Errorf("searchperson: a Writer is required")
	}
	if err := w.Consume(tableSchema); err != nil {
		return nil, fmt.Errorf("searchperson: create table: %w", err)
	}
	for _, stmt := range schemaMigrations {
		if err := w.Consume(stmt); err != nil {
			return nil, fmt.Errorf("searchperson: migrate table: %w", err)
		}
	}
	return &Table{w: w, r: r}, nil
}

// schemaMigrations brings an already-created table up to the current shape.
//
// Empty today and present from the start, because CREATE TABLE IF NOT EXISTS is a no-op
// wherever the table already exists: a column added to table.sql later is silently
// absent from every database that has booted once. Harmless in stage and production,
// where the database is cleared before deploy, and invisible in dev — which is the worst
// of the two. Entries must be idempotent; they run on every boot, against both shapes.
var schemaMigrations = []string{}

// CreateTableSql exposes the schema, matching the convention the other entities use.
func (t *Table) CreateTableSql() string { return tableSchema }

// Assert the consumer contract at compile time. The mux accepts anything shaped roughly
// right, so a drifting signature would surface as a projection that never runs — and an
// empty index is indistinguishable from an event with no participants.
var _ cqrs.Consumer = (*Table)(nil)
