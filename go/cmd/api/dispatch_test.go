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
	"nathejk.dk/nathejk/table/dispatch"
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
	err       error
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

func dispatchApp(cmd *fakeDispatchCommands, q *fakeDispatchQueries) *application {
	return &application{
		models: data.Models{
			Dispatch:   q,
			Section:    &fakeSectionQueries{},
			Vehicle:    &fakeVehicleQueries{},
			CrewMember: &fakeCrewQueries{},
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
