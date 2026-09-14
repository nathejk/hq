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
//     all in there, and a guardian field may hold two numbers at once, so matching has
//     to be against a normalized form computed once at write time rather than in an
//     unindexable expression per query.
//
// Note that speed is **not** on that list. It was the original argument, and measurement
// retired it: a scan over this table costs single-digit milliseconds at many times the
// current size, and since task 187 the phone lookup *is* a scan. What the projection buys
// is the two things above — people who were previously unfindable at all, and one place
// where normalisation is defined — plus one query instead of six.
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
	KindSenior  Kind = "senior"

	// KindGoegler and KindFriend are the two personnel populations. They are separate
	// kinds rather than one "personnel", because `personnel` is the *table's* name and
	// there is no such live entity token — a client depending on `personnel` would wait
	// forever.
	//
	// ASCII identifiers with Danish values, matching the rest of the codebase: no exported
	// Go identifier here carries an ø.
	KindGoegler Kind = "gøgler"
	KindFriend  Kind = "friend"

	KindCrew Kind = "crew"

	// The two contact-person kinds — the people this projection exists for as much as any.
	//
	// A contact person is not a row anywhere else in the platform. For a patrulje they are
	// three columns on the team; for a klan they are nowhere at all, since the klan table has
	// no contact columns. So the adult who submitted the team, and who is the one most likely
	// to ring, has until now been the hardest person here to look up.
	//
	// Keyed by teamId, because they have no id of their own. That is what kind is doing in
	// the primary key: a contact person and a member can share an id string without
	// colliding.
	KindPatruljeContact Kind = "patruljekontakt"
	KindKlanContact     Kind = "klankontakt"
)

// There is deliberately no KindBandit.
//
// A bandit is not a population of its own: `bandit.*.armNumber.assigned` is the only
// bandit-entity event, it carries neither a name nor a number, and both the senior and
// personnel projections consume it merely to stamp an arm number onto a row that already
// exists. The people themselves arrive as seniors. Adding a bandit kind would produce a
// table of empty rows and a live dependency that never fires.

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
