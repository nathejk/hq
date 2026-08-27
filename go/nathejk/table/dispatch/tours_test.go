package dispatch

import (
	"context"
	"strings"
	"testing"

	"github.com/nathejk/shared-go/types"
)

// The tour rules (task 111). What is refused, what is warned about, and — mostly — what the
// derived stop times come out as, because that is the number a dispatcher reads down a phone to
// a patrol standing in the dark.

// tourQueries serves one tour, and records nothing.
type tourQueries struct {
	stubQueries
	stops map[TaskID][]TaskStop
}

func (q tourQueries) StopsByTask(_ context.Context, _ types.YearSlug, ids []TaskID) (map[TaskID][]TaskStop, error) {
	out := map[TaskID][]TaskStop{}
	for _, id := range ids {
		if s, ok := q.stops[id]; ok {
			out[id] = s
		}
	}
	return out, nil
}

func departedAt(uts int64, stops ...TourStop) *Tour {
	return &Tour{ID: "tour-1", YearSlug: "2026", SectionSlug: "bil-2",
		DepartureUts: &uts, State: TourStatePlanned, Stops: stops}
}

func stopsOf(t *testing.T, p *recordingPublisher) StopsChanged {
	t.Helper()
	for _, body := range p.bodies {
		if s, ok := body.(*StopsChanged); ok {
			return *s
		}
	}
	t.Fatalf("no stops.changed was published: %v", p.subjects)
	return StopsChanged{}
}

func TestStopTimesAreDerivedFromDepartureAndTheLegAllowance(t *testing.T) {
	// The whole reason the tour model earns its complexity: a planned run with ordered stops
	// *is* an estimate, and a dispatcher building six stops at 3am must not have to type six
	// times.
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{tour: departedAt(1000)}}}

	if _, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{
		{Place: Place{Kind: PlaceCheckpoint, Label: "Post 2A"}},
		{Place: Place{Kind: PlaceText, Label: "ved Post 2B"}},
		{Place: Place{Kind: PlaceHQ, Label: "HQ"}},
	}); err != nil {
		t.Fatalf("SetStops: %v", err)
	}
	stops := stopsOf(t, p).Stops
	want := []int64{1000 + LegAllowance, 1000 + 2*LegAllowance, 1000 + 3*LegAllowance}
	for i, w := range want {
		if stops[i].PlannedUts == nil || *stops[i].PlannedUts != w {
			t.Errorf("stop %d planned %v, want %d", i, stops[i].PlannedUts, w)
		}
		if stops[i].Override {
			t.Errorf("stop %d marked as overridden, but nobody typed it", i)
		}
	}
}

func TestAnOverriddenStopIsMarkedAndTheRestReDeriveFromIt(t *testing.T) {
	// The override is what makes deriving acceptable: the moment the desk knows better it can
	// say so, and everything after it follows the correction rather than the guess.
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{tour: departedAt(1000)}}}
	override := int64(5000)

	if _, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{
		{Place: Place{Label: "Post 2A"}},
		{Place: Place{Label: "Post 2B"}, PlannedUts: &override},
		{Place: Place{Label: "HQ"}},
	}); err != nil {
		t.Fatalf("SetStops: %v", err)
	}
	stops := stopsOf(t, p).Stops
	if !stops[1].Override || *stops[1].PlannedUts != override {
		t.Errorf("the override was not kept or not marked: %+v", stops[1])
	}
	if want := override + LegAllowance; *stops[2].PlannedUts != want {
		t.Errorf("stop after the override planned %d, want %d", *stops[2].PlannedUts, want)
	}
}

func TestAVisitedStopAnchorsTheStopsAfterIt(t *testing.T) {
	// What actually happened beats what was planned. A tour running late must not keep quoting
	// the times it was going to make.
	visited := int64(9000)
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: departedAt(1000, TourStop{StopID: "stop-a", VisitedUts: &visited}),
	}}}

	if _, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{
		{StopID: "stop-a", Place: Place{Label: "Post 2A"}},
		{Place: Place{Label: "Post 2B"}},
	}); err != nil {
		t.Fatalf("SetStops: %v", err)
	}
	stops := stopsOf(t, p).Stops
	if stops[0].VisitedUts == nil || *stops[0].VisitedUts != visited {
		t.Errorf("the visit was lost in the rebuild: %+v", stops[0])
	}
	if want := visited + LegAllowance; *stops[1].PlannedUts != want {
		t.Errorf("stop after a visited one planned %d, want %d — it must anchor on the visit", *stops[1].PlannedUts, want)
	}
}

