package lok

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
)

type querier struct {
	db *sql.DB
}

func (q *querier) GetByID(ctx context.Context, lokID types.LokID) (*Lok, error) {
	query := `SELECT lokId, name, sortOrder, userIds, teamIds FROM lok WHERE lokId = ?`
	r := Lok{UserIDs: []types.UserID{}, TeamIDs: []types.TeamID{}}
	var userIDs, teamIDs string
	err := q.db.QueryRow(query, lokID).Scan(
		&r.LokID,
		&r.Name,
		&r.SortOrder,
		&userIDs,
		&teamIDs,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, tables.ErrRecordNotFound
		default:
			return nil, err
		}
	}
	if len(userIDs) > 0 {
		for _, id := range strings.Split(userIDs, ",") {
			r.UserIDs = append(r.UserIDs, types.UserID(id))
		}
	}
	if len(teamIDs) > 0 {
		for _, id := range strings.Split(teamIDs, ",") {
			r.TeamIDs = append(r.TeamIDs, types.TeamID(id))
		}
	}
	return &r, nil
}

func (q *querier) GetAll(ctx context.Context, filter Filter) ([]*Lok, Metadata, error) {
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT lokId, name, sortOrder, userIds, teamIds FROM lok
	WHERE (LOWER(year) = LOWER(?) OR ? = '') ORDER BY sortOrder`
	args := []any{filter.YearSlug, filter.YearSlug}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	loks := []*Lok{}
	for rows.Next() {
		r := Lok{UserIDs: []types.UserID{}, TeamIDs: []types.TeamID{}}
		var userIDs, teamIDs string
		err := rows.Scan(&r.LokID, &r.Name, &r.SortOrder, &userIDs, &teamIDs)
		if err != nil {
			return nil, Metadata{}, err
		}
		for _, id := range strings.Split(userIDs, ",") {
			r.UserIDs = append(r.UserIDs, types.UserID(id))
		}
		for _, id := range strings.Split(teamIDs, ",") {
			r.TeamIDs = append(r.TeamIDs, types.TeamID(id))
		}
		loks = append(loks, &r)
	}

	// When the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	//metadata := calculateMetadata(filters.Year, totalRecords, filters.Page, filters.PageSize)

	return loks, Metadata{}, nil
}
