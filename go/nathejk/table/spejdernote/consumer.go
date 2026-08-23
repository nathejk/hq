package spejdernote

import (
	"fmt"
	"log"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
)

type consumer struct {
	w cqrs.Writer
}

func (c *consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr("NATHEJK:*.spejder.*.commented"),
		subject.FromStr("NATHEJK:*.spejder.*.comment.updated"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch {
	// Five parts, so it is matched before the four-part pattern below. The reverse order has
	// bitten this codebase before — see the ordering note in spejderstatus/consumer.go — and
	// `NATHEJK.*.spejder.*.commented` would not in fact match `…comment.updated`, but keeping
	// specific-first is the habit that makes that irrelevant.
	case msg.Subject().Match("NATHEJK.*.spejder.*.comment.updated"):
		var body CommentUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.NoteID == "" {
			// Refused rather than written: a correction with no id cannot find its note, and an
			// UPDATE with an empty key would either hit nothing or, with a different WHERE, hit
			// everything. Loud, and not an error — a poison message must not wedge the replay.
			log.Printf("spejdernote: correction with no noteId for member %q", c.memberID(msg))
			return nil
		}
		// updatedAt moves, createdAt does not: the note was written when it was written, and a
		// trail that reordered itself when somebody fixed a typo would be worse than one with a
		// typo in it.
		return c.exec(goqu.Dialect("mysql").
			Update("spejdernote").
			Set(goqu.Record{
				"note":        body.Note,
				"actorUserId": string(c.actor(msg)),
				"updatedAt":   c.at(msg),
			}).
			Where(
				goqu.C("noteId").Eq(string(body.NoteID)),
				// Scoped to the member from the subject as well as the id. The command checks
				// this too (task 100); doing it here as well means a correction published by
				// anything else cannot reach another member's note either.
				goqu.C("memberId").Eq(c.memberID(msg)),
				goqu.C("year").Eq(c.year(msg)),
			))

	case msg.Subject().Match("NATHEJK.*.spejder.*.commented"):
		var body Commented
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.NoteID == "" {
			log.Printf("spejdernote: note with no noteId for member %q", c.memberID(msg))
			return nil
		}
		// Upsert, because replay re-delivers this on every boot. `note` and `actorUserId` are in
		// the update list so a replay reproduces the row exactly; `createdAt` is *not*, so the
		// original write time survives — and neither is `updatedAt`, because a correction may
		// already have moved it and replaying the original note must not undo that. Replay order
		// is therefore irrelevant, which is the only way an append-and-edit table can be rebuilt
		// safely.
		return c.upsert(goqu.Record{
			"noteId":      string(body.NoteID),
			"memberId":    c.memberID(msg),
			"year":        c.year(msg),
			"note":        body.Note,
			"actorUserId": string(c.actor(msg)),
			"createdAt":   c.at(msg),
			"updatedAt":   c.at(msg),
		}, "memberId", "year", "note", "actorUserId")

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
	}
	return nil
}

// upsert writes a note row, updating only the named columns if it already exists.
func (c *consumer) upsert(row goqu.Record, updates ...string) error {
	update := goqu.Record{}
	for _, col := range updates {
		update[col] = goqu.L(fmt.Sprintf("VALUES(%s)", col))
	}
	return c.exec(goqu.Dialect("mysql").
		Insert("spejdernote").Rows(row).
		OnConflict(goqu.DoUpdate("noteId", update)))
}

func (c *consumer) exec(ds interface {
	ToSQL() (string, []any, error)
}) error {
	sqlStr, _, err := ds.ToSQL()
	if err != nil {
		return err
	}
	return c.w.Consume(sqlStr)
}

// memberID comes from the subject rather than the body, so it cannot disagree with what the stream
// routed on.
func (c *consumer) memberID(msg stream.Message) string {
	return msg.Subject().Parts()[3]
}

// year comes from the subject too, never from msg.Time(): replay crosses year boundaries by
// definition, since that is how this table is built.
func (c *consumer) year(msg stream.Message) string {
	return msg.Subject().Parts()[1]
}

// actor is the user the event was published by, empty until the platform authenticates anybody
// (PRD 001 §6). Read from the metadata rather than the body's Actor for consistency with
// spejderstatus, which does the same — the metadata is what the publisher sets from the request.
func (c *consumer) actor(msg stream.Message) string {
	var meta struct {
		UserID string `json:"userId"`
	}
	if err := msg.Meta(&meta); err != nil {
		return ""
	}
	return meta.UserID
}

// at renders the event time as a MySQL DATETIME in UTC, matching every other projection: the driver
// reads DATETIME back as UTC, so the API emits a Z-offset timestamp and the browser converts to the
// crew's clock.
func (c *consumer) at(msg stream.Message) string {
	return msg.Time().UTC().Format(time.DateTime)
}
