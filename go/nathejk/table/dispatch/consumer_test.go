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

func TestConsumesTheDispatchTourAndDutySubjects(t *testing.T) {
	// The entity tokens the SPA depends on are derived from position 3 of these patterns, so they
	// are `dispatch`, `tour` and `dispatchduty` — not the projection's name, and not `section`.
	// A wildcard in that position would make the advertised set non-exhaustive, so there must not
	// be one.
	c := &consumer{}
	entities := map[string]int{}
	for _, s := range c.Consumes() {
		parts := strings.Split(s.Subject(), ".")
		if len(parts) < 3 {
			t.Fatalf("subject %q is off-convention", s.Subject())
		}
		entities[parts[2]]++
	}
	for _, want := range []string{"dispatch", "tour", "dispatchduty"} {
		if entities[want] == 0 {
			t.Errorf("no subscription emits %q; tokens are %v", want, entities)
		}
	}
	if len(entities) != 3 {
		t.Errorf("entity tokens %v, want exactly three", entities)
	}
}

func TestUnknownSubjectIsIgnored(t *testing.T) {
	// The mux may hand over anything matching a wildcard; an unrecognised subject must
	// be a no-op rather than an error that stalls the ordered consumer.
	w := &recordingWriter{}
	c := &consumer{w: w}

	if err := c.HandleMessage(msg("NATHEJK.2026.dispatch.disp-1.frobnicated", 5, struct{}{})); err != nil {
		t.Fatalf("unrecognised subject returned an error: %v", err)
	}
	if len(w.stmts) != 0 {
		t.Errorf("unrecognised subject wrote %d statements: %v", len(w.stmts), w.stmts)
	}
}

// --- tasks and tours (task 109) ---

func TestTaskCreatedInsertsTheTaskAndATimelineEntry(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.dispatch.disp-1.created", 11, TaskCreated{
		TaskID:      "disp-1",
		Kind:        KindPickup,
		Priority:    PriorityRed,
		Description: "spejder kan ikke gå videre",
		Pickup:      Place{Kind: PlaceText, Label: "ved Post 2B"},
		Dropoff:     Place{Kind: PlaceHQ, Label: "HQ"},
		CreatedUts:  1787862000,
	}))

	if len(w.stmts) != 2 {
		t.Fatalf("got %d statements, want 2 (task row + timeline entry): %v", len(w.stmts), w.stmts)
	}
	if !strings.Contains(w.stmts[0], "INTO `dispatch_task`") {
		t.Errorf("first statement is not an insert into dispatch_task: %s", w.stmts[0])
	}
	if !strings.Contains(w.stmts[0], "ON DUPLICATE KEY UPDATE") {
		t.Errorf("task insert is not an upsert, so replay would fail on the existing row: %s", w.stmts[0])
	}
	// The waiting clock comes from the event, not the message time: a task backdated by an
	// operator must wait from when they say it started.
	if !strings.Contains(w.stmts[0], "1787862000") {
		t.Errorf("createdUts not taken from the event body: %s", w.stmts[0])
	}
	if !strings.Contains(w.stmts[1], "`dispatch_activity`") {
		t.Errorf("second statement is not a timeline insert: %s", w.stmts[1])
	}
	// Keyed by stream sequence — this is what makes replay rebuild the log rather than
	// multiply it.
	if !strings.Contains(w.stmts[1], "11") {
		t.Errorf("timeline entry not keyed by stream sequence: %s", w.stmts[1])
	}
}

func TestReplayingCreatedDoesNotResetState(t *testing.T) {
	// The bug this table could most easily have had. Replay re-delivers `created` on every
	// boot, *after* the transitions that came later. With `state` in the upsert's update
	// list, every finished task would silently return to the queue on restart — and the
	// board would look like a night nobody had worked.
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.dispatch.disp-1.created", 11, TaskCreated{
		TaskID: "disp-1", Kind: KindDelivery, CreatedUts: 1787862000,
	}))

	if strings.Contains(w.stmts[0], "`state`=VALUES(state)") {
		t.Errorf("replaying created would reset state: %s", w.stmts[0])
	}
	for _, col := range []string{"doneUts", "pickedUpUts", "cancelledUts", "cancelReason"} {
		if strings.Contains(w.stmts[0], "`"+col+"`=VALUES("+col+")") {
			t.Errorf("replaying created would clear %s: %s", col, w.stmts[0])
		}
	}
}

func TestMemberlessTaskStoresAnEmptyListNotNull(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.dispatch.disp-1.created", 11, TaskCreated{
		TaskID: "disp-1", Kind: KindTransport, CreatedUts: 1,
	}))

	if strings.Contains(w.stmts[0], "'null'") {
		t.Errorf("memberIds stored as the string null: %s", w.stmts[0])
	}
	if !strings.Contains(w.stmts[0], "'[]'") {
		t.Errorf("memberIds not stored as an empty list: %s", w.stmts[0])
	}
}

