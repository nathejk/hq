package dispatch

import (
	"database/sql"
	"log"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"

	_ "embed"
)

type table struct {
	commander
	consumer
	querier
}

// New builds the dispatch read model and command surface.
//
// The publisher is only used by the commander, the writer only by the consumer and the
// *sql.DB only by the querier; three arguments rather than one "db" because the write
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
	if err := w.Consume(dispatchableSchema); err != nil {
		log.Printf("Error creating table %q", err)
	}
	if err := w.Consume(dutySchema); err != nil {
		log.Printf("Error creating table %q", err)
	}
	for _, stmt := range schemaMigrations {
		if err := w.Consume(stmt); err != nil {
			log.Printf("Error migrating dispatch tables %q", err)
		}
	}
	return t
}

// schemaMigrations brings an existing database up to the current shape.
//
// Empty today and present from the start, because CREATE TABLE IF NOT EXISTS is a no-op
// wherever the table already exists: a column added later is silently absent from every
// database that has already booted. PRD 009 §8 makes the point that this window closes
// the first night the desk runs.
//
// Entries must be idempotent: they run on every boot, against both shapes.
var schemaMigrations = []string{}

//go:embed table.sql
var tableSchema string

// The dispatchable-section flag lives in its own file rather than with the tasks and
// tours, because it is a different kind of thing: configuration of which organisation
// sections are dispatch units, not a record of anything that happened.
//
//go:embed dispatchable.sql
var dispatchableSchema string

// Duty windows likewise live apart from the tasks and tours: a roster agreed days in advance is
// configuration of capacity, not a record of what happened on the night.
//
//go:embed duty.sql
var dutySchema string

func (t *table) CreateTableSql() string { return tableSchema }

// Assert the consumer contract at compile time. The mux accepts anything shaped vaguely
// right, so a drifting signature would surface as a projection that silently never runs.
var _ cqrs.Consumer = (*table)(nil)
