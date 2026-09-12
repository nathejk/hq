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
	nathejktable "nathejk.dk/nathejk/table"
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

	// RemarkSeverities is the vocabulary for the "Info til banditter og postmandskab"
	// note. Served for the same reason MemberStatuses is: the slugs are persisted values,
	// so the picker must offer exactly what the command will accept.
	RemarkSeverities []SlugLabel `json:"remarkSeverities,omitempty"`
}

// RemarkSeverities is the note's severities with Danish labels, in the order offered.
//
// "Ikke aktiv" is last because it is the way *out* of using the feature, not a level of
// it: choosing it keeps the text on file while standing it down, which is why the client
// greys the textarea rather than clearing it.
func RemarkSeverities() []SlugLabel {
	return []SlugLabel{
		{Slug: patrulje.RemarkSeverityInformation, Label: "Information"},
		{Slug: patrulje.RemarkSeverityStop, Label: "Fuld stop"},
		{Slug: patrulje.RemarkSeverityInactive, Label: "Ikke aktiv"},
	}
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

// isParticipating reports whether a patrol counts as a real participant: it has paid
// something and it has been assigned a team number.
//
// A row in `patrulje` is created by the signup event, so an abandoned or half-finished
// signup leaves a named team that never paid and never got a number. Those are noise on
// the operator's list, in the export and in the dashboard count — and worse, the three
// used to disagree about which teams existed (the list showed everything, the front page
// required payment, the export required payment). One predicate so they cannot drift
// apart again.
//
// Applied by the callers rather than inside patrulje.GetAll on purpose: the race-side
// readers (started teams, transfer candidates, shelter) must keep seeing every row.
func isParticipating(p patrulje.Patrulje) bool {
	return p.PaidAmount > 0 && p.TeamNumber != ""
}

// participatingPatruljer filters a GetAll result down to the participants.
func participatingPatruljer(teams []patrulje.Patrulje) []patrulje.Patrulje {
	out := make([]patrulje.Patrulje, 0, len(teams))
	for _, t := range teams {
		if isParticipating(t) {
			out = append(out, t)
		}
	}
	return out
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

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"teams": participatingPatruljer(teams)}, nil)
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
		// The vocabulary for the "Info til banditter og postmandskab" note.
		RemarkSeverities: RemarkSeverities(),
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

	// The map sheets / QR codes handed to this patrulje, current and past (a sheet moves
	// with reassigned scouts when a team is discontinued, so "past" is real). Context like
	// the SOS list above: folded in here rather than given its own request, and a failure
	// is logged without taking the page down.
	maps, err := app.models.MapHandout.ByTeam(r.Context(), app.YearSlug(r), teamId)
	if err != nil {
		log.Printf("MapHandout.ByTeam %q", err)
		maps = nil
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config, "team": team, "contact": contact, "members": members, "payments": payments, "orders": orders, "sosCases": sosCases, "maps": maps}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// setPatruljeRemarkHandler stores the operational note for banditter and postmandskab.
//
// A no-op save answers 200 rather than an error: the client sends the whole note on every
// save, so "nothing changed" is an ordinary outcome of pressing Save twice, not something
// an operator needs telling about. The command still publishes nothing, which is what
// keeps a redundant save from making every open patrol page revalidate.
func (app *application) setPatruljeRemarkHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input struct {
		Remark   string `json:"remark"`
		Severity string `json:"severity"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	err := app.commands.Patrulje.SetRemark(r.Context(), teamID, input.Remark, input.Severity)
	switch {
	case err == nil, errors.Is(err, commands.ErrRemarkUnchanged):
	case errors.Is(err, commands.ErrRemarkSeverityInvalid):
		app.FailedValidationResponse(w, r, map[string]string{"severity": "ukendt værdi"})
		return
	// The patrulje querier's own not-found, from nathejk/table — not shared-go's
	// identically named one, which this would silently never match.
	case errors.Is(err, nathejktable.ErrRecordNotFound):
		app.NotFoundResponse(w, r)
		return
	default:
		app.ServerErrorResponse(w, r, err)
		return
	}

	// No echo of the saved note: the projection applies asynchronously, so a row read back
	// here could still be the old one. The client refetches — the live signal the event
	// produces tells it to.
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"teamId": teamID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// clearPatruljeRemarkHandler removes the note. See ClearRemark for why deleting is not the
// same act as standing the note down to "Ikke aktiv".
func (app *application) clearPatruljeRemarkHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}

	err := app.commands.Patrulje.ClearRemark(r.Context(), teamID)
	switch {
	// Deleting a note that is already gone is a success: the caller wanted no note, and
	// there is none.
	case err == nil, errors.Is(err, commands.ErrRemarkUnchanged):
	case errors.Is(err, nathejktable.ErrRecordNotFound):
		app.NotFoundResponse(w, r)
		return
	default:
		app.ServerErrorResponse(w, r, err)
		return
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"teamId": teamID}, nil); err != nil {
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

	// The raw scans carry a scannerId and a phone; the trail shows who that was
	// instead — a bandit by number, crew by section and post. See enrichScans.
	events := app.enrichScans(r.Context(), app.YearSlug(r), scans)

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"team": team, "scans": events}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
