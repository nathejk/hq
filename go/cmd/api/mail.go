package main

import (
	"context"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/klan"
	"nathejk.dk/nathejk/table/personnel"
)

type mailRecipient struct {
	ID    types.ID           `json:"id"`
	Name  string             `json:"name"`
	Email types.EmailAddress `json:"email"`
}

func (app *application) mailRecipientsHandler(w http.ResponseWriter, r *http.Request) {
	filter := personnel.Filter{YearSlug: "2025", UserTypes: []string{"gøgler"}}
	badutter, err := app.models.Personnel.GetAll(context.Background(), filter)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
	badutRecipients := []mailRecipient{}
	for _, badut := range badutter {
		badutRecipients = append(badutRecipients, mailRecipient{ID: types.ID(badut.ID), Name: badut.Name, Email: badut.Email})
	}

	klans, err := app.models.Klan.GetAll(context.Background(), klan.Filter{YearSlug: "2025"})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
	klanRecipients := []mailRecipient{}
	for _, t := range klans {
		klanRecipients = append(klanRecipients, mailRecipient{Name: t.Name}) //ID: types.ID(t.TeamID), Name: t.Name, Email: t.Email})
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"personnel": badutRecipients}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

/*
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
		}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}*/
