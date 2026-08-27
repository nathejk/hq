package dispatch

import (
	"context"
	"database/sql"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
)

// Queries is what the application may read. An interface so cmd/api depends on the
// queries rather than on the table implementation.
type Queries interface {
	DispatchableSections(context.Context, types.YearSlug) ([]types.Slug, error)
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

// DispatchableSections lists the organisation sections that are dispatch units for
// the given year.
//
// Empty by default and opted into per section, so a fresh year offers no unit until
// somebody says which subsections hold a car. An empty list is not an error: it means
// nobody has decided yet, and the desk can still write tasks down — which is the
// point of PRD 009's "no unit on duty" edge case.
func (q *querier) DispatchableSections(ctx context.Context, year types.YearSlug) ([]types.Slug, error) {
	query := `SELECT sectionSlug FROM dispatchable_section
		WHERE (year = ? OR ? = '') ORDER BY sectionSlug ASC`
	rows, err := q.db.QueryContext(ctx, query, year, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slugs := []types.Slug{}
	for rows.Next() {
		var s types.Slug
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		slugs = append(slugs, s)
	}
	return slugs, rows.Err()
}
