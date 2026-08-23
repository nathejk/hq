package spejdernote

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/nathejk/shared-go/types"
)

// ErrRecordNotFound is returned when a note id names nothing — a wrong id, or one belonging to
// another member's note.
var ErrRecordNotFound = errors.New("record not found")

type Queries interface {
	GetByMember(context.Context, types.YearSlug, types.MemberID) ([]Note, error)
	GetByID(context.Context, types.YearSlug, NoteID) (*Note, error)
	SummaryByMembers(context.Context, types.YearSlug, []types.MemberID) (map[types.MemberID]Summary, error)
}

type querier struct {
	db *sql.DB
	r  *goqu.Database
}

const selectNote = `SELECT noteId, memberId, year, note, actorUserId, createdAt, updatedAt
	FROM spejdernote`

// GetByMember reads one scout's trail, oldest first.
//
// Oldest first because a trail is a story: "rang the mother, agreed 06.00, she called back to say
// 05.30" only makes sense in the order it happened. The list on screen shows the newest note as a
// snippet, which is a different question answered by SummaryByMembers.
//
// noteId is the tiebreak, not decoration: two notes written in the same second — one crew member
// pasting, or two writing at once — would otherwise come back in whatever order the storage engine
// felt like, and two consecutive loads could disagree about the order of somebody's trail.
func (q *querier) GetByMember(ctx context.Context, year types.YearSlug, id types.MemberID) ([]Note, error) {
	if id == "" {
		return []Note{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := selectNote + `
		WHERE year = ? AND memberId = ?
		ORDER BY createdAt ASC, noteId ASC`

	rows, err := q.db.QueryContext(ctx, query, string(year), string(id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

// GetByID reads one note, for the command to dirty-check a correction against.
//
// It returns the note whatever member it belongs to; deciding whether that is the *right* member is
// the command's job, and it needs the row to make that decision. A query that filtered by member
// would make "no such note" and "not your note" indistinguishable, and those deserve different
// answers.
func (q *querier) GetByID(ctx context.Context, year types.YearSlug, id NoteID) (*Note, error) {
	if id == "" {
		return nil, ErrRecordNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := q.db.QueryContext(ctx, selectNote+` WHERE year = ? AND noteId = ?`, string(year), string(id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, ErrRecordNotFound
	}
	note, err := scanNote(rows)
	if err != nil {
		return nil, err
	}
	return &note, nil
}

// SummaryByMembers reports, per member, how many notes exist and the most recent one.
//
// This is what makes notes discoverable without opening anything: a list can show a count and a
// snippet, so nobody has to click through forty scouts to find the one with instructions.
//
// One query with an IN clause, not a lookup per member. The shelter screen asks about every scout
// it lists and is kept open all night, so a per-row query would be dozens of round trips on every
// revalidation — and revalidation happens on every event.
//
// **The reduction is done in Go, not in SQL.** "Latest row per group" in MySQL means either a
// correlated subquery or the SUBSTRING_INDEX(MAX(CONCAT(…))) trick, and I wrote the latter before
// deleting it: it is unreadable, it silently depends on the timestamp format sorting
// lexicographically, and it would be somebody's problem at 3am. Fetching the rows and folding them
// here is a few dozen rows for a screen that lists a few dozen scouts, and it is obvious.
func (q *querier) SummaryByMembers(ctx context.Context, year types.YearSlug, ids []types.MemberID) (map[types.MemberID]Summary, error) {
	out := map[types.MemberID]Summary{}
	if len(ids) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	args := make([]any, 0, len(ids)+1)
	args = append(args, string(year))
	placeholders := make([]string, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, string(id))
	}

	// Ordered oldest first so the fold below can simply keep overwriting: the last row seen for a
	// member is the newest. noteId breaks a shared second, as in GetByMember, so the "latest" note
	// is the same one on every load rather than whichever the engine returned last.
	query := `SELECT memberId, note, createdAt
		FROM spejdernote
		WHERE year = ? AND memberId IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY createdAt ASC, noteId ASC`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var memberID types.MemberID
		var note string
		var createdAt time.Time
		if err := rows.Scan(&memberID, &note, &createdAt); err != nil {
			return nil, err
		}
		s := out[memberID]
		s.Count++
		s.LatestNote = note
		s.LatestAt = createdAt
		out[memberID] = s
	}
	return out, rows.Err()
}

func scanNote(rows *sql.Rows) (Note, error) {
	var n Note
	var createdAt, updatedAt time.Time
	if err := rows.Scan(&n.NoteID, &n.MemberID, &n.YearSlug, &n.Note, &n.Actor, &createdAt, &updatedAt); err != nil {
		return n, err
	}
	n.CreatedAt = createdAt
	n.UpdatedAt = updatedAt
	return n, nil
}
