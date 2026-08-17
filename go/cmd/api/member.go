package main

import (
	"errors"
	"fmt"
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
	app.memberStatusOperation(w, r, false, func(ctx memberContext) (*spejderstatus.Change, error) {
		return app.commands.Member.RequestWithdrawal(ctx.r.Context(), ctx.actor, ctx.year, ctx.memberID)
	})
}

// resumeRacingHandler puts a member who changed their mind back into the race.
func (app *application) resumeRacingHandler(w http.ResponseWriter, r *http.Request) {
	app.memberStatusOperation(w, r, false, func(ctx memberContext) (*spejderstatus.Change, error) {
		return app.commands.Member.CancelWithdrawal(ctx.r.Context(), ctx.actor, ctx.year, ctx.memberID)
	})
}

// overrideMemberStatusHandler corrects a member's status by hand.
//
// Lenient about ordering by decision (2026-08-17): this is the out-of-sync repair
// tool, so it accepts any valid status from any other. Rejecting racing → sheltered
// because no pickup was logged would refuse exactly the correction it exists to make.
//
// **Mints its own case when the caller has none.** Every member command requires a
// `sosId` (PRD 006 §11) and the operator making a correction is on the patrol page, not
// in a case — so rather than carve out a case-less path, the handler opens a case, lets
// the correction land on its timeline, and closes it immediately. See mintCorrectionCase.
func (app *application) overrideMemberStatusHandler(w http.ResponseWriter, r *http.Request) {
	app.memberStatusOperation(w, r, true, func(ctx memberContext) (*spejderstatus.Change, error) {
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
	ctx, ok := app.memberContext(w, r, false)
	if !ok {
		return
	}
	if ctx.input.TeamID == "" {
		app.FailedValidationResponse(w, r, map[string]string{"teamId": "must be provided"})
		return
	}

	target, ok := app.moveTarget(w, r, ctx.input.TeamID)
	if !ok {
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
	// This and the bulk move are the two operations that need the check: the others act on
	// a member the operator is looking at, whereas these act on a *set* derived from a team
	// id alone, which is the difference between a mistake affecting one row and one
	// emptying a patrol.
	if !app.teamOnCase(w, r, sosID, teamID) {
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

// moveMembersHandler moves several members of one patrol to another in a single
// operation.
//
// One request rather than one per member, for the reason the collect endpoint exists: N
// calls from the browser can half-succeed, and an operator told only "something failed"
// cannot see which member is where. It also makes the operation one timeline entry, which
// is what it is — one decision about a below-strength patrol's remnants.
//
// The per-member endpoint remains for the single-row action and for corrections; this is
// the bulk case, not a replacement.
func (app *application) moveMembersHandler(w http.ResponseWriter, r *http.Request) {
	sosID := types.SosID(app.ReadNamedParam(r, "id"))
	teamID := types.TeamID(app.ReadNamedParam(r, "teamId"))
	if sosID == "" || teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input struct {
		MemberIDs []types.MemberID `json:"memberIds"`
		ToTeamID  types.TeamID     `json:"toTeamId"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if len(input.MemberIDs) == 0 {
		app.FailedValidationResponse(w, r, map[string]string{"memberIds": "must not be empty"})
		return
	}
	if input.ToTeamID == "" {
		app.FailedValidationResponse(w, r, map[string]string{"toTeamId": "must be provided"})
		return
	}
	year := app.YearSlug(r)

	// The origin team must be on the case, as for collect: this is a case-scoped action on
	// a set derived from a team id, so a stale id would move somebody else's patrol.
	if !app.teamOnCase(w, r, sosID, teamID) {
		return
	}

	target, ok := app.moveTarget(w, r, input.ToTeamID)
	if !ok {
		return
	}

	moves, err := app.commands.Member.MoveMembers(r.Context(), app.memberActor(r), year, input.MemberIDs, input.ToTeamID)
	if err != nil {
		app.memberCommandError(w, r, err)
		return
	}
	if len(moves) == 0 {
		if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"moves": []any{}}, nil); err != nil {
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	origin, _ := app.models.Teams.GetPatrulje(moves[0].FromTeamID)
	originName := ""
	if origin != nil {
		originName = origin.Name
	}
	// One roster read for the whole operation rather than one per member.
	names := map[types.MemberID]string{}
	if members, _, err := app.models.Members.GetSpejdere(data.Filters{TeamID: moves[0].FromTeamID}); err == nil {
		for _, m := range members {
			names[m.MemberID] = m.Name
		}
	}

	summary := sos.MembersMoved{
		SosID:            sosID,
		FromTeamID:       moves[0].FromTeamID,
		FromTeamName:     originName,
		FromTeamStrength: moves[0].FromTeamStrength,
	}
	for _, m := range moves {
		summary.Members = append(summary.Members, sos.MemberMove{
			MemberID:   m.MemberID,
			Name:       names[m.MemberID],
			ToTeamID:   m.ToTeamID,
			ToTeamName: target.Name,
		})
	}
	if err := app.commands.Sos.RecordMembersMoved(r.Context(), app.actor(r), year, summary); err != nil {
		log.Printf("sos summary for member operation failed: %v", err)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"moves": moves}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// teamOnCase reports whether the patrol is associated with the case, answering the client
// itself when it is not.
//
// Extracted when the bulk move needed the same guard the collect endpoint had grown: both
// act on a *set* derived from a team id alone, which is what makes a stale id dangerous
// rather than merely wrong.
func (app *application) teamOnCase(w http.ResponseWriter, r *http.Request, sosID types.SosID, teamID types.TeamID) bool {
	c, err := app.models.Sos.GetByID(r.Context(), sosID)
	if err != nil {
		app.handleSosError(w, r, err)
		return false
	}
	for _, t := range c.Teams {
		if t.TeamID == teamID {
			return true
		}
	}
	app.FailedValidationResponse(w, r, map[string]string{
		"teamId": "patruljen er ikke tilknyttet sagen",
	})
	return false
}

// moveTarget validates a move destination, answering the client itself when it is invalid.
//
// Required to have **started**, and deliberately *not* required to still have racing
// members. That extra condition was here first and it broke a guarantee PRD 006 §5 makes
// explicitly: moving a member back into a team with nobody left makes it active again — the
// reversibility the legacy `.splited` event had. A team emptied to zero is discontinued,
// and requiring `activeMemberCount > 0` meant **discontinuation could never be undone**,
// because the only action that reverses it was refused for exactly the teams that needed
// it. Found by trying to undo a test (task 077).
//
// The *picker* still offers only racing patrols, which is right for the survivors flow —
// but a UI convenience must not become a domain rule.
//
// `started` remains required: a patrol that never started has nobody on the route to join,
// so moving somebody into it would record a fiction.
func (app *application) moveTarget(w http.ResponseWriter, r *http.Request, teamID types.TeamID) (*data.Patrulje, bool) {
	target, err := app.models.Teams.GetPatrulje(teamID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.FailedValidationResponse(w, r, map[string]string{"teamId": "ukendt patrulje"})
			return nil, false
		}
		app.ServerErrorResponse(w, r, err)
		return nil, false
	}
	if target.SignupStatus != types.SignupStatusStarted {
		app.FailedValidationResponse(w, r, map[string]string{"teamId": "patruljen er ikke startet"})
		return nil, false
	}
	return target, true
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
func (app *application) memberContext(w http.ResponseWriter, r *http.Request, allowMinted bool) (memberContext, bool) {
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
		// The override is the one command allowed to arrive without a case, because the
		// operator making a correction is on the patrol page rather than in one. It does
		// not get a case-less path: the handler mints one (see mintCorrectionCase), which
		// is what keeps "every member change has a case" true without making the patrol
		// page open a case by hand first.
		if !allowMinted {
			app.FailedValidationResponse(w, r, map[string]string{
				"sosId": "must be provided: a member does not change status without a case explaining why",
			})
			return memberContext{}, false
		}
	}
	return memberContext{
		r:        r,
		actor:    app.memberActor(r),
		year:     app.YearSlug(r),
		memberID: memberID,
		input:    input,
	}, true
}

// mintCorrectionCase opens a case for a manual correction and closes it immediately.
//
// # Why a case at all
//
// Every member command requires a `sosId`, so that "what happened to this member?" is
// always answered by reading cases and there is no second, case-less way for a status to
// change. An operator correcting an out-of-sync status from the patrol page has no case,
// so one is made for them rather than punching a hole in that rule.
//
// # Why closed at once
//
// It is a record, not work: leaving it open would put a row in the nødtelefon's open list
// that nobody needs to handle, and operators who learn to ignore entries in that list are
// the failure this whole tool is built against. Closed, it still appears in the patrol
// page's "Kontakt med nødtelefon" card, which is where somebody looking for it would look.
//
// It also makes corrections **countable** — one recognisable case each — which is what
// PRD 006 §9's "overrides stay rare" metric needs to be a query rather than a guess.
//
// # Why not reuse an open case
//
// A correction is rarely part of the story an open case is telling, and "reuse if exactly
// one is open" needs a rule for when two are. Predictability wins.
func (app *application) mintCorrectionCase(r *http.Request, memberID types.MemberID, teamID types.TeamID, to types.MemberStatus) (types.SosID, error) {
	name := app.memberName(memberID, teamID)
	if name == "" {
		name = string(memberID)
	}
	// The headline marks it as machine-made and names who was corrected, so nobody
	// scanning the case list mistakes it for a call. Confirmed with the product owner
	// 2026-08-17.
	headline := fmt.Sprintf("Manuel rettelse — %s", name)
	description := fmt.Sprintf(
		"Status rettet manuelt til %q fra deltagerlisten. Oprettet automatisk som dokumentation.",
		to,
	)

	actor := app.actor(r)
	year := app.YearSlug(r)
	id, err := app.commands.Sos.Create(r.Context(), actor, year, headline, description)
	if err != nil {
		return "", err
	}
	// Associate the patrol so the correction is reachable from the patrol page's card,
	// which is the only place anybody would go looking for it.
	//
	// AssociateTeamAt rather than AssociateTeam: the case was created microseconds ago and
	// its `created` event has not been projected yet, so the ordinary command's read-back
	// returns not-found and skips the association silently. That is exactly what happened
	// before this was fixed — every minted case had an empty team list and was therefore
	// invisible on the patrol page it documented.
	if teamID != "" {
		if err := app.commands.Sos.AssociateTeamAt(r.Context(), actor, year, id, teamID); err != nil {
			log.Printf("correction case %q: associating team failed: %v", id, err)
		}
	}
	return id, nil
}

// closeCorrectionCase closes a minted case once the correction is on its timeline.
//
// Failure is logged rather than returned: the correction itself has been recorded, and
// telling the operator their fix failed because a bookkeeping case stayed open would be
// worse than an extra row in a list.
func (app *application) closeCorrectionCase(r *http.Request, id types.SosID) {
	closed := sos.StatusClosed
	if err := app.commands.Sos.Patch(r.Context(), app.actor(r), id, sos.PatchCommand{Status: &closed}); err != nil {
		log.Printf("correction case %q: closing failed: %v", id, err)
	}
}

// memberStatusOperation runs a status-changing command and records it on the case.
//
// Shared by the three status handlers because the shape is identical and the
// interesting part — which command — is the one line they differ by. It also keeps
// the publish-then-summarise order in one place rather than three.
func (app *application) memberStatusOperation(w http.ResponseWriter, r *http.Request, allowMinted bool, run func(memberContext) (*spejderstatus.Change, error)) {
	ctx, ok := app.memberContext(w, r, allowMinted)
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
	// intent holds — and deliberately before any case is minted, so a double click on a
	// correction does not litter the record with empty cases.
	if change == nil {
		if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"change": nil}, nil); err != nil {
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	// Mint the case now rather than earlier: only a change that actually happened is
	// worth documenting, and the member events are already published by this point, so
	// the summary still lands after them as the ordering requires.
	sosID := ctx.input.SosID
	minted := false
	if sosID == "" {
		id, err := app.mintCorrectionCase(r, change.MemberID, change.TeamID, change.To)
		if err != nil {
			// The correction is recorded; only its paper trail failed. Report success
			// with a log rather than telling the operator their fix did not happen.
			log.Printf("correction case for member %q could not be created: %v", change.MemberID, err)
		} else {
			sosID, minted = id, true
		}
	}

	team, _ := app.models.Teams.GetPatrulje(change.TeamID)
	teamName := ""
	if team != nil {
		teamName = team.Name
	}
	if sosID != "" {
		summary := sos.MemberStatusChanged{
			SosID:    sosID,
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
	}
	// Closed only after the correction is on its timeline, so the case is never briefly
	// closed-and-empty for a reader who catches it mid-flight.
	if minted {
		app.closeCorrectionCase(r, sosID)
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"change": change, "sosId": sosID}, nil); err != nil {
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

// showMemberHandler serves everything known about one member, for the detail modal.
//
// Its own endpoint rather than folding this into the case payload: a case with three
// patrols has eighteen members, and carrying each one's address, birthday and full status
// history would make the screen an operator stares at all night pay for detail they open
// one member at a time. The card sends what a row needs; this sends what a modal needs.
func (app *application) showMemberHandler(w http.ResponseWriter, r *http.Request) {
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	year := app.YearSlug(r)

	status, err := app.models.SpejderStatus.GetByMemberID(r.Context(), year, memberID)
	if err != nil && !errors.Is(err, spejderstatus.ErrRecordNotFound) {
		app.ServerErrorResponse(w, r, err)
		return
	}

	// The roster is keyed by team, so the member's own team has to come from their status
	// row — or, for somebody with no status row yet, from the team the request names.
	// A member with neither is genuinely unknown.
	teamID := types.TeamID(r.URL.Query().Get("teamId"))
	if status != nil {
		if status.InitialTeamID != "" {
			teamID = status.InitialTeamID
		}
		if status.CurrentTeamID != "" {
			teamID = status.CurrentTeamID
		}
	}
	if teamID == "" {
		app.NotFoundResponse(w, r)
		return
	}

	var member *data.Spejder
	roster, _, err := app.models.Members.GetSpejdere(data.Filters{TeamID: teamID})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	for _, m := range roster {
		if m.MemberID == memberID {
			member = m
			break
		}
	}
	// A member moved into this team is not on its roster, so fall back to the team they
	// started with — the same asymmetry the case card handles.
	if member == nil && status != nil && status.InitialTeamID != "" && status.InitialTeamID != teamID {
		if initial, _, err := app.models.Members.GetSpejdere(data.Filters{TeamID: status.InitialTeamID}); err == nil {
			for _, m := range initial {
				if m.MemberID == memberID {
					member = m
					break
				}
			}
		}
	}
	if member == nil {
		app.NotFoundResponse(w, r)
		return
	}

	history, err := app.models.SpejderStatus.GetHistory(r.Context(), year, memberID)
	if err != nil && !errors.Is(err, spejderstatus.ErrRecordNotFound) {
		app.ServerErrorResponse(w, r, err)
		return
	}
	if history == nil {
		history = []spejderstatus.StatusEvent{}
	}

	// The member's own team, named, so the modal can say which patrol they are with now
	// without a second request.
	teamName := ""
	if team, err := app.models.Teams.GetPatrulje(teamID); err == nil && team != nil {
		teamName = team.Name
	}

	envelope := jsonapi.Envelope{
		"member":         member,
		"teamId":         teamID,
		"teamName":       teamName,
		"history":        history,
		"memberStatuses": MemberStatuses(),
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
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
