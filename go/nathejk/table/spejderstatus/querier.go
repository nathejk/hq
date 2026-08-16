package spejderstatus

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
)

// ErrRecordNotFound is returned when a member has no status row.
//
// Its own sentinel rather than a copy of shared-go's: an errors.New duplicate
// would be a distinct value and errors.Is would silently stop matching, which is
// the trap the go-bff-layout notes record about internal/data. When this package
// is lifted (task 083) this alias goes and shared-go's is used directly.
var ErrRecordNotFound = errors.New("record not found")

type Filter struct {
	YearSlug types.YearSlug
	TeamID   types.TeamID
}

type Queries interface {
	GetByMemberID(context.Context, types.YearSlug, types.MemberID) (*SpejderStatus, error)
	GetByTeam(context.Context, Filter) ([]SpejderStatus, error)
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

// GetByMemberID reads one member's current place in the lifecycle.
//
// This is what the commands dirty-check against, which is the reason it exists:
// the resume action is valid only while the member is still self-carrying, and
// that can only be decided against the stored status rather than against whatever
// the operator's screen last showed.
func (q *querier) GetByMemberID(ctx context.Context, year types.YearSlug, id types.MemberID) (*SpejderStatus, error) {
	if id == "" {
		return nil, ErrRecordNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT id, year, initialTeamId, currentTeamId, status, updatedAt
		FROM spejderstatus WHERE year = ? AND id = ?`

	var s SpejderStatus
	var updatedAt time.Time
	err := q.db.QueryRowContext(ctx, query, string(year), string(id)).Scan(
		&s.MemberID, &s.YearSlug, &s.InitialTeamID, &s.CurrentTeamID, &s.Status, &updatedAt,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrRecordNotFound
	case err != nil:
		return nil, err
	}
	s.UpdatedAt = updatedAt
	return &s, nil
}

// GetByTeam reads every member currently attached to a team, whatever their
// status.
//
// Keyed on currentTeamId rather than initialTeamId, because the question callers
// ask is "who is with this patrol now?" — a member who was moved away is no longer
// this patrol's business, and a member moved in is.
func (q *querier) GetByTeam(ctx context.Context, f Filter) ([]SpejderStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT id, year, initialTeamId, currentTeamId, status, updatedAt
		FROM spejderstatus
		WHERE year = ? AND currentTeamId = ?
		ORDER BY id`

	rows, err := q.db.QueryContext(ctx, query, string(f.YearSlug), string(f.TeamID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []SpejderStatus{}
	for rows.Next() {
		var s SpejderStatus
		var updatedAt time.Time
		if err := rows.Scan(&s.MemberID, &s.YearSlug, &s.InitialTeamID, &s.CurrentTeamID, &s.Status, &updatedAt); err != nil {
			return nil, err
		}
		s.UpdatedAt = updatedAt
		members = append(members, s)
	}
	return members, rows.Err()
}