func TestUnplanningDoesNotTouchTheWaitingClock(t *testing.T) {
	// PRD 009 §5: a task dropped from a tour returns to the queue with its original clock
	// intact, because the scout has been waiting since the call and not since the re-plan.
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.dispatch.disp-1.unplanned", 21, TaskUnplanned{
		TaskID: "disp-1", TourID: "tour-1",
	}))

	if !strings.Contains(w.stmts[0], "'queued'") {
		t.Errorf("unplanned does not return the task to the queue: %s", w.stmts[0])
	}
	if strings.Contains(w.stmts[0], "createdUts") {
		t.Errorf("unplanned writes createdUts, which would reset the waiting clock: %s", w.stmts[0])
	}
}

func TestPickedUpRecordsTheMomentWithoutFinishingTheTask(t *testing.T) {
	// Custody changes when people get in the car; the task finishes when they are
	// delivered. Collapsing the two would make Hønsegården's *På vej* a list of people who
	// have already arrived.
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.dispatch.disp-1.pickedup", 31, TaskPickedUp{
		TaskID: "disp-1", SectionSlug: "bil-2", AtUts: 1787865300,
	}))

	if !strings.Contains(w.stmts[0], "pickedUpUts") || !strings.Contains(w.stmts[0], "1787865300") {
		t.Errorf("pickedup does not record the moment: %s", w.stmts[0])
	}
	if strings.Contains(w.stmts[0], "'done'") {
		t.Errorf("pickedup completes the task: %s", w.stmts[0])
	}
	// The unit is on the timeline, because "which car has my scout" is the question the
	// whole feature exists to answer.
	if !strings.Contains(w.stmts[1], "bil-2") {
		t.Errorf("the unit is not on the timeline entry: %s", w.stmts[1])
	}
}

func TestTaskWritesAreScopedToTheYear(t *testing.T) {
	// An event published for another year must not reach this year's row.
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.dispatch.disp-1.completed", 41, TaskCompleted{TaskID: "disp-1", AtUts: 5}))

	if !strings.Contains(w.stmts[0], "'2026'") || !strings.Contains(w.stmts[0], "'disp-1'") {
		t.Errorf("update is not scoped to both id and year: %s", w.stmts[0])
	}
}

func TestCancellingKeepsTheReason(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.dispatch.disp-1.cancelled", 51, TaskCancelled{
		TaskID: "disp-1", Reason: "spejderen fortsatte alligevel", AtUts: 9,
	}))

	if !strings.Contains(w.stmts[0], "fortsatte") {
		t.Errorf("the reason is not stored on the task: %s", w.stmts[0])
	}
	// And on the timeline, which is the part a handover reads.
	if !strings.Contains(w.stmts[1], "fortsatte") {
		t.Errorf("the reason is not on the timeline: %s", w.stmts[1])
	}
}

func TestStopsChangedRebuildsTheWholeList(t *testing.T) {
	w := &recordingWriter{}
	c := &consumer{w: w}

	visited := int64(1787862600)
	planned := int64(1787864100)
	handle(t, c, msg("NATHEJK.2026.tour.tour-1.stops.changed", 61, StopsChanged{
		TourID: "tour-1",
		Stops: []Stop{
			{StopID: "stop-a", Place: Place{Kind: PlaceCheckpoint, RefID: "cp-2a", Label: "Post 2A"},
				VisitedUts: &visited, Tasks: []StopTask{{TaskID: "disp-9", Role: RoleLoad}}},
			{StopID: "stop-b", Place: Place{Kind: PlaceText, Label: "ved Post 2B"},
				PlannedUts: &planned, Override: true,
				Tasks: []StopTask{{TaskID: "disp-1", Role: RoleLoad}, {TaskID: "disp-9", Role: RoleUnload}}},
		},
	}))

	// Delete-then-insert: a diff can leave behind a stop the new plan does not mention,
	// and the desk would then see a stop that is not on the tour.
	if !strings.HasPrefix(w.stmts[0], "DELETE") || !strings.Contains(w.stmts[0], "dispatch_stop_task") {
		t.Errorf("stop tasks are not cleared first: %s", w.stmts[0])
	}
	if !strings.HasPrefix(w.stmts[1], "DELETE") || !strings.Contains(w.stmts[1], "`dispatch_stop`") {
		t.Errorf("stops are not cleared: %s", w.stmts[1])
	}
	joined := strings.Join(w.stmts, "\n")
	// A visited stop survives a whole-list replacement, because the event carries its
	// visit. Without that, re-planning a running tour would un-visit its history.
	if !strings.Contains(joined, "1787862600") {
		t.Errorf("visitedUts lost in the rebuild: %s", joined)
	}
	// The slice order is the order: sortOrder is written from the index, so there is one
	// source of truth for the ordering.
	if !strings.Contains(joined, "'stop-a'") || !strings.Contains(joined, "'stop-b'") {
		t.Errorf("stops not inserted: %s", joined)
	}
	if strings.Count(joined, "'disp-9'") != 2 {
		t.Errorf("the task that is loaded and unloaded does not occupy two stops: %s", joined)
	}
	if !strings.Contains(joined, "`dispatch_activity`") {
		t.Errorf("no timeline entry for the re-plan: %s", joined)
	}
}

