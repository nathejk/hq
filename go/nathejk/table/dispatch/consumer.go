package dispatch

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

// consumer projects the kørsel events onto the read model: the tasks, the tours, their
// stops, what is actioned at each stop, and the timeline.
//
// Three properties, and everything else follows from them:
//
//   - **Every event appends to the timeline.** A dispatch desk is a log first. The rows
//     are current state; the timeline is what happened, and it is the only part a shift
//     handover can use.
//   - **Replay is idempotent.** Row writes are upserts, timeline entries are keyed by the
//     stream sequence of the event that produced them, and the stop list is rebuilt
//     wholesale from the event that describes it. Replaying the history rebuilds exactly
//     the same rows rather than duplicating the log.
//   - **A task's state comes from its own events, never from inspecting its stops.** The
//     tour publishes the plan; the task publishes what the plan did to it. That is why
//     `planned`, `unplanned`, `underway` and `completed` exist as task events at all, and
//     it is what keeps the projection free of derivation order.
type consumer struct {
	w cqrs.Writer
}

func (c *consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		// Configuration: which subsections are dispatch units.
		subject.FromStr("NATHEJK:*.dispatch.section.*.dispatchable"),

		// Tasks. The entity token derived from these is `dispatch`.
		subject.FromStr("NATHEJK:*.dispatch.*.created"),
		subject.FromStr("NATHEJK:*.dispatch.*.updated"),
		subject.FromStr("NATHEJK:*.dispatch.*.planned"),
		subject.FromStr("NATHEJK:*.dispatch.*.unplanned"),
		subject.FromStr("NATHEJK:*.dispatch.*.underway"),
		subject.FromStr("NATHEJK:*.dispatch.*.pickedup"),
		subject.FromStr("NATHEJK:*.dispatch.*.completed"),
		subject.FromStr("NATHEJK:*.dispatch.*.cancelled"),

		// Tours. A separate entity token, `tour`, so a client watching the board's tour
		// pane is not woken by every task edit — and vice versa.
		subject.FromStr("NATHEJK:*.tour.*.created"),
		subject.FromStr("NATHEJK:*.tour.*.updated"),
		subject.FromStr("NATHEJK:*.tour.*.stops.changed"),
		subject.FromStr("NATHEJK:*.tour.*.underway"),
		subject.FromStr("NATHEJK:*.tour.*.stop.visited"),
		subject.FromStr("NATHEJK:*.tour.*.completed"),
		subject.FromStr("NATHEJK:*.tour.*.cancelled"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch {
	// Checked before the task patterns: the section subject also has "dispatch" in third
	// position, so NATHEJK.*.dispatch.*.updated and friends would be candidate matches
	// for a five-part section subject. The same trap sos documents.
	case msg.Subject().Match("NATHEJK.*.dispatch.section.*.dispatchable"):
		return c.sectionDispatchable(msg)

	case msg.Subject().Match("NATHEJK.*.dispatch.*.created"):
		var body TaskCreated
		if err := msg.Body(&body); err != nil {
			return err
		}
		memberIDs, err := json.Marshal(body.MemberIDs)
		if err != nil {
			return err
		}
		// `[]` rather than `null` for a task that collects nobody: the column is read back
		// as a list, and this repo has been bitten three times by a null collection.
		if body.MemberIDs == nil {
			memberIDs = []byte("[]")
		}
		insert := goqu.Record{
			"id":              c.entityID(msg),
			"year":            c.year(msg),
			"kind":            string(body.Kind),
			"priority":        string(body.Priority),
			"description":     body.Description,
			"spaceNeeds":      body.SpaceNeeds,
			"pickupKind":      string(body.Pickup.Kind),
			"pickupRefId":     body.Pickup.RefID,
			"pickupLabel":     body.Pickup.Label,
			"dropoffKind":     string(body.Dropoff.Kind),
			"dropoffRefId":    body.Dropoff.RefID,
			"dropoffLabel":    body.Dropoff.Label,
			"state":           string(TaskStateQueued),
			"createdUts":      body.CreatedUts,
			"notBeforeUts":    body.NotBeforeUts,
			"deadlineUts":     body.DeadlineUts,
			"sosId":           string(body.SosID),
			"teamId":          string(body.TeamID),
			"memberIds":       string(memberIDs),
			"createdBy":       string(c.actor(msg)),
			"lastActivityUts": c.uts(msg),
		}
		// Everything except `state` and the clocks is replayed. State is excluded on
		// purpose: replay re-delivers `created` on every boot, *after* the transitions
		// that came later, and an upsert that reset state to queued would silently
		// un-complete every finished task on restart. The same trap task 099 hit with a
		// note's text.
		update := goqu.Record{}
		for _, col := range []string{"kind", "priority", "description", "spaceNeeds",
			"pickupKind", "pickupRefId", "pickupLabel", "dropoffKind", "dropoffRefId",
			"dropoffLabel", "createdUts", "notBeforeUts", "deadlineUts", "sosId",
			"teamId", "memberIds", "createdBy"} {
			update[col] = goqu.L("VALUES(" + col + ")")
		}
		if err := c.exec(goqu.Dialect("mysql").
			Insert("dispatch_task").Rows(insert).
			OnConflict(goqu.DoUpdate("id", update))); err != nil {
			return err
		}
		return c.activity(msg, ActivityTaskCreated, body.Description)

	case msg.Subject().Match("NATHEJK.*.dispatch.*.updated"):
		var body TaskUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		set := goqu.Record{
			"kind":            string(body.Kind),
			"priority":        string(body.Priority),
			"description":     body.Description,
			"spaceNeeds":      body.SpaceNeeds,
			"pickupKind":      string(body.Pickup.Kind),
			"pickupRefId":     body.Pickup.RefID,
			"pickupLabel":     body.Pickup.Label,
			"dropoffKind":     string(body.Dropoff.Kind),
			"dropoffRefId":    body.Dropoff.RefID,
			"dropoffLabel":    body.Dropoff.Label,
			"notBeforeUts":    body.NotBeforeUts,
			"deadlineUts":     body.DeadlineUts,
			"lastActivityUts": c.uts(msg),
		}
		if err := c.updateTask(msg, set); err != nil {
			return err
		}
		return c.activity(msg, ActivityTaskUpdated, strings.Join(body.Changed, ", "))

	case msg.Subject().Match("NATHEJK.*.dispatch.*.planned"):
		var body TaskPlanned
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.updateTask(msg, goqu.Record{
			"state":           string(TaskStatePlanned),
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTaskPlanned, string(body.TourID))

	case msg.Subject().Match("NATHEJK.*.dispatch.*.unplanned"):
		var body TaskUnplanned
		if err := msg.Body(&body); err != nil {
			return err
		}
		// Note what is *not* here: createdUts. The waiting clock survives a re-plan,
		// because the scout has been waiting since the call.
		if err := c.updateTask(msg, goqu.Record{
			"state":           string(TaskStateQueued),
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTaskUnplanned, string(body.TourID))

	case msg.Subject().Match("NATHEJK.*.dispatch.*.underway"):
		var body TaskUnderway
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.updateTask(msg, goqu.Record{
			"state":           string(TaskStateUnderway),
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTaskUnderway, string(body.TourID))

	case msg.Subject().Match("NATHEJK.*.dispatch.*.pickedup"):
		var body TaskPickedUp
		if err := msg.Body(&body); err != nil {
			return err
		}
		// pickedUpUts and not state: people aboard is a moment on the way to done, and a
		// task whose scouts are in the car is still underway.
		if err := c.updateTask(msg, goqu.Record{
			"pickedUpUts":     body.AtUts,
			"state":           string(TaskStateUnderway),
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTaskPickedUp, string(body.SectionSlug))

	case msg.Subject().Match("NATHEJK.*.dispatch.*.completed"):
		var body TaskCompleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.updateTask(msg, goqu.Record{
			"state":           string(TaskStateDone),
			"doneUts":         body.AtUts,
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTaskCompleted, "")

	case msg.Subject().Match("NATHEJK.*.dispatch.*.cancelled"):
		var body TaskCancelled
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.updateTask(msg, goqu.Record{
			"state":           string(TaskStateCancelled),
			"cancelledUts":    body.AtUts,
			"cancelReason":    body.Reason,
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTaskCancelled, body.Reason)

	case msg.Subject().Match("NATHEJK.*.tour.*.created"):
		var body TourCreated
		if err := msg.Body(&body); err != nil {
			return err
		}
		insert := goqu.Record{
			"id":              c.entityID(msg),
			"year":            c.year(msg),
			"sectionSlug":     string(body.SectionSlug),
			"departureUts":    body.DepartureUts,
			"notes":           body.Notes,
			"state":           string(TourStatePlanned),
			"createdUts":      body.CreatedUts,
			"createdBy":       string(c.actor(msg)),
			"lastActivityUts": c.uts(msg),
		}
		update := goqu.Record{}
		for _, col := range []string{"sectionSlug", "departureUts", "notes", "createdUts", "createdBy"} {
			update[col] = goqu.L("VALUES(" + col + ")")
		}
		if err := c.exec(goqu.Dialect("mysql").
			Insert("dispatch_tour").Rows(insert).
			OnConflict(goqu.DoUpdate("id", update))); err != nil {
			return err
		}
		return c.activity(msg, ActivityTourCreated, string(body.SectionSlug))

	case msg.Subject().Match("NATHEJK.*.tour.*.updated"):
		var body TourUpdated
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.updateTour(msg, goqu.Record{
			"sectionSlug":     string(body.SectionSlug),
			"departureUts":    body.DepartureUts,
			"notes":           body.Notes,
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTourUpdated, strings.Join(body.Changed, ", "))

	case msg.Subject().Match("NATHEJK.*.tour.*.stops.changed"):
		return c.stopsChanged(msg)

	case msg.Subject().Match("NATHEJK.*.tour.*.underway"):
		var body TourUnderway
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.updateTour(msg, goqu.Record{
			"state":           string(TourStateUnderway),
			"underwayUts":     body.AtUts,
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTourUnderway, "")

	case msg.Subject().Match("NATHEJK.*.tour.*.stop.visited"):
		var body StopVisited
		if err := msg.Body(&body); err != nil {
			return err
		}
		// Scoped to the tour as well as the stop, so an event published for another tour
		// cannot mark this one's stop visited.
		if err := c.exec(goqu.Dialect("mysql").
			Update("dispatch_stop").Set(goqu.Record{"visitedUts": body.AtUts}).
			Where(goqu.C("id").Eq(string(body.StopID)),
				goqu.C("tourId").Eq(c.entityID(msg)),
				goqu.C("year").Eq(c.year(msg)))); err != nil {
			return err
		}
		if err := c.updateTour(msg, goqu.Record{"lastActivityUts": c.uts(msg)}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTourStopVisited, string(body.StopID))

	case msg.Subject().Match("NATHEJK.*.tour.*.completed"):
		var body TourCompleted
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.updateTour(msg, goqu.Record{
			"state":           string(TourStateCompleted),
			"completedUts":    body.AtUts,
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTourCompleted, "")

	case msg.Subject().Match("NATHEJK.*.tour.*.cancelled"):
		var body TourCancelled
		if err := msg.Body(&body); err != nil {
			return err
		}
		if err := c.updateTour(msg, goqu.Record{
			"state":           string(TourStateCancelled),
			"cancelledUts":    body.AtUts,
			"cancelReason":    body.Reason,
			"lastActivityUts": c.uts(msg),
		}); err != nil {
			return err
		}
		return c.activity(msg, ActivityTourCancelled, body.Reason)
	}
	return nil
}

func (c *consumer) sectionDispatchable(msg stream.Message) error {
	var body SectionDispatchableSet
	if err := msg.Body(&body); err != nil {
		return err
	}
	slug := msg.Subject().Parts()[4]
	if body.Dispatchable {
		// OnConflict-do-nothing rather than an upsert of setAt: replay must not move the
		// timestamp, and nothing reads it as anything but "since when".
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

// stopsChanged rebuilds a tour's stop list from the event.
//
// Delete-then-insert rather than a diff. The event carries the whole list *including each
// stop's visitedUts*, so a rebuild loses nothing, and it is the only approach that cannot
// leave a stop behind that the new plan does not mention — which a diff can, and which
// would show the desk a stop no longer on the tour.
func (c *consumer) stopsChanged(msg stream.Message) error {
	var body StopsChanged
	if err := msg.Body(&body); err != nil {
		return err
	}
	tourID := c.entityID(msg)
	year := c.year(msg)

	if err := c.exec(goqu.Dialect("mysql").
		Delete("dispatch_stop_task").Where(goqu.C("tourId").Eq(tourID))); err != nil {
		return err
	}
	if err := c.exec(goqu.Dialect("mysql").
		Delete("dispatch_stop").Where(goqu.C("tourId").Eq(tourID))); err != nil {
		return err
	}
	for i, stop := range body.Stops {
		// The slice's order is the order. sortOrder is written from the index rather than
		// carried on the wire, so there is exactly one source of truth for the ordering.
		if err := c.exec(goqu.Dialect("mysql").
			Insert("dispatch_stop").Rows(goqu.Record{
			"id":              string(stop.StopID),
			"tourId":          tourID,
			"year":            year,
			"sortOrder":       i,
			"placeKind":       string(stop.Place.Kind),
			"placeRefId":      stop.Place.RefID,
			"placeLabel":      stop.Place.Label,
			"plannedUts":      stop.PlannedUts,
			"plannedOverride": stop.Override,
			"visitedUts":      stop.VisitedUts,
		})); err != nil {
			return err
		}
		for _, st := range stop.Tasks {
			if err := c.exec(goqu.Dialect("mysql").
				Insert("dispatch_stop_task").Rows(goqu.Record{
				"stopId": string(stop.StopID),
				"taskId": string(st.TaskID),
				"tourId": tourID,
				"year":   year,
				"role":   string(st.Role),
			}).OnConflict(goqu.DoNothing())); err != nil {
				return err
			}
		}
	}
	if err := c.updateTour(msg, goqu.Record{"lastActivityUts": c.uts(msg)}); err != nil {
		return err
	}
	return c.activity(msg, ActivityTourStopsChanged, "")
}

// updateTask scopes every task write to the id *and* the year, so an event published for
// another year cannot reach this year's row.
func (c *consumer) updateTask(msg stream.Message, set goqu.Record) error {
	return c.exec(goqu.Dialect("mysql").
		Update("dispatch_task").Set(set).
		Where(goqu.C("id").Eq(c.entityID(msg)), goqu.C("year").Eq(c.year(msg))))
}

func (c *consumer) updateTour(msg stream.Message, set goqu.Record) error {
	return c.exec(goqu.Dialect("mysql").
		Update("dispatch_tour").Set(set).
		Where(goqu.C("id").Eq(c.entityID(msg)), goqu.C("year").Eq(c.year(msg))))
}

// activity appends one timeline entry, keyed by the event's stream sequence — which is
// what makes replay rebuild the log rather than multiply it.
func (c *consumer) activity(msg stream.Message, kind ActivityType, value string) error {
	row := goqu.Record{
		"seq":         msg.Sequence(),
		"year":        c.year(msg),
		"type":        string(kind),
		"actorUserId": string(c.actor(msg)),
		"value":       value,
		"createdUts":  c.uts(msg),
		"taskId":      "",
		"tourId":      "",
	}
	if strings.HasPrefix(string(kind), "tour.") {
		row["tourId"] = c.entityID(msg)
	} else {
		row["taskId"] = c.entityID(msg)
	}
	return c.exec(goqu.Dialect("mysql").
		Insert("dispatch_activity").Rows(row).
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

// entityID is the task or tour id, taken from the subject rather than the body: the
// subject is what the stream routes on, so it is the one place the id cannot disagree
// with itself.
func (c *consumer) entityID(msg stream.Message) string {
	return msg.Subject().Parts()[3]
}

// year comes from the subject, never from msg.Time(): deriving it from the message
// timestamp is wrong the moment history is replayed across a year boundary.
func (c *consumer) year(msg stream.Message) string {
	return msg.Subject().Parts()[1]
}

// uts is the event time in unix seconds, which is how this entity stores time.
func (c *consumer) uts(msg stream.Message) int64 {
	return msg.Time().Unix()
}

// at renders the event time as a MySQL DATETIME in UTC, for the one table here that
// predates the switch to unix seconds (dispatchable_section).
func (c *consumer) at(msg stream.Message) string {
	return msg.Time().UTC().Format(time.DateTime)
}

// actor is the user the event was published by, empty until the platform authenticates
// anybody.
func (c *consumer) actor(msg stream.Message) types.UserID {
	var meta messages.Metadata
	if err := msg.Meta(&meta); err != nil {
		return ""
	}
	return meta.UserID
}
