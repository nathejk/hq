package track

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
)

// --- fakes ---

// recordingWriter captures the statements the projection produces. Asserting on emitted SQL rather
// than against a database is what the rest of this repo does, and it is a good fit here: the two
// things most worth pinning — that a batch is one insert per chunk, and that the latest-position row
// only ever moves forward — are both visible in the statement text.
type recordingWriter struct {
	stmts []string
}

func (w *recordingWriter) Consume(stmt string) error {
	w.stmts = append(w.stmts, stmt)
	return nil
}

func (w *recordingWriter) matching(substr string) []string {
	var out []string
	for _, s := range w.stmts {
		if strings.Contains(s, substr) {
			out = append(out, s)
		}
	}
	return out
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

// The real subject, from the payload pinned in PRD 011 §4a — uppercase TELEMETRY domain and the
// `track` entity token, both load-bearing.
func reported(body any) stream.Message {
	return fakeMessage{
		subject: subject.FromStr("TELEMETRY.2026.track.f30793d2-5393-4d90-bbfa-cf224bbc131b.reported"),
		body:    body,
		at:      time.Date(2026, 9, 3, 12, 18, 58, 0, time.UTC),
	}
}

func handle(t *testing.T, c *consumer, m stream.Message) {
	t.Helper()
	if err := c.HandleMessage(m); err != nil {
		t.Fatalf("HandleMessage(%s): %v", m.Subject().Subject(), err)
	}
}

// The exact message from PRD 011 §4a, so the parser is pinned against a real payload rather than an
// invented one.
func TestReportedWritesPointAndLatest(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, reported(Reported{
		PersonID: "f30793d2-5393-4d90-bbfa-cf224bbc131b",
		UserType: "gøgler",
		Year:     "2026",
		Points: []Point{
			{Ts: 1788437919856, Lat: 55.70915145018305, Lng: 12.600336777419688, Accuracy: 18.739823543347217},
		},
	}))

	points := w.matching("track_point")
	if len(points) != 1 {
		t.Fatalf("want 1 track_point statement, got %d: %v", len(points), w.stmts)
	}
	// INSERT IGNORE is what makes a client retry and a boot replay harmless.
	if !strings.HasPrefix(points[0], "INSERT IGNORE") {
		t.Errorf("track_point insert must be INSERT IGNORE, got: %s", points[0])
	}
	for _, want := range []string{"1788437919856", "55.709151", "12.6003367", "gøgler", "2026"} {
		if !strings.Contains(points[0], want) {
			t.Errorf("track_point insert missing %q: %s", want, points[0])
		}
	}

	latest := w.matching("track_latest")
	if len(latest) != 1 {
		t.Fatalf("want 1 track_latest statement, got %d", len(latest))
	}
	if !strings.Contains(latest[0], "ON DUPLICATE KEY UPDATE") {
		t.Errorf("track_latest must upsert, got: %s", latest[0])
	}
}

// Points arrive batched, and the latest-position row must follow the newest point in the batch —
// not the last one in the array, which is not the same thing when a backlog is flushed out of
// order.
func TestLatestFollowsNewestPointInBatch(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, reported(Reported{
		PersonID: "p-1",
		UserType: "spejder",
		Year:     "2026",
		Points: []Point{
			{Ts: 3000, Lat: 55.3, Lng: 12.3},
			{Ts: 9000, Lat: 55.9, Lng: 12.9}, // newest
			{Ts: 5000, Lat: 55.5, Lng: 12.5},
		},
	}))

	latest := w.matching("track_latest")[0]
	if !strings.Contains(latest, "9000") {
		t.Errorf("latest should carry the newest ts (9000): %s", latest)
	}
	if !strings.Contains(latest, "55.9") {
		t.Errorf("latest should carry the newest point's position: %s", latest)
	}
}

// The guard that stops a boot replay leaving "last seen" showing whichever message happened to be
// applied last. Every column is conditional on the incoming point being newer, so an older message
// arriving afterwards is a no-op rather than a regression.
func TestLatestOnlyAdvancesForNewerPoints(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, reported(Reported{
		PersonID: "p-1",
		Year:     "2026",
		Points:   []Point{{Ts: 5000, Lat: 55.5, Lng: 12.5}},
	}))

	latest := w.matching("track_latest")[0]
	for _, col := range []string{"latitude", "longitude", "accuracy", "personType", "year", "ts"} {
		want := "IF(VALUES(ts) > ts, VALUES(" + col + "), " + col + ")"
		if !strings.Contains(latest, want) {
			t.Errorf("update of %q must be guarded by %q, got: %s", col, want, latest)
		}
	}
}

// A batch whose every point was junk is legitimate: the producer drops bad points and keeps the
// batch, so that one bad reading cannot poison a member's whole track behind a retry loop. Such a
// message has nothing to write, and writing nothing is the correct response — in particular
// `advanceLatest` must not index into an empty slice.
func TestEmptyBatchWritesNothing(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, reported(Reported{PersonID: "p-1", Year: "2026", Points: nil}))
	handle(t, c, reported(Reported{PersonID: "", Year: "2026", Points: []Point{{Ts: 1, Lat: 1, Lng: 1}}}))

	if len(w.stmts) != 0 {
		t.Errorf("want no statements, got %v", w.stmts)
	}
}

// One statement per chunk, so a 2,000-point backlog does not become one enormous string.
func TestLargeBatchIsChunked(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	points := make([]Point, insertChunk*2+1)
	for i := range points {
		points[i] = Point{Ts: int64(i + 1), Lat: 55.5, Lng: 12.5}
	}
	handle(t, c, reported(Reported{PersonID: "p-1", Year: "2026", Points: points}))

	if got := len(w.matching("track_point")); got != 3 {
		t.Errorf("want 3 chunked inserts for %d points, got %d", len(points), got)
	}
	// Still exactly one latest-row write, however many points arrived.
	if got := len(w.matching("track_latest")); got != 1 {
		t.Errorf("want 1 track_latest statement, got %d", got)
	}
}

// The subject is the whole reason this projection reads a second JetStream stream, so it is pinned
// rather than assumed: the domain must be TELEMETRY and the entity token `track`.
func TestConsumesTelemetryTrackSubject(t *testing.T) {
	c := &consumer{}
	subs := c.Consumes()
	if len(subs) != 1 {
		t.Fatalf("want exactly 1 subject, got %d", len(subs))
	}
	if got := subs[0].Subject(); got != "TELEMETRY.*.track.*.reported" {
		t.Errorf("Consumes() = %q, want TELEMETRY.*.track.*.reported", got)
	}
	if got := subs[0].Domain(); got != "TELEMETRY" {
		t.Errorf("domain = %q, want TELEMETRY — the stream name is derived from it", got)
	}
}

// An off-convention subject must not panic or write; it is logged and ignored, like every other
// projection in this repo.
func TestUnhandledSubjectIsIgnored(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	m := fakeMessage{subject: subject.FromStr("TELEMETRY.2026.track.p-1.somethingelse")}
	if err := c.HandleMessage(m); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(w.stmts) != 0 {
		t.Errorf("want no statements, got %v", w.stmts)
	}
}
