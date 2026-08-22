package shelter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"

	"nathejk.dk/nathejk/table/spejderstatus"
)

// --- fakes ---

// Asserting on emitted SQL rather than against a database, which is what spejderstatus and
// the rest of the repo do. It pins the parts that matter in a projection replayed on every
// boot: whether a write is an upsert, which columns an event is entitled to overwrite, and
// whether a second identical event changes anything.
type recordingWriter struct {
	stmts []string
}

func (w *recordingWriter) Consume(stmt string) error {
	w.stmts = append(w.stmts, stmt)
	return nil
}

func (w *recordingWriter) only(t *testing.T) string {
	t.Helper()
	if len(w.stmts) != 1 {
		t.Fatalf("expected exactly one statement, got %d: %v", len(w.stmts), w.stmts)
	}
	return w.stmts[0]
}

type fakeMessage struct {
	subject stream.Subject
	body    any
	at      time.Time
}

func (m fakeMessage) Subject() stream.Subject { return m.subject }
func (m fakeMessage) Time() time.Time         { return m.at }
func (m fakeMessage) Sequence() uint64        { return 1 }
func (m fakeMessage) Body(dst any) error {
	b, err := json.Marshal(m.body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
func (m fakeMessage) Meta(any) error { return nil }
func (m fakeMessage) RawBody() any   { return m.body }
func (m fakeMessage) RawMeta() any   { return nil }

func msg(subj string, body any) stream.Message {
	return fakeMessage{
		subject: subject.FromStr(subj),
		body:    body,
		at:      time.Date(2026, 9, 26, 0, 42, 0, 0, time.UTC),
	}
}

func msgAt(subj string, body any, at time.Time) stream.Message {
	return fakeMessage{subject: subject.FromStr(subj), body: body, at: at}
}

func handle(t *testing.T, c *consumer, m stream.Message) {
	t.Helper()
	if err := c.HandleMessage(m); err != nil {
		t.Fatalf("HandleMessage(%s): %v", m.Subject().Subject(), err)
	}
}

func updateClause(t *testing.T, stmt string) string {
	t.Helper()
	i := strings.Index(stmt, "ON DUPLICATE KEY UPDATE")
	if i < 0 {
		t.Fatalf("statement is not an upsert, so replay would duplicate or clobber: %s", stmt)
	}
	return stmt[i:]
}

// --- tests ---

// An acceptance carrying a tent records both facts at once, which is the ordinary case: the
// crew meeting a car types the tent as they take the scouts in.
func TestAcceptanceWithPlaceringRecordsBoth(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.shelter.accepted",
		spejderstatus.ShelterAccepted{MemberID: "m-1", TeamID: "team-1", Placement: "Telt 4"}))

	stmt := w.only(t)
	for _, want := range []string{"'m-1'", "'2026'", "'team-1'", "'Telt 4'"} {
		if !strings.Contains(stmt, want) {
			t.Errorf("want %s in: %s", want, stmt)
		}
	}
	updateClause(t, stmt)
}

// A scout accepted with nowhere to go yet is a real state, and placedAt must stay NULL for
// them: the screen shows "arrived, not yet placed" as the crew's next job, and stamping
// placedAt with the arrival time would make every arrival look dealt with.
func TestAcceptanceWithoutPlaceringLeavesPlacedAtNull(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.shelter.accepted",
		spejderstatus.ShelterAccepted{MemberID: "m-1", TeamID: "team-1"}))

	stmt := w.only(t)
	if !strings.Contains(stmt, "NULL") {
		t.Errorf("expected a NULL placedAt for an unplaced arrival: %s", stmt)
	}
}

// The acceptance timestamp is when the shelter took charge of a child. Replay delivers the
// acceptance again on every boot, so it must not be in the update list — otherwise "in our
// care since 00:42" silently becomes "since the last time the API restarted".
func TestAcceptedAtIsNeverOverwritten(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.shelter.accepted",
		spejderstatus.ShelterAccepted{MemberID: "m-1", TeamID: "team-1", Placement: "Telt 4"}))

	if clause := updateClause(t, w.only(t)); strings.Contains(clause, "acceptedAt") {
		t.Errorf("acceptedAt is overwritten on replay: %s", clause)
	}
}

