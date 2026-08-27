package dispatch

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
)

// --- fakes ---

// recordingWriter captures the statements the projection produces. Testing the emitted
// SQL rather than a database is what the rest of the repo does, and it is the part worth
// pinning: whether the statement filters on the right key, and whether a replayed event
// adds a row.
type recordingWriter struct {
	stmts []string
}

func (w *recordingWriter) Consume(stmt string) error {
	w.stmts = append(w.stmts, stmt)
	return nil
}

type fakeMessage struct {
	subject stream.Subject
	body    any
	seq     uint64
	at      time.Time
}

func (m fakeMessage) Subject() stream.Subject { return m.subject }
func (m fakeMessage) Time() time.Time         { return m.at }
func (m fakeMessage) Sequence() uint64        { return m.seq }
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

func msg(subj string, seq uint64, body any) stream.Message {
	return fakeMessage{
		subject: subject.FromStr(subj),
		body:    body,
		seq:     seq,
		at:      time.Date(2026, 8, 27, 20, 30, 0, 0, time.UTC),
	}
}

func handle(t *testing.T, c *consumer, m stream.Message) {
	t.Helper()
	if err := c.HandleMessage(m); err != nil {
		t.Fatalf("HandleMessage(%s): %v", m.Subject().Subject(), err)
	}
}

// --- tests ---

func TestDispatchableSetInsertsTheSection(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.dispatch.section.bil-2.dispatchable", 3,
		SectionDispatchableSet{SectionSlug: "bil-2", Dispatchable: true}))

	if len(w.stmts) != 1 {
		t.Fatalf("got %d statements, want 1: %v", len(w.stmts), w.stmts)
	}
	if !strings.Contains(w.stmts[0], "INTO `dispatchable_section`") {
		t.Errorf("not an insert into dispatchable_section: %s", w.stmts[0])
	}
	if !strings.Contains(w.stmts[0], "'bil-2'") {
		t.Errorf("slug not taken from the subject: %s", w.stmts[0])
	}
	// The year must come from the subject, not from the message timestamp: replay across
	// a year boundary would otherwise file an old flag under the current year.
	if !strings.Contains(w.stmts[0], "'2026'") {
		t.Errorf("year not taken from the subject: %s", w.stmts[0])
	}
}

func TestReplayingASetDoesNotDuplicateTheRow(t *testing.T) {
	// A replay re-delivers the original event on every boot. The insert must therefore
	// tolerate the row existing, and must not move setAt — nothing reads that column as
	// anything but "since when".
	w := &recordingWriter{}
	c := &consumer{w: w}

	m := msg("NATHEJK.2026.dispatch.section.bil-2.dispatchable", 3,
		SectionDispatchableSet{SectionSlug: "bil-2", Dispatchable: true})
	handle(t, c, m)
	handle(t, c, m)

	for _, stmt := range w.stmts {
		if !strings.Contains(stmt, "ON DUPLICATE KEY UPDATE") && !strings.Contains(stmt, "IGNORE") {
			t.Errorf("insert does not tolerate an existing row: %s", stmt)
		}
		if strings.Contains(stmt, "setAt`=") {
			t.Errorf("replay updates setAt, which would move the timestamp: %s", stmt)
		}
	}
}

func TestDispatchableUnsetDeletesTheSection(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.dispatch.section.bil-2.dispatchable", 4,
		SectionDispatchableSet{SectionSlug: "bil-2", Dispatchable: false}))

	if len(w.stmts) != 1 {
		t.Fatalf("got %d statements, want 1: %v", len(w.stmts), w.stmts)
	}
	if !strings.HasPrefix(w.stmts[0], "DELETE") {
		t.Errorf("unset is not a delete: %s", w.stmts[0])
	}
	// Scoped to year *and* slug: an event for another year must not clear this one's flag.
	if !strings.Contains(w.stmts[0], "'2026'") || !strings.Contains(w.stmts[0], "'bil-2'") {
		t.Errorf("delete is not scoped to both year and slug: %s", w.stmts[0])
	}
}

func TestConsumesOnlyTheDispatchableSubject(t *testing.T) {
	// The entity token the SPA depends on is derived from position 3 of these patterns,
	// so it is `dispatch` — not the projection's name, and not `section`.
	c := &consumer{}
	subs := c.Consumes()
	if len(subs) != 1 {
		t.Fatalf("got %d subscriptions, want 1: %v", len(subs), subs)
	}
	if got := subs[0].Subject(); !strings.Contains(got, ".dispatch.section.") {
		t.Errorf("subscription %q is not the dispatchable-section subject", got)
	}
}

func TestUnknownSubjectIsIgnored(t *testing.T) {
	// The mux may hand over anything matching a wildcard; an unrecognised subject must
	// be a no-op rather than an error that stalls the ordered consumer.
	w := &recordingWriter{}
	c := &consumer{w: w}

	if err := c.HandleMessage(msg("NATHEJK.2026.dispatch.disp-1.created", 5, struct{}{})); err != nil {
		t.Fatalf("unrecognised subject returned an error: %v", err)
	}
	if len(w.stmts) != 0 {
		t.Errorf("unrecognised subject wrote %d statements: %v", len(w.stmts), w.stmts)
	}
}
