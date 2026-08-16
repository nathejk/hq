package spejderstatus

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

// --- fakes ---

// recordingWriter captures the statements the projection produces. Asserting on
// emitted SQL rather than against a database is what the rest of the repo does,
// and it pins the parts that actually matter here: whether a write is an upsert,
// which columns it is entitled to overwrite, and whether a second identical event
// changes anything.
type recordingWriter struct {
	stmts []string
}

func (w *recordingWriter) Consume(stmt string) error {
	w.stmts = append(w.stmts, stmt)
	return nil
}

func (w *recordingWriter) last() string {
	if len(w.stmts) == 0 {
		return ""
	}
	return w.stmts[len(w.stmts)-1]
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
		at:      time.Date(2026, 8, 11, 20, 30, 0, 0, time.UTC),
	}
}

// msgAt is for the one test that cares about the clock: a message published in a
// different year from the one in its subject.
func msgAt(subj string, body any, at time.Time) stream.Message {
	return fakeMessage{subject: subject.FromStr(subj), body: body, at: at}
}

func handle(t *testing.T, c *consumer, m stream.Message) {
	t.Helper()
	if err := c.HandleMessage(m); err != nil {
		t.Fatalf("HandleMessage(%s): %v", m.Subject().Subject(), err)
	}
}

// --- tests ---

// The event that makes the whole feature possible: racing is derived from a patrol
// starting, with no new producer anywhere in the platform.
func TestTeamStartedMakesEveryStartingMemberRacing(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.patrulje.team-1.started", messages.NathejkTeamStarted{
		TeamID: "team-1",
		Members: []messages.NathejkTeamStarted_Member{
			{MemberID: "m-1"},
			{MemberID: "m-2"},
		},
	}))

	if len(w.stmts) != 2 {
		t.Fatalf("got %d statements, want one per starting member: %v", len(w.stmts), w.stmts)
	}
	for _, stmt := range w.stmts {
		if !strings.Contains(stmt, "spejderstatus") {
			t.Errorf("statement does not write spejderstatus: %s", stmt)
		}
		if !strings.Contains(stmt, "racing") {
			t.Errorf("starting member not set racing: %s", stmt)
		}
		if !strings.Contains(stmt, "ON DUPLICATE KEY UPDATE") {
			t.Errorf("write is not an upsert, so replay would fail: %s", stmt)
		}
	}
	if !strings.Contains(w.stmts[0], "'m-1'") || !strings.Contains(w.stmts[1], "'m-2'") {
		t.Errorf("member ids not taken from the body: %v", w.stmts)
	}
}

// A member with no id is skipped rather than written as a row keyed on the empty
// string — which would collide with every other such member in the year, since the
// primary key is (year, id).
func TestTeamStartedSkipsMembersWithNoID(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.patrulje.team-1.started", messages.NathejkTeamStarted{
		TeamID:  "team-1",
		Members: []messages.NathejkTeamStarted_Member{{MemberID: ""}, {MemberID: "m-1"}},
	}))

	if len(w.stmts) != 1 {
		t.Fatalf("got %d statements, want 1: %v", len(w.stmts), w.stmts)
	}
}

// **The subtlest rule in the projection.** Replaying the start event after a member
// has been moved must not drag them back to the patrol they began with.
//
// The start event is replayed on every restart, and it legitimately knows the
// member's *initial* team — but nothing about where they are now. So its upsert may
// refresh initialTeamId and nothing else. If currentTeamId were in the update list,
// every restart would silently undo every move made during the event, and the
// strength of two patrols would be wrong with no trace of why.
func TestTeamStartedReplayDoesNotUndoAMove(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.patrulje.team-1.started", messages.NathejkTeamStarted{
		TeamID:  "team-1",
		Members: []messages.NathejkTeamStarted_Member{{MemberID: "m-1"}},
	}))

	stmt := w.last()
	update := stmt[strings.Index(stmt, "ON DUPLICATE KEY UPDATE"):]
	if strings.Contains(update, "currentTeamId") {
		t.Errorf("start event overwrites currentTeamId, so replay would undo moves: %s", update)
	}
	if !strings.Contains(update, "initialTeamId") {
		t.Errorf("start event should refresh initialTeamId: %s", update)
	}
}

// Non-starters are deleted, not left behind. A leftover row inflates its team's
// strength, so the 3-member requirement would be judged against members who never
// turned up.
func TestSpejderDeletedRemovesTheRow(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.spejder.m-1.deleted", nil))

	stmt := w.last()
	if !strings.HasPrefix(strings.ToUpper(stmt), "DELETE") {
		t.Errorf("want a DELETE, got: %s", stmt)
	}
	if !strings.Contains(stmt, "'m-1'") || !strings.Contains(stmt, "'2026'") {
		t.Errorf("delete is not scoped to the member and year: %s", stmt)
	}
}

