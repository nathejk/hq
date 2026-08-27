package dispatch

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jrgensen/stream"
	"github.com/nathejk/shared-go/types"
)

// --- fakes ---

// recordingPublisher captures the subjects published, which is what the dirty-check
// tests are really about: how many events a toggle produces, and which.
type recordingPublisher struct {
	subjects []string
	bodies   []any
}

func (p *recordingPublisher) MessageFunc() stream.MessageFunc {
	return func(s stream.Subject) stream.MutableMessage {
		return &mutableMessage{subject: s}
	}
}

func (p *recordingPublisher) Publish(m stream.Message) error {
	p.subjects = append(p.subjects, m.Subject().Subject())
	p.bodies = append(p.bodies, m.RawBody())
	return nil
}

type mutableMessage struct {
	subject stream.Subject
	body    any
	meta    any
	at      time.Time
}

func (m *mutableMessage) Subject() stream.Subject     { return m.subject }
func (m *mutableMessage) Time() time.Time             { return m.at }
func (m *mutableMessage) Sequence() uint64            { return 0 }
func (m *mutableMessage) Body(any) error              { return nil }
func (m *mutableMessage) Meta(any) error              { return nil }
func (m *mutableMessage) RawBody() any                { return m.body }
func (m *mutableMessage) RawMeta() any                { return m.meta }
func (m *mutableMessage) SetSubject(s stream.Subject) { m.subject = s }
func (m *mutableMessage) SetBody(b any) error         { m.body = b; return nil }
func (m *mutableMessage) SetMeta(v any) error         { m.meta = v; return nil }
func (m *mutableMessage) SetTime(t time.Time) error   { m.at = t; return nil }

// stubQueries stands in for the read model the commander dirty-checks against. Only the
// methods a command actually reads through return anything; the rest exist to satisfy
// Queries, and returning zero values from them is what makes a command that unexpectedly
// starts reading one show up as a failing test rather than as a plausible answer.
type stubQueries struct {
	dispatchable []types.Slug
	tasks        []*Task
	task         *Task
	tours        []*Tour
	tour         *Tour
	duty         []Duty
	err          error
}

func (q stubQueries) DispatchableSections(context.Context, types.YearSlug) ([]types.Slug, error) {
	return q.dispatchable, q.err
}

func (q stubQueries) Tasks(context.Context, Filter) ([]*Task, error) { return q.tasks, q.err }

func (q stubQueries) GetTask(context.Context, types.YearSlug, TaskID) (*Task, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.task, nil
}

func (q stubQueries) Tours(context.Context, TourFilter) ([]*Tour, error) { return q.tours, q.err }

func (q stubQueries) GetTour(context.Context, types.YearSlug, TourID) (*Tour, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.tour, nil
}

func (q stubQueries) StopsByTask(context.Context, types.YearSlug, []TaskID) (map[TaskID][]TaskStop, error) {
	return map[TaskID][]TaskStop{}, q.err
}

func (q stubQueries) DutyWindows(context.Context, types.YearSlug) ([]Duty, error) {
	return q.duty, q.err
}

// --- tests ---

func TestSetSectionDispatchablePublishesOnTheSubjectAndYear(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{}}

	if err := c.SetSectionDispatchable(context.Background(), Actor{}, "2026", "bil-2", true); err != nil {
		t.Fatalf("SetSectionDispatchable: %v", err)
	}
	if len(p.subjects) != 1 {
		t.Fatalf("got %d events, want 1: %v", len(p.subjects), p.subjects)
	}
	if want := "NATHEJK.2026.dispatch.section.bil-2.dispatchable"; p.subjects[0] != want {
		t.Errorf("subject %q, want %q", p.subjects[0], want)
	}
	body, ok := p.bodies[0].(*SectionDispatchableSet)
	if !ok {
		t.Fatalf("body is %T, want *SectionDispatchableSet", p.bodies[0])
	}
	// The new state is on the body rather than in two subjects, so a consumer cannot
	// handle one half of the fact and silently miss the other.
	if !body.Dispatchable || body.SectionSlug != "bil-2" {
		t.Errorf("body %+v does not carry the new state", body)
	}
}

func TestSettingWhatIsAlreadyTruePublishesNothing(t *testing.T) {
	// The Organisation page sends the state it wants without first working out whether
	// that is already the case, so the dirty-check is what keeps the log clean.
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{dispatchable: []types.Slug{"bil-1", "bil-2"}}}

	if err := c.SetSectionDispatchable(context.Background(), Actor{}, "2026", "bil-2", true); err != nil {
		t.Fatalf("SetSectionDispatchable: %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestUnsettingWhatIsAlreadyFalsePublishesNothing(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{dispatchable: []types.Slug{"bil-1"}}}

	if err := c.SetSectionDispatchable(context.Background(), Actor{}, "2026", "bil-2", false); err != nil {
		t.Fatalf("SetSectionDispatchable: %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v, want nothing", p.subjects)
	}
}

func TestUnsettingAnEnabledSectionPublishes(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{dispatchable: []types.Slug{"bil-2"}}}

	if err := c.SetSectionDispatchable(context.Background(), Actor{}, "2026", "bil-2", false); err != nil {
		t.Fatalf("SetSectionDispatchable: %v", err)
	}
	if len(p.subjects) != 1 {
		t.Fatalf("got %d events, want 1: %v", len(p.subjects), p.subjects)
	}
	if body := p.bodies[0].(*SectionDispatchableSet); body.Dispatchable {
		t.Errorf("body says dispatchable, want false")
	}
}

