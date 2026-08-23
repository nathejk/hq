// Package spejdernote is the prose trail about a scout: what was agreed with a guardian, what was
// said on the phone, what the next shift needs to know (PRD 008).
//
// It exists because none of that fits anywhere else. A status says `sheltered`, a placering says
// `Telt 4`, and neither answers the questions the crew member arriving at 04:00 will have — who
// was rung, what was agreed, who must not be called. The shelter's own work was, until this, on
// paper and in one person's head.
//
// # Why not on the case
//
// The obvious home was an sos comment, and it is the wrong one. The shelter deliberately has no
// case (PRD 007 made its whole write path case-free, because it may receive a scout nobody opened
// a case about), so a note that needed a case could not be written at the moment it matters.
// Notes are therefore facts about the *member*, and the case card reads them because it shows the
// member — nothing is copied onto a timeline, so there is one text and one place to correct it.
//
// # Events, not a form post
//
// Notes are published like everything else, on the scout's own subject, which buys two things
// without any work: the projection rebuilds from history on boot like its neighbours, and the live
// signal reaches the SPA on the `spejder` token that every member screen already depends on. A new
// entity token would have meant every client declaring a new dependency.
package spejdernote

import (
	"database/sql"
	"log"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/google/uuid"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/nathejk/shared-go/types"

	_ "embed"
)

// NoteID identifies one note.
//
// Minted by the server, as sos comment ids are, so a client cannot collide with an id it has not
// seen — and so a client cannot choose an id that already belongs to another member's note.
type NoteID string

func NewNoteID() NoteID {
	return NoteID("spejdernote-" + uuid.New().String())
}

// Note is one entry in a scout's trail.
//
// UpdatedAt equals CreatedAt until the note is corrected; the UI says "Rettet …" only when they
// differ. Keeping it non-nullable means every caller can render a timestamp without a branch.
type Note struct {
	NoteID    NoteID         `json:"noteId"`
	MemberID  types.MemberID `json:"memberId"`
	YearSlug  types.YearSlug `json:"year"`
	Note      string         `json:"note"`
	Actor     types.UserID   `json:"actorUserId"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// Edited reports whether the note has been corrected since it was written.
//
// A method rather than a field the client compares itself: the two timestamps are equal by
// construction for a fresh note, and every consumer would otherwise reimplement that comparison —
// including, eventually, one that used `!=` on a time.Time and got it subtly wrong.
func (n Note) Edited() bool { return n.UpdatedAt.After(n.CreatedAt) }

// Summary is what a list needs to show that notes exist without fetching them.
//
// The snippet is truncated by the caller that serves it (task 102), not here: how much fits is a
// question about a column on a screen, and the query should not encode one.
type Summary struct {
	Count      int       `json:"count"`
	LatestNote string    `json:"latestNote"`
	LatestAt   time.Time `json:"latestAt"`
}

type table struct {
	commander
	consumer
	querier
}

// New builds the projection.
//
// Publisher for the commander, writer for the consumer, *sql.DB for the querier — and the querier is
// handed to the commander, which dirty-checks corrections against it.
func New(p stream.Publisher, w cqrs.Writer, r *sql.DB) *table {
	q := querier{db: r, r: goqu.New("mysql", r)}
	t := &table{
		commander: commander{p: p, q: &q},
		consumer:  consumer{w: w},
		querier:   q,
	}
	if err := w.Consume(t.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	for _, stmt := range schemaMigrations {
		if err := w.Consume(stmt); err != nil {
			log.Printf("Error migrating spejdernote table %q", err)
		}
	}
	return t
}

// schemaMigrations brings an existing database up to the current shape.
//
// Empty today, and present anyway, for the reason spejderstatus learned the hard way: CREATE TABLE
// IF NOT EXISTS is a no-op wherever the table exists, so a column added to table.sql later is
// silently absent from every database that has already booted once. Entries must be idempotent —
// they run on every boot, against both the old shape and the new.
var schemaMigrations = []string{}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string { return tableSchema }

// Assert the consumer contract at compile time. The mux accepts anything shaped vaguely right, so
// a signature drifting out of step would surface as a projection that silently never runs — and an
// empty notes table looks exactly like a night nobody wrote anything down.
var _ cqrs.Consumer = (*table)(nil)
