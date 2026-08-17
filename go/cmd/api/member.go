package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/table/sos"
	"nathejk.dk/nathejk/table/spejderstatus"
)

// The member lifecycle write surface (PRD 006).
//
// # Why these handlers do two things
//
// Every member operation publishes one event per affected member on the member's own
// subject, and then one summarising event on the case's subject. The two halves live
// in two packages that may not import each other — spejderstatus and sos are each
// written to be lifted to shared-go independently — so this file is where they meet.
// That is not a workaround: composing domains is what a backend-for-frontend handler
// is for, and it keeps both packages movable.
//
// The order matters and is not incidental: the member events go first, so anything
// reading the summary is guaranteed the changes it describes are already in the log.
//
// # Why a sosId is always required
//
// Nothing changes a member's status or team without a case explaining why (PRD 006
// §11, decided 2026-08-17). It is the case that makes the lifecycle auditable from
// one place, and there is deliberately no case-less path — where an operator has
// none, the correction interface mints one (task 084).

// memberRequest is the body every member command takes.
//
// SosID is required on all of them. Status and TeamID are used by the override and
// the move respectively.
type memberRequest struct {
	SosID  types.SosID        `json:"sosId"`
	Status types.MemberStatus `json:"status"`
	TeamID types.TeamID       `json:"teamId"`
}

// requestWaitingHandler records that a member wants to leave the race.
func (app *application) requestWaitingHandler(w http.ResponseWriter, r *http.Request) {
	app.memberStatusOperation(w, r, func(ctx memberContext) (*spejderstatus.Change, error) {
		return app.commands.Member.RequestWithdrawal(ctx.r.Context(), ctx.actor, ctx.year, ctx.memberID)
	})
}

// resumeRacingHandler puts a member who changed their mind back into the race.
func (app *application) resumeRacingHandler(w http.ResponseWriter, r *http.Request) {
	app.memberStatusOperation(w, r, func(ctx memberContext) (*spejderstatus.Change, error) {
		return app.commands.Member.CancelWithdrawal(ctx.r.Context(), ctx.actor, ctx.year, ctx.memberID)
	})
}

// overrideMemberStatusHandler corrects a member's status by hand.
//
// Lenient about ordering by decision (2026-08-17): this is the out-of-sync repair
// tool, so it accepts any valid status from any other. Rejecting racing → sheltered
// because no pickup was logged would refuse exactly the correction it exists to make.
func (app *application) overrideMemberStatusHandler(w http.ResponseWriter, r *http.Request) {
	app.memberStatusOperation(w, r, func(ctx memberContext) (*spejderstatus.Change, error) {
		return app.commands.Member.OverrideStatus(ctx.r.Context(), ctx.actor, ctx.year, ctx.memberID, ctx.input.Status)
	})
}

