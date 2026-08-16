package spejderstatus

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

// SpejderStatus is one member's place in the lifecycle.
//
// InitialTeamID and CurrentTeamID are both kept, and the difference between them
// is the interesting part: a member moved to another patrol races on with a team
// that is not the one they signed up with, and both halves of that matter later —
// the current team for strength and discontinuation, the initial one for results
// and for answering "whose patrol was this?" months afterwards.
type SpejderStatus struct {
	MemberID      types.MemberID     `json:"memberId"`
	YearSlug      types.YearSlug     `json:"year"`
	InitialTeamID types.TeamID       `json:"initialTeamId"`
	CurrentTeamID types.TeamID       `json:"currentTeamId"`
	Status        types.MemberStatus `json:"status"`
	UpdatedAt     time.Time          `json:"updatedAt"`
}

// InOurCare reports whether Nathejk is currently responsible for this member's
// whereabouts. Delegates to shared-go rather than comparing statuses here, so the
// set has exactly one definition.
func (s SpejderStatus) InOurCare() bool { return s.Status.InOurCare() }

type table struct {
	consumer
	querier
}

// New builds the projection.
//
// It takes the publisher even though the read side does not need one, so that
// wiring in cmd/api does not change when the commands land in task 072.
func New(w cqrs.Writer, r *sql.DB) *table {
	q := querier{db: r, r: goqu.New("mysql", r)}
	t := &table{
		consumer: consumer{w: w},
		querier:  q,
	}
	if err := w.Consume(t.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	return t
}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string { return tableSchema }

// Assert the consumer contract at compile time. The mux accepts anything shaped
// vaguely right, so a signature drifting out of step would otherwise surface as a
// projection that silently never runs — and a member projection that never runs
// looks exactly like an event with nobody in our care.
var _ cqrs.Consumer = (*table)(nil)
