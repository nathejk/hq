package sos

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
)

// --- fakes ---

// recordingWriter captures the statements the projection produces. Testing the
// emitted SQL rather than a database is what the rest of the repo does, and it is
// the part worth pinning: whether the statement is an upsert, whether it filters
// on the right key, whether a second identical event adds a row.
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
		at:      time.Date(2026, 8, 11, 20, 30, 0, 0, time.UTC),
	}
}

func handle(t *testing.T, c *consumer, m stream.Message) {
	t.Helper()
	if err := c.HandleMessage(m); err != nil {
		t.Fatalf("HandleMessage(%s): %v", m.Subject().Subject(), err)
	}
}

// --- tests ---

func TestCreatedInsertsCaseAndTimelineEntry(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.sos.sos-1.created", 7, Created{
		SosID:       "sos-1",
		Headline:    "Forstuvet ankel ved post 4",
		Description: "Ringer fra stien",
	}))

	if len(w.stmts) != 2 {
		t.Fatalf("got %d statements, want 2 (case row + timeline entry): %v", len(w.stmts), w.stmts)
	}
	if !strings.Contains(w.stmts[0], "INTO `sos`") {
		t.Errorf("first statement is not an insert into sos: %s", w.stmts[0])
	}
	// Upsert, not a plain insert: replay must not fail on the row already existing.
	if !strings.Contains(w.stmts[0], "ON DUPLICATE KEY UPDATE") {
		t.Errorf("case insert is not an upsert: %s", w.stmts[0])
	}
	// The year must come from the subject, not from the message timestamp: replay
	// across a year boundary would otherwise file old cases under the current year.
	if !strings.Contains(w.stmts[0], "'2026'") {
		t.Errorf("year not taken from the subject: %s", w.stmts[0])
	}
	if !strings.Contains(w.stmts[1], "`sos_activity`") {
		t.Errorf("second statement is not a timeline insert: %s", w.stmts[1])
	}
	// Keyed by stream sequence — this is what makes replay idempotent.
	if !strings.Contains(w.stmts[1], "7") {
		t.Errorf("timeline entry not keyed by stream sequence: %s", w.stmts[1])
	}
}

func TestEveryEventAdvancesLastActivity(t *testing.T) {
	// The list sorts by lastActivityAt, so an event that failed to advance it would
	// make a case that just changed look untouched — which is precisely the case an
	// operator needs to see first.
	cases := []struct {
		subject string
		body    any
	}{
		{"NATHEJK.2026.sos.sos-1.created", Created{SosID: "sos-1", Headline: "h", Description: "d"}},
		{"NATHEJK.2026.sos.sos-1.headline.updated", HeadlineUpdated{SosID: "sos-1", Headline: "h2"}},
		{"NATHEJK.2026.sos.sos-1.description.updated", DescriptionUpdated{SosID: "sos-1", Description: "d2"}},
		{"NATHEJK.2026.sos.sos-1.commented", Commented{SosID: "sos-1", CommentID: "c1", Comment: "hej"}},
		{"NATHEJK.2026.sos.sos-1.comment.updated", CommentUpdated{SosID: "sos-1", CommentID: "c1", Comment: "hej igen"}},
		{"NATHEJK.2026.sos.sos-1.severity.specified", SeveritySpecified{SosID: "sos-1", Severity: SeverityRed}},
		{"NATHEJK.2026.sos.sos-1.assigned", Assigned{SosID: "sos-1", SectionSlug: "samarit"}},
		{"NATHEJK.2026.sos.sos-1.closed", Closed{SosID: "sos-1"}},
		{"NATHEJK.2026.sos.sos-1.reopened", Reopened{SosID: "sos-1"}},
		{"NATHEJK.2026.sos.sos-1.deleted", Deleted{SosID: "sos-1"}},
		{"NATHEJK.2026.sos.sos-1.team.associated", TeamAssociated{SosID: "sos-1", TeamID: "team-1"}},
		{"NATHEJK.2026.sos.sos-1.team.disassociated", TeamDisassociated{SosID: "sos-1", TeamID: "team-1"}},
	}

	for _, tc := range cases {
		w := &recordingWriter{}
		c := &consumer{w: w}
		handle(t, c, msg(tc.subject, 1, tc.body))

		all := strings.Join(w.stmts, "\n")
		if !strings.Contains(all, "lastActivityAt") {
			t.Errorf("%s did not touch lastActivityAt: %s", tc.subject, all)
		}
		if !strings.Contains(all, "`sos_activity`") {
			t.Errorf("%s produced no timeline entry: %s", tc.subject, all)
		}
	}
}

