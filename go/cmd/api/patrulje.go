package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/tables/payment"
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

	// MemberStatuses is the member lifecycle with Danish labels. Only the patrol page
	// populates it — klaner are not handled through the nødtelefon, so their members
	// have no lifecycle — hence omitempty rather than an empty array on every payload.
	MemberStatuses []SlugLabel `json:"memberStatuses,omitempty"`
}

// MemberStatuses is the member lifecycle with Danish labels, served to the SPA rather
// than hardcoded in a view (PRD 006 §6).
//
// Serving them is not ceremony. The strings are persisted values — changing one is a
// data migration, not a rename — and two screens show them (the case card and the
// patrol page's correction row). A label map in each view is how those two drift apart
// until one of them says "waiting" to an operator at 3am.
//
// Order is lifecycle order, so a picker reads as the journey rather than
// alphabetically. `finished` is deliberately absent: no correction may confer it, since
// only walking the route unaided earns it (types.MemberStatus.CanFinish), so offering it
// in a picker would invite the one edit the domain refuses.
func MemberStatuses() []SlugLabel {
	return []SlugLabel{
		{Slug: string(types.MemberStatusRegistered), Label: "Tilmeldt"},
		{Slug: string(types.MemberStatusSeated), Label: "Har plads"},
		{Slug: string(types.MemberStatusRacing), Label: "I løbet"},
		{Slug: string(types.MemberStatusWaiting), Label: "Venter på at blive hentet"},
		{Slug: string(types.MemberStatusTransit), Label: "I bil"},
		{Slug: string(types.MemberStatusSheltered), Label: "På HQ"},
		{Slug: string(types.MemberStatusReunited), Label: "Genforenet med patruljen"},
		{Slug: string(types.MemberStatusReleased), Label: "Hentet af forældre"},
	}
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
		// Return, do not fall through: without this the failing request answered with an
		// error envelope *and* a second `{"teams": null}` body, which is not JSON any
		// client can parse — so a database problem surfaced as a mystery in the SPA.
		app.ServerErrorResponse(w, r, err)
		return
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
	payments, err := app.models.Payment.GetAll(r.Context(), payment.Filter{TeamIDs: []types.TeamID{teamId}})
	if err != nil {
		log.Printf("GetSpejdere %q", err)
	}
	orders, err := app.models.Order.ListByOwner(r.Context(), app.YearSlug(r), types.TeamTypePatrulje, string(teamId))
	if err != nil {
		log.Printf("Order.ListByOwner %q", err)
	}

	config := TeamConfig{
		MinMemberCount: 3,
		MaxMemberCount: 7,
		MemberPrice:    250,
		TShirtPrice:    175,
		Korps:          Korps(),
		TShirtSizes:    TShirtSizes(),
		// The member lifecycle with Danish labels (PRD 006 §6), for the members table's
		// status column and the correction row beneath it. On the config rather than
		// loose in the envelope because that is where this page already looks for the
		// server's vocabulary — korps, t-shirt sizes, and now statuses.
		MemberStatuses: MemberStatuses(),
	}
	contact, _ := app.models.Teams.GetContact(teamId)

	// The patrol's SOS cases, for the "Kontakt med nødtelefon" card (PRD 001).
	//
	// Folded into this payload rather than given its own endpoint: this handler
	// already assembles members, payments and orders, so the card costs no extra
	// request and the page needs only `'sos'` added to its live dependsOn. A failure
	// here is logged and the page still renders — the case list is context, and losing
	// it should not take down a patrol's own page.
	sosCases, err := app.models.Sos.GetByTeam(r.Context(), app.YearSlug(r), teamId)
	if err != nil {
		log.Printf("Sos.GetByTeam %q", err)
		sosCases = nil
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "team": team, "contact": contact, "members": members, "payments": payments, "orders": orders, "sosCases": sosCases}, nil)
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
