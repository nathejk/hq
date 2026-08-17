package sos

import (
	"encoding/json"
	"log"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

// consumer projects the SOS events onto three tables: the case itself, the
// patrols associated with it, and its activity timeline.
//
// Two properties are worth stating because everything else follows from them:
//
//   - **Every event appends to the timeline.** The timeline is not a derived view
//     of the case row, it is the case's history, and it is the only part an
//     operator can use for a shift handover. The row is the current state; the
//     timeline is what happened.
//   - **Replay is idempotent.** Case-row writes are upserts, and each timeline
//     entry is keyed by the stream sequence of the event that produced it, so
//     replaying the whole history rebuilds exactly the same rows rather than
//     duplicating the log.
type consumer struct {
	w cqrs.Writer
}

func (c *consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr("NATHEJK:*.sos.*.created"),
		subject.FromStr("NATHEJK:*.sos.*.headline.updated"),
		subject.FromStr("NATHEJK:*.sos.*.description.updated"),
		subject.FromStr("NATHEJK:*.sos.*.commented"),
		subject.FromStr("NATHEJK:*.sos.*.comment.updated"),
		subject.FromStr("NATHEJK:*.sos.*.severity.specified"),
		subject.FromStr("NATHEJK:*.sos.*.assigned"),
		subject.FromStr("NATHEJK:*.sos.*.closed"),
		subject.FromStr("NATHEJK:*.sos.*.reopened"),
		subject.FromStr("NATHEJK:*.sos.*.deleted"),
		subject.FromStr("NATHEJK:*.sos.*.team.associated"),
		subject.FromStr("NATHEJK:*.sos.*.team.disassociated"),

		// The member lifecycle summaries (PRD 006). One per operation; the per-member
		// events go to the spejderstatus projection on the member's own subject.
		subject.FromStr("NATHEJK:*.sos.*.member.status.changed"),
		subject.FromStr("NATHEJK:*.sos.*.member.moved"),
		subject.FromStr("NATHEJK:*.sos.*.team.collected"),

		subject.FromStr("NATHEJK:*.sos.section.*.assignable"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch {
	// Checked before the case-event patterns: the section subject also has "sos"
	// in third position, and NATHEJK.*.sos.*.assigned would otherwise be a
	// candidate match for a five-part section subject.
	case msg.Subject().Match("NATHEJK.*.sos.section.*.assignable"):
		var body SectionAssignableSet
		if err := msg.Body(&body); err != nil {
			return err
		}
		slug := msg.Subject().Parts()[4]
		if body.Assignable {
			insert := goqu.Record{
				"year":        c.year(msg),
				"sectionSlug": slug,
				"setAt":       c.at(msg),
			}
			return c.exec(goqu.Dialect("mysql").
				Insert("sos_assignable_section").Rows(insert).
				OnConflict(goqu.DoNothing()))
		}
		return c.exec(goqu.Dialect("mysql").
			Delete("sos_assignable_section").
			Where(goqu.C("year").Eq(c.year(msg)), goqu.C("sectionSlug").Eq(slug)))

	case msg.Subject().Match("NATHEJK.*.sos.*.created"):
		var body Created
		if err := msg.Body(&body); err != nil {
			return err
		}
		insert := goqu.Record{
			"id":             c.sosID(msg),
			"year":           c.year(msg),
			"headline":       body.Headline,
			"description":    body.Description,
			"status":         string(StatusOpen),
			"createdAt":      c.at(msg),
			"createdBy":      string(c.actor(msg)),
			"lastActivityAt": c.at(msg),
		}
		update := goqu.Record{
			"headline":    goqu.L("VALUES(headline)"),
			"description": goqu.L("VALUES(description)"),
		}
		if err := c.exec(goqu.Dialect("mysql").
			Insert("sos").Rows(insert).
			OnConflict(goqu.DoUpdate("id", update))); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivityCreated, body.Headline, "", "")

	case msg.Subject().Match("NATHEJK.*.sos.*.headline.updated"):
		var body HeadlineUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.update(msg, goqu.Record{"headline": body.Headline}); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivityHeadlineUpdated, body.Headline, "", "")

	case msg.Subject().Match("NATHEJK.*.sos.*.description.updated"):
		var body DescriptionUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.update(msg, goqu.Record{"description": body.Description}); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivityDescriptionUpdated, body.Description, "", "")

	case msg.Subject().Match("NATHEJK.*.sos.*.commented"):
		var body Commented
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.touch(msg); err != nil {
			return err
		}
		// The comment's own id goes on the entry, because a later edit refers back
		// to it. Nothing else in the timeline is addressable yet.
		return c.appendActivity(msg, ActivityCommented, body.Comment, string(body.CommentID), "")

	case msg.Subject().Match("NATHEJK.*.sos.*.comment.updated"):
		var body CommentUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.touch(msg); err != nil {
			return err
		}
		// Appended, not applied to the original row: the original comment stays in
		// the timeline and this entry says what it became. With every operator
		// sharing one identity, an append-only log is the only thing making an
		// editable comment safe.
		return c.appendActivity(msg, ActivityCommentUpdated, body.Comment, "", string(body.CommentID))

	case msg.Subject().Match("NATHEJK.*.sos.*.severity.specified"):
		var body SeveritySpecified
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.update(msg, goqu.Record{"severity": string(body.Severity)}); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivitySeveritySpecified, string(body.Severity), "", "")

	case msg.Subject().Match("NATHEJK.*.sos.*.assigned"):
		var body Assigned
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.update(msg, goqu.Record{"assigneeSectionSlug": string(body.SectionSlug)}); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivityAssigned, string(body.SectionSlug), "", "")

	case msg.Subject().Match("NATHEJK.*.sos.*.closed"):
		if err := c.update(msg, goqu.Record{"status": string(StatusClosed)}); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivityClosed, "", "", "")

	case msg.Subject().Match("NATHEJK.*.sos.*.reopened"):
		if err := c.update(msg, goqu.Record{"status": string(StatusOpen)}); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivityReopened, "", "", "")

	case msg.Subject().Match("NATHEJK.*.sos.*.deleted"):
		// Soft: the row and its timeline stay, and every read path filters on
		// deletedAt. A case is deleted because it was created in error, which is
		// not a reason to lose the record that somebody did so.
		if err := c.update(msg, goqu.Record{"deletedAt": c.at(msg)}); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivityDeleted, "", "", "")

	case msg.Subject().Match("NATHEJK.*.sos.*.team.associated"):
		var body TeamAssociated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.touch(msg); err != nil {
			return err
		}
		insert := goqu.Record{
			"sosId":     c.sosID(msg),
			"teamId":    string(body.TeamID),
			"year":      c.year(msg),
			"createdAt": c.at(msg),
		}
		// Associating twice is a no-op rather than an error: two operators on the
		// same call will both reach for the patrol.
		if err := c.exec(goqu.Dialect("mysql").
			Insert("sos_team").Rows(insert).
			OnConflict(goqu.DoNothing())); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivityTeamAssociated, string(body.TeamID), "", "")

	case msg.Subject().Match("NATHEJK.*.sos.*.team.disassociated"):
		var body TeamDisassociated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.touch(msg); err != nil {
			return err
		}
		if err := c.exec(goqu.Dialect("mysql").
			Delete("sos_team").
			Where(goqu.C("sosId").Eq(c.sosID(msg)), goqu.C("teamId").Eq(string(body.TeamID)))); err != nil {
			return err
		}
		return c.appendActivity(msg, ActivityTeamDisassociated, string(body.TeamID), "", "")

	// --- The member lifecycle summaries (PRD 006) ---
	//
	// Each is one timeline entry for one operation, with the whole summary stored as
	// JSON in the entry's value column. No schema change: PRD 001 built this table to
	// be extended exactly here, and `value` is TEXT.
	//
	// The case row itself is only touched, never updated: a member changing status is
	// activity *on* the case, not a change *to* it. It still has to advance
	// lastActivityAt, because the list sorts by that and an operator's eye should land
	// on the case where somebody just left the race.
	case msg.Subject().Match("NATHEJK.*.sos.*.member.status.changed"):
		var body MemberStatusChanged
		if err := msg.Body(&body); err != nil {
			return err
		}
		return c.appendSummary(msg, ActivityMemberStatusChanged, body)

	case msg.Subject().Match("NATHEJK.*.sos.*.team.collected"):
		var body TeamCollected
		if err := msg.Body(&body); err != nil {
			return err
		}
		return c.appendSummary(msg, ActivityTeamCollected, body)

	case msg.Subject().Match("NATHEJK.*.sos.*.member.moved"):
		var body MembersMoved
		if err := msg.Body(&body); err != nil {
			return err
		}
		return c.appendSummary(msg, ActivityMembersMoved, body)
	}

	log.Printf("sos consumer: unhandled subject %q", msg.Subject().Subject())
	return nil
}

