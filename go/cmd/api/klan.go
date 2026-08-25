package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/tables/klan"
	"github.com/nathejk/shared-go/tables/payment"
	"github.com/nathejk/shared-go/tables/senior"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/commands"
	"nathejk.dk/nathejk/table/lok"
	"nathejk.dk/nathejk/table/personnel"
)

// klanStatusDeleted is the status a withdrawn klan carries.
//
// Not in types.SignupStatus, and lowercase where the rest are upper: it is what
// the shared-go klan entity's Delete publishes. Named here so the two list
// handlers filter on the same literal rather than repeating a bare string.
const klanStatusDeleted = types.SignupStatus("deleted")

// withoutDeleted drops withdrawn klans from a list.
//
// Needed because the klan projection soft-deletes: Delete sets signupStatus to
// "deleted", and the entity's own GetAll only excludes the empty status, so a
// withdrawn klan would otherwise keep appearing on the bandit page — draggable
// into a LOK, and counted in its total.
func withoutDeleted(klans []klan.Klan) []klan.Klan {
	out := make([]klan.Klan, 0, len(klans))
	for _, k := range klans {
		if k.Status == klanStatusDeleted {
			continue
		}
		out = append(out, k)
	}
	return out
}

func (app *application) showLoksHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teams, err := app.models.Klan.GetAll(ctx, klan.Filter{YearSlug: string(app.YearSlug(r))})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	teams = withoutDeleted(teams)
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
	teams = withoutDeleted(teams)

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

// orEmpty replaces a nil slice with an empty one.
//
// A nil slice marshals to `null`, not `[]`, and a client that does the obvious
// thing with a collection — `payload.orders.length` — throws on it. That is not a
// hypothetical: a klan with no seniors has no order lines and therefore no order,
// so ListByOwner returned nil and the klan dialog died mid-render, taking its own
// close button with it and trapping the operator in a modal.
//
// Guaranteed here, at the edge, rather than asked of every reader: a collection
// endpoint answering `null` for "none" is the API being wrong, not the client.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// klanStatusOption is one status an operator may set from the klan dialog.
type klanStatusOption struct {
	Slug  types.SignupStatus `json:"slug"`
	Label string             `json:"label"`
}

// The statuses the override offers, in lifecycle order.
//
// Served with the klan rather than hardcoded in the SPA for the same reason as
// the korps list: the set and its Danish wording are domain facts, and a copy in
// the frontend is a copy that drifts. Every status is offered deliberately — the
// whole point of an override is to reach a state the automatic flow will not
// produce, so a filtered list would defeat it.
//
// "deleted" is absent on purpose: it is not a status an operator sets, it is what
// the delete endpoint does, and offering it here would give two ways to delete a
// klan of which only one asks for confirmation.
var klanStatusOptions = []klanStatusOption{
	{types.SignupStatusNew, "Ny"},
	{types.SignupStatusOnHold, "Venteliste"},
	{types.SignupStatusPay, "Afventer betaling"},
	{types.SignupStatusSemipaid, "Delvist betalt"},
	{types.SignupStatusPaid, "Betalt"},
	{types.SignupStatusStarted, "Startet"},
	{types.SignupStatusOut, "Udgået"},
}

func klanStatusSettable(status types.SignupStatus) bool {
	for _, o := range klanStatusOptions {
		if o.Slug == status {
			return true
		}
	}
	return false
}

