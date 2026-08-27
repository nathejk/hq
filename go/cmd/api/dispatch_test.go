package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/vehicle"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/dispatch"
	"nathejk.dk/nathejk/table/lok"
)

// The dispatch endpoints at the HTTP boundary: what the board gets, what the SPA may send, and
// that refusals arrive as Danish the desk can act on. The rules themselves are tested in the
// dispatch package.

// --- fakes ---

type fakeDispatchQueries struct {
	dispatchable []types.Slug
	tasks        []*dispatch.Task
	task         *dispatch.Task
	tours        []*dispatch.Tour
	err          error
}

func (f *fakeDispatchQueries) DispatchableSections(context.Context, types.YearSlug) ([]types.Slug, error) {
	return f.dispatchable, f.err
}
func (f *fakeDispatchQueries) Tasks(context.Context, dispatch.Filter) ([]*dispatch.Task, error) {
	return f.tasks, f.err
}
func (f *fakeDispatchQueries) GetTask(context.Context, types.YearSlug, dispatch.TaskID) (*dispatch.Task, error) {
	if f.task == nil {
		return nil, tables.ErrRecordNotFound
	}
	return f.task, nil
}
func (f *fakeDispatchQueries) Tours(context.Context, dispatch.TourFilter) ([]*dispatch.Tour, error) {
	return f.tours, f.err
}
func (f *fakeDispatchQueries) GetTour(context.Context, types.YearSlug, dispatch.TourID) (*dispatch.Tour, error) {
	return nil, tables.ErrRecordNotFound
}
func (f *fakeDispatchQueries) StopsByTask(context.Context, types.YearSlug, []dispatch.TaskID) (map[dispatch.TaskID][]dispatch.TaskStop, error) {
	return map[dispatch.TaskID][]dispatch.TaskStop{}, nil
}

type fakeDispatchCommands struct {
	created   *dispatch.CreateTaskCommand
	patched   *dispatch.PatchTaskCommand
	pickedUp  int
	cancelled string
	unit      types.Slug

	tour        *dispatch.CreateTourCommand
	tourPatched *dispatch.PatchTourCommand
	stops       []dispatch.StopInput
	started     int
	visited     dispatch.StopID
	completed   int
	tourReason  string

	err error
}

func (f *fakeDispatchCommands) SetSectionDispatchable(context.Context, dispatch.Actor, types.YearSlug, types.Slug, bool) error {
	return f.err
}
func (f *fakeDispatchCommands) CreateTask(_ context.Context, _ dispatch.Actor, _ types.YearSlug, cmd dispatch.CreateTaskCommand) (dispatch.TaskID, error) {
	f.created = &cmd
	if f.err != nil {
		return "", f.err
	}
	return "dispatchtask-minted", nil
}
func (f *fakeDispatchCommands) PatchTask(_ context.Context, _ dispatch.Actor, _ types.YearSlug, _ dispatch.TaskID, cmd dispatch.PatchTaskCommand) error {
	f.patched = &cmd
	return f.err
}
func (f *fakeDispatchCommands) MarkPickedUp(_ context.Context, _ dispatch.Actor, _ types.YearSlug, _ dispatch.TaskID, unit types.Slug, _ int64) error {
	f.pickedUp++
	f.unit = unit
	return f.err
}
func (f *fakeDispatchCommands) CancelTask(_ context.Context, _ dispatch.Actor, _ types.YearSlug, _ dispatch.TaskID, reason string) error {
	f.cancelled = reason
	return f.err
}

