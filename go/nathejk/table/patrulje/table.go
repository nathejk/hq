package patrulje

import (
	"database/sql"
	"log"

	"github.com/doug-martin/goqu/v9"
	"github.com/jrgensen/cqrs"
	"github.com/nathejk/shared-go/types"

	_ "embed"
)

type Patrulje struct {
	TeamID       types.TeamID       `json:"teamId"`
	TeamNumber   string             `json:"teamNumber"`
	Year         string             `json:"year"`
	Name         string             `json:"name"`
	Group        string             `json:"group"`
	Korps        string             `json:"korps"`
	Liga         string             `json:"liga"`
	ContactName  string             `json:"contactName"`
	ContactPhone types.PhoneNumber  `json:"contactPhone"`
	ContactEmail types.EmailAddress `json:"contactEmail"`
	ContactRole  string             `json:"contactRole"`
	MemberCount  int                `json:"memberCount"`

	// ActiveMemberCount is how many of the team's members are still racing: its
	// strength on the route, and — when it reaches zero — the fact that the team is
	// discontinued (udgået). One number for both, so no caller derives either
	// independently and drifts.
	//
	// Maintained by the spejderstatus projection rather than by this package's
	// consumer, which owns the rest of the row. That is deliberate: the count is a
	// function of member rows, and a recompute here would race the member projection
	// over the same message — see the comment on recomputeActiveMemberCount in
	// table/spejderstatus/consumer.go.
	//
	// Distinct from MemberCount, which is frozen at who started and does not move.
	ActiveMemberCount int                `json:"activeMemberCount"`
	TshirtCount       int                `json:"tshirtCount"`
	SignupStatus      types.SignupStatus `json:"signupStatus"`
	PaidAmount        int                `json:"paidAmount"`
}

type table struct {
	consumer
	querier
}

func New(w cqrs.Writer, r *sql.DB) *table {
	q := querier{db: r, r: goqu.New("mysql", r)}
	table := &table{consumer: consumer{w: w}, querier: q}
	if err := w.Consume(table.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	for _, stmt := range schemaMigrations {
		if err := w.Consume(stmt); err != nil {
			log.Printf("Error migrating patrulje table %q", err)
		}
	}
	return table
}

// widenTextColumns brings an existing patrulje table up to the current schema.
//
// Needed because CREATE TABLE IF NOT EXISTS is a no-op wherever the table already
// exists, so widening a column in table.sql alone only takes effect on databases
// that get cleared before deploy. Without this, dev and any long-lived database
// keep the old width and keep failing.
//
// The failure this fixes was silent in the worst way: a name or group longer than
// 99 characters made the projection's UPDATE fail with "Error 1406: Data too long
// for column", so that patrulje's name, group, korps and contact details simply
// stopped being projected — the row kept whatever it held before. Teams entered as
// a merger of three groups exceed 99 easily.
//
// MODIFY is idempotent: re-running it against a column that is already VARCHAR(999)
// is accepted and changes nothing. ADD COLUMN IF NOT EXISTS is idempotent for the
// same reason, which is what lets a new column arrive without dropping the table.
var schemaMigrations = []string{
	`ALTER TABLE patrulje MODIFY COLUMN name VARCHAR(999) NOT NULL DEFAULT ""`,
	`ALTER TABLE patrulje MODIFY COLUMN groupName VARCHAR(999) NOT NULL DEFAULT ""`,

	// activeMemberCount (PRD 006). Added here as well as in table.sql because a
	// long-lived database — dev, and anything not cleared before deploy — would
	// otherwise never gain the column, and every recompute would fail with "Unknown
	// column". That is not hypothetical: it happened on the dev stack while task 065
	// was being built, and the projection wrote nothing at all until the table was
	// dropped by hand. A column nobody can add without manual intervention is a
	// column that will be missing in production one day.
	`ALTER TABLE patrulje ADD COLUMN IF NOT EXISTS activeMemberCount INT NOT NULL DEFAULT 0`,
}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string {
	return tableSchema
}