// Every lifecycle event resolves to its own status through the body, so one code
// path serves all of them — including the three this repo does not publish.
func TestLifecycleEventsWriteTheirStatus(t *testing.T) {
	tests := []struct {
		subject string
		body    any
		want    types.MemberStatus
	}{
		{"NATHEJK.2026.spejder.m-1.withdrawal.requested", WithdrawalRequested{MemberID: "m-1", TeamID: "team-1"}, types.MemberStatusWaiting},
		{"NATHEJK.2026.spejder.m-1.withdrawal.cancelled", WithdrawalCancelled{MemberID: "m-1", TeamID: "team-1"}, types.MemberStatusRacing},
		{"NATHEJK.2026.spejder.m-1.pickup.accepted", PickupAccepted{MemberID: "m-1", TeamID: "team-1", Car: "bil-3"}, types.MemberStatusTransit},
		{"NATHEJK.2026.spejder.m-1.shelter.accepted", ShelterAccepted{MemberID: "m-1", TeamID: "team-1"}, types.MemberStatusSheltered},
		{"NATHEJK.2026.spejder.m-1.status.overridden", StatusOverridden{MemberID: "m-1", TeamID: "team-1", To: types.MemberStatusSheltered}, types.MemberStatusSheltered},
		{"NATHEJK.2026.spejder.m-1.handover.completed", HandoverCompleted{MemberID: "m-1", TeamID: "team-1", To: types.MemberStatusReleased}, types.MemberStatusReleased},
	}
	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			w := &recordingWriter{}
			c := &consumer{w: w}
			handle(t, c, msg(tt.subject, tt.body))

			stmt := w.last()
			if stmt == "" {
				t.Fatalf("no statement emitted; subject unmatched?")
			}
			if !strings.Contains(stmt, string(tt.want)) {
				t.Errorf("want status %q in: %s", tt.want, stmt)
			}
			if !strings.Contains(stmt, "ON DUPLICATE KEY UPDATE") {
				t.Errorf("not an upsert: %s", stmt)
			}
		})
	}
}

// A move writes currentTeamId and must not touch initialTeamId — the member races
// on with a patrol that is not the one they signed up with, and both facts are
// wanted afterwards.
func TestTeamMovedChangesCurrentTeamOnly(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.spejder.m-1.team.moved", TeamMoved{
		MemberID: "m-1", FromTeamID: "team-1", ToTeamID: "team-2",
	}))

	stmt := w.last()
	if !strings.Contains(stmt, "'team-2'") {
		t.Errorf("destination team not written: %s", stmt)
	}
	update := stmt[strings.Index(stmt, "ON DUPLICATE KEY UPDATE"):]
	if !strings.Contains(update, "currentTeamId") {
		t.Errorf("move does not update currentTeamId: %s", update)
	}
	if strings.Contains(update, "initialTeamId") {
		t.Errorf("move overwrites initialTeamId, losing where the member started: %s", update)
	}
}

// An unrecognised status is refused rather than written, and the refusal is not an
// error: a poison message must not wedge the replay that rebuilds the whole table.
func TestUnknownStatusIsRefusedWithoutFailing(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.spejder.m-1.status.overridden", StatusOverridden{
		MemberID: "m-1", TeamID: "team-1", To: types.MemberStatus("nonsense"),
	}))

	if len(w.stmts) != 0 {
		t.Errorf("wrote an unreadable status into the read model: %v", w.stmts)
	}
}

// Replay is idempotent: the same event twice produces the same statement, so
// rebuilding from history cannot double-count anybody.
func TestReplayIsIdempotent(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	m := msg("NATHEJK.2026.spejder.m-1.withdrawal.requested", WithdrawalRequested{MemberID: "m-1", TeamID: "team-1"})
	handle(t, c, m)
	handle(t, c, m)

	if len(w.stmts) != 2 || w.stmts[0] != w.stmts[1] {
		t.Errorf("replay is not idempotent:\n%s\n%s", w.stmts[0], w.stmts[1])
	}
}

// The year comes from the subject, never from the message clock.
//
// This is the bug the old commented-out projection shipped with: it used
// msg.Time().Year(). Since the table is rebuilt by replaying all of history, a 2026
// event replayed in 2027 would be filed under the wrong year — and every
// year-scoped query, which is all of them, would miss it.
func TestYearComesFromSubjectNotMessageTime(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msgAt(
		"NATHEJK.2026.spejder.m-1.withdrawal.requested",
		WithdrawalRequested{MemberID: "m-1", TeamID: "team-1"},
		time.Date(2027, 3, 1, 12, 0, 0, 0, time.UTC),
	))

	stmt := w.last()
	if !strings.Contains(stmt, "'2026'") {
		t.Errorf("year not taken from the subject: %s", stmt)
	}
	if strings.Contains(stmt, "'2027'") {
		t.Errorf("year taken from the message clock: %s", stmt)
	}
}

// An unknown subject is a logged no-op rather than an error. The car and shelter
// interfaces will add subjects to this domain, and a projection that errored on
// anything unfamiliar would turn a forward-compatible deployment into a stuck
// replay.
func TestUnknownSubjectIsANoOp(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.spejder.m-1.teleported", nil))

	if len(w.stmts) != 0 {
		t.Errorf("unknown subject wrote something: %v", w.stmts)
	}
}

// Every subject the consumer declares must actually be handled.
//
// The two halves drift apart silently: a subject declared but unmatched means the
// projection subscribes to an event and quietly ignores it, which looks exactly
// like the event never being published.
func TestEverySubjectDeclaredIsHandled(t *testing.T) {
	c := &consumer{}
	for _, subj := range c.Consumes() {
		s := strings.ReplaceAll(subj.Subject(), ":", ".")
		s = strings.ReplaceAll(s, "*", "x")
		w := &recordingWriter{}
		c := &consumer{w: w}
		// Bodies differ per subject, so one permissive map stands in for all of
		// them: it carries every field any handler reads, including the members
		// list the start event needs to write anything at all. An unmatched subject
		// falls through to the default case and writes nothing, which is the
		// failure this test is looking for.
		body := map[string]any{
			"memberId": "m-1",
			"teamId":   "t-1",
			"toTeamId": "t-2",
			"to":       "racing",
			"members":  []map[string]any{{"memberId": "m-1"}},
		}
		if err := c.HandleMessage(msg(s, body)); err != nil {
			t.Fatalf("HandleMessage(%s): %v", s, err)
		}
		if len(w.stmts) == 0 {
			t.Errorf("subject %q is declared but not handled", subj.Subject())
		}
	}
}
