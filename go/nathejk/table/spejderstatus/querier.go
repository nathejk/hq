package spejderstatus

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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
	InOurCare(context.Context, types.YearSlug) (*Care, error)
}

// Care is how many members Nathejk is currently responsible for.
//
// The number that has to reach zero before the organisers can go home — which is why
// it is a type with a breakdown rather than an int: an operator who sees 3 needs to
// know whether that is three people waiting by a road or three asleep at HQ, because
// the first three are blocking three patrols and somebody has to drive.
type Care struct {
	Total    int                        `json:"total"`
	ByStatus map[types.MemberStatus]int `json:"byStatus"`

	// OldestWaitingAt is when the longest-waiting member asked to leave, or nil if
	// nobody is waiting. The threshold is applied by the caller, not here: it is
	// configuration and still unsettled (PRD 006 §11), so the query returns the fact
	// and lets the rule live where it can change.
	//
	// Only `waiting` is measured. A member in a car or asleep at HQ is accounted for
	// and their patrol has stopped waiting for them; a member by the trailside is
	// neither.
	OldestWaitingAt *time.Time `json:"oldestWaitingAt"`
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

// InOurCare counts the members Nathejk is responsible for right now, per status,
// with the oldest `waiting` timestamp for the alarm.
//
// The status set comes from InOurCareStatuses(), so this query does not name
// waiting/transit/sheltered anywhere — see the comment there for why that is not
// fussiness.
//
// Every in-care status is present in ByStatus even at zero. A breakdown that omitted
// empty states would make the display flicker between three rows and one as the night
// went on, and "transit: 0" is information: it says no car is currently carrying
// anybody.
func (q *querier) InOurCare(ctx context.Context, year types.YearSlug) (*Care, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	statuses := InOurCareStatuses()
	care := &Care{ByStatus: make(map[types.MemberStatus]int, len(statuses))}
	args := []any{string(year)}
	placeholders := make([]string, 0, len(statuses))
	for _, s := range statuses {
		care.ByStatus[s] = 0
		placeholders = append(placeholders, "?")
		args = append(args, string(s))
	}

	query := `SELECT status, COUNT(*), MIN(updatedAt)
		FROM spejderstatus
		WHERE year = ? AND status IN (` + strings.Join(placeholders, ",") + `)
		GROUP BY status`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status types.MemberStatus
		var count int
		var oldest sql.NullTime
		if err := rows.Scan(&status, &count, &oldest); err != nil {
			return nil, err
		}
		care.ByStatus[status] = count
		care.Total += count
		if status == types.MemberStatusWaiting && oldest.Valid {
			t := oldest.Time
			care.OldestWaitingAt = &t
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return care, nil
}

// GetByTeam reads every member currently attached to a team, whatever their status.
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
