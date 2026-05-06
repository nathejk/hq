package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table/klan"
	"nathejk.dk/nathejk/table/lok"
	"nathejk.dk/nathejk/table/personnel"
	"nathejk.dk/nathejk/table/senior"
)

func (app *application) showLoksHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teams, err := app.models.Klan.GetAll(ctx, klan.Filter{YearSlug: app.YearSlug(r)})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	users, err := app.models.Personnel.GetAll(ctx, personnel.Filter{Department: "Banditter"})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	loks, _, err := app.models.Lok.GetAll(ctx, lok.Filter{YearSlug: app.YearSlug(r)})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"loks": loks, "teams": teams, "users": users}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) updateLokHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Users []struct {
			UserID    types.UserID `json:"id"`
			ArmNumber string       `json:"armNumber"`
		} `json:"users"`
		Members []struct {
			MemberID  types.MemberID `json:"id"`
			ArmNumber string         `json:"armNumber"`
		} `json:"members"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	var count int
	for _, user := range input.Users {
		if err := app.commands.Lok.UpdateUser(user.UserID, user.ArmNumber); err == nil {
			count++
		}
	}
	for _, member := range input.Members {
		if err := app.commands.Lok.UpdateMember(member.MemberID, member.ArmNumber); err == nil {
			count++
		}
	}
	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"armNumberCount": count}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) updateLoksHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Loks []struct {
			LokID   types.LokID    `json:"lokId"`
			Name    string         `json:"name"`
			UserIDs []types.UserID `json:"userIds"`
			TeamIDs []types.TeamID `json:"teamIds"`
		} `json:"loks"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	for i, lok := range input.Loks {
		err := app.commands.Lok.UpdateLok(lok.LokID, lok.Name, i, lok.UserIDs, lok.TeamIDs)
		if err != nil {
			app.BadRequestResponse(w, r, err)
			return
		}
	}
	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": "team"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) deleteLokHandler(w http.ResponseWriter, r *http.Request) {
	lokID := types.LokID(app.ReadNamedParam(r, "id"))
	if err := app.commands.Lok.DeleteLok(lokID); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": "ok"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) showKlanListHandler(w http.ResponseWriter, r *http.Request) {
	filter := klan.Filter{}
	teams, err := app.models.Klan.GetAll(context.Background(), filter)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"teams": teams}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) showLokHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	lokID := types.LokID(app.ReadNamedParam(r, "id"))
	lok, err := app.models.Lok.GetByID(ctx, lokID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	teams, _ := app.models.Klan.GetAll(ctx, klan.Filter{TeamIDs: lok.TeamIDs})
	users, _ := app.models.Personnel.GetAll(ctx, personnel.Filter{UserIDs: lok.UserIDs})
	members, _ := app.models.Senior.GetAll(ctx, senior.Filter{TeamIDs: lok.TeamIDs})

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"lok": lok, "users": users, "teams": teams, "members": members}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) showKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamId := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamId == "" {
		app.NotFoundResponse(w, r)
		return
	}
	team, err := app.models.Teams.GetKlan(teamId)
	if err != nil {
		log.Printf("GetKlan %q", err)
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	members, _, err := app.models.Members.GetSeniore(data.Filters{TeamID: teamId})
	if err != nil {
		log.Printf("GetSenior %q", err)
	}

	config := TeamConfig{
		MinMemberCount: 1,
		MaxMemberCount: 4,
		MemberPrice:    250,
		TShirtPrice:    175,
		Korps:          Korps(),
		TShirtSizes:    TShirtSizes(),
	}
	//contact, _ := app.models.Teams.GetContact(teamId)

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "team": team, "members": members, "payments": []any{}}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) patchKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Lok *string `json:"lok"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	if _, err := app.models.Teams.GetKlan(teamID); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if input.Lok != nil {
		if err := app.commands.Team.AssignToLok(teamID, *input.Lok); err != nil {
			app.BadRequestResponse(w, r, err)
			return
		}
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) updateKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Team    commands.Klan     `json:"team"`
		Members []commands.Senior `json:"members"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	_, err := app.models.Teams.GetKlan(teamID)
	if err != nil {
		log.Printf("Signup.GetByID  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	err = app.commands.Team.UpdateKlan(teamID, input.Team, input.Members)
	if err != nil {
		log.Printf("UpdateKlan  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	team, _ := app.models.Teams.GetKlan(teamID)
	/*
		page := fmt.Sprintf("/patrulje/%s", input.TeamID)
		err = app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"team": map[string]string{"teamPage": page}}, nil)
		if err != nil {
			app.ServerErrorResponse(w, r, err)
		}*/
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
