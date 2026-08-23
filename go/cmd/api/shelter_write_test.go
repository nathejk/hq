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
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table/shelter"
	"nathejk.dk/nathejk/table/spejderstatus"
)

// The write endpoints, tested at the HTTP boundary: what the SPA sends, what comes back, and
// which refusals are worded for the crew rather than for a developer. The commands' own rules
// are tested in their packages; what is verified here is the wiring and the contract.

// --- fakes ---

// fakeMemberCommands records what the handler asked for. Only the two shelter methods do
// anything; the rest satisfy the interface.
type fakeMemberCommands struct {
	acceptedPlacement string
	acceptedCalls     int
	handoverTo        types.MemberStatus
	handoverCalls     int

	change *spejderstatus.Change
	err    error
}

func (f *fakeMemberCommands) AcceptIntoShelter(_ context.Context, _ spejderstatus.Actor, _ types.YearSlug, _ types.MemberID, placement string) (*spejderstatus.Change, error) {
	f.acceptedCalls++
	f.acceptedPlacement = placement
	return f.change, f.err
}

func (f *fakeMemberCommands) CompleteHandover(_ context.Context, _ spejderstatus.Actor, _ types.YearSlug, _ types.MemberID, to types.MemberStatus) (*spejderstatus.Change, error) {
	f.handoverCalls++
	f.handoverTo = to
	return f.change, f.err
}

func (f *fakeMemberCommands) RequestWithdrawal(context.Context, spejderstatus.Actor, types.YearSlug, types.MemberID) (*spejderstatus.Change, error) {
	return nil, nil
}
func (f *fakeMemberCommands) CancelWithdrawal(context.Context, spejderstatus.Actor, types.YearSlug, types.MemberID) (*spejderstatus.Change, error) {
	return nil, nil
}
func (f *fakeMemberCommands) OverrideStatus(context.Context, spejderstatus.Actor, types.YearSlug, types.MemberID, types.MemberStatus) (*spejderstatus.Change, error) {
	return nil, nil
}
func (f *fakeMemberCommands) MoveTeam(context.Context, spejderstatus.Actor, types.YearSlug, types.MemberID, types.TeamID) (*spejderstatus.Move, error) {
	return nil, nil
}
func (f *fakeMemberCommands) MoveMembers(context.Context, spejderstatus.Actor, types.YearSlug, []types.MemberID, types.TeamID) ([]spejderstatus.Move, error) {
	return nil, nil
}
func (f *fakeMemberCommands) CollectTeam(context.Context, spejderstatus.Actor, types.YearSlug, types.TeamID) ([]spejderstatus.Change, error) {
	return nil, nil
}

type fakeShelterCommands struct {
	placement string
	calls     int
	err       error
}

func (f *fakeShelterCommands) SetPlacement(_ context.Context, _ spejderstatus.Actor, _ types.YearSlug, _ types.MemberID, placement string) error {
	f.calls++
	f.placement = placement
	return f.err
}

// fakeSosCommands is deliberately absent: with no sosId on the request and caseOptional in
// force, no summary is attempted at all, which is the contract this file exists to pin. Task
// 097 adds the case-present path and its own fake.

// fakeTeams names the patrol for the summary line. Required rather than optional: the operation
// looks the team up unconditionally, and a nil Models.Teams takes the handler down.
type fakeTeams struct{}

func (fakeTeams) GetPatrulje(id types.TeamID) (*data.Patrulje, error) {
	return &data.Patrulje{Name: "Ulvene"}, nil
}
func (fakeTeams) GetKlan(types.TeamID) (*data.Klan, error)       { return nil, nil }
func (fakeTeams) GetContact(types.TeamID) (*data.Contact, error) { return nil, nil }
func (fakeTeams) RequestedSeniorCount() int                      { return 0 }

func writeApp(member *fakeMemberCommands, sh *fakeShelterCommands) *application {
	return &application{
		models: data.Models{
			Teams:   fakeTeams{},
			Members: &fakeShelterMembers{},
		},
		commands: commands.Commands{
			Member:  member,
			Shelter: sh,
		},
	}
}

