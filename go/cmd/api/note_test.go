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
	"nathejk.dk/nathejk/table/spejdernote"
)

// The note endpoints at the HTTP boundary: what the SPA sends, what comes back, and that refusals
// arrive as Danish the crew can act on rather than as domain errors. The rules themselves are tested
// in the spejdernote package.

// --- fakes ---

type fakeNoteCommands struct {
	note     string
	noteID   spejdernote.NoteID
	comments int
	updates  int
	err      error
}

func (f *fakeNoteCommands) Comment(_ context.Context, _ spejdernote.Actor, _ types.YearSlug, _ types.MemberID, note string) (spejdernote.NoteID, error) {
	f.comments++
	f.note = note
	if f.err != nil {
		return "", f.err
	}
	return "n-minted", nil
}

func (f *fakeNoteCommands) UpdateComment(_ context.Context, _ spejdernote.Actor, _ types.YearSlug, _ types.MemberID, id spejdernote.NoteID, note string) error {
	f.updates++
	f.noteID = id
	f.note = note
	return f.err
}

type fakeNoteQueries struct {
	notes []spejdernote.Note
}

func (f *fakeNoteQueries) GetByMember(context.Context, types.YearSlug, types.MemberID) ([]spejdernote.Note, error) {
	return f.notes, nil
}
func (f *fakeNoteQueries) GetByID(context.Context, types.YearSlug, spejdernote.NoteID) (*spejdernote.Note, error) {
	return nil, nil
}
func (f *fakeNoteQueries) SummaryByMembers(context.Context, types.YearSlug, []types.MemberID) (map[types.MemberID]spejdernote.Summary, error) {
	return nil, nil
}

func noteApp(cmd *fakeNoteCommands, q *fakeNoteQueries) *application {
	return &application{
		models:   data.Models{Note: q},
		commands: commands.Commands{Note: cmd},
	}
}

// Handlers are invoked directly: app.routes() installs app.Metrics, whose expvar.NewInt panics on a
// duplicate name, so it can be built at most once per process and stream_test.go already does. That
// one construction is also what proves these routes register without an httprouter conflict.
func noteRequest2(t *testing.T, method, memberID, noteID string, body any) *http.Request {
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
	req := httptest.NewRequest(method, "/api/member/"+memberID+"/notes", reader)
	req.Header.Set("X-YearSlug", "2026")

	params := httprouter.Params{}
	if memberID != "" {
		params = append(params, httprouter.Param{Key: "memberId", Value: memberID})
	}
	if noteID != "" {
		params = append(params, httprouter.Param{Key: "noteId", Value: noteID})
	}
	return req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))
}

// --- tests ---

func TestListNotesServesTheTrail(t *testing.T) {
	q := &fakeNoteQueries{notes: []spejdernote.Note{
		{NoteID: "n-1", MemberID: "m-1", Note: "Ringet til mor"},
	}}
	rec := httptest.NewRecorder()
	noteApp(&fakeNoteCommands{}, q).listMemberNotesHandler(rec, noteRequest2(t, http.MethodGet, "m-1", "", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Ringet til mor") {
		t.Errorf("the trail did not reach the client: %s", rec.Body.String())
	}
}

// An empty trail is an empty array, never null — a scout nobody has written about is the ordinary
// case, and every client would otherwise have to defend against it. Same lesson as the shelter
// sections in task 092, which shipped `null` to a browser before it was caught.
func TestListNotesReturnsAnArrayWhenEmpty(t *testing.T) {
	rec := httptest.NewRecorder()
	noteApp(&fakeNoteCommands{}, &fakeNoteQueries{}).listMemberNotesHandler(rec, noteRequest2(t, http.MethodGet, "m-1", "", nil))

	// Asserted on the raw JSON: decoding into []T turns both `null` and `[]` into a nil slice and
	// would pass either way.
	if strings.Contains(rec.Body.String(), `"notes": null`) {
		t.Errorf("an empty trail serialised as null: %s", rec.Body.String())
	}
}

func TestCreateNoteReturnsTheMintedID(t *testing.T) {
	cmd := &fakeNoteCommands{}
	rec := httptest.NewRecorder()
	noteApp(cmd, &fakeNoteQueries{}).createMemberNoteHandler(rec,
		noteRequest2(t, http.MethodPost, "m-1", "", map[string]string{"note": "Mor henter kl. 06"}))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	if cmd.comments != 1 || cmd.note != "Mor henter kl. 06" {
		t.Errorf("command saw %d calls with %q", cmd.comments, cmd.note)
	}
	if !strings.Contains(rec.Body.String(), "n-minted") {
		t.Errorf("the note id did not come back; the client needs it to edit: %s", rec.Body.String())
	}
}

// **No sosId.** The shelter may be looking after a scout nobody opened a case about, so a note must
// be writable without one — unlike the nødtelefon's own member endpoints.
func TestNoteWritesDoNotRequireACase(t *testing.T) {
	cmd := &fakeNoteCommands{}
	rec := httptest.NewRecorder()
	noteApp(cmd, &fakeNoteQueries{}).createMemberNoteHandler(rec,
		noteRequest2(t, http.MethodPost, "m-1", "", map[string]string{"note": "Ingen sag her"}))

	if rec.Code == http.StatusUnprocessableEntity {
		t.Fatalf("a case was demanded: %s", rec.Body.String())
	}
	if cmd.comments != 1 {
		t.Error("the command was not called")
	}
}

func TestUpdateNotePassesBothIDs(t *testing.T) {
	cmd := &fakeNoteCommands{}
	rec := httptest.NewRecorder()
	noteApp(cmd, &fakeNoteQueries{}).updateMemberNoteHandler(rec,
		noteRequest2(t, http.MethodPatch, "m-1", "n-1", map[string]string{"note": "Rettet"}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	if cmd.updates != 1 || cmd.noteID != "n-1" {
		t.Errorf("command saw %d updates for note %q", cmd.updates, cmd.noteID)
	}
}

// Refusals in Danish, on the field they concern, and never the raw domain error.
func TestNoteRefusalsAreWordedForTheCrew(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		field string
	}{
		{"empty", spejdernote.ErrEmptyNote, "note"},
		{"too long", spejdernote.ErrNoteTooLong, "note"},
		{"wrong member", spejdernote.ErrWrongMember, "noteId"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := noteApp(&fakeNoteCommands{err: tt.err}, &fakeNoteQueries{})
			rec := httptest.NewRecorder()
			app.createMemberNoteHandler(rec, noteRequest2(t, http.MethodPost, "m-1", "", map[string]string{"note": "x"}))

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.field) {
				t.Errorf("expected the message on %q, got %s", tt.field, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), tt.err.Error()) {
				t.Errorf("the raw domain error reached the client: %s", rec.Body.String())
			}
		})
	}
}

// A note about nobody, or a correction to nothing, is a 404 rather than a validation failure: the
// request names a resource that does not exist.
func TestNoteHandlersRequireTheirIDs(t *testing.T) {
	app := noteApp(&fakeNoteCommands{}, &fakeNoteQueries{})

	rec := httptest.NewRecorder()
	app.listMemberNotesHandler(rec, noteRequest2(t, http.MethodGet, "", "", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("list without a member = %d, want 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	app.updateMemberNoteHandler(rec, noteRequest2(t, http.MethodPatch, "m-1", "", map[string]string{"note": "x"}))
	if rec.Code != http.StatusNotFound {
		t.Errorf("correction without a note id = %d, want 404", rec.Code)
	}
}
