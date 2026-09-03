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

// kortRequest is the body for creating a sheet.
type kortRequest struct {
	KortsaetID kort.KortsaetID `json:"kortsaetId"`
	Name       string          `json:"name"`
}

// kortUpdateRequest is a partial edit to a sheet.
//
// Pointers so that an absent field is left alone: the settings dialog edits a sheet's description
// in one place and its checkpoints in another, and neither should have to restate what it never
// loaded. `extents: []` clears the extents, which is what happens when an operator decides a sheet
// is really a skitse — and is why it is a pointer to a slice rather than a slice.
type kortUpdateRequest struct {
	KortsaetID *kort.KortsaetID `json:"kortsaetId"`
	Name       *string          `json:"name"`
	Format     *kort.Format     `json:"format"`
	Note       *string          `json:"note"`
	Extents    *[]kort.Extent   `json:"extents"`
}

// kortCheckpointsRequest replaces the checkpoints drawn on a sheet.
type kortCheckpointsRequest struct {
	CheckpointIDs []types.CheckpointID `json:"checkpointIds"`
}

// kortSortRequest is the body for reordering a set's sheets.
type kortSortRequest struct {
	KortIDs []kort.KortID `json:"kortIds"`
}

// showKortHandler serves the year's sets with their sheets.
//
//	@Summary		The year's map sheets, grouped by set
//	@Description	Every set defined for the year, in order, each with its sheets in handout order. This is the endpoint the hej-app reads. Three things about it are worth knowing before writing a client. (1) Find the patrol sheets via a set's `teamType`, never by its name — names are Danish free text an organizer may rename mid-season. (2) `teamType` is nullable and is *not* unique: it means "this set is specifically for this team type", so an unmarked set is the general crew set, which klaner also draw from. Filtering by `klan` will usually return nothing, and that is not an error — fall back to the unmarked set. (3) A sheet's `checkpointIds` are the checkpoints drawn on it, and are what may be revealed once the sheet is known to be in a team's hands; ids that no longer resolve are filtered out here, so the list is always live. `orphanKort` holds sheets whose set is unknown — normally empty, and present so a mis-assigned sheet cannot become invisible. Year comes from the X-YearSlug header, or the current year.
//	@Tags			kort
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}	"envelope with \"kortsaet\" and \"orphanKort\" arrays"
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/kort [get]
func (app *application) showKortHandler(w http.ResponseWriter, r *http.Request) {
	year := app.YearSlug(r)

	sets, err := app.models.Kort.Sets(r.Context(), year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	maps, err := app.models.Kort.Maps(r.Context(), year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	nested, orphans := kort.Nest(sets, maps)
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{
		"kortsaet":   nested,
		"orphanKort": orphans,
	}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// createKortHandler adds a sheet to a set.
//
//	@Summary		Create a map sheet
//	@Description	Creates a sheet in a set, with a name. A sheet is one thing physically handed to a team — one QR code, one reveal — so a double-sided A3 is one sheet with two extents, not two sheets. Only the set and the name are given here: a sheet is described after it exists, because an operator adds "Kort 3" before knowing its format or drawing its extent, and a half-known sheet is exactly when writing it down is most useful. The set is not required to exist yet.
//	@Tags			kort
//	@Accept			json
//	@Produce		json
//	@Param			body	body		kortRequest	true	"The sheet"
//	@Success		201		{object}	map[string]interface{}	"envelope with \"kortId\""
//	@Failure		422		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/api/kort [post]
func (app *application) createKortHandler(w http.ResponseWriter, r *http.Request) {
	var input kortRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	id, err := app.commands.Kort.Create(r.Context(), app.kortActor(r), app.YearSlug(r), input.KortsaetID, input.Name)
	if err != nil {
		app.kortCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"kortId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updateKortHandler edits a sheet's description.
//
//	@Summary		Update a map sheet
//	@Description	Edits any of name, set, format (a4/a3/skitse/andet), note and extents. Absent fields are left alone, so the checkpoint picker and this endpoint do not overwrite each other. Extents are the ground the sheet shows: none for a skitse, one for a normal sheet, two for a double-sided one — the two are simply two areas, with no front/back distinction and no per-side checkpoints, because both sides are handed over at once. Corners may be given either way round and are stored as a true north-west/south-east pair, so re-sending the same rectangle with its corners swapped is not an edit. A rectangle with no area is refused: it draws as nothing, which reads as a failed save. Changing `kortsaetId` moves the sheet to another set; ordering is separate. Nothing changed means no event and no live signal.
//	@Tags			kort
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Map id"
//	@Param			body	body		kortUpdateRequest	true	"The fields to change"
//	@Success		200		{object}	map[string]interface{}	"envelope with \"kortId\""
//	@Failure		404		{object}	map[string]interface{}
//	@Failure		422		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/api/kort/{id} [put]
func (app *application) updateKortHandler(w http.ResponseWriter, r *http.Request) {
	id := kort.KortID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input kortUpdateRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	err := app.commands.Kort.Update(r.Context(), app.kortActor(r), app.YearSlug(r), id, kort.UpdateRequest{
		KortsaetID: input.KortsaetID,
		Name:       input.Name,
		Format:     input.Format,
		Note:       input.Note,
		Extents:    input.Extents,
	})
	if err != nil {
		app.kortCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"kortId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// deleteKortHandler removes a sheet.
//
//	@Summary		Delete a map sheet
//	@Description	Deletes a sheet. Its checkpoints are untouched — they exist independently of any map and are almost certainly drawn on another sheet too, since the crew map covers the same ground as every patrol sheet put together.
//	@Tags			kort
//	@Produce		json
//	@Param			id	path		string	true	"Map id"
//	@Success		200	{object}	map[string]interface{}	"envelope with \"kortId\""
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Router			/api/kort/{id} [delete]
func (app *application) deleteKortHandler(w http.ResponseWriter, r *http.Request) {
	id := kort.KortID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if err := app.commands.Kort.Delete(r.Context(), app.kortActor(r), app.YearSlug(r), id); err != nil {
		app.kortCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"kortId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// setKortCheckpointsHandler replaces the checkpoints drawn on a sheet.
//
//	@Summary		Set a map sheet's checkpoints
//	@Description	Replaces the whole list — the UI is a set of tick-boxes, so "these ones" is the operator's intent, and incremental adds and removes would let two concurrent editors interleave into a list neither chose. Repeats are ignored and order is preserved. An empty list is allowed: an overview map for drivers legitimately shows no checkpoints. Ids are not checked for existence, because unresolvable ids are filtered when the maps are read — which also covers a checkpoint deleted after this save. A checkpoint may be on any number of sheets, including several in one set: adjacent sheets overlap by design. Re-saving an unchanged selection publishes nothing.
//	@Tags			kort
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Map id"
//	@Param			body	body		kortCheckpointsRequest	true	"The checkpoints drawn on the sheet"
//	@Success		200		{object}	map[string]interface{}	"envelope with \"kortId\""
//	@Failure		404		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/api/kort/{id}/checkpoints [put]
func (app *application) setKortCheckpointsHandler(w http.ResponseWriter, r *http.Request) {
	id := kort.KortID(app.ReadNamedParam(r, "id"))
	if id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input kortCheckpointsRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	err := app.commands.Kort.SetCheckpoints(r.Context(), app.kortActor(r), app.YearSlug(r), id, input.CheckpointIDs)
	if err != nil {
		app.kortCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"kortId": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// sortKortHandler reorders a set's sheets.
//
//	@Summary		Reorder a set's map sheets
//	@Description	Records the handout order of sheets along the route, as one event for the whole collection rather than one update per sheet. Sheets not named keep their position, so reordering one set need not restate the others. Moving a sheet to another set is not a reorder — use PUT /api/kort/{id} with a new kortsaetId. This lives under the set rather than at /api/kort/sorted because httprouter refuses a static segment beside a wildcard at the same level, and because handout order is only meaningful within a set.
//	@Tags			kort
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string			true	"Set id"
//	@Param			body	body		kortSortRequest	true	"The map ids, in handout order"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		422		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Router			/api/kortsaet/{id}/kort [put]
func (app *application) sortKortHandler(w http.ResponseWriter, r *http.Request) {
	var input kortSortRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.commands.Kort.SortMaps(r.Context(), app.kortActor(r), app.YearSlug(r), input.KortIDs); err != nil {
		app.kortCommandError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"kortIds": input.KortIDs}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
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
	case errors.Is(err, kort.ErrTooManyExtents):
		app.FailedValidationResponse(w, r, map[string]string{
			"extents": fmt.Sprintf("et kort kan højst have %d områder — for- og bagside", kort.MaxExtents),
		})
	case errors.Is(err, kort.ErrDegenerateExtent):
		app.FailedValidationResponse(w, r, map[string]string{
			"extents": "området har ingen udstrækning — vælg to forskellige hjørner",
		})
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
