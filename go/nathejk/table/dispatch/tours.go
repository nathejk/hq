package dispatch

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/types"
)

// The tour write side (PRD 009 §6, task 111).
//
// A tour is what makes "when?" answerable without GPS: a planned run with ordered stops *is*
// an estimate, made by a human who knows the roads. Everything here therefore exists to keep
// one property true — **the plan on screen is a plan somebody could actually drive**:
//
//   - visited stops are fixed, because history is not a plan;
//   - a task's unload cannot come before its load, because that is not a route;
//   - the whole stop list is set in one call, because a reorder is one intent and three
//     finer-grained calls would make a half-applied reorder representable;
//   - times are derived so a dispatcher does not type six of them at 3am, and overridable so
//     the derivation can always be corrected.
//
// What is deliberately *not* here: refusing anything the desk might really need to do. Seat
// overruns and out-of-hours planning are warnings (returned, not raised), because seats fold
// down and the race does not stop for a roster — and a system that refuses the real world
// gets worked around, which is worse than a warning nobody read.

// LegAllowance is how long one leg of a tour is assumed to take.
//
// One number for every vehicle and every road, deliberately (PRD 009 §8, open question 10). A
// minibus and an estate do not drive alike and we are ignoring that: the difference between
// two cars is far smaller than the error in an estimate that knows nothing about distance,
// traffic or where the car currently is, so encoding it would add a column and a maintenance
// burden while moving the number by less than its own error.
//
// It is the desk's starting point, not its answer: every derived time can be overridden, and
// the measured gap between planned and visited stop times (PRD 009 §9) is the evidence that
// would justify changing this.
const LegAllowance int64 = 15 * 60

// TourCommands is the tour write surface.
//
// Each transition is its own method, and each gets its own route, because the driver's app
// will call exactly these (PRD 009 §8). SetStops is the one bulk operation, for the same
// reason `/api/sections/sorted` is.
type TourCommands interface {
	CreateTour(ctx context.Context, actor Actor, year types.YearSlug, cmd CreateTourCommand) (TourID, error)
	PatchTour(ctx context.Context, actor Actor, year types.YearSlug, id TourID, cmd PatchTourCommand) error
	SetStops(ctx context.Context, actor Actor, year types.YearSlug, id TourID, stops []StopInput) ([]Warning, error)
	StartTour(ctx context.Context, actor Actor, year types.YearSlug, id TourID) error
	VisitStop(ctx context.Context, actor Actor, year types.YearSlug, id TourID, stop StopID, atUts int64) error
	CompleteTour(ctx context.Context, actor Actor, year types.YearSlug, id TourID) error
	CancelTour(ctx context.Context, actor Actor, year types.YearSlug, id TourID, reason string) error
}

// CreateTourCommand opens a tour for one unit.
type CreateTourCommand struct {
	// SectionSlug is the dispatchable subsection running it. Required: a tour nobody is
	// driving is not a plan, and "which car" is the question the board exists to answer.
	SectionSlug types.Slug

	DepartureUts *int64
	Notes        string
}

// PatchTourCommand is a partial update.
type PatchTourCommand struct {
	SectionSlug  *types.Slug
	DepartureUts **int64
	Notes        *string
}

// StopInput is one stop as the client sends it.
type StopInput struct {
	// StopID is empty for a new stop and the server mints one. Sent back for an existing
	// stop, which is what lets a reorder keep its identity — and what lets the driver app's
	// "stop reached" call refer to a stop rather than to a position that has since moved.
	StopID StopID

	Place Place

	// PlannedUts, when set, is an override: a dispatcher saying "no, 22:35". Absent means
	// derive it.
	PlannedUts *int64

	Tasks []StopTask
}