func TestAnInvalidSlugIsRejectedBeforePublishing(t *testing.T) {
	// A slug that cannot round-trip would produce a subject nothing can route and a row
	// nobody can turn off from the UI.
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{}}

	if err := c.SetSectionDispatchable(context.Background(), Actor{}, "2026", "Bil 2!", true); err == nil {
		t.Fatal("an invalid slug was accepted")
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v despite an invalid slug", p.subjects)
	}
}

// --- duty windows (task 115) ---

func TestSetDutyMintsAnIDAndPublishesTheWholeWindow(t *testing.T) {
	// A whole window per event rather than a start and a later end: the roster is agreed in
	// advance, so both ends are known, and a half-open window would make "who is on now" depend
	// on an event that may never arrive.
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{}}

	id, err := c.SetDuty(context.Background(), Actor{}, "2026", "", "bil-2", 1000, 2000)
	if err != nil {
		t.Fatalf("SetDuty: %v", err)
	}
	if id == "" {
		t.Fatal("no id minted")
	}
	if len(p.subjects) != 1 || !strings.HasSuffix(p.subjects[0], ".set") {
		t.Fatalf("published %v, want one dispatchduty set", p.subjects)
	}
	if !strings.Contains(p.subjects[0], ".dispatchduty.") {
		t.Errorf("subject %q is not on the dispatchduty entity", p.subjects[0])
	}
	body := p.bodies[0].(*DutySet)
	if body.StartUts != 1000 || body.EndUts != 2000 || body.SectionSlug != "bil-2" {
		t.Errorf("body %+v does not carry the window", body)
	}
}

func TestSetDutyWithAnIDEditsThatWindow(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{}}

	id, err := c.SetDuty(context.Background(), Actor{}, "2026", "dispatchduty-1", "bil-2", 1000, 2000)
	if err != nil {
		t.Fatalf("SetDuty: %v", err)
	}
	if id != "dispatchduty-1" {
		t.Errorf("id = %q, want the one given", id)
	}
	if !strings.Contains(p.subjects[0], "dispatchduty-1") {
		t.Errorf("subject %q is not the edited window", p.subjects[0])
	}
}

func TestADutyWindowMustEndAfterItStarts(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{}}

	for _, window := range [][2]int64{{2000, 1000}, {1000, 1000}} {
		if _, err := c.SetDuty(context.Background(), Actor{}, "2026", "", "bil-2", window[0], window[1]); err != ErrBadWindow {
			t.Errorf("window %v: err = %v, want ErrBadWindow", window, err)
		}
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v despite a bad window", p.subjects)
	}
}

func TestADutyWindowNeedsAUnit(t *testing.T) {
	// Per unit, not per person (PRD 009 §6): the unit is what is available or asleep.
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{}}

	if _, err := c.SetDuty(context.Background(), Actor{}, "2026", "", "", 1000, 2000); err != ErrUnitRequired {
		t.Fatalf("err = %v, want ErrUnitRequired", err)
	}
}

func TestOverlappingWindowsAreAllowed(t *testing.T) {
	// A unit rostered twice over the same hour is on duty either way, and refusing it would block
	// an operator fixing a roster in whatever order makes sense to them.
	p := &recordingPublisher{}
	c := commander{p: p, q: stubQueries{duty: []Duty{
		{ID: "dispatchduty-1", SectionSlug: "bil-2", StartUts: 1000, EndUts: 3000},
	}}}

	if _, err := c.SetDuty(context.Background(), Actor{}, "2026", "", "bil-2", 2000, 4000); err != nil {
		t.Fatalf("SetDuty over an existing window: %v", err)
	}
	if len(p.subjects) != 1 {
		t.Errorf("published %v, want the overlapping window", p.subjects)
	}
}

func TestDutyCoversIsHalfOpen(t *testing.T) {
	// A shift ending at 22.00 does not include 22.00, so two consecutive windows do not both
	// claim the same minute — which would answer "who is on now" twice for one unit and read as
	// a configuration error on the capacity strip.
	d := Duty{StartUts: 1000, EndUts: 2000}
	if !d.Covers(1000) {
		t.Error("the first second of a window is not covered")
	}
	if !d.Covers(1999) {
		t.Error("the last second before the end is not covered")
	}
	if d.Covers(2000) {
		t.Error("the end instant is covered, so consecutive windows overlap")
	}
	if d.Covers(999) {
		t.Error("an instant before the window is covered")
	}
}
