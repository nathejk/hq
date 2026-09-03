package track

import (
	"context"
	"database/sql"

	"github.com/doug-martin/goqu/v9"
)

type Queries interface {
	// Presence reports every person who has ever reported a position in a year, with when they
	// last did. This is what the position glyph on every people list is drawn from.
	Presence(context.Context, string) ([]Latest, error)

	// Points reads one person's raw history, ordered by time. Segmenting and reducing it happens
	// above this (tasks 145, 146) — the storage layer returns what arrived.
	Points(context.Context, Filter) ([]Point, error)

	// LatestFor reads one person's last known position, or nil.
	LatestFor(context.Context, string) (*Latest, error)
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

// Presence returns one row per person, and deliberately nothing else.
//
// No join to a name, and no filter beyond the year. Both are what make this endpoint cheap enough to
// fetch on nearly every page: a `personId` is either a memberID or a crewmemberID, both opaque and
// non-colliding, and every people-list row in HQ already carries the id it needs to look up. So the
// client asks "is my row's id in here?" and no identity mapping exists anywhere.
//
// Reads `track_latest` only. Answering this from the history table would mean a group-by over
// millions of rows to produce a few hundred, on every page load, which is the whole reason the two
// tables are separate.
func (q *querier) Presence(ctx context.Context, year string) ([]Latest, error) {
	sqlStr, _, err := goqu.Dialect("mysql").
		From("track_latest").
		Select("personId", "personType", "year", "latitude", "longitude", "accuracy", "ts").
		Where(goqu.C("year").Eq(year)).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Latest
	for rows.Next() {
		var l Latest
		if err := rows.Scan(&l.PersonID, &l.PersonType, &l.Year, &l.Lat, &l.Lng, &l.Accuracy, &l.Ts); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// Points reads one person's history in time order.
//
// Ordered by `ts` rather than by arrival, because the two differ: batches carry backlogs, retries
// repeat, and replay re-reads everything. The primary key `(personId, ts)` serves this query
// directly, so the ordering costs nothing and no separate index is needed.
//
// Year is not a parameter. The time bounds already select an event far more precisely than a year
// slug does, and a person's points are keyed by person and time alone — adding year to the where
// clause would only skip the index this query exists to use.
func (q *querier) Points(ctx context.Context, f Filter) ([]Point, error) {
	if f.PersonID == "" {
		return nil, nil
	}

	ds := goqu.Dialect("mysql").
		From("track_point").
		Select("ts", "latitude", "longitude", "accuracy").
		Where(goqu.C("personId").Eq(f.PersonID)).
		Order(goqu.C("ts").Asc())
	if f.FromTs > 0 {
		ds = ds.Where(goqu.C("ts").Gte(f.FromTs))
	}
	if f.ToTs > 0 {
		ds = ds.Where(goqu.C("ts").Lte(f.ToTs))
	}

	sqlStr, _, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Point
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.Ts, &p.Lat, &p.Lng, &p.Accuracy); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LatestFor reads one person's last known position.
//
// Returns (nil, nil) for a person who has never reported — which is not an error and must not be
// reported as one. "Never reported" is a normal and meaningful state: it is what makes the absence
// of a glyph informative rather than a gap in the data.
func (q *querier) LatestFor(ctx context.Context, personID string) (*Latest, error) {
	if personID == "" {
		return nil, nil
	}

	sqlStr, _, err := goqu.Dialect("mysql").
		From("track_latest").
		Select("personId", "personType", "year", "latitude", "longitude", "accuracy", "ts").
		Where(goqu.C("personId").Eq(personID)).
		Limit(1).
		ToSQL()
	if err != nil {
		return nil, err
	}

	var l Latest
	err = q.db.QueryRowContext(ctx, sqlStr).
		Scan(&l.PersonID, &l.PersonType, &l.Year, &l.Lat, &l.Lng, &l.Accuracy, &l.Ts)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}
