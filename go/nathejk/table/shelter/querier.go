package shelter

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
)

// ErrRecordNotFound is returned when a member has no shelter row — which is the ordinary
// case for everybody not currently in the shelter's care, not an exceptional one.
var ErrRecordNotFound = errors.New("record not found")

type Queries interface {
	GetByMemberIDs(context.Context, types.YearSlug, []types.MemberID) (map[types.MemberID]Placement, error)
	DistinctPlacements(context.Context, types.YearSlug) ([]Zone, error)
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

const selectPlacement = `SELECT id, year, teamId, placement, acceptedAt, placedAt
	FROM shelter`

// GetByMemberIDs reads the placeringer of specific members.
//
// Plural and batched by design: the shelter screen has just fetched a list of sheltered
// members from spejderstatus and needs all their placeringer, and a lookup per member is a
// query per row on a page the crew keeps open all night.
//
// A member with no row is simply absent from the map. Callers must treat that as "not in the
// shelter" rather than an error — a scout in a car has no placering and that is not a problem
// to report.
func (q *querier) GetByMemberIDs(ctx context.Context, year types.YearSlug, ids []types.MemberID) (map[types.MemberID]Placement, error) {
	out := map[types.MemberID]Placement{}
	if len(ids) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	args := make([]any, 0, len(ids)+1)
	args = append(args, string(year))
	placeholders := make([]string, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, string(id))
	}

	query := selectPlacement + `
		WHERE year = ? AND id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		p, err := scanPlacement(rows)
		if err != nil {
			return nil, err
		}
		out[p.MemberID] = p
	}
	return out, rows.Err()
}

// DistinctPlacements is the placering vocabulary in use tonight, most-used first.
//
// This is the whole zone model. The zones scouts are kept in are not known until race start
// (PRD 007 §6), so there is no zone table to read and nothing to configure: the suggestions
// are whatever the crew has already typed. The first scout into a tent is typed, every one
// after that is picked from here, and "Telt 4", "telt4" and "t4" therefore do not become
// three places without anybody having set anything up.
//
// Ordered by count descending because that is what makes a typo harmless: the real tent, with
// four scouts in it, sits above the fat-fingered one with a single scout. Name ascending as
// the tiebreak, so a list of equal counts is stable between loads rather than reshuffling
// under the cursor.
//
// Empty placeringer are excluded. A scout accepted but not yet bedded down is not evidence of
// a zone called "", and offering it as a suggestion would be offering to un-place somebody.
// distinctPlacementsQuery is a const so it can be asserted on without a database, the same
// way the projection's statements are. What is worth pinning is the ordering (a typo must sort
// below the real tent) and the exclusion of the empty placering.
const distinctPlacementsQuery = `SELECT placement, COUNT(*) AS c
	FROM shelter
	WHERE year = ? AND placement <> ''
	GROUP BY placement
	ORDER BY c DESC, placement ASC`

func (q *querier) DistinctPlacements(ctx context.Context, year types.YearSlug) ([]Zone, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := q.db.QueryContext(ctx, distinctPlacementsQuery, string(year))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zones := []Zone{}
	for rows.Next() {
		var z Zone
		if err := rows.Scan(&z.Placement, &z.Count); err != nil {
			return nil, err
		}
		zones = append(zones, z)
	}
	return zones, rows.Err()
}

// scanPlacement reads one row, mapping a NULL placedAt to a nil pointer rather than to the
// zero time — which the SPA would render as 1970 and an operator would read as a bug.
func scanPlacement(rows *sql.Rows) (Placement, error) {
	var p Placement
	var acceptedAt time.Time
	var placedAt sql.NullTime
	if err := rows.Scan(&p.MemberID, &p.YearSlug, &p.TeamID, &p.Placement, &acceptedAt, &placedAt); err != nil {
		return p, err
	}
	p.AcceptedAt = acceptedAt
	if placedAt.Valid {
		t := placedAt.Time
		p.PlacedAt = &t
	}
	return p, nil
}
