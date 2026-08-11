package sos

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/types"
)

// Queries is what the application may read. Declared as an interface so cmd/api
// depends on the queries rather than on the table implementation.
type Queries interface {
	GetAll(context.Context, Filter) ([]*Sos, error)
	GetByID(context.Context, types.SosID) (*Sos, error)
	GetByTeam(context.Context, types.YearSlug, types.TeamID) ([]*Sos, error)
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

// GetAll lists cases for the list view: no timeline, no teams, ordered so the
// case that just moved is first.
//
// Both open and closed cases come back in one query and the caller groups them.
// Two queries would be two round trips to render one screen, and the volume here
// is tens of rows per event, not thousands.
func (q *querier) GetAll(ctx context.Context, f Filter) ([]*Sos, error) {
	query := `SELECT id, year, headline, description, status, severity,
		assigneeSectionSlug, createdAt, createdBy, lastActivityAt
		FROM sos WHERE (year = ? OR ? = '')`
	args := []any{f.YearSlug, f.YearSlug}
	if !f.IncludeDeleted {
		query += ` AND deletedAt IS NULL`
	}
	if f.Status != "" {
		query += ` AND status = ?`
		args = append(args, string(f.Status))
	}
	query += ` ORDER BY lastActivityAt DESC, createdAt DESC`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cases := []*Sos{}
	for rows.Next() {
		s, err := scanSos(rows)
		if err != nil {
			return nil, err
		}
		cases = append(cases, s)
	}
	return cases, rows.Err()
}

// GetByID returns one case with its full timeline and associated teams.
//
// Three queries rather than a join: a join would multiply the case row by
// timeline entries and teams, and the timeline is unbounded in a way the case is
// not. Cheap either way — this is one case, opened by one operator.
func (q *querier) GetByID(ctx context.Context, id types.SosID) (*Sos, error) {
	if id == "" {
		return nil, tables.ErrRecordNotFound
	}
	query := `SELECT id, year, headline, description, status, severity,
		assigneeSectionSlug, createdAt, createdBy, lastActivityAt
		FROM sos WHERE id = ? AND deletedAt IS NULL`

	s, err := scanSos(q.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// A soft-deleted case is not found, not forbidden: the operator who has
			// it open should be told it is gone, and the id should stop resolving.
			return nil, tables.ErrRecordNotFound
		}
		return nil, err
	}

	if s.Timeline, err = q.timeline(ctx, id); err != nil {
		return nil, err
	}
	if s.Teams, err = q.teams(ctx, id); err != nil {
		return nil, err
	}
	return s, nil
}

// GetByTeam returns the cases a patrol is involved in, for the patrol's own page.
func (q *querier) GetByTeam(ctx context.Context, year types.YearSlug, teamID types.TeamID) ([]*Sos, error) {
	if teamID == "" {
		return []*Sos{}, nil
	}
	query := `SELECT s.id, s.year, s.headline, s.description, s.status, s.severity,
		s.assigneeSectionSlug, s.createdAt, s.createdBy, s.lastActivityAt
		FROM sos s JOIN sos_team t ON t.sosId = s.id
		WHERE t.teamId = ? AND (s.year = ? OR ? = '') AND s.deletedAt IS NULL
		ORDER BY s.createdAt DESC`

	rows, err := q.db.QueryContext(ctx, query, teamID, year, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cases := []*Sos{}
	for rows.Next() {
		s, err := scanSos(rows)
		if err != nil {
			return nil, err
		}
		cases = append(cases, s)
	}
	return cases, rows.Err()
}

func (q *querier) timeline(ctx context.Context, id types.SosID) ([]Activity, error) {
	query := `SELECT seq, sosId, type, actorUserId, activityId, refActivityId, value, createdAt
		FROM sos_activity WHERE sosId = ? ORDER BY seq ASC`
	rows, err := q.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timeline := []Activity{}
	for rows.Next() {
		var a Activity
		var value sql.NullString
		if err := rows.Scan(&a.Seq, &a.SosID, &a.Type, &a.Actor, &a.ActivityID,
			&a.RefActivityID, &value, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.Value = value.String
		timeline = append(timeline, a)
	}
	return timeline, rows.Err()
}

func (q *querier) teams(ctx context.Context, id types.SosID) ([]Team, error) {
	query := `SELECT teamId, createdAt FROM sos_team WHERE sosId = ? ORDER BY createdAt ASC`
	rows, err := q.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := []Team{}
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.TeamID, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

// scanner is what *sql.Row and *sql.Rows have in common, so one scan function
// serves both the list and the single-case read and they cannot drift apart.
type scanner interface {
	Scan(dest ...any) error
}

func scanSos(s scanner) (*Sos, error) {
	var r Sos
	var description sql.NullString
	var createdAt, lastActivityAt time.Time
	if err := s.Scan(&r.ID, &r.YearSlug, &r.Headline, &description, &r.Status,
		&r.Severity, &r.AssigneeSectionSlug, &createdAt, &r.CreatedBy,
		&lastActivityAt); err != nil {
		return nil, err
	}
	r.Description = description.String
	r.CreatedAt = createdAt
	r.LastActivityAt = lastActivityAt
	return &r, nil
}
