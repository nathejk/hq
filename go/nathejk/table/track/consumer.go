package track

import (
	"log"

	"github.com/doug-martin/goqu/v9"
	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
)

// The one subject this package consumes.
//
// Note the domain: `TELEMETRY`, not `NATHEJK`. The stream library derives the JetStream *stream
// name* from a subject's domain, so this string is also what makes HQ subscribe to a second stream
// — and why that stream must already exist, correctly cased, or `mux.Run` fails at boot and the api
// never starts (task 139).
//
// The entity token is `track`. Not `position`, not `telemetry`: this is what
// `live.SignalFromSubject` derives, so it is also the token every frontend `dependsOn` must name.
const subjectReported = "TELEMETRY.*.track.*.reported"

// insertChunk bounds one INSERT statement.
//
// A batch may carry up to 2,000 points, and the Writer takes a rendered SQL string rather than
// arguments, so an unchunked insert would be one very long statement. 500 rows keeps each one well
// inside any packet limit while still being a handful of round trips rather than hundreds.
const insertChunk = 500

type consumer struct {
	w cqrs.Writer
}

func (c *consumer) Consumes() []stream.Subject {
	return []stream.Subject{
		subject.FromStr(subjectReported),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch {
	case msg.Subject().Match(subjectReported):
		var body Reported
		if err := msg.Body(&body); err != nil {
			return err
		}
		return c.apply(body)

	default:
		log.Printf("track: unhandled message %q", msg.Subject().Subject())
	}
	return nil
}

// apply writes one batch: every point into the history, and the newest of them into the
// latest-position row.
//
// A batch with no points is not an error. The producer rejects an empty request, but a batch whose
// every point was junk is legitimate — `Clean` drops points and keeps the batch, deliberately, so
// that one bad reading cannot poison a member's whole track behind a retry loop. Such a message
// arrives here carrying nothing to write, and the right response is to do nothing.
func (c *consumer) apply(body Reported) error {
	if body.PersonID == "" || len(body.Points) == 0 {
		return nil
	}

	if err := c.insertPoints(body); err != nil {
		return err
	}
	return c.advanceLatest(body)
}

// insertPoints appends the batch to the history, ignoring points already held.
//
// INSERT IGNORE against the `(personId, ts)` primary key is doing real work rather than being
// defensive: the client retries a batch until the server accepts it, and HQ replays the whole
// stream on every api restart, so the same point arriving twice is normal operation. Making that a
// no-op at the storage layer is what keeps every caller above it free of dedupe logic.
func (c *consumer) insertPoints(body Reported) error {
	rows := make([]any, 0, len(body.Points))
	for _, p := range body.Points {
		rows = append(rows, goqu.Record{
			"personId":   body.PersonID,
			"ts":         p.Ts,
			"personType": body.UserType,
			"year":       body.Year,
			"latitude":   p.Lat,
			"longitude":  p.Lng,
			"accuracy":   p.Accuracy,
		})
	}

	for len(rows) > 0 {
		n := min(insertChunk, len(rows))
		ds := goqu.Dialect("mysql").
			Insert("track_point").
			Rows(rows[:n]...).
			// The mysql dialect renders DoNothing as INSERT IGNORE.
			OnConflict(goqu.DoNothing())
		if err := c.exec(ds); err != nil {
			return err
		}
		rows = rows[n:]
	}
	return nil
}

// advanceLatest moves the person's latest-position row forward to the newest point in the batch —
// but only if it really is newer than what is stored.
//
// The guard matters because out-of-order arrival is routine here, not exotic. Points are batched, so
// a message can carry an older backlog than one already applied; the client retries, so messages
// repeat; and replay re-reads the whole stream from the beginning on every restart. Without the
// condition, a boot replay would leave every person's "last seen" showing whichever message
// happened to be applied last rather than whichever position is actually most recent — and the
// position glyph would then quietly lie about staleness on every page in HQ.
//
// Expressed as `IF(VALUES(ts) > ts, …)` per column so the decision is made by the database in one
// statement. A read-then-write would be a race and, at one round trip per person per batch, the
// most expensive thing in this package.
func (c *consumer) advanceLatest(body Reported) error {
	newest := body.Points[0]
	for _, p := range body.Points[1:] {
		if p.Ts > newest.Ts {
			newest = p
		}
	}

	insert := goqu.Record{
		"personId":   body.PersonID,
		"personType": body.UserType,
		"year":       body.Year,
		"latitude":   newest.Lat,
		"longitude":  newest.Lng,
		"accuracy":   newest.Accuracy,
		"ts":         newest.Ts,
		"updatedAt":  goqu.L("CURRENT_TIMESTAMP"),
	}
	newer := func(col string) goqu.Expression {
		return goqu.L("IF(VALUES(ts) > ts, VALUES(" + col + "), " + col + ")")
	}
	update := goqu.Record{
		"personType": newer("personType"),
		"year":       newer("year"),
		"latitude":   newer("latitude"),
		"longitude":  newer("longitude"),
		"accuracy":   newer("accuracy"),
		"updatedAt":  goqu.L("IF(VALUES(ts) > ts, CURRENT_TIMESTAMP, updatedAt)"),
		// ts last: every expression above compares against the stored value, so it must still be
		// the old one while they are evaluated. MariaDB assigns left to right, and goqu renders a
		// Record's keys sorted, which puts "ts" after every column that reads it — true today by
		// alphabet rather than by intent, so the assignment is written to be correct either way:
		// it compares VALUES(ts) with ts, which is a no-op if ts has already been advanced.
		"ts": newer("ts"),
	}

	return c.exec(goqu.Dialect("mysql").
		Insert("track_latest").
		Rows(insert).
		OnConflict(goqu.DoUpdate("personId", update)))
}

type toSQLer interface {
	ToSQL() (string, []any, error)
}

func (c *consumer) exec(ds toSQLer) error {
	sqlStr, _, err := ds.ToSQL()
	if err != nil {
		return err
	}
	return c.w.Consume(sqlStr)
}