// Warning is something the desk should know but that must not block the plan.
//
// Returned rather than raised. Every one of these is a case where the platform knows less
// than the person at the keyboard: seats get folded down, a member sits with a leader, a
// driver agrees to one more run at the end of their window. Refusing would be wrong *and*
// counterproductive — the job would be done anyway and simply not written down, which is the
// one failure this whole feature exists to prevent.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var (
	// ErrUnitRequired: a tour belongs to a unit, not to a car or a person.
	ErrUnitRequired = errors.New("a dispatch unit is required")
	// ErrTourFinished: a completed or cancelled tour is history.
	ErrTourFinished = errors.New("the tour is already finished")
	// ErrVisitedStopChanged: a visited stop cannot be moved, removed or re-placed. The plan
	// may change; what happened may not.
	ErrVisitedStopChanged = errors.New("a visited stop cannot be reordered or removed")
	// ErrUnloadBeforeLoad: a task cannot be delivered before it is collected.
	ErrUnloadBeforeLoad = errors.New("a task's unload is ordered before its load")
	// ErrStopsRemaining: refuse to complete a tour with stops nobody has visited. Not
	// pedantry: the alternative is a tour marked done with a task silently stranded on it,
	// and the desk would never see the job it dropped. Remove the stop, or cancel the tour.
	ErrStopsRemaining = errors.New("the tour still has unvisited stops")
	// ErrUnknownStop is a stop that is not on this tour.
	ErrUnknownStop = errors.New("no such stop on this tour")
	// ErrStopAlreadyVisited is harmless and reported anyway: see VisitStop.
	ErrStopAlreadyVisited = errors.New("the stop is already visited")
)

