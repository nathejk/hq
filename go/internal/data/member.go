package data

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/davecgh/go-spew/spew"
	"nathejk.dk/internal/validator"
	"nathejk.dk/nathejk/types"
)

// https://github.com/sunary/sqlize

type Member struct {
}

func (p *Member) Validate(v validator.Validator) {
	//v.Check(p.Timestamp.IsZero(), "timestamp", "must be provided")
}

type MemberModel struct {
	DB *sql.DB
}

type Spejder struct {
	ID            types.MemberID     `json:"id"`
	InitialTeamID types.TeamID       `json:"teamId"`
	CurrentTeamID types.TeamID       `json:"teamId"`
	Status        types.MemberStatus `json:"status"`
	Name          string             `json:"name"`
	Address       string             `json:"address"`
	PostalCode    string             `json:"postalCode"`
	City          string             `json:"city"`
	Email         string             `json:"email"`
	Phone         string             `json:"phone"`
	PhoneParent   string             `json:"phoneParent"`
	Birthday      types.Date         `json:"birthday"`
	Returning     bool               `json:"returning"`
}

func (m MemberModel) GetSpejdere(filters Filters) ([]*Spejder, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `Select 
  s.memberId, 
  s.teamId, 
  IF(pm.parentTeamId IS NULL, s.teamId, pm.parentTeamId) AS currentTeamId, 
  IF(ss.status IS NULL, IF(ps.startedUts > 0, 'started', 'paid'), ss.status) AS status,
  name,
  address,
  postalCode,
  city,
  email,
  phone,
  phoneParent,
  birthday,
  ` + "`returning`" + `
from spejder s
join patruljestatus ps on s.teamId = ps.teamId
left join patruljemerged pm on s.teamId = pm.teamId
left join spejderstatus ss on s.memberId = ss.id and s.year = ss.year
WHERE  (LOWER(s.year) = LOWER(?) OR ? = '') AND  (s.teamId = ? OR ? = '')`
	args := []any{filters.Year, filters.Year, filters.TeamID, filters.TeamID}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Print(err)
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	spejdere := []*Spejder{}
	for rows.Next() {
		var s Spejder
		if err := rows.Scan(&s.ID, &s.InitialTeamID, &s.CurrentTeamID, &s.Status, &s.Name, &s.Address, &s.PostalCode, &s.City, &s.Email, &s.Phone, &s.PhoneParent, &s.Birthday, &s.Returning); err != nil {
			log.Print(err)
			return nil, Metadata{}, err
		}
		spejdere = append(spejdere, &s)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return spejdere, metadata, nil
}

type SpejderStatus struct {
	MemberID  types.MemberID
	TeamID    types.TeamID
	Status    types.MemberStatus
	Name      string
	TeamName  string
	UpdatedAt string
}

func (m MemberModel) GetInactive(filters Filters) ([]*SpejderStatus, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := sq.Select("s.memberId", "s.name", "s.teamId", "p.name", "ss.status", "ss.updatedAt").
		From("spejder s").
		Join("patrulje p ON s.teamId = p.teamId").
		Join("spejderstatus ss ON s.memberId = ss.id AND s.year = ss.year")
	if filters.Year != "" {
		query = query.Where(sq.Eq{"s.year": filters.Year})
	}
	if values, found := filters.Search["status"]; found && len(values) > 0 {
		query = query.Where(sq.Eq{"ss.status": values})
	}

	/*
			Where(Or{Expr("s.year = LOWER(?)", 10), And{Eq{"k": 11}, Expr("true")}}).
		active := users.Where(sq.Eq{"deleted_at": nil})
	*/
	sql, args, err := query.ToSql()

	spew.Dump(sql, args, err)
	/*
	   	query := `
	   	select s.memberId, s.name, s.teamId, p.name, ss.status, ss.updatedAt
	   from spejder s
	   join patrulje p on s.teamId = p.teamId
	   join spejderstatus ss on s.memberId = ss.id and s.year = ss.year
	   WHERE (LOWER(s.year) = LOWER(?) OR ? = '')` // AND (ss.status IN (?) )`

	   	args := []any{filters.Year, filters.Year} //, filters.TeamID, filters.TeamID}
	   	rows, err := m.DB.QueryContext(ctx, query, args...)
	*/
	rows, err := m.DB.QueryContext(ctx, sql, args...)
	if err != nil {
		log.Print(err)
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	sss := []*SpejderStatus{}
	for rows.Next() {
		var s SpejderStatus
		if err := rows.Scan(&s.MemberID, &s.Name, &s.TeamID, &s.TeamName, &s.Status, &s.UpdatedAt); err != nil {
			return nil, Metadata{}, err
		}
		sss = append(sss, &s)
		totalRecords++
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return sss, metadata, nil
}

func (m TeamModel) GetSpejder(teamID types.TeamID) (*Patrulje, error) {
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
