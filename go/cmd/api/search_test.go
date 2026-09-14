package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/jsonlog"
	"nathejk.dk/nathejk/table/searchperson"
)

// fakeSearch records the query it was handed, which is most of what is worth asserting about
// this handler: the endpoint's job is to turn an HTTP request into a Query faithfully, and the
// year scope in particular must not be reachable by accident.
type fakeSearch struct {
	got     searchperson.Query
	results searchperson.Results
	err     error
}

func (f *fakeSearch) Search(_ context.Context, q searchperson.Query) (searchperson.Results, error) {
	f.got = q
	return f.results, f.err
}

func searchApp(f *fakeSearch) *application {
	return &application{models: data.Models{SearchPerson: f}}
}

func searchRequest(target string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.Header.Set("X-YearSlug", "2026")
	return r
}

func TestSearchPersonPassesTheQueryThrough(t *testing.T) {
	f := &fakeSearch{}
	w := httptest.NewRecorder()
	searchApp(f).searchPersonHandler(w, searchRequest("/api/search/person?q=12345678"))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if f.got.Text != "12345678" {
		t.Errorf("Text = %q, want the query verbatim", f.got.Text)
	}
	if f.got.Year != "2026" {
		t.Errorf("Year = %q, want it from X-YearSlug", f.got.Year)
	}
}

// The year scope is the default with a privacy dimension: this endpoint returns contact details
// for minors, and "this year" is a far smaller disclosure than "every year we have ever run".
// So it must be unreachable by anything short of the exact opt-in — a typo, a truthy-looking
// value or a missing parameter all leave the search where it was.
func TestSearchPersonOnlyWidensTheYearOnAnExactOptIn(t *testing.T) {
	cases := map[string]bool{
		"":                           false,
		"&includeOtherYears=true":    true,
		"&includeOtherYears=false":   false,
		"&includeOtherYears=1":       false,
		"&includeOtherYears=yes":     false,
		"&includeOtherYears=TRUE":    false,
		"&includeOtherYears=":        false,
		"&includeOtherYears=true%20": false,
		"&includePreviousYears=true": false, // the name this parameter nearly had
	}

	for suffix, want := range cases {
		f := &fakeSearch{}
		w := httptest.NewRecorder()
		searchApp(f).searchPersonHandler(w, searchRequest("/api/search/person?q=Anders"+suffix))

		if f.got.IncludeOtherYears != want {
			t.Errorf("%q: IncludeOtherYears = %v, want %v", suffix, f.got.IncludeOtherYears, want)
		}
	}
}

// An empty query is answered, not rejected: the SPA holds the search box open with nothing in
// it, and a 400 there would be noise. The query layer reports tooShort.
func TestSearchPersonAcceptsAnEmptyQuery(t *testing.T) {
	f := &fakeSearch{results: searchperson.Results{TooShort: true}}
	w := httptest.NewRecorder()
	searchApp(f).searchPersonHandler(w, searchRequest("/api/search/person"))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var envelope struct {
		Search searchperson.Results `json:"search"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !envelope.Search.TooShort {
		t.Error("tooShort not surfaced; the UI cannot tell 'keep typing' from 'ingen match'")
	}
}

// A missing projection is a 500, not an empty result. An empty result would read to an operator
// as "this person does not exist", which is the one answer this feature must never give wrongly.
func TestSearchPersonWithoutTheIndexIsAnError(t *testing.T) {
	// A real logger, discarded: ServerErrorResponse logs, and a nil Logger hangs rather than
	// panicking cleanly — which cost this test a 600-second timeout before it was noticed.
	app := &application{JsonApi: jsonapi.JsonApi{Logger: jsonlog.New(io.Discard, jsonlog.LevelOff)}}

	w := httptest.NewRecorder()
	app.searchPersonHandler(w, searchRequest("/api/search/person?q=Anders"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if strings.Contains(w.Body.String(), `"results"`) {
		t.Error("answered with a result shape; that reads as 'nobody found'")
	}
}

// The response must not grow the fields search deliberately leaves out. Asserted on the wire
// rather than on the struct, because a later embedded type could reintroduce them.
func TestSearchPersonResponseCarriesNoSensitiveExtras(t *testing.T) {
	f := &fakeSearch{results: searchperson.Results{Results: []searchperson.Result{{
		Kind: searchperson.KindSpejder, ID: "m-1", Name: "Anders", Phone: "12345678",
	}}}}
	w := httptest.NewRecorder()
	searchApp(f).searchPersonHandler(w, searchRequest("/api/search/person?q=Anders"))

	body := strings.ToLower(w.Body.String())
	for _, forbidden := range []string{"address", "postalcode", "birthday", "note"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("the response carries %q; that belongs on the detail page", forbidden)
		}
	}
}

// Route registration is deliberately **not** asserted here.
//
// `app.routes()` calls `app.Metrics`, which calls `expvar.NewInt` — and `expvar.Publish`
// panics on a duplicate name, so routes() can only be built once per test binary.
// `stream_test.go` already does that, and a second call from here made *that* test fail while
// passing in isolation. The route is verified against the running API instead (see task 181's
// log), which is better evidence anyway: it proves the wiring in main.go too.
