package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"nathejk.dk/internal/validator"
	"nathejk.dk/nathejk/types"
)

type Team struct {
}

func (p *Team) Validate(v validator.Validator) {
	//v.Check(p.Timestamp.IsZero(), "timestamp", "must be provided")
}

type TeamModel struct {
	DB *sql.DB
}

func (m *TeamModel) query(filters Filters, query string, args []any) ([]types.TeamID, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	teamIDs := []types.TeamID{}
	for rows.Next() {
		var teamID types.TeamID
		if err := rows.Scan(&teamID); err != nil {
			return nil, Metadata{}, err
		}
		teamIDs = append(teamIDs, teamID)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return teamIDs, metadata, nil
}

func (m TeamModel) GetStartedTeamIDs(filters Filters) ([]types.TeamID, Metadata, error) {
	sql := `SELECT teamId FROM patruljestatus WHERE startedUts > 0 AND (LOWER(year) = LOWER(?) OR ? = '')`
	args := []any{filters.Year, filters.Year}
	return m.query(filters, sql, args)
}

func (m TeamModel) GetDiscontinuedTeamIDs(filters Filters) ([]types.TeamID, Metadata, error) {
	//sql := "SELECT teamId FROM patruljestatus WHERE startedUts > 0 AND (LOWER(year) = LOWER($1) OR $1 = '')"
	sql := `SELECT DISTINCT m.teamId FROM patruljemerged m JOIN patruljestatus s ON m.teamId = s.teamId WHERE s.startedUts > 0 AND (LOWER(year) = LOWER(?) OR ? = '')`
	args := []any{filters.Year, filters.Year}
	return m.query(filters, sql, args)
}

type PatruljeStatus struct {
	Year        string
	Status      string
	TeamCount   int
	MemberCount int
}

func (m TeamModel) GetStatus(filters Filters) ([]PatruljeStatus, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `SELECT COUNT(teamId), SUM(memberCount), year, signupStatus FROM patrulje WHERE (LOWER(year) = LOWER(?) OR ? = '') GROUP BY signupStatus`
	args := []any{filters.Year, filters.Year}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	ps := []PatruljeStatus{}
	for rows.Next() {
		var s PatruljeStatus
		if err := rows.Scan(&s.TeamCount, &s.MemberCount, &s.Year, &s.Status); err != nil {
			return nil, Metadata{}, err
		}
		ps = append(ps, s)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	return ps, Metadata{}, nil
}

type Patrulje struct {
	ID          types.TeamID `json:"id"`
	Number      string       `json:"number"`
	Status      string       `json:"status"`
	Name        string       `json:"name"`
	Group       string       `json:"group"`
	Corps       string       `json:"corps"`
	MemberCount int          `json:"memberCount"`
}

func (m TeamModel) GetPatruljer(filters Filters) ([]*Patrulje, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `SELECT p.teamId, p.teamNumber, p.name, p.groupName, p.korps, p.memberCount, IF(pm.parentTeamId IS NOT NULL, "JOIN", IF(startedUts > 0, "STARTED",  signupStatus)) 
		FROM patrulje p 
		JOIN patruljestatus ps ON p.teamId = ps.teamID AND (LOWER(p.year) = LOWER(?) OR ? = '')
		LEFT JOIN patruljemerged pm ON p.teamId = pm.teamId`
	args := []any{filters.Year, filters.Year}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	patruljer := []*Patrulje{}
	for rows.Next() {
		var p Patrulje
		if err := rows.Scan(&p.ID, &p.Number, &p.Name, &p.Group, &p.Corps, &p.MemberCount, &p.Status); err != nil {
			return nil, Metadata{}, err
		}
		patruljer = append(patruljer, &p)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return patruljer, metadata, nil
}

func (m TeamModel) GetPatrulje(teamID types.TeamID) (*Patrulje, error) {
	if len(teamID) == 0 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT p.teamId, p.teamNumber, p.name, p.groupName, p.korps, p.memberCount, IF(pm.parentTeamId IS NOT NULL, "JOIN", IF(startedUts > 0, "STARTED",  signupStatus)) 
		FROM patrulje p 
		JOIN patruljestatus ps ON p.teamId = ps.teamID
		LEFT JOIN patruljemerged pm ON p.teamId = pm.teamId
		WHERE p.teamId = ?`
	var p Patrulje
	err := m.DB.QueryRow(query, teamID).Scan(
		&p.ID,
		&p.Number,
		&p.Name,
		&p.Group,
		&p.Corps,
		&p.MemberCount,
		&p.Status,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &p, nil
}
