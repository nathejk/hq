package dispatch

import (
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

// consumer projects the dispatch events onto the read model.
//
// Today that is only the dispatchable-section flag; tasks, tours and stops arrive in
// task 109 and land on the same consumer, because they are one aggregate.
type consumer struct {
	w cqrs.Writer
}

func (c *consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr("NATHEJK:*.dispatch.section.*.dispatchable"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch {
	case msg.Subject().Match("NATHEJK.*.dispatch.section.*.dispatchable"):
		var body SectionDispatchableSet
		if err := msg.Body(&body); err != nil {
			return err
		}
		slug := msg.Subject().Parts()[4]
		if body.Dispatchable {
			// OnConflict-do-nothing rather than an upsert of setAt: replay must not
			// move the timestamp, and nothing reads it as anything but "since when".
			return c.exec(goqu.Dialect("mysql").
				Insert("dispatchable_section").Rows(goqu.Record{
				"year":        c.year(msg),
				"sectionSlug": slug,
				"setAt":       c.at(msg),
			}).OnConflict(goqu.DoNothing()))
		}
		return c.exec(goqu.Dialect("mysql").
			Delete("dispatchable_section").
			Where(goqu.C("year").Eq(c.year(msg)), goqu.C("sectionSlug").Eq(slug)))
	}
	return nil
}

type toSQL interface {
	ToSQL() (string, []any, error)
}

func (c *consumer) exec(ds toSQL) error {
	sqlStr, _, err := ds.ToSQL()
	if err != nil {
		return err
	}
	return c.w.Consume(sqlStr)
}

// year comes from the subject, never from msg.Time(): deriving it from the message
// timestamp is wrong the moment history is replayed across a year boundary.
func (c *consumer) year(msg stream.Message) string {
	return msg.Subject().Parts()[1]
}

// at renders the event time as a MySQL DATETIME in UTC, matching every other table
// here — the driver reads DATETIME back as UTC, so storing local wall-clock would
// make every timestamp two hours wrong in summer.
func (c *consumer) at(msg stream.Message) string {
	return msg.Time().UTC().Format(time.DateTime)
}

// actor is the user the event was published by, empty until the platform
// authenticates anybody.
func (c *consumer) actor(msg stream.Message) types.UserID {
	var meta messages.Metadata
	if err := msg.Meta(&meta); err != nil {
		return ""
	}
	return meta.UserID
}
