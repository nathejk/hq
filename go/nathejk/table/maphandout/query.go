package maphandout

import (
	"context"
	"database/sql"
	"time"

	"github.com/nathejk/shared-go/types"
)

type Queries interface {
	// ByTeam lists every map/QR ever handed to a team, with each code's current holder,
	// so the caller can show whether the team still has it.
	ByTeam(ctx context.Context, year types.YearSlug, teamID types.TeamID) ([]Handout, error)
}

// Handout is one map/QR code this team was given, and where that code is now.
type Handout struct {
	QrID    string `json:"qrId"`
	MapID   string `json:"mapId"`
	MapName string `json:"mapName"`

	// RegisteredBy is the scanner who bound the code to this team; First/LastUts the
	// interval the team held it (equal for a code handed over once).
	RegisteredBy string `json:"registeredBy"`
	FirstUts     int64  `json:"firstUts"`
	LastUts      int64  `json:"lastUts"`

	// Current is true when this team is still the code's holder. When false the code was
	// reassigned — to a discontinued team's successor, typically — and the CurrentTeam*
	// fields say who holds it now.
	Current           bool   `json:"current"`
	CurrentTeamID     string `json:"currentTeamId"`
	CurrentTeamNumber string `json:"currentTeamNumber"`
	CurrentTeamName   string `json:"currentTeamName"`
}

type querier struct {
	db *sql.DB
}

func (q *querier) ByTeam(ctx context.Context, year types.YearSlug, teamID types.TeamID) ([]Handout, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// The current holder of a code is its newest binding. Derived here with a window
	// function rather than stored, because "current" is a property of the whole set of a
	// code's bindings, not of any one row, and a stored flag would have to be flipped on
	// every row of a code whenever it moved. teamId breaks ties so the pick is stable if
	// two bindings share a timestamp.
	//
	// The map name is joined from kort (the same sheet ids skan chose from, since kort is
	// event-sourced from one stream); a sheet since deleted, or an unknown "" sheet,
	// simply reads blank. The current team's number and name come from patrulje when it
	// is one — falling back to the number recorded at registration for a non-patrulje.
	const query = `
		SELECT mh.qrId, mh.mapId, COALESCE(k.name, '') AS mapName,
		       mh.registeredBy, mh.firstUts, mh.lastUts,
		       cur.teamId AS currentTeamId,
		       COALESCE(NULLIF(p.teamNumber, ''), cur.teamNumber) AS currentTeamNumber,
		       COALESCE(p.name, '') AS currentTeamName
		FROM maphandout mh
		LEFT JOIN kort k ON k.id = mh.mapId AND k.year = mh.year
		JOIN (
			SELECT year, qrId, teamId, teamNumber,
			       ROW_NUMBER() OVER (PARTITION BY year, qrId ORDER BY lastUts DESC, teamId DESC) rn
			FROM maphandout
		) cur ON cur.year = mh.year AND cur.qrId = mh.qrId AND cur.rn = 1
		LEFT JOIN patrulje p ON p.teamId = cur.teamId
		WHERE mh.year = ? AND mh.teamId = ?
		ORDER BY mh.lastUts, mh.qrId`

	rows, err := q.db.QueryContext(ctx, query, string(year), string(teamID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Handout{}
	for rows.Next() {
		var h Handout
		if err := rows.Scan(
			&h.QrID, &h.MapID, &h.MapName,
			&h.RegisteredBy, &h.FirstUts, &h.LastUts,
			&h.CurrentTeamID, &h.CurrentTeamNumber, &h.CurrentTeamName,
		); err != nil {
			return nil, err
		}
		h.Current = h.CurrentTeamID == string(teamID)
		out = append(out, h)
	}
	return out, rows.Err()
}
