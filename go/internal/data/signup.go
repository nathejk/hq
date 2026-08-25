package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/nathejk/shared-go/types"
)

type Signup struct {
	TeamID       types.TeamID        `json:"teamId"`
	TeamType     types.TeamType      `json:"teamType"`
	Name         string              `json:"name"`
	Email        *types.EmailAddress `json:"email"`
	EmailPending types.EmailAddress  `json:"emailPending"`
	Phone        *types.PhoneNumber  `json:"phone"`
	PhonePending types.PhoneNumber   `json:"phonePending"`
	Pincode      string              `json:"-"`
	CreatedAt    string              `json:"createdAt"`
}

type SignupModel struct {
	DB *sql.DB
}

// TeamIDsByType lists the teams of one type that reached signup in a year.
//
// Returned as a set of ids rather than whole Signup rows because that is all the
// caller needs: whether a given team has a signup at all. For crew members that
// is the difference between somebody who filled in the public form — and so has
// a signup page worth linking to — and somebody an HQ operator typed in by hand,
// who has none. Both are plain UUIDs, so the id alone cannot tell them apart.
func (m SignupModel) TeamIDsByType(ctx context.Context, year types.YearSlug, teamType types.TeamType) (map[types.TeamID]bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx,
		`SELECT teamId FROM signup WHERE year = ? AND teamType = ?`,
		string(year), string(teamType),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := map[types.TeamID]bool{}
	for rows.Next() {
		var id types.TeamID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

func (m SignupModel) GetByID(teamID types.TeamID) (*Signup, error) {
	if len(teamID) == 0 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT teamId, teamType, name, email, emailPending, phone, phonePending, pincode, createdAt
		FROM signup
		WHERE teamId = ?`
	var p Signup
	err := m.DB.QueryRow(query, teamID).Scan(
		&p.TeamID,
		&p.TeamType,
		&p.Name,
		&p.Email,
		&p.EmailPending,
		&p.Phone,
		&p.PhonePending,
		&p.Pincode,
		&p.CreatedAt,
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