func TestReplayIsIdempotent(t *testing.T) {
	// Tables are created with CREATE TABLE IF NOT EXISTS and the stream is replayed
	// on every restart, so the same event arrives again with the same sequence. The
	// timeline insert must not duplicate.
	w := &recordingWriter{}
	c := &consumer{w: w}
	m := msg("NATHEJK.2026.sos.sos-1.commented", 42, Commented{SosID: "sos-1", CommentID: "c1", Comment: "hej"})

	handle(t, c, m)
	handle(t, c, m)

	for _, stmt := range w.stmts {
		if strings.Contains(stmt, "`sos_activity`") &&
			!strings.Contains(stmt, "ON DUPLICATE KEY UPDATE") &&
			!strings.Contains(stmt, "IGNORE") {
			t.Errorf("timeline insert is not replay-safe: %s", stmt)
		}
	}
}

func TestAssociateIsIdempotent(t *testing.T) {
	// Two operators on the same call will both reach for the patrol; the second must
	// not be an error.
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.sos.sos-1.team.associated", 1,
		TeamAssociated{SosID: "sos-1", TeamID: "team-1"}))

	var insert string
	for _, stmt := range w.stmts {
		if strings.Contains(stmt, "`sos_team`") && !strings.HasPrefix(stmt, "DELETE") {
			insert = stmt
		}
	}
	if insert == "" {
		t.Fatal("no insert into sos_team")
	}
	if !strings.Contains(insert, "ON DUPLICATE KEY UPDATE") && !strings.Contains(insert, "IGNORE") {
		t.Errorf("association insert is not idempotent: %s", insert)
	}
}

func TestDeleteIsSoft(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}
	handle(t, c, msg("NATHEJK.2026.sos.sos-1.deleted", 1, Deleted{SosID: "sos-1"}))

	all := strings.Join(w.stmts, "\n")
	if strings.Contains(strings.ToUpper(all), "DELETE FROM `SOS`") {
		t.Errorf("case was hard-deleted: %s", all)
	}
	if !strings.Contains(all, "deletedAt") {
		t.Errorf("deletedAt not set: %s", all)
	}
}

func TestUnknownSubjectIsIgnored(t *testing.T) {
	// PRD 006 adds subjects to this domain. An unrecognised one must be a no-op
	// rather than an error, or a future producer takes the projection down.
	w := &recordingWriter{}
	c := &consumer{w: w}

	if err := c.HandleMessage(msg("NATHEJK.2026.sos.sos-1.something.new", 1, nil)); err != nil {
		t.Errorf("unknown subject returned an error: %v", err)
	}
	if len(w.stmts) != 0 {
		t.Errorf("unknown subject wrote %d statements, want 0: %v", len(w.stmts), w.stmts)
	}
}

// --- the member lifecycle summaries (PRD 006) ---