// CreateTour opens a tour. The server mints the id.
func (c commander) CreateTour(ctx context.Context, actor Actor, year types.YearSlug, cmd CreateTourCommand) (TourID, error) {
	if cmd.SectionSlug == "" || !cmd.SectionSlug.Valid() {
		return "", ErrUnitRequired
	}
	id := NewTourID()
	err := c.publishTour(actor, year, id, "created", &TourCreated{
		TourID:       id,
		SectionSlug:  cmd.SectionSlug,
		DepartureUts: cmd.DepartureUts,
		Notes:        strings.TrimSpace(cmd.Notes),
		CreatedUts:   nowUts(),
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

// PatchTour edits departure, unit or notes, dirty-checked field by field.
//
// Changing the departure does *not* re-derive the stop times here. That is deliberate: the
// stops are set by their own call, and silently rewriting six planned times because somebody
// nudged the departure by five minutes would throw away the overrides a dispatcher typed. The
// board asks for the stops again after a departure change if it wants them re-derived.
func (c commander) PatchTour(ctx context.Context, actor Actor, year types.YearSlug, id TourID, cmd PatchTourCommand) error {
	current, err := c.q.GetTour(ctx, year, id)
	if err != nil {
		return err
	}
	if current == nil {
		return tables.ErrRecordNotFound
	}
	if current.State == TourStateCompleted || current.State == TourStateCancelled {
		return ErrTourFinished
	}
	next := TourUpdated{
		TourID:       id,
		SectionSlug:  current.SectionSlug,
		DepartureUts: current.DepartureUts,
		Notes:        current.Notes,
	}
	changed := []string{}
	if cmd.SectionSlug != nil && *cmd.SectionSlug != current.SectionSlug {
		if *cmd.SectionSlug == "" || !cmd.SectionSlug.Valid() {
			return ErrUnitRequired
		}
		next.SectionSlug, changed = *cmd.SectionSlug, append(changed, "sectionSlug")
	}
	if cmd.DepartureUts != nil && !sameUts(*cmd.DepartureUts, current.DepartureUts) {
		next.DepartureUts, changed = *cmd.DepartureUts, append(changed, "departureUts")
	}
	if cmd.Notes != nil {
		if notes := strings.TrimSpace(*cmd.Notes); notes != current.Notes {
			next.Notes, changed = notes, append(changed, "notes")
		}
	}
	if len(changed) == 0 {
		return nil
	}
	next.Changed = changed
	return c.publishTour(actor, year, id, "updated", &next)
}

// SetStops replaces the tour's whole ordered stop list.
//
// Order of operations matters and is the reason this is not four smaller commands: the stop
// list is validated as a whole, published as a whole, and only then are the per-task
// consequences published. A task therefore never says "planned" for a plan that was rejected.
func (c commander) SetStops(ctx context.Context, actor Actor, year types.YearSlug, id TourID, stops []StopInput) ([]Warning, error) {
	current, err := c.q.GetTour(ctx, year, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, tables.ErrRecordNotFound
	}
	if current.State == TourStateCompleted || current.State == TourStateCancelled {
		return nil, ErrTourFinished
	}

	visited := map[StopID]*int64{}
	// Visited stops always occupy the leading positions of a tour — a car visits them in order
	// from the start — so "fixed" means literally *in the same position*, not merely "still
	// present in the same relative order". The weaker check passes a plan that moves an
	// unvisited stop in front of a visited one, which claims the car will drive somewhere before
	// a place it has already been. Found by the test.
	fixed := map[int]StopID{}
	for i, s := range current.Stops {
		if s.VisitedUts != nil {
			visited[s.StopID] = s.VisitedUts
			fixed[i] = s.StopID
		}
	}

	out := make([]Stop, 0, len(stops))
	seenVisited := 0
	var previous *int64
	for i := range stops {
		in := stops[i]
		stopID := in.StopID
		if stopID == "" {
			stopID = NewStopID()
		}
		stop := Stop{
			StopID: stopID,
			Place:  in.Place,
			Tasks:  in.Tasks,
		}
		if stop.Tasks == nil {
			stop.Tasks = []StopTask{}
		}
		if at, ok := visited[stopID]; ok {
			stop.VisitedUts = at
			seenVisited++
			// A visited stop's own time is the anchor for everything after it: what actually
			// happened beats what was planned, which is the whole reason to record visits.
			stop.PlannedUts = at
			previous = at
			out = append(out, stop)
			continue
		}
		switch {
		case in.PlannedUts != nil:
			stop.PlannedUts = in.PlannedUts
			stop.Override = true
			previous = in.PlannedUts
		case previous != nil:
			t := *previous + LegAllowance
			stop.PlannedUts = &t
			previous = &t
		case current.DepartureUts != nil:
			t := *current.DepartureUts + LegAllowance
			stop.PlannedUts = &t
			previous = &t
		default:
			// No departure and no override: the tour is a plan with no clock yet, and a
			// derived time would be a fabricated one. The board shows the stop with no time
			// rather than inventing one — an invented time gets read down a phone to a patrol
			// standing in the dark.
			stop.PlannedUts = nil
		}
		out = append(out, stop)
	}

	// Every visited stop must still be there, in the position it was in.
	if seenVisited != len(fixed) {
		return nil, ErrVisitedStopChanged
	}
	for i, id := range fixed {
		if i >= len(out) || out[i].StopID != id {
			return nil, ErrVisitedStopChanged
		}
	}
	if err := checkLoadBeforeUnload(out); err != nil {
		return nil, err
	}

	if err := c.publishTour(actor, year, id, "stops.changed", &StopsChanged{TourID: id, Stops: out}); err != nil {
		return nil, err
	}

	// The consequences for the tasks. Published after the plan, so nothing can read "lagt i
	// tur" for a tour whose stops were rejected.
	before := taskIDs(current.Stops)
	after := map[TaskID]bool{}
	for _, s := range out {
		for _, st := range s.Tasks {
			after[st.TaskID] = true
		}
	}
	for taskID := range after {
		if !before[taskID] {
			if err := c.publishTask(actor, year, taskID, "planned", &TaskPlanned{TaskID: taskID, TourID: id}); err != nil {
				return nil, err
			}
		}
	}
	for taskID := range before {
		if after[taskID] {
			continue
		}
		// Only unplanned if it is not on some *other* tour: a task moved from one tour to
		// another has not returned to the queue, and saying so would put it in two places on
		// the board.
		elsewhere, err := c.onAnotherTour(ctx, year, taskID, id)
		if err != nil {
			return nil, err
		}
		if elsewhere {
			continue
		}
		if err := c.publishTask(actor, year, taskID, "unplanned", &TaskUnplanned{TaskID: taskID, TourID: id}); err != nil {
			return nil, err
		}
	}
	return []Warning{}, nil
}

// StartTour records that the car has set off.
func (c commander) StartTour(ctx context.Context, actor Actor, year types.YearSlug, id TourID) error {
	current, err := c.q.GetTour(ctx, year, id)
	if err != nil {
		return err
	}
	if current == nil {
		return tables.ErrRecordNotFound
	}
	if current.State == TourStateCompleted || current.State == TourStateCancelled {
		return ErrTourFinished
	}
	if current.State == TourStateUnderway {
		return nil
	}
	at := nowUts()
	if err := c.publishTour(actor, year, id, "underway", &TourUnderway{TourID: id, AtUts: at}); err != nil {
		return err
	}
	for taskID := range taskIDs(current.Stops) {
		if err := c.publishTask(actor, year, taskID, "underway", &TaskUnderway{TaskID: taskID, TourID: id}); err != nil {
			return err
		}
	}
	return nil
}

// VisitStop marks a stop reached, and progresses the tasks actioned there.
//
// A task is completed by its *unload* (or by the single action), never by its load: a scout
// collected at Post 2B is aboard, not delivered, and completing the task there would take them
// off the board while they are still in a car.
func (c commander) VisitStop(ctx context.Context, actor Actor, year types.YearSlug, id TourID, stopID StopID, atUts int64) error {
	current, err := c.q.GetTour(ctx, year, id)
	if err != nil {
		return err
	}
	if current == nil {
		return tables.ErrRecordNotFound
	}
	if current.State == TourStateCompleted || current.State == TourStateCancelled {
		return ErrTourFinished
	}
	var stop *TourStop
	for i := range current.Stops {
		if current.Stops[i].StopID == stopID {
			stop = &current.Stops[i]
			break
		}
	}
	if stop == nil {
		return ErrUnknownStop
	}
	if stop.Visited() {
		// Idempotent, and reported as such rather than as an error: a dispatcher ticking off
		// a stop the driver mentioned twice must not see a red toast.
		return nil
	}
	if atUts == 0 {
		atUts = nowUts()
	}
	// A stop reached implies the tour has set off. Recording the visit without that would
	// leave a tour "planned" while its stops are being ticked off — and the desk would be
	// looking for a car it thinks has not left.
	if current.State == TourStatePlanned {
		if err := c.publishTour(actor, year, id, "underway", &TourUnderway{TourID: id, AtUts: atUts}); err != nil {
			return err
		}
		for taskID := range taskIDs(current.Stops) {
			if err := c.publishTask(actor, year, taskID, "underway", &TaskUnderway{TaskID: taskID, TourID: id}); err != nil {
				return err
			}
		}
	}
	if err := c.publishTour(actor, year, id, "stop.visited", &StopVisited{TourID: id, StopID: stopID, AtUts: atUts}); err != nil {
		return err
	}
	for _, st := range stop.Tasks {
		if st.Role == RoleLoad {
			continue
		}
		if err := c.publishTask(actor, year, st.TaskID, "completed", &TaskCompleted{TaskID: st.TaskID, AtUts: atUts}); err != nil {
			return err
		}
	}
	return nil
}

// CompleteTour closes a tour whose stops have all been visited.
func (c commander) CompleteTour(ctx context.Context, actor Actor, year types.YearSlug, id TourID) error {
	current, err := c.q.GetTour(ctx, year, id)
	if err != nil {
		return err
	}
	if current == nil {
		return tables.ErrRecordNotFound
	}
	if current.State == TourStateCompleted {
		return nil
	}
	if current.State == TourStateCancelled {
		return ErrTourFinished
	}
	for _, s := range current.Stops {
		if !s.Visited() {
			return ErrStopsRemaining
		}
	}
	return c.publishTour(actor, year, id, "completed", &TourCompleted{TourID: id, AtUts: nowUts()})
}

// CancelTour abandons a tour, returning its unvisited work to the queue.
//
// The car breaking down needs no special modelling (PRD 009 §5): this is it. Each task keeps
// its own waiting clock, because the scout has been waiting since the call.
func (c commander) CancelTour(ctx context.Context, actor Actor, year types.YearSlug, id TourID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrReasonRequired
	}
	current, err := c.q.GetTour(ctx, year, id)
	if err != nil {
		return err
	}
	if current == nil {
		return tables.ErrRecordNotFound
	}
	if current.State == TourStateCancelled {
		return nil
	}
	if current.State == TourStateCompleted {
		return ErrTourFinished
	}
	if err := c.publishTour(actor, year, id, "cancelled", &TourCancelled{TourID: id, Reason: reason, AtUts: nowUts()}); err != nil {
		return err
	}
	// Only the tasks whose work here is unfinished go back to the queue. A task already
	// unloaded at a visited stop is done, and re-queueing it would send a second car for a
	// scout who is asleep in Hønsegården.
	for _, s := range current.Stops {
		if s.Visited() {
			continue
		}
		for _, st := range s.Tasks {
			done, err := c.taskFinished(ctx, year, st.TaskID)
			if err != nil {
				return err
			}
			if done {
				continue
			}
			elsewhere, err := c.onAnotherTour(ctx, year, st.TaskID, id)
			if err != nil {
				return err
			}
			if elsewhere {
				continue
			}
			if err := c.publishTask(actor, year, st.TaskID, "unplanned", &TaskUnplanned{TaskID: st.TaskID, TourID: id}); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- helpers ---

func (c commander) publishTour(actor Actor, year types.YearSlug, id TourID, event string, body any) error {
	return c.publish(actor, fmt.Sprintf("NATHEJK.%s.tour.%s.%s", year, id, event), body)
}

func (c commander) onAnotherTour(ctx context.Context, year types.YearSlug, taskID TaskID, exclude TourID) (bool, error) {
	stops, err := c.q.StopsByTask(ctx, year, []TaskID{taskID})
	if err != nil {
		return false, err
	}
	for _, s := range stops[taskID] {
		if s.TourID != exclude {
			return true, nil
		}
	}
	return false, nil
}

func (c commander) taskFinished(ctx context.Context, year types.YearSlug, id TaskID) (bool, error) {
	task, err := c.q.GetTask(ctx, year, id)
	if err != nil {
		if errors.Is(err, tables.ErrRecordNotFound) {
			// A stop referring to a task that does not exist is not worth failing a
			// cancellation over: the tour is going away either way.
			return true, nil
		}
		return false, err
	}
	if task == nil {
		return true, nil
	}
	return task.State == TaskStateDone || task.State == TaskStateCancelled, nil
}

func taskIDs(stops []TourStop) map[TaskID]bool {
	ids := map[TaskID]bool{}
	for _, s := range stops {
		for _, st := range s.Tasks {
			ids[st.TaskID] = true
		}
	}
	return ids
}

// checkLoadBeforeUnload refuses a plan in which something is delivered before it is collected.
func checkLoadBeforeUnload(stops []Stop) error {
	loadedAt := map[TaskID]int{}
	for i, s := range stops {
		for _, st := range s.Tasks {
			if st.Role == RoleLoad {
				if _, seen := loadedAt[st.TaskID]; !seen {
					loadedAt[st.TaskID] = i
				}
			}
		}
	}
	for i, s := range stops {
		for _, st := range s.Tasks {
			if st.Role != RoleUnload {
				continue
			}
			load, seen := loadedAt[st.TaskID]
			// An unload with no load on this tour is allowed: the load may be another tour's
			// job, or may already have happened. Only an ordering that contradicts itself
			// within one plan is refused.
			if seen && load > i {
				return ErrUnloadBeforeLoad
			}
		}
	}
	return nil
}
