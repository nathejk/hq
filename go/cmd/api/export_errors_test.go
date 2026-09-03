package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
	"nathejk.dk/internal/jsonlog"
	"nathejk.dk/nathejk/table/personnel"
)

// failingPersonnelQueries stands in for a database that is down, slow past its timeout, or
// otherwise unable to answer — the situation the patrol list actually hit in production.
type failingPersonnelQueries struct {
	data.PersonnelInterface
	err error
}

func (f *failingPersonnelQueries) GetAll(context.Context, personnel.Filter) ([]*personnel.Person, error) {
	return nil, f.err
}

func failingExportApp(err error) *application {
	app := &application{
		models: data.Models{
			Personnel: &failingPersonnelQueries{err: err},
		},
	}
	// ServerErrorResponse logs, so a nil logger would panic and hide what is being tested.
	app.Logger = jsonlog.New(io.Discard, jsonlog.LevelOff)
	return app
}

// A failed query must produce exactly one response body.
//
// `GET /api/patrulje` once answered with an error envelope followed by `{"teams": null}`,
// because the handler called ServerErrorResponse and then carried on. Two JSON documents
// in one body parse as neither, so a plain database timeout reached the operator as an
// unexplained failure. Six sibling handlers had the same omission.
//
// Asserted by decoding and then asking whether anything is left, rather than by matching
// the body text: it is the *number of documents* that broke the client.
func TestFailedListQueryWritesOneBody(t *testing.T) {
	app := failingExportApp(errTestQuery)
	rec := httptest.NewRecorder()

	app.showBadutListHandler(rec, httptest.NewRequest(http.MethodGet, "/api/badut", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	assertSingleJSONDocument(t, rec.Body.Bytes())
	if bytes.Contains(rec.Body.Bytes(), []byte(`"personnel"`)) {
		t.Errorf("body carries a success envelope as well as the error: %s", rec.Body.String())
	}
}

// A failed query must not produce a spreadsheet.
//
// Worse than a malformed body: excelPersonnelHandler used to keep going and build a whole
// workbook from the nil slice, then write it over a response already carrying an error
// status. The operator downloads a plausible, correctly named, silently empty xlsx — a
// wrong answer presented as a right one, since nobody checks a download's status code.
func TestFailedExcelQuerySendsNoSpreadsheet(t *testing.T) {
	app := failingExportApp(errTestQuery)
	rec := httptest.NewRecorder()

	app.excelPersonnelHandler(rec, httptest.NewRequest(http.MethodGet, "/api/excel/personnel", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != "" {
		t.Errorf("Content-Disposition = %q, want none — a failure must not present itself as a download", cd)
	}
	if ct := rec.Header().Get("Content-Type"); ct == "application/octet-stream" {
		t.Errorf("Content-Type = %q, want a JSON error rather than a binary payload", ct)
	}
	// Every xlsx is a zip, and every zip starts "PK\x03\x04". Checking the magic bytes
	// catches a workbook appended to the error body, which a status-code assertion cannot.
	if bytes.Contains(rec.Body.Bytes(), []byte("PK\x03\x04")) {
		t.Errorf("body contains a zip/xlsx payload after the error response")
	}
	assertSingleJSONDocument(t, rec.Body.Bytes())
}

// The success path still produces a real workbook, so the guard above cannot be satisfied
// by never writing one.
func TestSucceedingExcelStillSendsSpreadsheet(t *testing.T) {
	app := &application{
		models: data.Models{
			Personnel: &okPersonnelQueries{},
			Signup:    &noSignupQueries{},
		},
	}
	app.Logger = jsonlog.New(io.Discard, jsonlog.LevelOff)

	rec := httptest.NewRecorder()
	app.excelPersonnelHandler(rec, httptest.NewRequest(http.MethodGet, "/api/excel/personnel", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("PK\x03\x04")) {
		t.Errorf("body is not an xlsx (zip) payload")
	}
	if cd := rec.Header().Get("Content-Disposition"); cd == "" {
		t.Errorf("Content-Disposition is empty, so the browser will not save the export")
	}
	// Set from the buffer, and worth asserting: it is what a proxy needs in order not to
	// have to buffer or chunk the download itself.
	if cl := rec.Header().Get("Content-Length"); cl == "" || cl == "0" {
		t.Errorf("Content-Length = %q, want the workbook's size", cl)
	}
}

type okPersonnelQueries struct {
	data.PersonnelInterface
}

func (okPersonnelQueries) GetAll(context.Context, personnel.Filter) ([]*personnel.Person, error) {
	return []*personnel.Person{{
		ID:          types.UserID("user-1"),
		UserType:    "gøgler",
		Name:        "Viktor Madsen",
		Additionals: map[string]any{},
	}}, nil
}

type noSignupQueries struct{}

func (noSignupQueries) GetByID(types.TeamID) (*data.Signup, error) { return &data.Signup{}, nil }
func (noSignupQueries) TeamIDsByType(context.Context, types.YearSlug, types.TeamType) (map[types.TeamID]bool, error) {
	return map[types.TeamID]bool{}, nil
}

// assertSingleJSONDocument fails if body is not exactly one JSON value.
func assertSingleJSONDocument(t *testing.T, body []byte) {
	t.Helper()

	dec := json.NewDecoder(bytes.NewReader(body))
	var first any
	if err := dec.Decode(&first); err != nil {
		t.Fatalf("body is not JSON: %v (%s)", err, body)
	}
	var second any
	if err := dec.Decode(&second); err == nil {
		t.Errorf("body contains a second document, so no client can parse it: %s", body)
	} else if err != io.EOF {
		t.Errorf("trailing bytes after the JSON document: %v (%s)", err, body)
	}
}

var errTestQuery = errTestQueryType{}

type errTestQueryType struct{}

func (errTestQueryType) Error() string { return "context deadline exceeded" }