func (f *fakeDispatchCommands) CreateTour(_ context.Context, _ dispatch.Actor, _ types.YearSlug, cmd dispatch.CreateTourCommand) (dispatch.TourID, error) {
	f.tour = &cmd
	if f.err != nil {
		return "", f.err
	}
	return "dispatchtour-minted", nil
}
func (f *fakeDispatchCommands) PatchTour(_ context.Context, _ dispatch.Actor, _ types.YearSlug, _ dispatch.TourID, cmd dispatch.PatchTourCommand) error {
	f.tourPatched = &cmd
	return f.err
}
func (f *fakeDispatchCommands) SetStops(_ context.Context, _ dispatch.Actor, _ types.YearSlug, _ dispatch.TourID, stops []dispatch.StopInput) ([]dispatch.Warning, error) {
	f.stops = stops
	if f.err != nil {
		return nil, f.err
	}
	return []dispatch.Warning{}, nil
}
func (f *fakeDispatchCommands) StartTour(context.Context, dispatch.Actor, types.YearSlug, dispatch.TourID) error {
	f.started++
	return f.err
}
func (f *fakeDispatchCommands) VisitStop(_ context.Context, _ dispatch.Actor, _ types.YearSlug, _ dispatch.TourID, stop dispatch.StopID, _ int64) error {
	f.visited = stop
	return f.err
}
func (f *fakeDispatchCommands) CompleteTour(context.Context, dispatch.Actor, types.YearSlug, dispatch.TourID) error {
	f.completed++
	return f.err
}
func (f *fakeDispatchCommands) CancelTour(_ context.Context, _ dispatch.Actor, _ types.YearSlug, _ dispatch.TourID, reason string) error {
	f.tourReason = reason
	return f.err
}

// fakeSectionQueries and fakeCrewQueries already exist in checkpersonnel_test.go, embedding the
// shared-go interfaces so only the method under test has to be written. Reused rather than
// duplicated — two fakes for one collection drift, and the compiler cannot tell.
type fakeVehicleQueries struct {
	vehicle.Queries
	vehicles []vehicle.Vehicle
}

func (f *fakeVehicleQueries) GetAll(context.Context, vehicle.Filter) ([]vehicle.Vehicle, error) {
	return f.vehicles, nil
}

// fakeCheckpointQueries and fakeLokQueries stand in for the place vocabulary the dialog's picker
// offers. Empty by default: a board with no checkpoints still has HQ and free text, which is what
// makes the picker usable before anything is configured.
type fakeCheckpointQueries struct{ checkpoints []checkpoint.Checkpoint }

func (f *fakeCheckpointQueries) GetAll(context.Context, checkpoint.Filter) ([]checkpoint.Checkpoint, error) {
	return f.checkpoints, nil
}
func (f *fakeCheckpointQueries) GetByID(context.Context, types.CheckpointID) (*checkpoint.Checkpoint, error) {
	return nil, tables.ErrRecordNotFound
}

type fakeLokQueries struct{ loks []*lok.Lok }

func (f *fakeLokQueries) GetAll(context.Context, lok.Filter) ([]*lok.Lok, lok.Metadata, error) {
	return f.loks, lok.Metadata{}, nil
}
func (f *fakeLokQueries) GetByID(context.Context, types.LokID) (*lok.Lok, error) {
	return nil, tables.ErrRecordNotFound
}

func dispatchApp(cmd *fakeDispatchCommands, q *fakeDispatchQueries) *application {
	return &application{
		models: data.Models{
			Dispatch:   q,
			Section:    &fakeSectionQueries{},
			Vehicle:    &fakeVehicleQueries{},
			CrewMember: &fakeCrewQueries{},
			Checkpoint: &fakeCheckpointQueries{},
			Lok:        &fakeLokQueries{},
		},
		commands: commands.Commands{Dispatch: cmd},
	}
}

// Handlers are invoked directly: app.routes() installs app.Metrics, whose expvar.NewInt panics
// on a duplicate name, so it can be built at most once per process and stream_test.go already
// does. That one construction is also what proves these routes register without an httprouter
// conflict — which matters here, because /api/dispatch/task/:id/pickedup sits beside
// /api/dispatch/task/:id.
func dispatchRequest(t *testing.T, method, path, id string, body any) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encoding body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("X-YearSlug", "2026")
	params := httprouter.Params{}
	if id != "" {
		params = append(params, httprouter.Param{Key: "id", Value: id})
	}
	return req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))
}

// dispatchRawRequest sends a body the Go structs could not produce — which is the only way to
// test the difference between an absent field and an explicit null.
func dispatchRawRequest(t *testing.T, method, path, id, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-YearSlug", "2026")
	params := httprouter.Params{{Key: "id", Value: id}}
	return req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))
}

// --- tests ---

