package data

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/nathejk/shared-go/types"
)

type MemberModel struct {
	DB *sql.DB
}

type Spejder struct {
	ID       types.MemberID `json:"id"`
	MemberID types.MemberID `json:"memberId"`

	// InitialTeamID is the team the member signed up with, CurrentTeamID the one
	// they are racing for now. They differ only for a member who was moved — see
	// PRD 006. CurrentTeamID falls back to the signup team for a member with no
	// status row, because "not tracked yet" means they are still with their own
	// patrol.
	InitialTeamID types.TeamID       `json:"teamId"`
	CurrentTeamID types.TeamID       `json:"currentTeamId"`
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
	TShirtSize    string             `json:"tshirtSize"`
}

// GetSpejdere reads a patrol's members with their lifecycle status.
//
// The status comes from the spejderstatus projection and nowhere else. This query
// used to invent one when the join found no row:
//
//	IF(ss.status IS NULL, IF(ps.startedUts > 0, 'started', 'paid'), ss.status)
//
// Neither 'started' nor 'paid' is a types.MemberStatus. They were not in the
// documented legacy mapping either, so every lifecycle helper — Valid(),
// CanFinish(), InOurCare() — returned false for them, and the field was typed
// types.MemberStatus while holding values that type has never defined. The
// fallback also leaned on patruljestatus.startedUts, which despite its name is
// hardcoded to 1 on *signup*, so 'started' did not even mean started.
//
// A member with no row now reads as MemberStatusNone, which is what it means: no
// status recorded. Callers that want to show something for that ("ikke startet")
// decide so themselves rather than having the database decide for them.
func (m MemberModel) GetSpejdere(filters Filters) ([]*Spejder, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `Select
  s.memberId,
  s.teamId,
  COALESCE(ss.currentTeamId, s.teamId) AS currentTeamId,
  COALESCE(ss.status, '') AS status,
  name,
  address,
  postalCode,
  city,
  email,
  phone,
  phoneParent,
  birthday,
  ` + "`returning`" + `,
  tshirtsize
from spejder s
join patruljestatus ps on s.teamId = ps.teamId
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
		if err := rows.Scan(&s.ID, &s.InitialTeamID, &s.CurrentTeamID, &s.Status, &s.Name, &s.Address, &s.PostalCode, &s.City, &s.Email, &s.Phone, &s.PhoneParent, &s.Birthday, &s.Returning, &s.TShirtSize); err != nil {
			log.Print(err)
			return nil, Metadata{}, err
		}
		s.MemberID = s.ID
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

type Senior struct {
	ID         types.MemberID `json:"id"`
	MemberID   types.MemberID `json:"memberId"`
	TeamID     types.TeamID   `json:"teamId"`
	Name       string         `json:"name"`
	Address    string         `json:"address"`
	PostalCode string         `json:"postalCode"`
	City       string         `json:"city"`
	Email      string         `json:"email"`
	Phone      string         `json:"phone"`
	Birthday   types.Date     `json:"birthday"`
	Diet       string         `json:"diet"`
	TShirtSize string         `json:"tshirtSize"`
}

func (m MemberModel) GetSeniore(filters Filters) ([]*Senior, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `Select
  s.memberId,
  s.teamId,
  name,
  address,
  postalCode,
  city,
  email,
  phone,
  birthday,
  diet,
  tshirtsize
from senior s
WHERE  s.teamId = ?`
	args := []any{filters.TeamID}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Print(err)
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	members := []*Senior{}
	for rows.Next() {
		var s Senior
		if err := rows.Scan(&s.ID, &s.TeamID, &s.Name, &s.Address, &s.PostalCode, &s.City, &s.Email, &s.Phone, &s.Birthday, &s.Diet, &s.TShirtSize); err != nil {
			log.Print(err)
			return nil, Metadata{}, err
		}
		s.MemberID = s.ID
		members = append(members, &s)
	}
	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return members, metadata, nil
}