func TestStopVisitedIsScopedToItsOwnTour(t *testing.T) {
	// Stop ids are minted, but the scoping is what stops an event published for another
	// tour from marking this one's stop visited.
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.tour.tour-1.stop.visited", 71, StopVisited{
		TourID: "tour-1", StopID: "stop-b", AtUts: 1787864200,
	}))

	if !strings.Contains(w.stmts[0], "'stop-b'") || !strings.Contains(w.stmts[0], "'tour-1'") {
		t.Errorf("visit is not scoped to the stop and its tour: %s", w.stmts[0])
	}
}

func TestTourEventsLandOnTheTourTimelineNotTheTaskTimeline(t *testing.T) {
	// The two columns are what make "this task's history" and "this tour's history" both
	// indexed lookups. A tour event filed under taskId would appear on a task's timeline
	// as an entry about something else.
	w := &recordingWriter{}
	c := &consumer{w: w}

	handle(t, c, msg("NATHEJK.2026.tour.tour-1.underway", 81, TourUnderway{TourID: "tour-1", AtUts: 3}))

	last := w.stmts[len(w.stmts)-1]
	if !strings.Contains(last, "`tourId`") || !strings.Contains(last, "'tour-1'") {
		t.Errorf("tour event not filed under tourId: %s", last)
	}
	if !strings.Contains(last, "'tour.underway'") {
		t.Errorf("unexpected activity type: %s", last)
	}
}

func TestReplayingTheWholeHistoryTwiceProducesTheSameStatements(t *testing.T) {
	// The property that matters on every boot: the projection is rebuilt from the log, so
	// two passes must produce identical SQL. Anything that depended on current state — a
	// counter, a computed order, an appended list — would drift here.
	history := []stream.Message{
		msg("NATHEJK.2026.dispatch.disp-1.created", 1, TaskCreated{TaskID: "disp-1", Kind: KindPickup, CreatedUts: 100}),
		msg("NATHEJK.2026.tour.tour-1.created", 2, TourCreated{TourID: "tour-1", SectionSlug: "bil-2", CreatedUts: 110}),
		msg("NATHEJK.2026.tour.tour-1.stops.changed", 3, StopsChanged{TourID: "tour-1", Stops: []Stop{
			{StopID: "stop-a", Place: Place{Kind: PlaceHQ, Label: "HQ"}, Tasks: []StopTask{{TaskID: "disp-1", Role: RoleUnload}}},
		}}),
		msg("NATHEJK.2026.dispatch.disp-1.planned", 4, TaskPlanned{TaskID: "disp-1", TourID: "tour-1"}),
		msg("NATHEJK.2026.tour.tour-1.underway", 5, TourUnderway{TourID: "tour-1", AtUts: 120}),
		msg("NATHEJK.2026.tour.tour-1.stop.visited", 6, StopVisited{TourID: "tour-1", StopID: "stop-a", AtUts: 130}),
		msg("NATHEJK.2026.dispatch.disp-1.completed", 7, TaskCompleted{TaskID: "disp-1", AtUts: 130}),
		msg("NATHEJK.2026.tour.tour-1.completed", 8, TourCompleted{TourID: "tour-1", AtUts: 131}),
	}

	first, second := &recordingWriter{}, &recordingWriter{}
	for _, w := range []*recordingWriter{first, second} {
		c := &consumer{w: w}
		for _, m := range history {
			handle(t, c, m)
		}
	}
	if len(first.stmts) != len(second.stmts) {
		t.Fatalf("replay produced %d statements, first pass %d", len(second.stmts), len(first.stmts))
	}
	for i := range first.stmts {
		if first.stmts[i] != second.stmts[i] {
			t.Fatalf("statement %d differs on replay:\n first: %s\nsecond: %s", i, first.stmts[i], second.stmts[i])
		}
	}
}
