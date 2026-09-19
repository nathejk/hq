package checkpersonnel

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

type CheckpointScan struct {
	ScanID types.ScanID `json:"scanId"`
	TeamID types.TeamID `json:"teamId"`
	Time   time.Time    `json:"time"`
}
type Checkpersonnel struct {
	ID           types.CheckpersonnelID `json:"id" db:"id"`
	Year         types.YearSlug         `json:"year" db:"year"`
	CheckgroupID types.CheckgroupID     `json:"checkgroupId" db:"checkgroupId"`
	CheckpointID types.CheckpointID     `json:"checkpointId" db:"checkpointId"`
	UserID       types.UserID           `json:"userId" db:"userId"`
	Start        time.Time              `json:"start" db:"startUts"`
	End          time.Time              `json:"end" db:"endUts"`
	Scans        []CheckpointScan       `json:"scans"`
}

// CheckPersonnel is an alias kept for backward compatibility with external references.
type CheckPersonnel = Checkpersonnel

type table struct {
	commander
	consumer
	querier
}

func New(p stream.Publisher, w cqrs.Writer, r *sql.DB) *table {
	q := querier{db: r, r: goqu.New("mysql", r)}
	table := &table{commander: commander{p: p, q: &q}, consumer: consumer{w: w}, querier: q}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	for _, stmt := range schemaMigrations {
		if err := w.Consume(stmt); err != nil {
			log.Printf("Error migrating checkpersonnel table %q", err)
		}
	}
	return table
}

// schemaMigrations brings an existing checkpersonnel table up to the current schema.
//
// Needed because CREATE TABLE IF NOT EXISTS is a no-op wherever the table already exists,
// so an index added to table.sql alone only takes effect on databases that get cleared
// before deploy — which silently leaves dev, and any long-lived database, on the old
// schema. The same reason patrulje carries a list like this.
//
// ADD INDEX IF NOT EXISTS is idempotent, so re-running against a table that already has
// it is accepted and changes nothing.
var schemaMigrations = []string{
	// See the comment on this index in table.sql: without it, attributing scans to posts
	// rescans every shift for every scan.
	`ALTER TABLE checkpersonnel ADD INDEX IF NOT EXISTS idx_checkpersonnel_user (userId, startUts, endUts)`,
}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string {
	return tableSchema
}