func TestATourWithNoDepartureGetsNoInventedTimes(t *testing.T) {
	// An invented time gets read down a phone to a patrol standing in the dark, and then they
	// stop making their own plans. No departure means no planned times, not "now-ish".
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: &Tour{ID: "tour-1", SectionSlug: "bil-2", State: TourStatePlanned},
	}}}

	if _, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{
		{Place: Place{Label: "Post 2A"}},
	}); err != nil {
		t.Fatalf("SetStops: %v", err)
	}
	if got := stopsOf(t, p).Stops[0].PlannedUts; got != nil {
		t.Errorf("planned time %v invented for a tour with no departure", *got)
	}
}

func TestReorderingAVisitedStopIsRefused(t *testing.T) {
	// The plan may change; what happened may not.
	visited := int64(9000)
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: departedAt(1000,
			TourStop{StopID: "stop-a", VisitedUts: &visited},
			TourStop{StopID: "stop-b"},
		),
	}}}

	_, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{
		{StopID: "stop-b", Place: Place{Label: "Post 2B"}},
		{StopID: "stop-a", Place: Place{Label: "Post 2A"}},
	})
	if err != ErrVisitedStopChanged {
		t.Fatalf("err = %v, want ErrVisitedStopChanged", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v despite refusing the plan", p.subjects)
	}
}

func TestDroppingAVisitedStopIsRefused(t *testing.T) {
	visited := int64(9000)
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: departedAt(1000, TourStop{StopID: "stop-a", VisitedUts: &visited}),
	}}}

	if _, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{
		{Place: Place{Label: "et andet sted"}},
	}); err != ErrVisitedStopChanged {
		t.Fatalf("err = %v, want ErrVisitedStopChanged", err)
	}
}

func TestUnloadingBeforeLoadingIsRefused(t *testing.T) {
	// Not a route. The board must not let a dispatcher build one.
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{tour: departedAt(1000)}}}

	_, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{
		{Place: Place{Label: "Post 2B"}, Tasks: []StopTask{{TaskID: "disp-9", Role: RoleUnload}}},
		{Place: Place{Label: "Post 2A"}, Tasks: []StopTask{{TaskID: "disp-9", Role: RoleLoad}}},
	})
	if err != ErrUnloadBeforeLoad {
		t.Fatalf("err = %v, want ErrUnloadBeforeLoad", err)
	}
}

func TestAnUnloadWithNoLoadOnThisTourIsAllowed(t *testing.T) {
	// The load may be another tour's job, or may already have happened. Only a plan that
	// contradicts itself is refused.
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{tour: departedAt(1000)}}}

	if _, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{
		{Place: Place{Kind: PlaceHQ, Label: "HQ"}, Tasks: []StopTask{{TaskID: "disp-9", Role: RoleUnload}}},
	}); err != nil {
		t.Fatalf("SetStops: %v", err)
	}
}

func TestPlanningATaskPublishesItsOwnPlannedEvent(t *testing.T) {
	// A task's state comes from its own events, never from inspecting its stops — and the
	// task event is published *after* the plan, so nothing can read "lagt i tur" for a plan
	// that was rejected.
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{tour: departedAt(1000)}}}

	if _, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{
		{Place: Place{Label: "Post 2B"}, Tasks: []StopTask{{TaskID: "disp-1", Role: RoleLoad}}},
	}); err != nil {
		t.Fatalf("SetStops: %v", err)
	}
	if len(p.subjects) != 2 {
		t.Fatalf("published %v, want the plan and one task event", p.subjects)
	}
	if !strings.Contains(p.subjects[0], "tour.tour-1.stops.changed") {
		t.Errorf("the plan is not published first: %v", p.subjects)
	}
	if !strings.Contains(p.subjects[1], "dispatch.disp-1.planned") {
		t.Errorf("the task was not told it is planned: %v", p.subjects)
	}
}

