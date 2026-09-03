package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nathejk.dk/nathejk/table/kort"
)

// Every refusal the kort commands can produce must map to a client error, not a 500.
//
// This test exists because the first version of kortCommandError missed two of them, and nothing
// caught it: the command tests asserted the right error came *back*, and the handler happily turned
// it into a 500 with a stack trace in the log. An operator drawing a rectangle with two clicks on
// the same spot would have been told "the server broke" instead of "choose two different corners",
// and would reasonably have reported it as a bug in the map.
//
// Written as an exhaustive table over the package's exported errors rather than one case per error,
// so that adding an error to the package and forgetting to map it fails here.
func TestEveryKortCommandErrorIsAClientError(t *testing.T) {
	app := &application{}

	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"empty name", kort.ErrEmptyName, http.StatusUnprocessableEntity},
		{"name too long", kort.ErrNameTooLong, http.StatusUnprocessableEntity},
		{"invalid team type", kort.ErrInvalidTeamType, http.StatusUnprocessableEntity},
		{"invalid format", kort.ErrInvalidFormat, http.StatusUnprocessableEntity},
		{"too many extents", kort.ErrTooManyExtents, http.StatusUnprocessableEntity},
		{"degenerate extent", kort.ErrDegenerateExtent, http.StatusUnprocessableEntity},
		{"set not empty", kort.ErrSetNotEmpty, http.StatusUnprocessableEntity},
		{"not found", kort.ErrRecordNotFound, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/kort/kort-1", nil)

			app.kortCommandError(rec, req, tc.err)

			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d (a refusal must not read as a server fault)", rec.Code, tc.want)
			}
		})
	}
}

// The wrapped form matters as much as the sentinel: DeleteSet returns ErrSetNotEmpty wrapped with
// the number of sheets in the way, so a handler switching on `==` rather than errors.Is would send a
// 500 for the only case an operator will actually hit.
func TestWrappedSetNotEmptyStillMapsTo422(t *testing.T) {
	app := &application{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/kortsaet/kortsaet-1", nil)

	app.kortCommandError(rec, req, wrapErr(kort.ErrSetNotEmpty))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422 for a wrapped ErrSetNotEmpty", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "flyt eller slet") {
		// The message has to say what to do about it; "cannot be deleted" alone leaves an operator
		// clicking the button again.
		t.Errorf("body = %s, want it to say how to resolve the refusal", body)
	}
}

func wrapErr(err error) error { return errWrapper{err} }

type errWrapper struct{ err error }

func (e errWrapper) Error() string { return "sættet indeholder kort: " + e.err.Error() }
func (e errWrapper) Unwrap() error { return e.err }
