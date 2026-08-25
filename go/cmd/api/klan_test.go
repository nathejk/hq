package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
)

// --- fakes ---
//
// Everything empty by default, because "a klan with nothing yet" is the case that
// broke: no seniors, so no order lines, so no order at all.

type fakeKlanQueries struct {
	klans []klan.Klan
}

func (f *fakeKlanQueries) GetAll(context.Context, klan.Filter) ([]klan.Klan, error) {
	return f.klans, nil
}
func (f *fakeKlanQueries) GetByID(context.Context, types.TeamID) (*klan.Klan, error) {
	if len(f.klans) == 0 {
		return &klan.Klan{}, nil
	}
	return &f.klans[0], nil
}

type fakeSeniorQueries struct {
	members []*senior.Senior
}

func (f *fakeSeniorQueries) GetAll(context.Context, senior.Filter) ([]*senior.Senior, error) {
	return f.members, nil
}
func (f *fakeSeniorQueries) GetByID(context.Context, types.MemberID) (*senior.Senior, error) {
	return nil, nil
}

// fakeSignupQueries reports "no signup", which a klan created by other means has.
type fakeSignupQueries struct{}

func (f *fakeSignupQueries) GetByID(types.TeamID) (*data.Signup, error) {
	return nil, data.ErrRecordNotFound
}
func (f *fakeSignupQueries) TeamIDsByType(context.Context, types.YearSlug, types.TeamType) (map[types.TeamID]bool, error) {
	return nil, nil
}

// fakeOrderQueries returns a nil slice from ListByOwner — precisely what the real
// querier does for an owner with no orders, and the source of the crash.
type fakeOrderQueries struct {
	order.Queries
}

func (f *fakeOrderQueries) ListByOwner(context.Context, types.YearSlug, types.TeamType, string) ([]order.Order, error) {
	return nil, nil
}

type fakePaymentQueries struct {
	payment.Queries
}

func (f *fakePaymentQueries) GetAll(context.Context, payment.Filter) ([]payment.Payment, error) {
	return nil, nil
}

func klanApp() *application {
	return &application{
		models: data.Models{
			Klan:    &fakeKlanQueries{klans: []klan.Klan{{ID: "team-1", Name: "Nøkken", Status: types.SignupStatusPay}}},
			Senior:  &fakeSeniorQueries{},
			Signup:  &fakeSignupQueries{},
			Order:   &fakeOrderQueries{},
			Payment: &fakePaymentQueries{},
		},
	}
}

func klanRequest(t *testing.T, teamID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/klan/"+teamID, nil)
	req.Header.Set("X-YearSlug", "2026")
	params := httprouter.Params{{Key: "id", Value: teamID}}
	return req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))
}

// --- tests ---

// A klan with no seniors, no orders and no payments is an ordinary state on the
// bandit page — most klans are in it before the weekend — and every collection in
// the payload must still be an array.
//
// This is the third time this bug class has been found in this repo (see the
// shelter sections, and TestListNotesReturnsAnArrayWhenEmpty). Here it cost more
// than a warning: `orders: null` made `orders.length` throw during the klan
// dialog's render, which took the dialog's own close button down with it and left
// the operator trapped in a modal.
func TestShowKlanServesArraysWhenTheKlanIsEmpty(t *testing.T) {
	rec := httptest.NewRecorder()
	klanApp().showKlanHandler(rec, klanRequest(t, "team-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}

	// Asserted on the raw JSON: decoding into []T turns both `null` and `[]` into a
	// nil slice, so a decoded assertion would pass either way.
	body := rec.Body.String()
	for _, field := range []string{"members", "orders", "payments", "statusOptions"} {
		if strings.Contains(body, `"`+field+`": null`) {
			t.Errorf("%q serialised as null, which throws in the browser: %s", field, body)
		}
	}
}

// The dialog is opened to investigate klans in odd states, so a missing signup must
// not fail the request — only leave that section out.
func TestShowKlanToleratesAMissingSignup(t *testing.T) {
	rec := httptest.NewRecorder()
	klanApp().showKlanHandler(rec, klanRequest(t, "team-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"signup": null`) {
		t.Errorf("expected a null signup rather than a failure: %s", rec.Body.String())
	}
}

func TestShowKlanIsNotFoundForAnUnknownKlan(t *testing.T) {
	app := &application{
		models: data.Models{
			Klan:    &fakeKlanQueries{},
			Senior:  &fakeSeniorQueries{},
			Signup:  &fakeSignupQueries{},
			Order:   &fakeOrderQueries{},
			Payment: &fakePaymentQueries{},
		},
	}
	rec := httptest.NewRecorder()
	app.showKlanHandler(rec, klanRequest(t, "team-1"))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

// Withdrawn klans must not reappear on the bandit page: the klan projection
// soft-deletes, and the entity's own GetAll only excludes the empty status.
func TestWithoutDeletedDropsWithdrawnKlans(t *testing.T) {
	in := []klan.Klan{
		{ID: "a", Status: types.SignupStatusPay},
		{ID: "b", Status: klanStatusDeleted},
		{ID: "c", Status: types.SignupStatusPaid},
	}
	out := withoutDeleted(in)

	if len(out) != 2 {
		t.Fatalf("kept %d klans, want 2: %+v", len(out), out)
	}
	for _, k := range out {
		if k.ID == "b" {
			t.Errorf("a withdrawn klan survived the filter")
		}
	}
}

// An empty list is an empty list, not null: the bandit page iterates it.
func TestWithoutDeletedReturnsAnEmptySliceNotNil(t *testing.T) {
	if out := withoutDeleted(nil); out == nil {
		t.Errorf("withoutDeleted(nil) = nil, want an empty slice")
	}
}

func TestKlanStatusSettableRejectsDeleteAndUnknowns(t *testing.T) {
	// "deleted" is reachable only through the delete endpoint, which asks for
	// confirmation; offering it as a status would be a second, silent way to
	// delete a klan.
	if klanStatusSettable(klanStatusDeleted) {
		t.Errorf(`"deleted" must not be settable as a status`)
	}
	if klanStatusSettable("BOGUS") {
		t.Errorf("an unknown status must not be settable")
	}
	if !klanStatusSettable(types.SignupStatusPaid) {
		t.Errorf("PAID must be settable")
	}
}
