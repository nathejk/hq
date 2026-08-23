package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/spejdernote"
)

// Member notes (PRD 008): the prose trail about a scout — what was agreed with a guardian, what was
// said on the phone, what the next shift needs to know.
//
// # Why these are not part of GET /api/member/:memberId
//
// That endpoint feeds a modal opened one scout at a time and already carries address, birthday and
// the full status history. The trail is a separate resource because it is wanted separately: the SPA
// caches and invalidates it per member, and the shelter list wants a *summary* of it for every row
// without any of this detail. Folding them together would mean the row summaries could not be served
// without the whole payload, and a note written anywhere would invalidate everything.
//
// # No case, ever
//
// None of these takes a sosId, and notes are never written to an SOS timeline. The shelter has no
// case by design (PRD 007), and one text with one place to correct it beats a copy on a case that
// diverges the first time somebody fixes a typo.

// noteRequest is the body for writing and correcting.
type noteRequest struct {
	Note string `json:"note"`
}

// listMemberNotesHandler serves one scout's trail.
//
// @Summary     A scout's note trail
// @Description Every note about the scout, oldest first — a trail is a story and reads in the order it happened. Notes are prose written by the crew: what was agreed with a guardian, what was said on the phone, what the next shift needs to know. Year comes from the X-YearSlug header, or the current year.
// @Tags        notes
// @Produce     json
// @Param       memberId path string true "Member id"
// @Success     200 {object} map[string]interface{} "envelope with a \"notes\" array"
// @Failure     404 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/member/{memberId}/notes [get]
func (app *application) listMemberNotesHandler(w http.ResponseWriter, r *http.Request) {
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	notes, err := app.models.Note.GetByMember(r.Context(), app.YearSlug(r), memberID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	// An empty trail is an empty array, never null: a scout nobody has written about is the
	// ordinary case, and every client would otherwise have to defend against it — the same lesson
	// as the shelter sections (task 092).
	if notes == nil {
		notes = []spejdernote.Note{}
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"notes": notes}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// createMemberNoteHandler adds a note.
//
// @Summary     Write a note about a scout
// @Description Adds a note to the scout's trail. Prose, up to 2000 characters, trimmed; an empty note is refused. Requires no SOS case — the shelter may be looking after a scout nobody opened a case about. Allowed for any member of the year, including one still racing: the nødtelefon takes calls about scouts who have not dropped out. Until HQ has login the note is recorded with no author, which is accepted.
// @Tags        notes
// @Accept      json
// @Produce     json
// @Param       memberId path string true "Member id"
// @Param       body body noteRequest true "The note"
// @Success     201 {object} map[string]interface{} "envelope with \"noteId\""
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/member/{memberId}/notes [post]
func (app *application) createMemberNoteHandler(w http.ResponseWriter, r *http.Request) {
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input noteRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	noteID, err := app.commands.Note.Comment(r.Context(), app.noteActor(r), app.YearSlug(r), memberID, input.Note)
	if err != nil {
		app.noteCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"noteId": noteID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updateMemberNoteHandler corrects a note.
//
// @Summary     Correct a note
// @Description Amends a note's text. Intended for typos — the trail is a record, so the UI encourages a further note rather than a rewrite — and every version stays in the event stream either way. The note must belong to the scout named in the path. Resubmitting identical text changes nothing and answers 200. Not restricted to the note's author, because until login there is no author to compare against.
// @Tags        notes
// @Accept      json
// @Produce     json
// @Param       memberId path string true "Member id"
// @Param       noteId path string true "Note id"
// @Param       body body noteRequest true "The corrected note"
// @Success     200 {object} map[string]interface{} "envelope with \"noteId\""
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/member/{memberId}/notes/{noteId} [patch]
func (app *application) updateMemberNoteHandler(w http.ResponseWriter, r *http.Request) {
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	noteID := spejdernote.NoteID(app.ReadNamedParam(r, "noteId"))
	if memberID == "" || noteID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input noteRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	err := app.commands.Note.UpdateComment(r.Context(), app.noteActor(r), app.YearSlug(r), memberID, noteID, input.Note)
	if err != nil {
		app.noteCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"noteId": noteID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// noteActor is who wrote the note, as far as the platform knows.
//
// Empty until HQ has login (PRD 001 §6), and recorded anyway so notes start being attributed the day
// accounts exist with no change here. An unsigned trail on race day is accepted (PRD 008 §5) —
// weaker as accountability, still far better than paper.
func (app *application) noteActor(r *http.Request) spejdernote.Actor {
	user := app.actor(r)
	return spejdernote.Actor{UserID: user.UserID, Name: user.Name}
}

// noteCommandError maps the note commands' refusals onto responses.
//
// Danish, and phrased as what is wrong with the request rather than as the rule it broke. The one
// worth distinguishing is ErrWrongMember: telling an operator "no such note" about a note plainly on
// their screen would send them looking for a bug that is not there.
func (app *application) noteCommandError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, spejdernote.ErrEmptyNote):
		app.FailedValidationResponse(w, r, map[string]string{"note": "noten kan ikke være tom"})
	case errors.Is(err, spejdernote.ErrNoteTooLong):
		app.FailedValidationResponse(w, r, map[string]string{
			"note": fmt.Sprintf("noten må højst være %d tegn", spejdernote.MaxNoteLength),
		})
	case errors.Is(err, spejdernote.ErrWrongMember):
		app.FailedValidationResponse(w, r, map[string]string{
			"noteId": "noten hører til en anden spejder",
		})
	case errors.Is(err, spejdernote.ErrRecordNotFound):
		app.NotFoundResponse(w, r)
	default:
		app.ServerErrorResponse(w, r, err)
	}
}
