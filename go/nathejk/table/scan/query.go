package scan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
)

type querier struct {
	db *sql.DB
}

func (q *querier) GetByID(ctx context.Context, qrID types.QrID) (*QR, error) {
	query := `SELECT id, teamNumber, mapCreatedBy, mapCreatedAt
		FROM scan
		WHERE id = ?`
	var r QR
	var id int
	err := q.db.QueryRow(query, qrID).Scan(
		&id,
		&r.TeamNumber,
		&r.CreatedBy,
		&r.CreatedAt,
	)
	r.ID = types.QrID(fmt.Sprintf("%d", id))
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, tables.ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &r, nil
}

func (q *querier) GetAll(ctx context.Context, filter Filter) ([]*Scan, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT qrId, teamId, teamNumber, scannerId, scannerPhone, uts, latitude, longitude FROM scan
	WHERE (teamId = ? OR ? = '') ORDER BY uts`
	args := []any{filter.TeamID, filter.TeamID}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	ss := []*Scan{}
	for rows.Next() {
		var r Scan
		// uts is scanned straight into the row.
		//
		// It used to be scanned into a `var uts types.UnixtimeInteger` declared outside this loop and
		// then never assigned to `r`, so every scan this query returned carried uts=0 while the table
		// held the real value. Found while building the patrol track map (PRD 011, task 149), where it
		// mattered visibly: tracks and scans share one time axis there, so every scan marker landed in
		// 1970. It had been silently wrong for `GET /api/patrulje/:id/scans` too.
		err := rows.Scan(&r.QrID, &r.TeamID, &r.TeamNumber, &r.ScannerID, &r.ScannerPhone, &r.Uts, &r.Latitude, &r.Longitude)
		if err != nil {
			return nil, Metadata{}, err
		}
		ss = append(ss, &r)
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	//metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return ss, Metadata{}, nil
}

// TeamPosition is one scan reduced to a position: where a team was seen, and when.
//
// Units and types are the table's own, not the map's. `Uts` is **seconds** while telemetry is
// milliseconds, and the coordinates are VARCHAR and may be junk ("", "0"). Both are converted and
// validated once, at the read boundary that merges scans with telemetry — see cmd/api/telemetry.go.
// Normalising here would put a second, divergent copy of those rules in the storage layer.
type TeamPosition struct {
	TeamID    types.TeamID
	Uts       int64
	Latitude  string
	Longitude string
}

// TeamPositions reports every usable scan position in a year, one row per scan, in time order.
//
// The obviously unusable coordinates are excluded here as well as at the boundary, and that is not
// redundancy for its own sake: a scanner that saved without a fix writes "" or "0", such rows are
// common, and letting them through would mean carrying thousands of rows across the wire for the
// caller to throw away. The boundary still re-checks, because VARCHAR admits shapes SQL cannot
// enumerate.
//
// Ordered ascending on `uts` so a fold that overwrites keeps a team's newest scan. The
// `year_teamId (year, teamId, uts)` index serves the year predicate; the sort is a few thousand rows.
func (q *querier) TeamPositions(ctx context.Context, year string) ([]TeamPosition, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := q.db.QueryContext(ctx, `SELECT teamId, uts, latitude, longitude FROM scan
		WHERE LOWER(year) = LOWER(?)
		  AND latitude NOT IN ('', '0') AND longitude NOT IN ('', '0')
		ORDER BY uts ASC`, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []TeamPosition{}
	for rows.Next() {
		var p TeamPosition
		if err := rows.Scan(&p.TeamID, &p.Uts, &p.Latitude, &p.Longitude); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type CheckgroupScan struct {
	ScanID          types.ScanID
	UserID          types.UserID
	TeamID          types.TeamID
	Time            time.Time
	CheckgroupID    types.CheckgroupID
	CheckpointIndex int
	Coordinate      types.Coordinate
}

func (q *querier) GetCheckgroupsScans(ctx context.Context, filters Filter) ([]*CheckgroupScan, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT s.id, controlGroupId, controlIndex, userId, teamId, teamNumber, uts FROM controlgroup cg
	JOIN controlgroup_user cgu ON cg.id=cgu.controlGroupId AND (LOWER(cg.year) = LOWER(?) OR ? = '')
	JOIN scan s ON cgu.userId = s.scannerId AND cgu.startUts < s.uts AND cgu.endUts >s.uts`
	args := []any{filters.Year, filters.Year}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	css := []*CheckgroupScan{}
	for rows.Next() {
		var r struct {
			ScanID          types.ScanID
			CheckgroupID    types.CheckgroupID
			CheckpointIndex int
			UserID          types.UserID
			TeamID          types.TeamID
			TeamNumber      string
			Uts             types.UnixtimeInteger
			Lat             float64
			Lng             float64
		}

		err := rows.Scan(&r.ScanID, &r.CheckgroupID, &r.CheckpointIndex, &r.UserID, &r.TeamID, &r.TeamNumber, &r.Uts)
		if err != nil {
			return nil, Metadata{}, err
		}
		css = append(css, &CheckgroupScan{
			ScanID:          r.ScanID,
			UserID:          r.UserID,
			TeamID:          r.TeamID,
			Time:            *r.Uts.Time(),
			CheckgroupID:    r.CheckgroupID,
			CheckpointIndex: r.CheckpointIndex,
			Coordinate: types.Coordinate{
				Latitude:  r.Lat,
				Longitude: r.Lng,
			},
		})
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return css, metadata, nil
}
