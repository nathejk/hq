package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/scan"
)

type SlugLabel struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}
type TeamConfig struct {
	MinMemberCount int         `json:"minMemberCount"`
	MaxMemberCount int         `json:"maxMemberCount"`
	MemberPrice    int         `json:"memberPrice"`
	TShirtPrice    int         `json:"tshirtPrice"`
	Korps          []SlugLabel `json:"korps"`
	TShirtSizes    []SlugLabel `json:"tshirtSizes"`
}

func Korps() []SlugLabel {
	return []SlugLabel{
		{Slug: "dds", Label: "Det Danske Spejderkorps"},
		{Slug: "kfum", Label: "KFUM-Spejderne"},
		{Slug: "kfuk", Label: "De grønne pigespejdere"},
		{Slug: "dbs", Label: "Danske Baptisters Spejderkorps"},
		{Slug: "dgs", Label: "De Gule Spejdere"},
		{Slug: "dss", Label: "Dansk Spejderkorps Sydslesvig"},
		{Slug: "fdf", Label: "FDF / FPF"},
		{Slug: "andet", Label: "Andet"},
	}
}
func TShirtSizes() []SlugLabel {
	return []SlugLabel{
		{Slug: "", Label: "Ingen"},
		{Slug: "xs", Label: "X-Small"},
		{Slug: "s", Label: "Small"},
		{Slug: "m", Label: "Medium"},
		{Slug: "l", Label: "Large"},
		{Slug: "xl", Label: "X-Large"},
		{Slug: "xxl", Label: "XX-Large"},
	}
}

func (app *application) showPatruljeListHandler(w http.ResponseWriter, r *http.Request) {
	filter := patrulje.Filter{YearSlug: app.YearSlug(r)}
	teams, err := app.models.Patrulje.GetAll(r.Context(), filter)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"teams": teams}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) showPatruljeHandler(w http.ResponseWriter, r *http.Request) {
	teamId := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamId == "" {
		app.NotFoundResponse(w, r)
		return
	}
	team, err := app.models.Teams.GetPatrulje(teamId)
	if err != nil {
		log.Printf("GetPatrulje %q", err)
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	members, _, err := app.models.Members.GetSpejdere(data.Filters{TeamID: teamId})
	if err != nil {
		log.Printf("GetSpejdere %q", err)
	}
	payments, err := app.models.Payment.GetAll(r.Context(), teamId)
	if err != nil {
		log.Printf("GetSpejdere %q", err)
	}

	config := TeamConfig{
		MinMemberCount: 3,
		MaxMemberCount: 7,
		MemberPrice:    250,
		TShirtPrice:    175,
		Korps:          Korps(),
		TShirtSizes:    TShirtSizes(),
	}
	contact, _ := app.models.Teams.GetContact(teamId)

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "team": team, "contact": contact, "members": members, "payments": payments}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) updatePatruljeHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Team    commands.Patrulje  `json:"team"`
		Contact commands.Contact   `json:"contact"`
		Members []commands.Spejder `json:"members"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	team, err := app.models.Teams.GetPatrulje(teamID)
	if err != nil {
		log.Printf("Signup.GetByID  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	err = app.commands.Team.UpdatePatrulje(teamID, input.Team, input.Contact, input.Members)
	if err != nil {
		log.Printf("UpdatePatrulje  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
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
func (app *application) startPatruljeHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		TeamID  types.TeamID `json:"teamId"`
		Members []struct {
			MemberID    types.MemberID    `json:"memberId"`
			Name        string            `json:"name"`
			Phone       types.PhoneNumber `json:"phone"`
			PhoneParent types.PhoneNumber `json:"phoneParent"`
			Starter     bool              `json:"starter"`
		} `json:"members"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	var members []commands.StartPatruljeMember
	for _, m := range input.Members {
		members = append(members, commands.StartPatruljeMember{MemberID: m.MemberID, Phone: m.Phone, PhoneParent: m.PhoneParent, Starter: m.Starter})
	}
	err := app.commands.Team.StartPatrulje(teamID, members)
	if err != nil {
		log.Printf("StartPatrulje  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	team, err := app.models.Teams.GetPatrulje(teamID)
	if err != nil {
		log.Printf("Teams.GetPatrulje  %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
func (app *application) scansPatruljeHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))

	team, err := app.models.Teams.GetPatrulje(teamID)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	scans, _, err := app.models.Scan.GetAll(r.Context(), scan.Filter{TeamID: teamID})
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team, "scans": scans}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
