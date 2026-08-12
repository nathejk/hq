package data

import (
	"database/sql"
	"errors"

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