// showKlanHandler serves everything the klan dialog shows: the team, its members,
// how it signed up, and the money.
//
// The klan projection is the source for the team rather than models.Teams.GetKlan,
// which is what this handler used to use: that query JOINs patruljestatus, a table
// the klan entity does not own, so a klan with no row there answered 404 — and the
// dialog is opened precisely to investigate klans in odd states. GetAll with a
// single id is used because only it computes paidAmount.
func (app *application) showKlanHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}

	teams, err := app.models.Klan.GetAll(ctx, klan.Filter{TeamIDs: []types.TeamID{teamID}})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	if len(teams) == 0 {
		app.NotFoundResponse(w, r)
		return
	}
	team := teams[0]

	// GetAll computes memberCount and paidAmount but does not select the year, so
	// the row it returns carries an empty one. Taken from GetByID rather than from
	// the request's year: the orders below are looked up by it, and a klan opened
	// while another year is selected would otherwise silently show no money at all.
	// The computed counts are kept in preference to GetByID's stored column so the
	// dialog cannot disagree with the list it was opened from.
	stored, err := app.models.Klan.GetByID(ctx, teamID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	team.Year = stored.Year

	members, err := app.models.Senior.GetAll(ctx, senior.Filter{TeamIDs: []types.TeamID{teamID}})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	// How the klan reached us: the contact email and phone live on the signup, not
	// on any member, so without this the dialog could show a klan nobody could ring.
	// Absent for a klan created by other means, which is not an error.
	var signup *data.Signup
	if s, err := app.models.Signup.GetByID(teamID); err == nil {
		signup = s
	}

	// The money, in full, because it is the evidence for or against an override:
	// an operator about to mark a klan Betalt should be able to see what the system
	// thinks it has received, and from where.
	orders, err := app.models.Order.ListByOwner(ctx, team.Year, types.TeamTypeKlan, string(teamID))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	payments, err := app.models.Payment.GetAll(ctx, payment.Filter{TeamIDs: []types.TeamID{teamID}})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	config := TeamConfig{
		MinMemberCount: 1,
		MaxMemberCount: 4,
		MemberPrice:    250,
		TShirtPrice:    175,
		Korps:          Korps(),
		TShirtSizes:    TShirtSizes(),
	}

	envelope := jsonapi.Envelope{
		"config": config,
		"team":   team,
		// Every collection normalised, not just the one that broke: the same nil is
		// possible for each of them, and a klan with nothing yet is an ordinary state
		// on this screen rather than an edge case.
		"members":       orEmpty(members),
		"signup":        signup,
		"orders":        orEmpty(orders),
		"payments":      orEmpty(payments),
		"statusOptions": klanStatusOptions,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// deleteKlanHandler withdraws a klan.
//
// A soft delete in the read model — the shared-go entity records it as a status
// change to "deleted" — so the event trail for a klan that paid and then withdrew
// survives, which matters when the money has to be found again later.
func (app *application) deleteKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if _, err := app.models.Klan.GetByID(r.Context(), teamID); err != nil {
		app.NotFoundResponse(w, r)
		return
	}
	if err := app.commands.Klan.Delete(r.Context(), teamID); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": "ok"}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// patchKlanHandler applies partial changes: the lok assignment, and the status
// override.
func (app *application) patchKlanHandler(w http.ResponseWriter, r *http.Request) {
	teamID := types.TeamID(app.ReadNamedParam(r, "id"))
	var input struct {
		Lok    *string             `json:"lok"`
		Status *types.SignupStatus `json:"status"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		log.Printf("ReadJSON %q", err)
		app.BadRequestResponse(w, r, err)
		return
	}

	// The klan projection, not models.Teams.GetKlan: see showKlanHandler for why
	// that query cannot find every klan this endpoint must be able to act on.
	if _, err := app.models.Klan.GetByID(r.Context(), teamID); err != nil {
		app.NotFoundResponse(w, r)
		return
	}
	if input.Lok != nil {
		if err := app.commands.Team.AssignToLok(teamID, *input.Lok); err != nil {
			app.BadRequestResponse(w, r, err)
			return
		}
	}
	if input.Status != nil {
		// Checked against the offered set rather than passed through: the projection
		// writes the status verbatim, so an unrecognised value would leave a klan in
		// a state no screen can render and no filter can find.
		if !klanStatusSettable(*input.Status) {
			app.BadRequestResponse(w, r, errFromString(fmt.Sprintf("invalid status %q", *input.Status)))
			return
		}
		if err := app.commands.Klan.SetStatus(r.Context(), teamID, *input.Status); err != nil {
			app.BadRequestResponse(w, r, err)
			return
		}
	}
	// An acknowledgement, not the row.
	//
	// This used to answer with the klan read back immediately, which is a trap on the
	// write side of a CQRS split: the projection applies asynchronously, so the echoed
	// row still carried the *old* status — and its memberCount came from the stored
	// column, which is 0 for every klan because the real count is a subquery over
	// seniors. A caller trusting either would be wrong twice. Clients refetch (or wait
	// for the live signal, which cannot precede the write).
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"teamId": teamID}, nil); err != nil {
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