// withMemberParam puts the route parameter where ReadNamedParam looks for it.
//
// The handlers read `:memberId` from httprouter's context rather than the path, so invoking
// them directly means supplying it here. httprouter exposes exactly this for the purpose.
func withMemberParam(r *http.Request, memberID string) *http.Request {
	params := httprouter.Params{}
	if memberID != "" {
		params = append(params, httprouter.Param{Key: "memberId", Value: memberID})
	}
	return r.WithContext(context.WithValue(r.Context(), httprouter.ParamsKey, params))
}

// put invokes a handler directly. app.routes() cannot be built here — it installs app.Metrics,
// which calls expvar.NewInt, and expvar panics on a duplicate name, so routes() may be
// constructed at most once per process and stream_test.go already does it. Route registration
// is covered by that one construction: a static/wildcard conflict would panic there.
func put(t *testing.T, handler http.HandlerFunc, memberID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encoding body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/member/"+memberID+"/x", bytes.NewReader(encoded))
	req.Header.Set("X-YearSlug", "2026")
	req = withMemberParam(req, memberID)
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

// --- tests ---

// The placering travels with the acceptance, which is the crew's actual gesture: they type the
// tent as they take the scouts in.
func TestAcceptIntoShelterPassesThePlacering(t *testing.T) {
	member := &fakeMemberCommands{change: &spejderstatus.Change{
		MemberID: "m-1", TeamID: "team-1",
		From: types.MemberStatusTransit, To: types.MemberStatusSheltered,
	}}
	app := writeApp(member, &fakeShelterCommands{})

	rec := put(t, app.acceptIntoShelterHandler, "m-1", map[string]string{"placement": "  Telt 4  "})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if member.acceptedCalls != 1 {
		t.Fatalf("expected one acceptance, got %d", member.acceptedCalls)
	}
	// Trimmed before it reaches the command, so a stray space cannot become a second zone in
	// the suggestion list.
	if member.acceptedPlacement != "Telt 4" {
		t.Errorf("placement = %q, want it trimmed", member.acceptedPlacement)
	}
}

// **No sosId required.** The shelter may receive a scout nobody opened a case about, so a
// request without one must go through rather than being rejected the way the nødtelefon's own
// endpoints reject it.
func TestShelterWritesDoNotRequireACase(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler func(*application) http.HandlerFunc
		body    any
		calls   func(*fakeMemberCommands) int
	}{
		{
			"accept",
			func(a *application) http.HandlerFunc { return a.acceptIntoShelterHandler },
			map[string]string{"placement": "Telt 4"},
			func(f *fakeMemberCommands) int { return f.acceptedCalls },
		},
		{
			"handover",
			func(a *application) http.HandlerFunc { return a.completeHandoverHandler },
			map[string]string{"to": "released"},
			func(f *fakeMemberCommands) int { return f.handoverCalls },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			member := &fakeMemberCommands{change: &spejderstatus.Change{MemberID: "m-1", TeamID: "team-1"}}
			app := writeApp(member, &fakeShelterCommands{})

			rec := put(t, tc.handler(app), "m-1", tc.body)

			if rec.Code == http.StatusUnprocessableEntity {
				t.Fatalf("a case was demanded: %s", rec.Body.String())
			}
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
			}
			if tc.calls(member) != 1 {
				t.Errorf("the command was not called")
			}
		})
	}
}

// Which ending it was reaches the command unchanged. Guessing it from the hour was the
// alternative and would be wrong in both directions.
func TestHandoverPassesTheEnding(t *testing.T) {
	for _, to := range []types.MemberStatus{types.MemberStatusReleased, types.MemberStatusReunited} {
		t.Run(string(to), func(t *testing.T) {
			member := &fakeMemberCommands{change: &spejderstatus.Change{MemberID: "m-1", TeamID: "team-1"}}
			app := writeApp(member, &fakeShelterCommands{})

			put(t, app.completeHandoverHandler, "m-1", map[string]string{"to": string(to)})

			if member.handoverTo != to {
				t.Errorf("to = %q, want %q", member.handoverTo, to)
			}
		})
	}
}

