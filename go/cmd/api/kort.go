package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/kort"
)

// Map sets (PRD 010): the sheets we print are grouped into sets — most often "Patruljer" and one
// for everybody else.
//
// # Why the set is an entity with its own endpoints
//
// Because `teamType` belongs to the set as a whole. Held on each sheet instead, five sheets in one
// set would each carry a copy that can disagree, and "which set is the spejder set?" would have
// five possibly-conflicting answers.
//
// # There is no GET here
//
// Sets are read as part of `GET /api/kort` (task 125), which returns the year's sets with their
// sheets nested. The whole year is a handful of records, so one response serves both the settings
// modal and the hej-app, and a separate set listing would be a second code path with no caller.

// kortsaetRequest is the body for creating and updating a set.
//
// TeamType is a pointer so that an absent field and an explicit null are both "not for a specific
// team type" — which is the ordinary case, since the crew set is unmarked and klaner draw from it.
// An empty string means the same thing and is normalised by the command.
type kortsaetRequest struct {
	Name     string          `json:"name"`
	TeamType *types.TeamType `json:"teamType"`
}

// kortsaetSortRequest is the body for reordering sets.
type kortsaetSortRequest struct {
	KortsaetIDs []kort.KortsaetID `json:"kortsaetIds"`
}

// createKortsaetHandler adds a set of map sheets.
//
//	@Summary		Create a map set
//	@Description	Creates a set of map sheets for the year — most years there are two, one for patruljer and one for everybody else, but a year may have three. Sets are named by the operator, never chosen from a fixed list. `teamType` optionally marks which team type the set is *specifically for*, and is what the hej-app matches on instead of the Danish name; leave it out for the general crew set, which klaner also draw from. Several sets may carry the same team type. Year comes from the X-YearSlug header, or the current year.
//	@Tags			kort
//	@Accept			json
//	@Produce		json
//	@Param			body	body		kortsaetRequest	true	"The set"
//	@Success		201		{object}	map[string]interface{}	"envelope with \"kortsaetId\""
//	@Failure		422		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/api/kortsaet [post]
func (app *application) createKortsaetHandler(w http.ResponseWriter, r *http.Request) {
	var input kortsaetRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	id, err := app.commands.KortSet.CreateSet(r.Context(), app.kortActor(r), app.YearSlug(r), input.Name, input.TeamType)
	if err != nil {
		app.kortCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"kortsaetId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updateKortsaetHandler renames a set or changes its team type.
//
//	@Summary		Update a map set
//	@Description	Sets the set's name and team type. Both are always submitted: the whole record is carried so that clearing the team type is expressible at all — with a patch, "clear it" and "do not touch it" would both be an absent field. Resubmitting identical values changes nothing, publishes no event and answers 200, so an operator who opens and closes the editor does not make every other session refetch. Sort order is not set here; reorder with PUT /api/kortsaet.
//	@Tags			kort
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string			true	"Set id"
//	@Param			body	body		kortsaetRequest	true	"The set"
//	@Success		200		{object}	map[string]interface{}	"envelope with \"kortsaetId\""
//	@Failure		404		{object}	map[string]interface{}
//	@Failure		422		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/api/kortsaet/{id} [put]
func (app *application) updateKortsaetHandler(w http.ResponseWriter, r *http.Request) {
	id := kort.KortsaetID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input kortsaetRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	err := app.commands.KortSet.UpdateSet(r.Context(), app.kortActor(r), app.YearSlug(r), id, input.Name, input.TeamType)
	if err != nil {
		app.kortCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"kortsaetId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// deleteKortsaetHandler removes an empty set.
//
//	@Summary		Delete a map set
//	@Description	Deletes a set. Refused with 422 while the set still holds sheets — deliberately not a cascade, because a mis-click in a list would otherwise cost a season of map definitions and an event stream offers no undo. Move or delete the sheets first.
//	@Tags			kort
//	@Produce		json
//	@Param			id	path		string	true	"Set id"
//	@Success		200	{object}	map[string]interface{}	"envelope with \"kortsaetId\""
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		422	{object}	map[string]interface{}	"the set still holds maps"
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/kortsaet/{id} [delete]
func (app *application) deleteKortsaetHandler(w http.ResponseWriter, r *http.Request) {
	id := kort.KortsaetID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if err := app.commands.KortSet.DeleteSet(r.Context(), app.kortActor(r), app.YearSlug(r), id); err != nil {
		app.kortCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"kortsaetId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// sortKortsaetHandler reorders the year's sets.
//
//	@Summary		Reorder map sets
//	@Description	Records a new order for the year's sets, as one event for the whole collection rather than one update per set — a drag is one gesture, and N events would let a replay observe orders that never existed on screen. Sets not named keep their current position. This is PUT on the collection rather than /api/kortsaet/sorted because httprouter refuses a static segment beside a wildcard at the same level, and Danish gives "kortsæt" no plural to move the sort route to.
//	@Tags			kort
//	@Accept			json
//	@Produce		json
//	@Param			body	body		kortsaetSortRequest	true	"The set ids, in order"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		422		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/api/kortsaet [put]
func (app *application) sortKortsaetHandler(w http.ResponseWriter, r *http.Request) {
	var input kortsaetSortRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.commands.KortSet.SortSets(r.Context(), app.kortActor(r), app.YearSlug(r), input.KortsaetIDs); err != nil {
		app.kortCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"kortsaetIds": input.KortsaetIDs}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// kortActor is who made the change, as far as the platform knows.
//
// Empty until HQ has login (PRD 001 §6), recorded anyway so map edits start being attributed the
// day accounts exist with no change here.
func (app *application) kortActor(r *http.Request) kort.Actor {
	user := app.actor(r)
	return kort.Actor{UserID: user.UserID, Name: user.Name}
}

// kortCommandError maps the kort commands' refusals onto responses.
//
// Danish, and phrased as what is wrong rather than as the rule that was broken. ErrSetNotEmpty is
// the one that has to say what to do about it: "kan ikke slettes" alone would leave an operator
// clicking the button again.
func (app *application) kortCommandError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, kort.ErrEmptyName):
		app.FailedValidationResponse(w, r, map[string]string{"name": "navnet kan ikke være tomt"})
	case errors.Is(err, kort.ErrNameTooLong):
		app.FailedValidationResponse(w, r, map[string]string{
			"name": fmt.Sprintf("navnet må højst være %d tegn", kort.MaxNameLength),
		})
	case errors.Is(err, kort.ErrInvalidTeamType):
		app.FailedValidationResponse(w, r, map[string]string{"teamType": "ukendt holdtype"})
	case errors.Is(err, kort.ErrInvalidFormat):
		app.FailedValidationResponse(w, r, map[string]string{"format": "ukendt format"})
	case errors.Is(err, kort.ErrSetNotEmpty):
		app.FailedValidationResponse(w, r, map[string]string{
			"kortsaetId": "sættet indeholder stadig kort — flyt eller slet dem først",
		})
	case errors.Is(err, kort.ErrRecordNotFound):
		app.NotFoundResponse(w, r)
	default:
		app.ServerErrorResponse(w, r, err)
	}
}
