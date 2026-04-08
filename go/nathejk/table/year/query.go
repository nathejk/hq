package year

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/nathejk/shared-go/types"
	tables "nathejk.dk/nathejk/table"
)

type Queries interface {
	GetByID(context.Context, types.YearSlug) (*Year, error)
	GetAll(context.Context, Filter) ([]Year, error)
}

type querier struct {
	db *sql.DB
}

func (q *querier) GetByID(ctx context.Context, slug types.YearSlug) (*Year, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT headline, description, cityDeparture, cityDestination, dateStart, dateEnd FROM years WHERE slug = ?`

	y := Year{Slug: slug}
	err := q.db.QueryRow(query, slug).Scan(&y.Headline, &y.Description, &y.CityDeparture, &y.CityDestination, &y.DateStart, &y.DateEnd)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, tables.ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &y, nil
}

func (q *querier) GetAll(ctx context.Context, filter Filter) ([]Year, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT slug, headline, description, cityDeparture, cityDestination, dateStart, dateEnd FROM years ORDER BY dateStart DESC`
	args := []any{}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	years := []Year{}
	for rows.Next() {
		var y Year
		err := rows.Scan(&y.Slug, &y.Headline, &y.Description, &y.CityDeparture, &y.CityDestination, &y.DateStart, &y.DateEnd)
		if err != nil {
			return nil, err
		}
		years = append(years, y)
	}
	return years, nil
}
