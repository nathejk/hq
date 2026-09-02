// Package kort is the maps we print and hand out during the event: what each sheet is called,
// which set it belongs to, what ground it shows, and which checkpoints are drawn on it (PRD 010).
//
// # Why a map is an entity
//
// The tempting shortcut is to say the reveal unit is the checkgroup, which HQ already models, and
// skip this package. It does not hold. A skitse shows a subset of one group; a double-sided A3
// spans two. The thing a patrol holds is a *sheet*, and the sheet is what a post hands over, so
// the sheet is what gets modelled. The checkgroup then stays what it is — a timing and staffing
// concept — instead of being quietly overloaded with a printing concern.
//
// Both are reveal units, and they do not nest: every checkpoint in a checkgroup is revealed at
// once, *and* the checkpoints on a sheet are revealed when its QR is linked to a team. This
// package makes both expressible and implements neither — the rule lives in the app that asks
// the question (PRD 010 §8).
//
// # What this package deliberately cannot answer
//
// "Which maps contain checkpoint X?" There is no index for it and no query for it, because
// checkpoints are a JSON array on the row rather than a join table. That is a decision, not an
// omission (see table.sql), and the one place it costs anything is the delete cascade.
//
// "Where was this sheet handed out?" Nothing records that, and nothing should: it varies, it is
// not reliably known, and it is not needed — the sheet a team holds is established when its QR
// code is linked to the team, not by inferring it from where they had been.
package kort

import (
	"database/sql"
	"log"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/google/uuid"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/nathejk/shared-go/types"

	_ "embed"
)

// KortID identifies one printed sheet.
//
// Minted by the server, like spejdernote's NoteID: a client cannot collide with an id it has not
// seen.
type KortID string

func NewKortID() KortID {
	return KortID("kort-" + uuid.New().String())
}

// KortsaetID identifies a set of sheets (task 122).
//
// Declared here rather than in the set's own file so that Created can name it without the two
// halves of this package having to be written in one go.
type KortsaetID string

func NewKortsaetID() KortsaetID {
	return KortsaetID("kortsaet-" + uuid.New().String())
}

// Kort is one printed sheet.
//
// CheckpointIDs and Extents are always non-nil after a read, so every caller — and the JSON
// encoder serving the hej-app — sees `[]` rather than `null` for an empty one.
type Kort struct {
	KortID     KortID         `json:"id"`
	KortsaetID KortsaetID     `json:"kortsaetId"`
	YearSlug   types.YearSlug `json:"year"`
	Version    uint64         `json:"version"`

	Name      string `json:"name"`
	Format    Format `json:"format"`
	Note      string `json:"note"`
	SortOrder int    `json:"sortOrder"`

	// CheckpointIDs is what the hej-app is really here for: the checkpoints drawn on this sheet,
	// and therefore what may be revealed when the sheet is known to be in a team's hands.
	CheckpointIDs []types.CheckpointID `json:"checkpointIds"`

	// Extents is the ground the sheet shows — zero rectangles for a skitse, one for a normal
	// sheet, two for a double-sided one. Front and back are not distinguished; they are simply
	// two areas.
	Extents []Extent `json:"extents"`
}

// HasCheckpoint reports whether this sheet shows the given checkpoint.
//
// A method on the row rather than a query, which is the whole shape of this package: the
// question is only ever asked about maps already in hand, and answering it in SQL would need
// the index the JSON column deliberately does not have.
func (k Kort) HasCheckpoint(id types.CheckpointID) bool {
	for _, cp := range k.CheckpointIDs {
		if cp == id {
			return true
		}
	}
	return false
}

// ContainsAll reports whether every one of the given checkpoints is on this sheet.
//
// This is the primitive behind the split-checkgroup warning (task 133), and it is deliberately
// existential rather than partitioning: the warning fires when *no single* map contains a whole
// checkgroup, so two overlapping sheets that both contain it are fine. Overlap between adjacent
// sheets is designed in, so a test that treated it as an error would fire constantly and be
// ignored.
//
// An empty list is contained by every sheet, which is the right answer for a checkgroup with no
// checkpoints yet: it is not a coverage failure, it is an unfinished course.
func (k Kort) ContainsAll(ids []types.CheckpointID) bool {
	for _, id := range ids {
		if !k.HasCheckpoint(id) {
			return false
		}
	}
	return true
}

type table struct {
	consumer
	querier
}

// New builds the projection.
//
// The publisher argument is accepted now and used by the commander in task 124, so registering
// this in main.go does not have to change shape twice.
func New(p stream.Publisher, w cqrs.Writer, r *sql.DB) *table {
	q := querier{db: r, r: goqu.New("mysql", r)}
	t := &table{
		consumer: consumer{w: w},
		querier:  q,
	}
	if err := w.Consume(t.CreateTableSql()); err != nil {
		log.Printf("Error creating table %q", err)
	}
	for _, stmt := range schemaMigrations {
		if err := w.Consume(stmt); err != nil {
			log.Printf("Error migrating kort tables %q", err)
		}
	}
	return t
}

// schemaMigrations brings an existing database up to the current shape.
//
// Empty today and present from the start, because CREATE TABLE IF NOT EXISTS is a no-op wherever
// the table already exists: a column added to table.sql later is silently absent from every
// database that has already booted once (`.rules`, and the way spejderstatus learned it).
//
// This package will need it sooner than most — the set table arrives in task 122 and the columns
// it references are already declared here — so entries must be idempotent: they run on every
// boot, against both the old shape and the new.
var schemaMigrations = []string{}

//go:embed table.sql
var tableSchema string

func (t *table) CreateTableSql() string { return tableSchema }

// Assert the consumer contract at compile time. The mux accepts anything shaped vaguely right,
// so a signature drifting out of step would surface as a projection that silently never runs —
// and an empty kort table looks exactly like a year nobody has drawn the maps for yet.
var _ cqrs.Consumer = (*table)(nil)