func TestATaskDroppedFromATourIsUnplanned(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: departedAt(1000, TourStop{StopID: "stop-a", Tasks: []StopTask{{TaskID: "disp-1", Role: RoleLoad}}}),
	}}}

	if _, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{}); err != nil {
		t.Fatalf("SetStops: %v", err)
	}
	found := false
	for _, s := range p.subjects {
		if strings.Contains(s, "dispatch.disp-1.unplanned") {
			found = true
		}
	}
	if !found {
		t.Errorf("a dropped task was not returned to the queue: %v", p.subjects)
	}
}

func TestATaskMovedToAnotherTourIsNotUnplanned(t *testing.T) {
	// Saying "unplanned" for a task that is on a different tour would put it in two places on
	// the board at once: back in the queue, and on the tour it was moved to.
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{
		stubQueries: stubQueries{
			tour: departedAt(1000, TourStop{StopID: "stop-a", Tasks: []StopTask{{TaskID: "disp-1", Role: RoleLoad}}}),
		},
		stops: map[TaskID][]TaskStop{"disp-1": {{TourID: "tour-2", StopID: "stop-x"}}},
	}}

	if _, err := c.SetStops(context.Background(), Actor{}, "2026", "tour-1", []StopInput{}); err != nil {
		t.Fatalf("SetStops: %v", err)
	}
	for _, s := range p.subjects {
		if strings.Contains(s, "unplanned") {
			t.Errorf("a task on another tour was returned to the queue: %v", p.subjects)
		}
	}
}

func TestVisitingAStopCompletesItsUnloadsButNotItsLoads(t *testing.T) {
	// A scout collected at Post 2B is aboard, not delivered. Completing the task at the load
	// would take them off the board while they are still in a car.
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: &Tour{ID: "tour-1", SectionSlug: "bil-2", State: TourStateUnderway, Stops: []TourStop{
			{StopID: "stop-a", Tasks: []StopTask{
				{TaskID: "disp-load", Role: RoleLoad},
				{TaskID: "disp-unload", Role: RoleUnload},
				{TaskID: "disp-action", Role: RoleAction},
			}},
		}},
	}}}

	if err := c.VisitStop(context.Background(), Actor{}, "2026", "tour-1", "stop-a", 4242); err != nil {
		t.Fatalf("VisitStop: %v", err)
	}
	joined := strings.Join(p.subjects, " ")
	if !strings.Contains(joined, "dispatch.disp-unload.completed") {
		t.Errorf("the unloaded task was not completed: %v", p.subjects)
	}
	if !strings.Contains(joined, "dispatch.disp-action.completed") {
		t.Errorf("the task actioned here was not completed: %v", p.subjects)
	}
	if strings.Contains(joined, "dispatch.disp-load.completed") {
		t.Errorf("a loaded task was completed at its load: %v", p.subjects)
	}
}

func TestVisitingAStopStartsATourThatWasStillPlanned(t *testing.T) {
	// Otherwise the desk is looking for a car it thinks has not left.
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: departedAt(1000, TourStop{StopID: "stop-a"}),
	}}}

	if err := c.VisitStop(context.Background(), Actor{}, "2026", "tour-1", "stop-a", 4242); err != nil {
		t.Fatalf("VisitStop: %v", err)
	}
	if !strings.Contains(p.subjects[0], "tour.tour-1.underway") {
		t.Errorf("the tour was not started by its first visit: %v", p.subjects)
	}
}

func TestVisitingAnAlreadyVisitedStopIsHarmless(t *testing.T) {
	// A dispatcher ticking off a stop the driver mentioned twice must not see a red toast, and
	// must not put two visits on the log.
	visited := int64(9000)
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: &Tour{ID: "tour-1", State: TourStateUnderway, Stops: []TourStop{{StopID: "stop-a", VisitedUts: &visited}}},
	}}}

	if err := c.VisitStop(context.Background(), Actor{}, "2026", "tour-1", "stop-a", 9100); err != nil {
		t.Fatalf("VisitStop: %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v for a stop that was already visited", p.subjects)
	}
}

