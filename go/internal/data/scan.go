package data

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"nathejk.dk/internal/validator"
	"nathejk.dk/nathejk/types"
)

type CheckgroupScan struct {
	ScanID            types.ScanID
	UserID            types.UserID
	TeamID            types.TeamID
	Time              time.Time
	ControlGroupID    types.ControlGroupID
	ControlpointIndex int
	Coordinate        types.Coordinate
}

func (cs *CheckgroupScan) Validate(v validator.Validator) {
	//v.Check(p.Timestamp.IsZero(), "timestamp", "must be provided")
}

type ScanModel struct {
	DB *sql.DB
}

func (m ScanModel) GetCheckgroupsScans(filters Filters) ([]*CheckgroupScan, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `SELECT s.id, controlGroupId, controlIndex, userId, teamId, teamNumber, uts FROM controlgroup cg
	JOIN controlgroup_user cgu ON cg.id=cgu.controlGroupId AND (LOWER(cg.year) = LOWER(?) OR ? = '')
	JOIN scan s ON cgu.userId = s.scannerId AND cgu.startUts < s.uts AND cgu.endUts >s.uts`
	args := []any{filters.Year, filters.Year}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	css := []*CheckgroupScan{}
	for rows.Next() {
		var r struct {
			ScanID            types.ScanID
			ControlGroupID    types.ControlGroupID
			ControlpointIndex int
			UserID            types.UserID
			TeamID            types.TeamID
			TeamNumber        string
			Uts               types.UnixtimeInteger
			Lat               float64
			Lng               float64
		}

		err := rows.Scan(&r.ScanID, &r.ControlGroupID, &r.ControlpointIndex, &r.UserID, &r.TeamID, &r.TeamNumber, &r.Uts)
		if err != nil {
			return nil, Metadata{}, err
		}
		css = append(css, &CheckgroupScan{
			ScanID:            r.ScanID,
			UserID:            r.UserID,
			TeamID:            r.TeamID,
			Time:              *r.Uts.Time(),
			ControlGroupID:    r.ControlGroupID,
			ControlpointIndex: r.ControlpointIndex,
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

type CheckgroupTeamTime map[types.ControlGroupID]map[types.TeamID]time.Time

func (m ScanModel) GetNewestCheckgroupTeamTime(filters Filters) (CheckgroupTeamTime, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `SELECT cg.id, s.teamId, MAX(s.uts) FROM controlgroup cg 
	JOIN controlpoint cp ON cg.id = cp.relativeControlGroupId AND (LOWER(cg.year) = LOWER(?) OR ? = '')
	JOIN controlgroup_user cgu ON cg.id = cgu.controlGroupId AND cp.controlIndex = cgu.controlIndex 
	JOIN scan s ON cgu.userId = s.scannerId AND cgu.startUts <= s.uts AND cgu.endUts >= s.uts
	GROUP BY cg.id, s.teamId`
	args := []any{filters.Year, filters.Year}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	ctt := CheckgroupTeamTime{}
	for rows.Next() {
		var r struct {
			ControlGroupID types.ControlGroupID
			TeamID         types.TeamID
			Uts            types.UnixtimeInteger
		}
		err := rows.Scan(&r.ControlGroupID, &r.TeamID, &r.Uts)
		if err != nil {
			return nil, Metadata{}, err
		}
		if _, found := ctt[r.ControlGroupID]; !found {
			ctt[r.ControlGroupID] = map[types.TeamID]time.Time{}
		}
		ctt[r.ControlGroupID][r.TeamID] = *r.Uts.Time()
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return ctt, metadata, nil
}

type TeamScan struct {
	TeamID         types.TeamID     `json:"teamId"`
	ScanID         types.ScanID     `json:"scanId"`
	Time           *time.Time       `json:"time"`
	Coordinate     types.Coordinate `json:"coordinate"`
	UserID         types.UserID     `json:"userId"`
	UserName       *string          `json:"userName"`
	UserTeamName   *string          `json:"userTeamName"`
	CheckpointName *string          `json:"checkpointName"`
	Bandit         bool             `json:"bandit"`
}

func (m ScanModel) GetTeamScans(teamID types.TeamID) ([]*TeamScan, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `select p.teamId, s.id, uts, s.latitude, s.longitude, s.scannerId, u.name, u.department, cp.controlName, u.userId IS NULL
from scan s
join patrulje p on s.teamId = p.teamId
left join controlgroup_user cgu on s.scannerId = cgu.userId and s.uts <= cgu.endUts AND s.uts >= cgu.startUts
left join controlpoint cp on  cgu.controlGroupId = cp.controlGroupId AND cgu.controlIndex = cp.controlIndex
left join personnel u on s.scannerId = u.userId
where p.teamId = ? ORDER BY uts`
	args := []any{teamID}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tss := []*TeamScan{}
	for rows.Next() {
		var r TeamScan
		var lat, lng string
		var uts types.UnixtimeInteger

		err := rows.Scan(&r.TeamID, &r.ScanID, &uts, &lat, &lng, &r.UserID, &r.UserName, &r.UserTeamName, &r.CheckpointName, &r.Bandit)
		if err != nil {
			return nil, err
		}
		r.Time = uts.Time()
		if s, err := strconv.ParseFloat(lat, 64); err == nil {
			r.Coordinate.Latitude = s
		}
		if s, err := strconv.ParseFloat(lng, 64); err == nil {
			r.Coordinate.Longitude = s
		}
		tss = append(tss, &r)
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tss, nil
}
