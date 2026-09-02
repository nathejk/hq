package kort

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
)

// ErrRecordNotFound is returned when a kort id names nothing.
var ErrRecordNotFound = errors.New("record not found")

type Queries interface {
	// Maps lists a year's sheets in handout order.
	Maps(context.Context, types.YearSlug) ([]Kort, error)
	// GetByID reads one sheet, which the commands need in order to dirty-check (task 124).
	GetByID(context.Context, types.YearSlug, KortID) (*Kort, error)
	// Sets lists a year's sets in their own order, without their sheets.
	Sets(context.Context, types.YearSlug) ([]Kortsaet, error)
	// GetSetByID reads one set, for the set commands' dirty-check.
	GetSetByID(context.Context, types.YearSlug, KortsaetID) (*Kortsaet, error)
	// CountMapsInSet reports how many sheets a set holds, which is what makes deleting a
	// non-empty set refusable.
	CountMapsInSet(context.Context, types.YearSlug, KortsaetID) (int, error)
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

const selectKort = `SELECT id, year, version, kortsaetId, name, format, note, sortOrder,
	checkpointIds, extents
	FROM kort`

// Maps lists a year's sheets, grouped by set and in handout order within it.
//
// One query for the whole year with no pagination and no filter beyond the year, because there
// are on the order of fifteen rows: `GET /api/kort` returns all of them, and both the settings
// modal and the hej-app work from that single response (PRD 010 §8). A per-set query would be a
// second code path serving no caller.
//
// `id` is the tiebreak after sortOrder so that two sheets sharing a sort order — easy to produce
// mid-reorder — come back in a stable order rather than whatever the storage engine feels like.
// Two consecutive loads disagreeing about the order of the maps would look like a bug in the
// drag-and-drop.
func (q *querier) Maps(ctx context.Context, year types.YearSlug) ([]Kort, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := selectKort + `
		WHERE (year = ? OR ? = '')
		ORDER BY kortsaetId ASC, sortOrder ASC, id ASC`

	rows, err := q.db.QueryContext(ctx, query, string(year), string(year))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	maps := []Kort{}
	for rows.Next() {
		k, err := scanKort(rows)
		if err != nil {
			return nil, err
		}
		maps = append(maps, *k)
	}
	return maps, rows.Err()
}

// GetByID reads one sheet.
//
// Year-scoped as well as keyed by id: ids are unique, but a request carrying the wrong year
// should not silently reach another year's map.
func (q *querier) GetByID(ctx context.Context, year types.YearSlug, id KortID) (*Kort, error) {
	if id == "" {
		return nil, ErrRecordNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row := q.db.QueryRowContext(ctx,
		selectKort+` WHERE id = ? AND (year = ? OR ? = '')`,
		string(id), string(year), string(year))

	k, err := scanKort(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecordNotFound
	}
	return k, err
}

type scanner interface {
	Scan(dest ...any) error
}

const selectKortsaet = `SELECT id, year, version, name, teamType, sortOrder
	FROM kortsaet`

// Sets lists a year's sets in their own order.
//
// Without their sheets: nesting is done by the read path that serves them together (task 125),
// from one query of each table rather than a query per set.
func (q *querier) Sets(ctx context.Context, year types.YearSlug) ([]Kortsaet, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := q.db.QueryContext(ctx,
		selectKortsaet+`
		WHERE (year = ? OR ? = '')
		ORDER BY sortOrder ASC, id ASC`,
		string(year), string(year))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sets := []Kortsaet{}
	for rows.Next() {
		s, err := scanKortsaet(rows)
		if err != nil {
			return nil, err
		}
		sets = append(sets, *s)
	}
	return sets, rows.Err()
}

// GetSetByID reads one set.
func (q *querier) GetSetByID(ctx context.Context, year types.YearSlug, id KortsaetID) (*Kortsaet, error) {
	if id == "" {
		return nil, ErrRecordNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row := q.db.QueryRowContext(ctx,
		selectKortsaet+` WHERE id = ? AND (year = ? OR ? = '')`,
		string(id), string(year), string(year))

	s, err := scanKortsaet(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecordNotFound
	}
	return s, err
}

// CountMapsInSet reports how many sheets a set holds.
//
// A count rather than a list, because the only caller asks a yes/no question: may this set be
// deleted? Reading the sheets to answer it would pull every checkpoint list off disk to look at
// len().
func (q *querier) CountMapsInSet(ctx context.Context, year types.YearSlug, id KortsaetID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var count int
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM kort WHERE kortsaetId = ? AND (year = ? OR ? = '')`,
		string(id), string(year), string(year)).Scan(&count)
	return count, err
}

// scanKortsaet reads one set row.
//
// teamType is read through sql.NullString and left nil when absent, never coerced to "": the
// difference between "this set is for patruljer" and "this set is the general one" is the whole
// point of the column.
func scanKortsaet(row scanner) (*Kortsaet, error) {
	var s Kortsaet
	var teamType sql.NullString
	if err := row.Scan(&s.KortsaetID, &s.YearSlug, &s.Version, &s.Name, &teamType, &s.SortOrder); err != nil {
		return nil, err
	}
	if teamType.Valid && teamType.String != "" {
		t := types.TeamType(teamType.String)
		s.TeamType = &t
	}
	return &s, nil
}

// scanKort reads one row, decoding the two JSON columns.
//
// A malformed JSON column yields an empty list rather than an error, and that is deliberate: a
// map whose extents cannot be parsed is still a map with a name and a checkpoint list, and
// failing the whole read would take the entire settings screen down over one bad row. The
// alternative — refusing to load — would also make the row impossible to fix from the UI.
func scanKort(row scanner) (*Kort, error) {
	var k Kort
	var checkpointIDs, extents sql.NullString
	err := row.Scan(
		&k.KortID,
		&k.YearSlug,
		&k.Version,
		&k.KortsaetID,
		&k.Name,
		&k.Format,
		&k.Note,
		&k.SortOrder,
		&checkpointIDs,
		&extents,
	)
	if err != nil {
		return nil, err
	}

	k.CheckpointIDs = []types.CheckpointID{}
	if checkpointIDs.Valid && checkpointIDs.String != "" {
		_ = json.Unmarshal([]byte(checkpointIDs.String), &k.CheckpointIDs)
		if k.CheckpointIDs == nil {
			k.CheckpointIDs = []types.CheckpointID{}
		}
	}

	k.Extents = []Extent{}
	if extents.Valid && extents.String != "" {
		_ = json.Unmarshal([]byte(extents.String), &k.Extents)
		if k.Extents == nil {
			k.Extents = []Extent{}
		}
	}

	return &k, nil
}
