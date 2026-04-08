package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/validator"
	"nathejk.dk/nathejk/table/checkgroup"
	"nathejk.dk/nathejk/table/checkpersonnel"
	"nathejk.dk/nathejk/table/checkpoint"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/personnel"
)

type CheckgroupStats struct {
	CheckgroupID types.CheckgroupID `json:"checkgroupId"`
	OnTime       int                `json:"onTime"`
	Late         int                `json:"late"`
	Expired      int                `json:"expired"`
	Missing      int                `json:"missing"`
}

func (app *application) listCheckgroupsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		checkgroup.Filter
	}

	v := validator.New()
	qs := r.URL.Query()
	input.Filter.Year = app.YearSlug(r)
	input.Filter.Page = app.ReadInt(qs, "page", 1, v)
	input.Filter.PageSize = app.ReadInt(qs, "page_size", 1000, v)
	input.Filter.Sort = app.ReadString(qs, "sort", "id")
	input.Filter.SortSafelist = []string{"id"}

	if input.Filter.Validate(v); !v.Valid() {
		app.FailedValidationResponse(w, r, v.Errors)
		return
	}

	//status, metadata, err := app.models.GetCheckgroupsStatus(input.Filters)
	checkgroups, err := app.models.Checkgroup.GetAll(r.Context(), input.Filter) //, startedTeamIDs, discontinuedTeamIDs)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	filter := checkpoint.Filter{CheckgroupIDs: []types.CheckgroupID{}}
	for _, cg := range checkgroups {
		filter.CheckgroupIDs = append(filter.CheckgroupIDs, cg.ID)
	}
	checkpoints, err := app.models.Checkpoint.GetAll(r.Context(), filter)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	cpFilter := checkpersonnel.Filter{CheckgroupIDs: filter.CheckgroupIDs}
	assignedPersonnel, _ := app.models.Checkpersonnel.GetAll(r.Context(), cpFilter)
	if assignedPersonnel == nil {
		assignedPersonnel = []checkpersonnel.Checkpersonnel{}
	}
	availablePersonnel, _ := app.models.Personnel.GetAll(r.Context(), personnel.Filter{YearSlug: input.Filter.Year, UserTypes: []string{"friend"}})

	// Compute scan stats per checkgroup
	startedTeamIDs, err := app.models.Patrulje.GetStartedTeamIDs(r.Context(), patrulje.Filter{YearSlug: string(input.Filter.Year)})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	startedTeamCount := len(startedTeamIDs)

	// For each checkgroup, find teams scanned on-time vs late
	checkgroupStatsMap := make(map[types.CheckgroupID]*CheckgroupStats)
	for _, cg := range checkgroups {
		checkgroupStatsMap[cg.ID] = &CheckgroupStats{CheckgroupID: cg.ID, Missing: startedTeamCount}
	}

	if len(filter.CheckgroupIDs) > 0 {
		statsQuery := `
			SELECT cpt.checkgroupId, s.teamId,
				MAX(CASE WHEN s.uts >= cpt.openFromUts AND s.uts <= cpt.openUntilUts THEN 1 ELSE 0 END) as wasOnTime
			FROM scan s
			JOIN checkpersonnel cpn ON s.scannerId = cpn.userId AND s.uts >= cpn.startUts AND s.uts <= cpn.endUts
			JOIN checkpoint cpt ON cpn.checkpointId = cpt.id
			WHERE cpt.checkgroupId IN (?` + strings.Repeat(",?", len(filter.CheckgroupIDs)-1) + `)
			GROUP BY cpt.checkgroupId, s.teamId`
		args := make([]any, len(filter.CheckgroupIDs))
		for i, id := range filter.CheckgroupIDs {
			args[i] = string(id)
		}
		statsRows, err := app.db.DB().QueryContext(r.Context(), statsQuery, args...)
		if err == nil {
			defer statsRows.Close()
			for statsRows.Next() {
				var cgID types.CheckgroupID
				var teamID types.TeamID
				var wasOnTime int
				if err := statsRows.Scan(&cgID, &teamID, &wasOnTime); err != nil {
					continue
				}
				st := checkgroupStatsMap[cgID]
				if st == nil {
					continue
				}
				if wasOnTime == 1 {
					st.OnTime++
				} else {
					st.Late++
				}
				st.Missing--
			}
		}
	}

	checkgroupStats := make([]CheckgroupStats, 0, len(checkgroups))
	for _, cg := range checkgroups {
		if st, ok := checkgroupStatsMap[cg.ID]; ok {
			checkgroupStats = append(checkgroupStats, *st)
		}
	}

	envelope := jsonapi.Envelope{
		//"metadata": metadata,
		"checkgroups":       checkgroups,
		"checkpoints":       checkpoints,
		"assignedPersonnel": assignedPersonnel,
		"personnel":         availablePersonnel,
		"startedTeamCount":  startedTeamCount,
		"checkgroupStats":   checkgroupStats,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) createCheckgroupHandler(w http.ResponseWriter, r *http.Request) {
	yearSlug := app.YearSlug(r)
	checkgroupID, err := app.commands.Checkgroup.Create(r.Context(), yearSlug)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"checkgroupId": checkgroupID,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) checkgroupHandler(w http.ResponseWriter, r *http.Request) {
	cgID := types.CheckgroupID(app.ReadNamedParam(r, "id"))
	cg, err := app.models.Checkgroup.GetByID(r.Context(), cgID)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	cps, _ := app.models.Checkpoint.GetAll(r.Context(), checkpoint.Filter{CheckgroupIDs: []types.CheckgroupID{cgID}})
	assignedPersonnel, _ := app.models.Checkpersonnel.GetAll(r.Context(), checkpersonnel.Filter{CheckgroupIDs: []types.CheckgroupID{cgID}})
	availablePersonnel, _ := app.models.Personnel.GetAll(r.Context(), personnel.Filter{YearSlug: cg.YearSlug, UserTypes: []string{"friend"}})
	year, _ := app.models.Year.GetByID(r.Context(), cg.YearSlug)

	envelope := jsonapi.Envelope{
		"checkgroup":         cg,
		"checkpoints":        cps,
		"assignedPersonnel":  assignedPersonnel,
		"availablePersonnel": availablePersonnel,
		"year":               year,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) deleteCheckgroupHandler(w http.ResponseWriter, r *http.Request) {
	cgID := types.CheckgroupID(app.ReadNamedParam(r, "id"))
	cg, err := app.models.Checkgroup.GetByID(r.Context(), cgID)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.commands.Checkgroup.Delete(r.Context(), cgID); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": "ok", "cg": cg}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

type CheckgroupScheme string

func (s CheckgroupScheme) Scheme() types.CheckgroupScheme {
	return types.CheckgroupScheme(strings.Split(string(s), ":")[0])
}
func (s CheckgroupScheme) RelativeCheckgroupID() *types.CheckgroupID {
	scheme := strings.Split(string(s), ":")
	if len(scheme) != 2 {
		return nil
	}
	checkgroupID := types.CheckgroupID(scheme[1])
	return &checkgroupID
}
func (app *application) updateCheckgroupHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        *string           `json:"name"`
		Scheme      *CheckgroupScheme `json:"scheme"`
		ShowOnMap   *bool             `json:"showOnMap"`
		Mandatory   *bool             `json:"mandatory"`
		Checkpoints []struct {
			CheckpointID types.CheckpointID `json:"id"`
			Deleted      bool               `json:"deleted"`
			Created      bool               `json:"created"`
			Name         *string            `json:"name"`
			OpenFrom     *time.Time         `json:"openFrom"`
			OpenUntil    *time.Time         `json:"openUntil"`
			OpenDuration *int               `json:"openDuration"`
			Address      *string            `json:"address"`
			Description  *string            `json:"description"`
			Latitude     *float64           `json:"latitude"`
			Longitude    *float64           `json:"longitude"`
		} `json:"checkpoints"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	checkgroupID := types.CheckgroupID(app.ReadNamedParam(r, "id"))
	checkgroup := checkgroup.UpdateCommand{
		Name:      input.Name,
		ShowOnMap: input.ShowOnMap,
		Mandatory: input.Mandatory,
	}
	if input.Scheme != nil {
		scheme := input.Scheme.Scheme()
		checkgroup.Scheme = &scheme
		checkgroup.RelativeCheckgroupID = input.Scheme.RelativeCheckgroupID()
	}
	if err := app.commands.Checkgroup.Update(r.Context(), checkgroupID, checkgroup); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	log.Printf("antal checkpoints %d", len(input.Checkpoints))
	for _, cp := range input.Checkpoints {
		spew.Dump(cp)
		if cp.Deleted {
			app.commands.Checkpoint.Delete(r.Context(), cp.CheckpointID)
			log.Println("deleted")
			continue
		}
		if cp.Created {
			var err error
			cp.CheckpointID, err = app.commands.Checkpoint.Create(r.Context(), app.YearSlug(r), checkgroupID)
			if err != nil {
				log.Printf("error creating checkpoint %v", err)
			}
		}

		var duration time.Duration
		if cp.OpenDuration != nil {
			duration = time.Duration(*cp.OpenDuration) * time.Minute
		}
		cmd := checkpoint.UpdateCommand{
			Name:         cp.Name,
			OpenFrom:     cp.OpenFrom,
			OpenUntil:    cp.OpenUntil,
			OpenDuration: &duration,
			Address:      cp.Address,
			Description:  cp.Description,
		}
		if cp.Latitude != nil && cp.Longitude != nil {
			cmd.Position = &types.Position{
				Latitude:  types.Latitude(*cp.Latitude),
				Longitude: types.Longitude(*cp.Longitude),
			}
		}
		err := app.commands.Checkpoint.Update(r.Context(), cp.CheckpointID, cmd)
		if err != nil {
			log.Printf("error udating checkpoint %#v", err)
		}
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"updated": "ok", "checkgroup": checkgroup}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) sortCheckgroupsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CheckgroupIDs []types.CheckgroupID `json:"checkgroupIds"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.commands.Checkgroup.Sort(r.Context(), input.CheckgroupIDs); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"sorted": "ok"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