// appendSummary writes one timeline entry whose value is a JSON summary of an
// operation.
//
// The existing entries store a bare string — a headline, a severity, a team id —
// because one event changed one thing. A member operation changes several members at
// once and the line has to name them, so the payload is structured. Marshalling it
// rather than adding columns keeps `sos_activity` a log of *what happened* instead of
// a widening union of every event's fields, which is what PRD 001 had in mind when it
// required the table to be extensible without a schema change.
func (c *consumer) appendSummary(msg stream.Message, t ActivityType, summary any) error {
	if err := c.touch(msg); err != nil {
		return err
	}
	value, err := json.Marshal(summary)
	if err != nil {
		// The event is in the log and cannot be un-published, so failing here would
		// wedge the replay on every restart from now on. Log and skip the entry: a
		// missing timeline line is recoverable, a projection that cannot finish is not.
		log.Printf("sos consumer: cannot marshal %s summary for case %q: %v", t, c.sosID(msg), err)
		return nil
	}
	return c.appendActivity(msg, t, string(value), "", "")
}

// update applies fields to the case row and advances lastActivityAt, which every
// event does — the list sorts by it, so an event that did not touch it would make
// a case look untouched.
func (c *consumer) update(msg stream.Message, fields goqu.Record) error {
	fields["lastActivityAt"] = c.at(msg)
	return c.exec(goqu.Dialect("mysql").
		Update("sos").Set(fields).
		Where(goqu.C("id").Eq(c.sosID(msg))))
}