func TestBoardServesTasksToursAndUnits(t *testing.T) {
	q := &fakeDispatchQueries{
		tasks: []*dispatch.Task{{ID: "disp-1", Kind: dispatch.KindPickup, Description: "spejder ved Post 2B"}},
		tours: []*dispatch.Tour{{ID: "tour-1", SectionSlug: "bil-2", Stops: []dispatch.TourStop{}}},
	}
	rec := httptest.NewRecorder()
	dispatchApp(&fakeDispatchCommands{}, q).showDispatchBoardHandler(rec, dispatchRequest(t, http.MethodGet, "/api/dispatch", "", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"disp-1", "tour-1", `"units"`, `"kinds"`, `"priorities"`} {
		if !strings.Contains(body, want) {
			t.Errorf("board payload is missing %s: %s", want, body)
		}
	}
}

// Every collection is an array, never null. This has bitten the repo three times, most
// memorably when `orders: null` threw during render and took the klan dialog's own close button
// with it, trapping the operator in a modal.
func TestBoardCollectionsAreArraysWhenEmpty(t *testing.T) {
	rec := httptest.NewRecorder()
	dispatchApp(&fakeDispatchCommands{}, &fakeDispatchQueries{}).showDispatchBoardHandler(rec, dispatchRequest(t, http.MethodGet, "/api/dispatch", "", nil))

	// Asserted on the raw JSON: decoding into []T turns both null and [] into a nil slice and
	// would pass either way.
	for _, key := range []string{"tasks", "tours", "units"} {
		if strings.Contains(rec.Body.String(), `"`+key+`": null`) {
			t.Errorf("%s serialised as null: %s", key, rec.Body.String())
		}
	}
}

func TestUnitsAreBuiltFromTheOrganisationTree(t *testing.T) {
	// A dispatch unit is a subsection holding a vehicle and crew — it needs no entity of its
	// own, which is the cheapest part of PRD 009's design and worth a test that it stays true.
	app := dispatchApp(&fakeDispatchCommands{}, &fakeDispatchQueries{dispatchable: []types.Slug{"bil-2"}})
	app.models.Section = &fakeSectionQueries{sections: []section.Section{{Slug: "bil-2", Label: "Bil 2"}}}
	app.models.Vehicle = &fakeVehicleQueries{vehicles: []vehicle.Vehicle{
		{VehicleID: "v-1", SectionSlug: "bil-2", LicensePlate: "DK+AB12345", SeatCount: 4},
		{VehicleID: "v-2", SectionSlug: "logistik"},
	}}
	app.models.CrewMember = &fakeCrewQueries{crew: []crewmember.CrewMember{
		{UserID: "u-1", Name: "Ib", SectionSlug: "bil-2"},
		{UserID: "u-2", Name: "Ida", SectionSlug: "koekken"},
	}}

	rec := httptest.NewRecorder()
	app.showDispatchBoardHandler(rec, dispatchRequest(t, http.MethodGet, "/api/dispatch", "", nil))

	var got struct {
		Units []dispatchUnit `json:"units"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding: %v; body: %s", err, rec.Body.String())
	}
	if len(got.Units) != 1 {
		t.Fatalf("got %d units, want 1: %+v", len(got.Units), got.Units)
	}
	unit := got.Units[0]
	if unit.Label != "Bil 2" {
		t.Errorf("label = %q, want the section's label", unit.Label)
	}
	if len(unit.Vehicles) != 1 || unit.Vehicles[0].VehicleID != "v-1" {
		t.Errorf("vehicles = %+v, want only the one in this subsection", unit.Vehicles)
	}
	if len(unit.People) != 1 || unit.People[0].Name != "Ib" {
		t.Errorf("people = %+v, want only the crew in this subsection", unit.People)
	}
}

func TestAUnitWhoseSectionIsGoneFallsBackToItsSlug(t *testing.T) {
	// Units drifting from the organisation is a named risk (PRD 009 §8). A unit with no
	// section must be visible and identifiable, not an empty row.
	app := dispatchApp(&fakeDispatchCommands{}, &fakeDispatchQueries{dispatchable: []types.Slug{"bil-9"}})
	rec := httptest.NewRecorder()
	app.showDispatchBoardHandler(rec, dispatchRequest(t, http.MethodGet, "/api/dispatch", "", nil))

	if !strings.Contains(rec.Body.String(), "bil-9") {
		t.Errorf("a unit with no section vanished from the board: %s", rec.Body.String())
	}
}

func TestCreateTaskPassesTheWholeCommandAndReturnsTheMintedID(t *testing.T) {
	cmd := &fakeDispatchCommands{}
	rec := httptest.NewRecorder()
	deadline := int64(1787869200)
	dispatchApp(cmd, &fakeDispatchQueries{}).createDispatchTaskHandler(rec,
		dispatchRequest(t, http.MethodPost, "/api/dispatch/task", "", map[string]any{
			"kind":        "delivery",
			"priority":    "yellow",
			"description": "aftensmad til Lok 3",
			"pickup":      map[string]any{"kind": "hq", "label": "HQ"},
			"dropoff":     map[string]any{"kind": "lok", "refId": "lok-3", "label": "Lok 3"},
			"deadlineUts": deadline,
		}))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "dispatchtask-minted") {
		t.Errorf("the minted id did not reach the client: %s", rec.Body.String())
	}
	if cmd.created == nil {
		t.Fatal("no command reached the domain")
	}
	if cmd.created.Kind != dispatch.KindDelivery || cmd.created.Priority != dispatch.PriorityYellow {
		t.Errorf("kind/priority lost in translation: %+v", cmd.created)
	}
	// The place's label travels with the task, so a lok renamed later does not rewrite what
	// the desk was told to do.
	if cmd.created.Dropoff.Label != "Lok 3" || cmd.created.Dropoff.RefID != "lok-3" {
		t.Errorf("dropoff place lost: %+v", cmd.created.Dropoff)
	}
	if cmd.created.DeadlineUts == nil || *cmd.created.DeadlineUts != deadline {
		t.Errorf("deadline lost: %+v", cmd.created.DeadlineUts)
	}
}

func TestCreateTaskRefusalsArriveAsDanishValidationErrors(t *testing.T) {
	cmd := &fakeDispatchCommands{err: dispatch.ErrEmptyDescription}
	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).createDispatchTaskHandler(rec,
		dispatchRequest(t, http.MethodPost, "/api/dispatch/task", "", map[string]any{"kind": "pickup"}))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "beskrivelse") {
		t.Errorf("the operator is not told what is wrong, in Danish: %s", rec.Body.String())
	}
}

func TestPatchDistinguishesAnAbsentFieldFromAnExplicitNull(t *testing.T) {
	// The entire point of PATCH here: an absent deadline leaves the deadline alone, and
	// `"deadlineUts": null` clears it. If these decoded to the same thing, editing a task's
	// description would silently drop its dinner deadline.
	cmd := &fakeDispatchCommands{}
	app := dispatchApp(cmd, &fakeDispatchQueries{task: &dispatch.Task{ID: "disp-1"}})

	rec := httptest.NewRecorder()
	app.patchDispatchTaskHandler(rec, dispatchRawRequest(t, http.MethodPatch,
		"/api/dispatch/task/disp-1", "disp-1", `{"description":"kort til postlinje 2"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if cmd.patched.DeadlineUts != nil {
		t.Errorf("an absent deadline was sent to the domain as a change: %+v", cmd.patched.DeadlineUts)
	}

	rec = httptest.NewRecorder()
	app.patchDispatchTaskHandler(rec, dispatchRawRequest(t, http.MethodPatch,
		"/api/dispatch/task/disp-1", "disp-1", `{"deadlineUts":null}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if cmd.patched.DeadlineUts == nil {
		t.Fatal("an explicit null did not reach the domain as a clear")
	}
	if *cmd.patched.DeadlineUts != nil {
		t.Errorf("an explicit null reached the domain as a value: %v", **cmd.patched.DeadlineUts)
	}
}

func TestPickedUpPassesTheUnit(t *testing.T) {
	// "Which car has my scout" is the question the feature exists to answer, and the unit is
	// what answers it — a section slug, so it survives a car being swapped mid-night.
	cmd := &fakeDispatchCommands{}
	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).dispatchTaskPickedUpHandler(rec,
		dispatchRequest(t, http.MethodPost, "/api/dispatch/task/disp-1/pickedup", "disp-1",
			map[string]any{"sectionSlug": "bil-2"}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if cmd.pickedUp != 1 || cmd.unit != "bil-2" {
		t.Errorf("unit = %q after %d calls, want bil-2 once", cmd.unit, cmd.pickedUp)
	}
}

func TestPickedUpOnADeliveryIsRefusedInDanish(t *testing.T) {
	cmd := &fakeDispatchCommands{err: dispatch.ErrNotPickup}
	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).dispatchTaskPickedUpHandler(rec,
		dispatchRequest(t, http.MethodPost, "/api/dispatch/task/disp-1/pickedup", "disp-1", map[string]any{}))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "hentning") {
		t.Errorf("unhelpful refusal: %s", rec.Body.String())
	}
}

func TestCancelRequiresAReason(t *testing.T) {
	// A cancelled task with no explanation is the one thing a handover cannot recover from.
	cmd := &fakeDispatchCommands{err: dispatch.ErrReasonRequired}
	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).cancelDispatchTaskHandler(rec,
		dispatchRequest(t, http.MethodPost, "/api/dispatch/task/disp-1/cancelled", "disp-1", map[string]any{"reason": "  "}))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "årsag") {
		t.Errorf("the operator is not asked for a reason: %s", rec.Body.String())
	}
}

func TestCancelPassesTheReasonThrough(t *testing.T) {
	cmd := &fakeDispatchCommands{}
	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).cancelDispatchTaskHandler(rec,
		dispatchRequest(t, http.MethodPost, "/api/dispatch/task/disp-1/cancelled", "disp-1",
			map[string]any{"reason": "spejderen fortsatte alligevel"}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if cmd.cancelled != "spejderen fortsatte alligevel" {
		t.Errorf("reason = %q, not passed through", cmd.cancelled)
	}
}

func TestShowTaskAnswers404ForAnUnknownID(t *testing.T) {
	rec := httptest.NewRecorder()
	dispatchApp(&fakeDispatchCommands{}, &fakeDispatchQueries{}).showDispatchTaskHandler(rec,
		dispatchRequest(t, http.MethodGet, "/api/dispatch/task/nope", "nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

// --- tours (task 111) ---

func TestSetStopsPassesTheOrderedListThrough(t *testing.T) {
	// The array order is the order. If the handler reordered or dropped anything, a
	// dispatcher's drag would land somewhere else — and the board would be lying about a
	// route somebody has to drive.
	cmd := &fakeDispatchCommands{}
	rec := httptest.NewRecorder()
	planned := int64(1787864100)
	dispatchApp(cmd, &fakeDispatchQueries{}).setDispatchTourStopsHandler(rec,
		dispatchRequest(t, http.MethodPut, "/api/dispatch/tour/tour-1/stops", "tour-1", map[string]any{
			"stops": []any{
				map[string]any{
					"stopId": "stop-a",
					"place":  map[string]any{"kind": "checkpoint", "refId": "cp-2a", "label": "Post 2A"},
					"tasks":  []any{map[string]any{"taskId": "disp-9", "role": "load"}},
				},
				map[string]any{
					"place":      map[string]any{"kind": "text", "label": "ved Post 2B"},
					"plannedUts": planned,
					"tasks": []any{
						map[string]any{"taskId": "disp-1", "role": "load"},
						map[string]any{"taskId": "disp-9", "role": "unload"},
					},
				},
			},
		}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if len(cmd.stops) != 2 {
		t.Fatalf("got %d stops, want 2: %+v", len(cmd.stops), cmd.stops)
	}
	if cmd.stops[0].StopID != "stop-a" {
		t.Errorf("an existing stop lost its id, so a reorder would lose its identity: %+v", cmd.stops[0])
	}
	if cmd.stops[1].StopID != "" {
		t.Errorf("a new stop arrived with an id; the server mints those: %+v", cmd.stops[1])
	}
	if cmd.stops[1].PlannedUts == nil || *cmd.stops[1].PlannedUts != planned {
		t.Errorf("the override time was lost: %+v", cmd.stops[1].PlannedUts)
	}
	if len(cmd.stops[1].Tasks) != 2 || cmd.stops[1].Tasks[1].Role != dispatch.RoleUnload {
		t.Errorf("stop tasks and roles lost: %+v", cmd.stops[1].Tasks)
	}
}

func TestSetStopsAnswersWithAWarningsArrayEvenWhenEmpty(t *testing.T) {
	rec := httptest.NewRecorder()
	dispatchApp(&fakeDispatchCommands{}, &fakeDispatchQueries{}).setDispatchTourStopsHandler(rec,
		dispatchRequest(t, http.MethodPut, "/api/dispatch/tour/tour-1/stops", "tour-1", map[string]any{"stops": []any{}}))

	if strings.Contains(rec.Body.String(), `"warnings": null`) {
		t.Errorf("warnings serialised as null: %s", rec.Body.String())
	}
}

func TestMovingAVisitedStopIsRefusedInDanish(t *testing.T) {
	cmd := &fakeDispatchCommands{err: dispatch.ErrVisitedStopChanged}
	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).setDispatchTourStopsHandler(rec,
		dispatchRequest(t, http.MethodPut, "/api/dispatch/tour/tour-1/stops", "tour-1", map[string]any{"stops": []any{}}))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "besøgt stop") {
		t.Errorf("unhelpful refusal: %s", rec.Body.String())
	}
}

func TestCompletingATourWithUnvisitedStopsIsRefused(t *testing.T) {
	// A tour marked done with a stop nobody visited strands the task on it, and the desk
	// never sees the job it dropped.
	cmd := &fakeDispatchCommands{err: dispatch.ErrStopsRemaining}
	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).completeDispatchTourHandler(rec,
		dispatchRequest(t, http.MethodPost, "/api/dispatch/tour/tour-1/completed", "tour-1", nil))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ikke er besøgt") {
		t.Errorf("unhelpful refusal: %s", rec.Body.String())
	}
}

func TestVisitingAStopWorksWithNoBody(t *testing.T) {
	// "Reached, now" is the ordinary call, and a driver app sending no body must not get a 400
	// for it.
	cmd := &fakeDispatchCommands{}
	req := httptest.NewRequest(http.MethodPost, "/api/dispatch/tour/tour-1/stop/stop-a/visited", nil)
	req.Header.Set("X-YearSlug", "2026")
	req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, httprouter.Params{
		{Key: "id", Value: "tour-1"}, {Key: "stopId", Value: "stop-a"},
	}))

	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).visitDispatchTourStopHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if cmd.visited != "stop-a" {
		t.Errorf("visited stop = %q, want stop-a", cmd.visited)
	}
}

func TestCancellingATourRequiresAReason(t *testing.T) {
	cmd := &fakeDispatchCommands{err: dispatch.ErrReasonRequired}
	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).cancelDispatchTourHandler(rec,
		dispatchRequest(t, http.MethodPost, "/api/dispatch/tour/tour-1/cancelled", "tour-1", map[string]any{"reason": ""}))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
}

func TestATourWithoutAUnitIsRefused(t *testing.T) {
	// A tour nobody is driving is not a plan, and "which car" is the question the board exists
	// to answer.
	cmd := &fakeDispatchCommands{err: dispatch.ErrUnitRequired}
	rec := httptest.NewRecorder()
	dispatchApp(cmd, &fakeDispatchQueries{}).createDispatchTourHandler(rec,
		dispatchRequest(t, http.MethodPost, "/api/dispatch/tour", "", map[string]any{}))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "enhed") {
		t.Errorf("the operator is not asked for a unit: %s", rec.Body.String())
	}
}

func TestSeatOverrunIsAWarningNotARefusal(t *testing.T) {
	// PRD 009 §11 answer 8: warn, never refuse. Seats fold down, a member sits with a leader,
	// and a platform that refuses the real world gets worked around — which means the job
	// happens and is not written down, the one failure this feature exists to prevent.
	app := dispatchApp(&fakeDispatchCommands{}, &fakeDispatchQueries{})
	stop := dispatch.TourStop{StopID: "stop-a", Tasks: []dispatch.StopTask{
		{TaskID: "disp-1", Role: dispatch.RoleLoad},
	}}
	q := &seatQueries{
		tour: &dispatch.Tour{ID: "tour-1", SectionSlug: "bil-2", Stops: []dispatch.TourStop{stop}},
		task: &dispatch.Task{ID: "disp-1", Kind: dispatch.KindPickup, MemberIDs: []types.MemberID{"m-1", "m-2", "m-3", "m-4", "m-5"}},
	}
	app.models.Dispatch = q
	app.models.Vehicle = &fakeVehicleQueries{vehicles: []vehicle.Vehicle{{VehicleID: "v-1", SectionSlug: "bil-2", SeatCount: 4}}}

	rec := httptest.NewRecorder()
	app.setDispatchTourStopsHandler(rec, dispatchRequest(t, http.MethodPut,
		"/api/dispatch/tour/tour-1/stops", "tour-1", map[string]any{"stops": []any{}}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — a seat overrun must not block the plan; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "pladser") {
		t.Errorf("no seat warning reached the desk: %s", rec.Body.String())
	}
}

func TestAPickupNamingNobodyStillCountsAsAPerson(t *testing.T) {
	// Linking the member record is common to skip at 3am. Counting an unlinked pickup as zero
	// people is how a full car looks empty.
	app := dispatchApp(&fakeDispatchCommands{}, &fakeDispatchQueries{})
	app.models.Dispatch = &seatQueries{
		tour: &dispatch.Tour{ID: "tour-1", SectionSlug: "bil-2", Stops: []dispatch.TourStop{
			{StopID: "s", Tasks: []dispatch.StopTask{{TaskID: "disp-1", Role: dispatch.RoleLoad}}},
		}},
		task: &dispatch.Task{ID: "disp-1", Kind: dispatch.KindPickup},
	}
	app.models.Vehicle = &fakeVehicleQueries{vehicles: []vehicle.Vehicle{{SectionSlug: "bil-2", SeatCount: 0}}}

	// With no seats recorded there is nothing to compare against, so nothing is warned about —
	// asserted so the "unlinked counts as one" rule cannot be mistaken for "no seats warns".
	rec := httptest.NewRecorder()
	app.setDispatchTourStopsHandler(rec, dispatchRequest(t, http.MethodPut,
		"/api/dispatch/tour/tour-1/stops", "tour-1", map[string]any{"stops": []any{}}))
	if strings.Contains(rec.Body.String(), "pladser") {
		t.Errorf("warned about seats for a unit whose car has none recorded: %s", rec.Body.String())
	}
}

// seatQueries serves one tour and one task, for the seat arithmetic.
type seatQueries struct {
	fakeDispatchQueries
	tour *dispatch.Tour
	task *dispatch.Task
}

func (s *seatQueries) GetTour(context.Context, types.YearSlug, dispatch.TourID) (*dispatch.Tour, error) {
	return s.tour, nil
}
func (s *seatQueries) GetTask(context.Context, types.YearSlug, dispatch.TaskID) (*dispatch.Task, error) {
	return s.task, nil
}

// --- the place vocabulary (task 114) ---

func TestPlacesOfferHQEvenWithNothingConfigured(t *testing.T) {
	// The picker must be usable on a fresh year: HQ exists whatever the organisers have set up,
	// and free text is typed rather than offered. A picker with no options is one an operator
	// types around — into the description, where nothing can read it.
	rec := httptest.NewRecorder()
	dispatchApp(&fakeDispatchCommands{}, &fakeDispatchQueries{}).showDispatchBoardHandler(rec,
		dispatchRequest(t, http.MethodGet, "/api/dispatch", "", nil))

	var got struct {
		Places []struct {
			Kind  string `json:"kind"`
			Label string `json:"label"`
		} `json:"places"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding: %v; body: %s", err, rec.Body.String())
	}
	if len(got.Places) != 1 || got.Places[0].Kind != "hq" {
		t.Errorf("places = %+v, want just HQ", got.Places)
	}
}

func TestPlacesGroupCheckpointsAndLoks(t *testing.T) {
	app := dispatchApp(&fakeDispatchCommands{}, &fakeDispatchQueries{})
	app.models.Checkpoint = &fakeCheckpointQueries{checkpoints: []checkpoint.Checkpoint{
		{ID: "cp-1", Name: "Post 2A"},
		// A checkpoint with no name is still somewhere a car may have to go; an unlabelled
		// option is one nobody can pick, so it falls back to its id.
		{ID: "cp-2"},
	}}
	app.models.Lok = &fakeLokQueries{loks: []*lok.Lok{{LokID: "lok-3", Name: "Lok 3"}}}

	rec := httptest.NewRecorder()
	app.showDispatchBoardHandler(rec, dispatchRequest(t, http.MethodGet, "/api/dispatch", "", nil))

	body := rec.Body.String()
	for _, want := range []string{"Post 2A", "cp-2", "Lok 3", `"checkpoint"`, `"lok"`, `"hq"`} {
		if !strings.Contains(body, want) {
			t.Errorf("places are missing %s: %s", want, body)
		}
	}
}