func TestCompletingIsRefusedWhileAStopIsUnvisited(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: &Tour{ID: "tour-1", State: TourStateUnderway, Stops: []TourStop{{StopID: "stop-a"}}},
	}}}

	if err := c.CompleteTour(context.Background(), Actor{}, "2026", "tour-1"); err != ErrStopsRemaining {
		t.Fatalf("err = %v, want ErrStopsRemaining", err)
	}
}

func TestCancellingATourReturnsItsUnfinishedWorkToTheQueue(t *testing.T) {
	// The broken-down car needs no special modelling: this is it. And a task already unloaded
	// at a visited stop stays done — re-queueing it would send a second car for a scout who is
	// asleep in Hønsegården.
	visited := int64(9000)
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{
		tour: &Tour{ID: "tour-1", State: TourStateUnderway, Stops: []TourStop{
			{StopID: "stop-done", VisitedUts: &visited, Tasks: []StopTask{{TaskID: "disp-done", Role: RoleUnload}}},
			{StopID: "stop-todo", Tasks: []StopTask{{TaskID: "disp-todo", Role: RoleLoad}}},
		}},
		task: &Task{ID: "disp-todo", State: TaskStatePlanned},
	}}}

	if err := c.CancelTour(context.Background(), Actor{}, "2026", "tour-1", "bilen brød sammen"); err != nil {
		t.Fatalf("CancelTour: %v", err)
	}
	joined := strings.Join(p.subjects, " ")
	if !strings.Contains(joined, "tour.tour-1.cancelled") {
		t.Errorf("the tour was not cancelled: %v", p.subjects)
	}
	if !strings.Contains(joined, "dispatch.disp-todo.unplanned") {
		t.Errorf("unfinished work was not returned to the queue: %v", p.subjects)
	}
	if strings.Contains(joined, "dispatch.disp-done.unplanned") {
		t.Errorf("work already delivered was returned to the queue: %v", p.subjects)
	}
}

func TestCancellingATourRequiresAReason(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{tour: departedAt(1000)}}}

	if err := c.CancelTour(context.Background(), Actor{}, "2026", "tour-1", "   "); err != ErrReasonRequired {
		t.Fatalf("err = %v, want ErrReasonRequired", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v without a reason", p.subjects)
	}
}

func TestATourNeedsAUnit(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{}}

	if _, err := c.CreateTour(context.Background(), Actor{}, "2026", CreateTourCommand{}); err != ErrUnitRequired {
		t.Fatalf("err = %v, want ErrUnitRequired", err)
	}
}

func TestPatchingATourWithNothingNewPublishesNothing(t *testing.T) {
	// The board can send the state it has without first working out what changed.
	departure := int64(1000)
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{tour: departedAt(1000)}}}
	slug := types.Slug("bil-2")
	notes := ""

	if err := c.PatchTour(context.Background(), Actor{}, "2026", "tour-1", PatchTourCommand{
		SectionSlug: &slug, Notes: &notes, DepartureUts: ptrptr(&departure),
	}); err != nil {
		t.Fatalf("PatchTour: %v", err)
	}
	if len(p.subjects) != 0 {
		t.Errorf("published %v for an unchanged patch", p.subjects)
	}
}

func TestClearingATourDepartureIsDistinctFromLeavingItAlone(t *testing.T) {
	p := &recordingPublisher{}
	c := commander{p: p, q: tourQueries{stubQueries: stubQueries{tour: departedAt(1000)}}}

	var cleared *int64
	if err := c.PatchTour(context.Background(), Actor{}, "2026", "tour-1", PatchTourCommand{
		DepartureUts: &cleared,
	}); err != nil {
		t.Fatalf("PatchTour: %v", err)
	}
	if len(p.subjects) != 1 {
		t.Fatalf("published %v, want one update", p.subjects)
	}
	body := p.bodies[0].(*TourUpdated)
	if body.DepartureUts != nil {
		t.Errorf("the departure was not cleared: %v", *body.DepartureUts)
	}
}

func ptrptr(v *int64) **int64 { return &v }