// One operation, one timeline entry — however many members it touched.
//
// This is the property the whole N-events-plus-one-summary design exists to get: the
// member events are per member because that is the grain the projection works at, and
// the case entry is per operation because that is the grain an operator reads at.
func TestTeamCollectedIsOneEntryForThreeMembers(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.sos.sos-1.team.collected", 42, TeamCollected{
		SosID:    "sos-1",
		TeamID:   "team-1",
		TeamName: "Ulvene",
		Members: []MemberChange{
			{MemberID: "m-1", Name: "Ida", From: "racing", To: "waiting"},
			{MemberID: "m-2", Name: "Bo", From: "racing", To: "waiting"},
			{MemberID: "m-3", Name: "Sol", From: "racing", To: "waiting"},
		},
		TeamStrength: 0,
	}))

	var entries int
	for _, s := range w.stmts {
		if strings.Contains(s, "sos_activity") {
			entries++
		}
	}
	if entries != 1 {
		t.Fatalf("got %d timeline entries for one collection, want 1: %v", entries, w.stmts)
	}

	all := strings.Join(w.stmts, "\n")
	if !strings.Contains(all, string(ActivityTeamCollected)) {
		t.Errorf("entry is not typed team.collected: %s", all)
	}
	// All three members must be named in the stored summary, or the line cannot say
	// who left without a join.
	for _, name := range []string{"Ida", "Bo", "Sol"} {
		if !strings.Contains(all, name) {
			t.Errorf("member %q missing from the summary: %s", name, all)
		}
	}
	if !strings.Contains(all, "lastActivityAt") {
		t.Errorf("case was not touched, so the list would show it as untouched: %s", all)
	}
}

// The summary is self-contained: statuses, names and the resulting strength are all
// stored, so the line never has to be re-derived from current state.
//
// This is the property that keeps a timeline honest. Storing ids only would mean a
// member moved twice has their *first* move described using their second team — an
// entry that changes meaning after the fact, which is worse than no entry.
func TestMemberSummaryStoresEverythingNeededToRenderIt(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.sos.sos-1.member.status.changed", 43, MemberStatusChanged{
		SosID:    "sos-1",
		TeamID:   "team-1",
		TeamName: "Ulvene",
		Members: []MemberChange{
			{MemberID: "m-1", Name: "Ida", From: "racing", To: "waiting"},
		},
		TeamStrength: 2,
	}))

	all := strings.Join(w.stmts, "\n")
	for _, want := range []string{"Ida", "racing", "waiting", "Ulvene", `\"teamStrength\":2`} {
		if !strings.Contains(all, want) {
			t.Errorf("summary is missing %s: %s", want, all)
		}
	}
}

// Members moved in one operation may have different destinations.
func TestMembersMovedRecordsPerMemberDestinations(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.sos.sos-1.member.moved", 44, MembersMoved{
		SosID:        "sos-1",
		FromTeamID:   "team-1",
		FromTeamName: "Ulvene",
		Members: []MemberMove{
			{MemberID: "m-1", Name: "Ida", ToTeamID: "team-2", ToTeamName: "Bj\u00f8rnene"},
			{MemberID: "m-2", Name: "Bo", ToTeamID: "team-3", ToTeamName: "Ravnene"},
		},
		FromTeamStrength: 0,
	}))

	all := strings.Join(w.stmts, "\n")
	if !strings.Contains(all, string(ActivityMembersMoved)) {
		t.Errorf("entry is not typed member.moved: %s", all)
	}
	// Two different destinations in one entry — the case the flow must not flatten.
	if !strings.Contains(all, "team-2") || !strings.Contains(all, "team-3") {
		t.Errorf("per-member destinations lost: %s", all)
	}
}

// Replaying a summary lands on the same row, like every other entry: the stream
// sequence is the key.
func TestMemberSummaryReplayIsIdempotent(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	m := msg("NATHEJK.2026.sos.sos-1.member.status.changed", 43, MemberStatusChanged{
		SosID: "sos-1", TeamID: "team-1",
		Members: []MemberChange{{MemberID: "m-1", From: "racing", To: "waiting"}},
	})
	handle(t, c, m)
	handle(t, c, m)

	var entries []string
	for _, s := range w.stmts {
		if strings.Contains(s, "sos_activity") {
			entries = append(entries, s)
		}
	}
	if len(entries) != 2 || entries[0] != entries[1] {
		t.Errorf("replay is not idempotent:\n%v", entries)
	}
	if !strings.Contains(entries[0], "43") {
		t.Errorf("entry is not keyed by the stream sequence: %s", entries[0])
	}
}
