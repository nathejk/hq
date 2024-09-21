package data

import (
	"context"
	"database/sql"
	"log"
	"time"

	"nathejk.dk/internal/validator"
	"nathejk.dk/nathejk/types"
)

type Sos struct {
	SosID       types.SosID `json:"sosId"`
	Year        string      `json:"year"`
	Headline    string      `json:"headline"`
	Description string      `json:"description"`
	Severity    string      `json:"severity"`
	Status      string      `json:"status"`

	CreatedAt       time.Time    `json:"createdAt"`
	CreatedByUserID types.UserID `json:"createdByUserId"`
}

func (cs *Sos) Validate(v validator.Validator) {
	//v.Check(p.Timestamp.IsZero(), "timestamp", "must be provided")
}

type SosModel struct {
	DB *sql.DB
}

func (m SosModel) GetByTeam(teamID types.TeamID) ([]*Sos, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `SELECT id, year, headline, description, createdAt, createdBy, status, severity FROM sos s JOIN sos_team st ON s.id = st.sosId WHERE st.teamId = ?`
	args := []any{teamID}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ss := []*Sos{}
	for rows.Next() {
		var s Sos
		var ts string
		err := rows.Scan(&s.SosID, &s.Year, &s.Headline, &s.Description, &ts, &s.CreatedByUserID, &s.Status, &s.Severity)
		if err != nil {
			return nil, err
		}
		s.CreatedAt, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			log.Printf("Datofejl: %q", err)
		}
		ss = append(ss, &s)
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ss, nil
}