// The bug this projection is most likely to have.
//
// Replay order is not guaranteed between the acceptance and a later placering, and the
// acceptance is re-delivered on every boot. If an acceptance carrying no placering
// overwrote the column, a scout moved into Telt 4 at 01:10 would be back to nowhere the next
// time the API restarted — and the crew would be looking for a child the screen says is
// nowhere.
func TestEmptyPlaceringDoesNotWipeAStoredOne(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.shelter.accepted",
		spejderstatus.ShelterAccepted{MemberID: "m-1", TeamID: "team-1"}))

	clause := updateClause(t, w.only(t))
	if !strings.Contains(clause, "IF(VALUES(placement)") {
		t.Errorf("placement is updated unconditionally, so replay can blank it: %s", clause)
	}
}

// A placering event for a member this projection never saw arrive must still land. The
// acceptance may be missing from a truncated history or arrive later in a replay, and a scout
// with a placering and no recorded arrival is a better read model than no scout at all — the
// crew can find them, which is the point of the table.
func TestPlacedCreatesTheRowWhenTheArrivalIsMissing(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.shelter.placed",
		spejderstatus.ShelterPlaced{MemberID: "m-1", TeamID: "team-1", Placement: "Telt 4"}))

	stmt := w.only(t)
	if !strings.HasPrefix(stmt, "INSERT") {
		t.Errorf("expected an insert so a missing arrival cannot lose the scout: %s", stmt)
	}
	updateClause(t, stmt)
}

// Handover empties the bed. Deleted rather than flagged: the table answers "where is this
// child now", and a released child is not anywhere of ours. Their history survives in
// spejderstatuslog.
func TestHandoverRemovesTheRow(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.handover.completed",
		spejderstatus.HandoverCompleted{MemberID: "m-1", TeamID: "team-1", To: "released"}))

	stmt := w.only(t)
	if !strings.HasPrefix(stmt, "DELETE") {
		t.Errorf("expected a delete: %s", stmt)
	}
	for _, want := range []string{"'2026'", "'m-1'"} {
		if !strings.Contains(stmt, want) {
			t.Errorf("want %s in: %s", want, stmt)
		}
	}
}

// Replay must be a no-op, not an accumulation. This table is rebuilt from the full history on
// every API boot, so identical statements are the correctness condition rather than a
// nice-to-have.
func TestReplayIsIdempotent(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	m := msg("NATHEJK.2026.spejder.m-1.shelter.accepted",
		spejderstatus.ShelterAccepted{MemberID: "m-1", TeamID: "team-1", Placement: "Telt 4"})
	handle(t, c, m)
	handle(t, c, m)

	if len(w.stmts) != 2 || w.stmts[0] != w.stmts[1] {
		t.Errorf("replay did not reproduce the same statement: %v", w.stmts)
	}
}

// The year comes from the subject, never from the message clock. Replay crosses year
// boundaries by definition — that is how the table is built — and msg.Time().Year() would
// file every historical event under the current year, quietly merging two races.
func TestYearComesFromSubjectNotMessageTime(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msgAt("NATHEJK.2025.spejder.m-1.shelter.accepted",
		spejderstatus.ShelterAccepted{MemberID: "m-1", TeamID: "team-1", Placement: "Telt 4"},
		time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)))

	if stmt := w.only(t); !strings.Contains(stmt, "'2025'") {
		t.Errorf("expected the subject's year: %s", stmt)
	}
}

// A subject nobody handles must not fail the replay: an unknown message is logged and
// skipped, because a poison message that returns an error would wedge the whole rebuild.
func TestUnknownSubjectIsANoOp(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.spejder.m-1.pickup.accepted", struct{}{}))

	if len(w.stmts) != 0 {
		t.Errorf("expected no statements, got %v", w.stmts)
	}
}

// Every subject declared must be handled. A subscription with no case is silent: the
// projection would receive the event, log "unhandled", and leave the crew with an empty
// screen and no error anywhere.
func TestEverySubjectDeclaredIsHandled(t *testing.T) {
	c := &consumer{}
	for _, s := range c.Consumes() {
		// Subjects are declared with a colon after the prefix (the subscription form) and
		// matched with a dot, so compare on the parts rather than the string.
		declared := strings.Replace(s.Subject(), ":", ".", 1)
		concrete := strings.Replace(strings.Replace(declared, "*.spejder", "2026.spejder", 1), "spejder.*", "spejder.m-1", 1)

		w := &recordingWriter{}
		c := &consumer{w: w}
		handle(t, c, msg(concrete, map[string]string{"memberId": "m-1", "teamId": "team-1"}))
		if len(w.stmts) == 0 {
			t.Errorf("subject %q is subscribed but produced no statement", concrete)
		}
	}
}
