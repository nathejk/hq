package shelter

import (
	"fmt"
	"log"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"

	"nathejk.dk/nathejk/table/spejderstatus"
)

type consumer struct {
	w cqrs.Writer
}

// Consumes the shelter's half of the member lifecycle.
//
// The event bodies are spejderstatus's, not redeclared here: the wire format has one
// definition, and a local copy would drift the first time a field was added. When
// spejderstatus is lifted to shared-go (task 083) this import moves with it, which is a
// one-line change.
//
// Notably absent: pickup.accepted and withdrawal.requested. A scout in a car or by a road is
// not in the shelter and has no placering, so this projection has nothing to say about them —
// the screen reads their status from spejderstatus.
func (c *consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr("NATHEJK:*.spejder.*.shelter.accepted"),
		subject.FromStr("NATHEJK:*.spejder.*.shelter.placed"),
		subject.FromStr("NATHEJK:*.spejder.*.handover.completed"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch {
	case msg.Subject().Match("NATHEJK.*.spejder.*.shelter.accepted"):
		var body spejderstatus.ShelterAccepted
		if err := msg.Body(&body); err != nil {
			return err
		}
		row := goqu.Record{
			"id":         c.memberID(msg),
			"year":       c.year(msg),
			"teamId":     string(body.TeamID),
			"placement":  body.Placement,
			"acceptedAt": c.at(msg),
			"placedAt":   c.placedAt(msg, body.Placement),
		}
		// acceptedAt is not in the update list: it records when the shelter took charge,
		// and a replayed or repeated acceptance must not move that timestamp forward.
		//
		// The placering *is* updatable, but only because an acceptance carrying one is a
		// statement about where the scout is. An acceptance carrying none is not a
		// statement that they are nowhere — so an empty placement must not wipe a
		// placering set afterwards by shelter.placed. Hence the conditional below rather
		// than a plain VALUES(placement).
		return c.upsert(msg, row, "teamId", "placement", "placedAt")

	case msg.Subject().Match("NATHEJK.*.spejder.*.shelter.placed"):
		var body spejderstatus.ShelterPlaced
		if err := msg.Body(&body); err != nil {
			return err
		}
		// Upsert rather than update, because this projection must tolerate an event for a
		// member it never saw arrive: the acceptance may be missing from a truncated
		// history, or a replay may deliver the two out of order. A scout with a placering
		// and no recorded arrival is a better read model than no scout at all — the crew
		// can find them, which is the entire purpose.
		row := goqu.Record{
			"id":         c.memberID(msg),
			"year":       c.year(msg),
			"teamId":     string(body.TeamID),
			"placement":  body.Placement,
			"acceptedAt": c.at(msg),
			"placedAt":   c.at(msg),
		}
		return c.upsert(msg, row, "teamId", "placement", "placedAt")

	// Somebody else has taken charge of the scout, so they are no longer anywhere of ours
	// and the bed is free. Deleted rather than flagged: see table.sql for why, and note that
	// their history survives in spejderstatuslog either way.
	case msg.Subject().Match("NATHEJK.*.spejder.*.handover.completed"):
		return c.exec(goqu.Dialect("mysql").
			Delete("shelter").
			Where(
				goqu.C("year").Eq(c.year(msg)),
				goqu.C("id").Eq(c.memberID(msg)),
			))

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
	}
	return nil
}

// placedAt is the event time when the acceptance named a placering, and NULL when it did not.
//
// The two cases are different facts and the screen shows them differently: a scout accepted
// with a tent is bedded down, a scout accepted without one is standing in the doorway and is
// the crew's next job. Writing the acceptance time into placedAt regardless would erase that
// distinction and make every arrival look dealt with.
func (c *consumer) placedAt(msg stream.Message, placement string) any {
	if placement == "" {
		return nil
	}
	return c.at(msg)
}

// upsert writes a row, updating only the named columns if it already exists.
//
// The explicit column list is the same discipline spejderstatus applies: an upsert that
// overwrote everything would make replay order-dependent, so each caller says exactly which
// facts its event is entitled to change.
//
// `placement` and `placedAt` are updated conditionally rather than unconditionally, which is
// the one subtlety here. An acceptance carrying no placering must not blank a placering that
// shelter.placed set later, because replay delivers the acceptance again on every boot and
// the two events are not ordered with respect to each other in that regard. IF(VALUES(x)=”,
// x, VALUES(x)) keeps the incoming value when there is one and the stored value when there is
// not — the same conditional-update trick the older projections use for optional fields.
func (c *consumer) upsert(msg stream.Message, row goqu.Record, updates ...string) error {
	update := goqu.Record{}
	for _, col := range updates {
		switch col {
		case "placement":
			update[col] = goqu.L("IF(VALUES(placement) = '', placement, VALUES(placement))")
		case "placedAt":
			update[col] = goqu.L("IF(VALUES(placement) = '', placedAt, VALUES(placedAt))")
		default:
			update[col] = goqu.L(fmt.Sprintf("VALUES(%s)", col))
		}
	}
	return c.exec(goqu.Dialect("mysql").
		Insert("shelter").Rows(row).
		OnConflict(goqu.DoUpdate("year,id", update)))
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

// memberID comes from the subject rather than the body, so it cannot disagree with what the
// stream routed on.
func (c *consumer) memberID(msg stream.Message) string {
	return msg.Subject().Parts()[3]
}

// year comes from the subject too, never from msg.Time(): replay across a year boundary is
// how this table is built, and msg.Time().Year() would file old events under this year.
func (c *consumer) year(msg stream.Message) string {
	return msg.Subject().Parts()[1]
}

// at renders the event time as a MySQL DATETIME in UTC, matching spejderstatus and sos. The
// driver reads DATETIME back as UTC, so the API emits a Z-offset timestamp and the browser
// converts to the crew's clock; local wall-clock would be an hour out all summer.
func (c *consumer) at(msg stream.Message) string {
	return msg.Time().UTC().Format(time.DateTime)
}