// moveMemberTeamHandler moves a member to another patrol.
//
// The destination is validated here rather than in the member package, which may not
// import patrulje. Any patrol in the same year that started and still has racing
// members is a valid target — no proximity or size rule, because crew in the field
// agree the destination and the operator is recording it, not choosing it.
func (app *application) moveMemberTeamHandler(w http.ResponseWriter, r *http.Request) {
	ctx, ok := app.memberContext(w, r)
	if !ok {
		return
	}
	if ctx.input.TeamID == "" {
		app.FailedValidationResponse(w, r, map[string]string{"teamId": "must be provided"})
		return
	}

	target, err := app.models.Teams.GetPatrulje(ctx.input.TeamID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.FailedValidationResponse(w, r, map[string]string{"teamId": "ukendt patrulje"})
			return
		}
		app.ServerErrorResponse(w, r, err)
		return
	}
	// "Still racing" is what makes a patrol able to receive somebody. A team that
	// never started has nobody on the route to join, and one that has been emptied
	// is discontinued — moving a member into either would be recording a fiction.
	if target.SignupStatus != types.SignupStatusStarted || target.ActiveMemberCount == 0 {
		app.FailedValidationResponse(w, r, map[string]string{
			"teamId": "patruljen er ikke i løbet",
		})
		return
	}

	move, err := app.commands.Member.MoveTeam(ctx.r.Context(), ctx.actor, ctx.year, ctx.memberID, ctx.input.TeamID)
	if err != nil {
		app.memberCommandError(w, r, err)
		return
	}

	// The summary needs names for both the member and the destination, so the line
	// reads without a join now or later.
	name := app.memberName(ctx.memberID, move.FromTeamID)
	origin, _ := app.models.Teams.GetPatrulje(move.FromTeamID)
	originName := ""
	if origin != nil {
		originName = origin.Name
	}
	summary := sos.MembersMoved{
		SosID:        ctx.input.SosID,
		FromTeamID:   move.FromTeamID,
		FromTeamName: originName,
		Members: []sos.MemberMove{{
			MemberID:   move.MemberID,
			Name:       name,
			ToTeamID:   move.ToTeamID,
			ToTeamName: target.Name,
		}},
		FromTeamStrength: move.FromTeamStrength,
	}
	if err := app.commands.Sos.RecordMembersMoved(ctx.r.Context(), app.actor(r), ctx.year, summary); err != nil {
		// The move is already published and cannot be recalled, so this is logged
		// rather than returned: losing the timeline line is bad, telling the operator
		// the move failed when it did not would be worse.
		log.Printf("sos summary for member operation failed: %v", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"move": move}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// collectTeamHandler takes a whole patrol out of the race in one action.
//
// One request, not one per member: three separate calls from the browser could
// half-succeed and leave a team split across two states with nobody noticing.
//
// Renders as a single timeline entry — "hele patruljen hentes" — because the N member
// events are summarised by one case event, which is the general rule from task 071
// applied to the case that motivated it.
func (app *application) collectTeamHandler(w http.ResponseWriter, r *http.Request) {
	sosID := types.SosID(app.ReadNamedParam(r, "id"))
	teamID := types.TeamID(app.ReadNamedParam(r, "teamId"))
	if sosID == "" || teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	year := app.YearSlug(r)

	// The team must actually be on this case.
	//
	// Found missing while verifying task 076: the endpoint happily emptied a patrol that
	// had never been associated. Nothing in the URL enforces the relationship it asserts,
	// so a stale or copy-pasted teamId would take a patrol out of the race from a case
	// that has nothing to do with it — and the summary would land on a timeline whose team
	// card does not even list them, making it invisible where it matters.
	//
	// This is the one member operation that needs the check: the others act on a member
	// the operator is looking at, whereas this one acts on a *set* derived from a team id
	// alone, which is the difference between a mistake affecting one row and one emptying
	// a patrol.
	case_, err := app.models.Sos.GetByID(r.Context(), sosID)
	if err != nil {
		app.handleSosError(w, r, err)
		return
	}
	associated := false
	for _, t := range case_.Teams {
		if t.TeamID == teamID {
			associated = true
			break
		}
	}
	if !associated {
		app.FailedValidationResponse(w, r, map[string]string{
			"teamId": "patruljen er ikke tilknyttet sagen",
		})
		return
	}

	changes, err := app.commands.Member.CollectTeam(r.Context(), app.memberActor(r), year, teamID)
	if err != nil {
		app.memberCommandError(w, r, err)
		return
	}
	// Nobody left to collect. Answered as success with an empty list: the operator's
	// intent holds, and a double click must not produce a second timeline entry.
	if len(changes) == 0 {
		if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"collected": []any{}}, nil); err != nil {
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	team, _ := app.models.Teams.GetPatrulje(teamID)
	teamName := ""
	if team != nil {
		teamName = team.Name
	}
	// One lookup for the whole team rather than one per member: memberName would
	// otherwise re-read the same roster once per collected member.
	names := map[types.MemberID]string{}
	if members, _, err := app.models.Members.GetSpejdere(data.Filters{TeamID: teamID}); err == nil {
		for _, m := range members {
			names[m.MemberID] = m.Name
		}
	}

	summary := sos.TeamCollected{
		SosID:        sosID,
		TeamID:       teamID,
		TeamName:     teamName,
		TeamStrength: 0,
	}
	for _, ch := range changes {
		summary.Members = append(summary.Members, sos.MemberChange{
			MemberID: ch.MemberID,
			Name:     names[ch.MemberID],
			From:     ch.From,
			To:       ch.To,
		})
	}
	if err := app.commands.Sos.RecordTeamCollected(r.Context(), app.actor(r), year, summary); err != nil {
		log.Printf("sos summary for member operation failed: %v", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"collected": changes}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// memberContext is everything a member handler needs after validation.
type memberContext struct {
	r        *http.Request
	actor    spejderstatus.Actor
	year     types.YearSlug
	memberID types.MemberID
	input    memberRequest
}

// memberContext reads and validates what every member command requires.
func (app *application) memberContext(w http.ResponseWriter, r *http.Request) (memberContext, bool) {
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if memberID == "" {
		app.NotFoundResponse(w, r)
		return memberContext{}, false
	}
	var input memberRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return memberContext{}, false
	}
	if input.SosID == "" {
		app.FailedValidationResponse(w, r, map[string]string{
			"sosId": "must be provided: a member does not change status without a case explaining why",
		})
		return memberContext{}, false
	}
	return memberContext{
		r:        r,
		actor:    app.memberActor(r),
		year:     app.YearSlug(r),
		memberID: memberID,
		input:    input,
	}, true
}

// memberStatusOperation runs a status-changing command and records it on the case.
//
// Shared by the three status handlers because the shape is identical and the
// interesting part — which command — is the one line they differ by. It also keeps
// the publish-then-summarise order in one place rather than three.
func (app *application) memberStatusOperation(w http.ResponseWriter, r *http.Request, run func(memberContext) (*spejderstatus.Change, error)) {
	ctx, ok := app.memberContext(w, r)
	if !ok {
		return
	}
	change, err := run(ctx)
	if err != nil {
		app.memberCommandError(w, r, err)
		return
	}
	// A no-op: the member was already in that state, so nothing was published and
	// nothing goes on the timeline. Answered as success, because the caller's
	// intent holds.
	if change == nil {
		if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"change": nil}, nil); err != nil {
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	team, _ := app.models.Teams.GetPatrulje(change.TeamID)
	teamName := ""
	if team != nil {
		teamName = team.Name
	}
	summary := sos.MemberStatusChanged{
		SosID:    ctx.input.SosID,
		TeamID:   change.TeamID,
		TeamName: teamName,
		Members: []sos.MemberChange{{
			MemberID: change.MemberID,
			Name:     app.memberName(change.MemberID, change.TeamID),
			From:     change.From,
			To:       change.To,
		}},
		TeamStrength: change.TeamStrength,
	}
	if err := app.commands.Sos.RecordMemberStatusChanged(ctx.r.Context(), app.actor(r), ctx.year, summary); err != nil {
		log.Printf("sos summary for member operation failed: %v", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"change": change}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// memberName resolves a member's name for a timeline summary.
//
// Best-effort: an empty name yields a line that reads a little worse, which is much
// better than refusing to record that somebody left the race because a lookup failed.
func (app *application) memberName(id types.MemberID, teamID types.TeamID) string {
	members, _, err := app.models.Members.GetSpejdere(data.Filters{TeamID: teamID})
	if err != nil {
		return ""
	}
	for _, m := range members {
		if m.MemberID == id {
			return m.Name
		}
	}
	return ""
}

// memberCommandError maps the domain's refusals onto responses an operator can act
// on.
//
// ErrAlreadyCollected is the one that matters: it means a car has the scout, and the
// answer is to tell the caller that rather than to retry.
func (app *application) memberCommandError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, spejderstatus.ErrRecordNotFound):
		app.NotFoundResponse(w, r)
	case errors.Is(err, spejderstatus.ErrAlreadyCollected):
		app.FailedValidationResponse(w, r, map[string]string{"status": "allerede hentet"})
	case errors.Is(err, spejderstatus.ErrNotWaiting):
		app.FailedValidationResponse(w, r, map[string]string{"status": "deltageren venter ikke"})
	case errors.Is(err, spejderstatus.ErrNotSelfCarrying):
		app.FailedValidationResponse(w, r, map[string]string{"status": "deltageren er allerede ude af løbet"})
	case errors.Is(err, spejderstatus.ErrCannotFinish):
		app.FailedValidationResponse(w, r, map[string]string{"status": "gennemført kan ikke sættes manuelt"})
	case errors.Is(err, spejderstatus.ErrInvalidStatus):
		app.FailedValidationResponse(w, r, map[string]string{"status": "ukendt status"})
	case errors.Is(err, spejderstatus.ErrSameTeam):
		app.FailedValidationResponse(w, r, map[string]string{"teamId": "deltageren er allerede i patruljen"})
	default:
		app.ServerErrorResponse(w, r, err)
	}
}

// showMemberCareHandler serves the count of members Nathejk is currently
// responsible for: the number that has to reach zero before the organisers can go
// home.
//
// Year-scoped from X-YearSlug like everything else. It is served as an event-wide
// figure rather than per case, because that is what it is — a member in our care is
// our problem whether or not anybody has opened a case about them.
//
// The response carries the oldest `waiting` timestamp rather than a "somebody has
// waited too long" boolean, so the threshold stays in one place and can change
// without a new deploy of this endpoint. It is still unsettled (PRD 006 §11, task
// 082).
func (app *application) showMemberCareHandler(w http.ResponseWriter, r *http.Request) {
	care, err := app.models.SpejderStatus.InOurCare(r.Context(), app.YearSlug(r))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"care": care}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