// touch advances lastActivityAt without changing the case itself — for comments
// and team associations, which are activity on the case rather than changes to it.
func (c *consumer) touch(msg stream.Message) error {
	return c.exec(goqu.Dialect("mysql").
		Update("sos").Set(goqu.Record{"lastActivityAt": c.at(msg)}).
		Where(goqu.C("id").Eq(c.sosID(msg))))
}

// appendActivity writes one timeline entry, keyed by the event's stream sequence.
//
// That key is what makes replay idempotent: the same event always lands on the
// same row, so rebuilding from the whole history reproduces the timeline instead
// of duplicating it. Using a generated id here instead would silently double
// every entry on each restart.
func (c *consumer) appendActivity(msg stream.Message, t ActivityType, value, activityID, refActivityID string) error {
	insert := goqu.Record{
		"seq":           msg.Sequence(),
		"sosId":         c.sosID(msg),
		"year":          c.year(msg),
		"type":          string(t),
		"actorUserId":   string(c.actor(msg)),
		"activityId":    activityID,
		"refActivityId": refActivityID,
		"value":         value,
		"createdAt":     c.at(msg),
	}
	return c.exec(goqu.Dialect("mysql").
		Insert("sos_activity").Rows(insert).
		OnConflict(goqu.DoNothing()))
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

// sosID is taken from the subject rather than the body: the subject is what the
// stream routes on, so it is the one place the id cannot disagree with itself.
func (c *consumer) sosID(msg stream.Message) string {
	return msg.Subject().Parts()[3]
}

// year comes from the subject too, never from msg.Time(). The old commented-out
// spejderstatus projection derived the year from the message timestamp, which is
// wrong the moment history is replayed across a year boundary.
func (c *consumer) year(msg stream.Message) string {
	return msg.Subject().Parts()[1]
}

// at renders the event time as a MySQL DATETIME in UTC.
//
// UTC deliberately: the driver reads DATETIME back as UTC (parseTime with the
// default location), so the API marshals an explicit Z-offset timestamp and the
// browser converts it to the operator's own clock. Storing local wall-clock here
// would make every timestamp two hours wrong in summer, in a log whose entire
// purpose is establishing when things happened.
func (c *consumer) at(msg stream.Message) string {
	return msg.Time().UTC().Format(time.DateTime)
}

// actor is the user the event was published by, empty until the platform
// authenticates anybody (PRD 001 §6 Auth).
func (c *consumer) actor(msg stream.Message) types.UserID {
	var meta messages.Metadata
	if err := msg.Meta(&meta); err != nil {
		return ""
	}
	return meta.UserID
}
