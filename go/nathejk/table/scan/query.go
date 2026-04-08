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

	var uts types.UnixtimeInteger
	ss := []*Scan{}
	for rows.Next() {
		var r Scan
		err := rows.Scan(&r.QrID, &r.TeamID, &r.TeamNumber, &r.ScannerID, &r.ScannerPhone, &uts, &r.Latitude, &r.Longitude)
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