// A no-op answers 200 with a null change, not an error. The second crew member to press
// Modtaget should see the state they wanted, not a failure.
func TestShelterNoOpAnswersOK(t *testing.T) {
	member := &fakeMemberCommands{change: nil} // the command's way of saying "nothing to do"
	app := writeApp(member, &fakeShelterCommands{})

	rec := put(t, app.acceptIntoShelterHandler, "m-1", map[string]string{})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for a no-op; body: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if payload["change"] != nil {
		t.Errorf("change = %v, want null so the client can tell nothing happened", payload["change"])
	}
}

// The command's refusals arrive as messages the crew can act on. "Not started" is a mistyped
// identity; the wrong ending means the UI let them submit something meaningless.
func TestShelterRefusalsAreWordedForTheCrew(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		handler func(*application) http.HandlerFunc
		field   string
	}{
		{"not started", spejderstatus.ErrNotStarted, func(a *application) http.HandlerFunc { return a.acceptIntoShelterHandler }, "status"},
		{"not an ending", spejderstatus.ErrNotAnEnding, func(a *application) http.HandlerFunc { return a.completeHandoverHandler }, "to"},
		{"cannot finish", spejderstatus.ErrCannotFinish, func(a *application) http.HandlerFunc { return a.completeHandoverHandler }, "status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := writeApp(&fakeMemberCommands{err: tt.err}, &fakeShelterCommands{})

			rec := put(t, tt.handler(app), "m-1", map[string]string{"to": "finished"})

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.field) {
				t.Errorf("expected the message on field %q, got %s", tt.field, rec.Body.String())
			}
			// Danish, because the operator reads it. An English domain error leaking to the
			// screen is the failure this mapping exists to prevent.
			if strings.Contains(rec.Body.String(), tt.err.Error()) {
				t.Errorf("the raw domain error reached the client: %s", rec.Body.String())
			}
		})
	}
}

// An over-long placering is refused before the command, so the 64-character limit is one rule
// enforced in one place rather than a database error surfacing as a 500.
func TestAcceptRejectsAnOverLongPlacering(t *testing.T) {
	member := &fakeMemberCommands{}
	app := writeApp(member, &fakeShelterCommands{})

	rec := put(t, app.acceptIntoShelterHandler, "m-1", map[string]string{
		"placement": strings.Repeat("a", shelter.MaxPlacementLength+1),
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	if member.acceptedCalls != 0 {
		t.Errorf("the command ran despite the placering being too long")
	}
}

// The placering endpoint hands the value to the shelter's own commander, which is where the
// dirty-check against the current placering lives.
func TestSetPlacementCallsTheShelterCommand(t *testing.T) {
	sh := &fakeShelterCommands{}
	app := writeApp(&fakeMemberCommands{}, sh)

	rec := put(t, app.setPlacementHandler, "m-1", map[string]string{"placement": "Telt 7"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if sh.calls != 1 || sh.placement != "Telt 7" {
		t.Errorf("placement = %q after %d calls, want one call with the tent", sh.placement, sh.calls)
	}
}

// Setting a placering for somebody not in the shelter tells the crew what to do instead —
// press Modtaget — rather than reporting a rule violation.
func TestSetPlacementForSomebodyNotShelteredExplainsItself(t *testing.T) {
	app := writeApp(&fakeMemberCommands{}, &fakeShelterCommands{err: shelter.ErrNotSheltered})

	rec := put(t, app.setPlacementHandler, "m-1", map[string]string{"placement": "Telt 7"})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "modtag") {
		t.Errorf("expected the message to say what to do next, got %s", rec.Body.String())
	}
}

func TestSetPlacementRequiresAMemberID(t *testing.T) {
	app := writeApp(&fakeMemberCommands{}, &fakeShelterCommands{})

	rec := put(t, app.setPlacementHandler, "", map[string]string{"placement": "Telt 7"})

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for a request naming no member", rec.Code)
	}
}
