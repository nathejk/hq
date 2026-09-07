package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/tables/order"
	"github.com/nathejk/shared-go/tables/transfer"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	hqtables "nathejk.dk/nathejk/table"
	patruljetable "nathejk.dk/nathejk/table/patrulje"
)

// Pre-race reassignment: moving a member, and the money paid for them, to another
// patrulje (PRD 012).
//
// # Why this is not moveMemberTeamHandler
//
// There are two moves and they are different facts. `PUT /api/member/:id/team` is the
// nødtelefon's race-time move: somebody *started* with one patrol and continues with
// another, so the origin keeps them on its roster, `initialTeamId` is preserved, both
// teams have started, and a case explains why. This one is administrative: a contact
// person reshuffles teams before anyone sets off, the member simply belongs to the
// other patrol now, and there is no incident to record. Reusing either for the other
// would make `initialTeamId` — the record of where somebody started — a lie.
//
// Hence no `sosId`: an operator acting on a contact person's request is not handling
// an emergency, and minting a case per reshuffle would bury the real calls in the
// nødtelefon's list. That is the opposite trade-off from the manual status correction
// on the same page, which *does* mint one, because a status changing without a reason
// is exactly what that rule exists to prevent.
//
// # What is enforced here rather than in the domain
//
// The transfer command refuses only what it alone can know: an unknown member, and a
// move to the team they are already on. Everything else is enforced here because this
// is the layer that owns the team read models and the Danish wording an operator
// reads. Deliberately absent: any lower bound on the origin's member count. A team
// below three may not start, and the way out is to move its last one or two members to
// a team that has not started — so refusing to empty a team would block the case this
// feature exists for.

// maxPatruljeMemberCount is the upper bound on a patrol's size.
//
// The same 7 that TeamConfig serves to the SPA (see showPatruljeHandler). Stated as a
// constant here because this is the first place it is *enforced* rather than displayed,
// and a rule that only exists as a display value is a rule the server does not have.
const maxPatruljeMemberCount = 7

type reassignRequest struct {
	TeamID types.TeamID `json:"teamId"`
}

// transferCandidate is one destination the operator may pick, and why they may not.
//
// Ineligible teams are returned rather than filtered out: an operator looking for
// "Ulvene" needs to see that it is full or already started, not to wonder whether they
// misremembered the name. The reason is server-authored so the list and the refusal
// cannot word the same rule differently.
type transferCandidate struct {
	TeamID      types.TeamID `json:"teamId"`
	TeamNumber  string       `json:"teamNumber"`
	Name        string       `json:"name"`
	Group       string       `json:"group"`
	MemberCount int          `json:"memberCount"`
	Eligible    bool         `json:"eligible"`
	Reason      string       `json:"reason,omitempty"`
}

// transferLine is one thing that will move with the member, priced as it was paid for.
type transferLine struct {
	ProductSKU  string `json:"productSku"`
	ProductName string `json:"productName"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int    `json:"unitPrice"`
	LineTotal   int    `json:"lineTotal"`

	// Size is lifted out of the line's attributes because it is the one attribute an
	// operator needs to see: "t-shirt (xl)" is a different thing to hand over than
	// "t-shirt".
	Size string `json:"size,omitempty"`
}

// showTransferCandidatesHandler answers "where may this member go, and what moves with
// them?".
//
// @Summary     Destinations for a pre-race member transfer
// @Description Every patrulje that has been accepted into the race (it has a team number), annotated with whether this member may be moved there and, if not, why — teams that have started or are already full are returned as ineligible rather than hidden, so an operator can see that the team they were looking for exists. Also returns the paid lines that would move with the member (the participation seat and any merchandise, at the price actually paid) and their total, so the dialog can state what the transfer will do before it is confirmed. An empty `lines` with `amount` 0 is an ordinary outcome: nobody has paid for this member yet, so there is nothing to move.
// @Tags        member
// @Produce     json
// @Param       memberId path string true "Member id"
// @Success     200 {object} map[string]interface{} "envelope with \"fromTeamId\", \"candidates\", \"lines\", \"amount\", \"maxMemberCount\""
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/member/{memberId}/transfer-candidates [get]
func (app *application) showTransferCandidatesHandler(w http.ResponseWriter, r *http.Request) {
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	year := app.YearSlug(r)

	fromTeamID, ok := app.reassignOrigin(w, r, year, memberID)
	if !ok {
		return
	}

	teams, err := app.models.Patrulje.GetAll(r.Context(), patruljetable.Filter{YearSlug: year})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	candidates := make([]transferCandidate, 0, len(teams))
	for _, t := range teams {
		// No team number means the patrol has not been accepted into the race, so it
		// is not a destination at all — not even a struck-through one. Showing every
		// half-finished signup would drown the list in teams that are not coming.
		if t.TeamNumber == "" || t.TeamID == fromTeamID {
			continue
		}
		c := transferCandidate{
			TeamID:      t.TeamID,
			TeamNumber:  t.TeamNumber,
			Name:        t.Name,
			Group:       t.Group,
			MemberCount: t.MemberCount,
			Eligible:    true,
		}
		switch {
		case t.SignupStatus == types.SignupStatusStarted:
			c.Eligible, c.Reason = false, "startet"
		case t.MemberCount >= maxPatruljeMemberCount:
			c.Eligible, c.Reason = false, "fuld"
		}
		candidates = append(candidates, c)
	}

	lines, amount, err := app.transferableLines(r.Context(), year, fromTeamID, memberID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{
		"fromTeamId":     fromTeamID,
		"candidates":     candidates,
		"lines":          lines,
		"amount":         amount,
		"maxMemberCount": maxPatruljeMemberCount,
	}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// reassignMemberHandler moves a member to another patrulje, with their money.
//
// @Summary     Move a paid member to another patrulje before the race
// @Description Reassigns the member to another patrulje and moves what was paid for them along with them: the participation seat and any merchandise are credited back to the origin and charged to the destination, covered by internal-transfer payments rather than a card payment, so no money enters or leaves the system and the pair nets to zero. Requires that neither team has started and that the destination has been accepted into the race and has room. Deliberately requires no `sosId` — an administrative reshuffle is not an incident. Deliberately does not enforce a minimum on the origin: emptying a team out one member at a time is how a team that may not start hands its members over. A member nobody has paid for is reassigned with no orders created, which the response reports as a zero `amount`.
// @Tags        member
// @Accept      json
// @Produce     json
// @Param       memberId path string true "Member id"
// @Param       body body reassignRequest true "Destination team"
// @Success     200 {object} map[string]interface{} "envelope with \"transfer\""
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     422 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/member/{memberId}/reassign [put]
func (app *application) reassignMemberHandler(w http.ResponseWriter, r *http.Request) {
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input reassignRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if input.TeamID == "" {
		app.FailedValidationResponse(w, r, map[string]string{"teamId": "must be provided"})
		return
	}
	year := app.YearSlug(r)

	if _, ok := app.reassignOrigin(w, r, year, memberID); !ok {
		return
	}
	if !app.reassignTarget(w, r, year, input.TeamID) {
		return
	}

	res, err := app.commands.Transfer.TransferMember(r.Context(), transfer.Transfer{
		MemberID: memberID,
		ToTeamID: input.TeamID,
		TeamType: types.TeamTypePatrulje,
		Actor:    app.transferActor(r),
	})
	if err != nil {
		app.transferCommandError(w, r, err)
		return
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"transfer": res}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// reassignOrigin resolves the team the member is on and refuses if it has started.
//
// The origin is read from the roster rather than taken from the request: it is the only
// thing that knows, and a stale browser tab must not be able to name a team the member
// has already left. The transfer command reads it the same way for the same reason.
//
// A started origin is refused here, not in the domain, and the refusal points at the
// other move: once a patrol is on the route, a member leaving it is a race-time event
// with a case behind it, which is `PUT /api/member/:id/team`.
func (app *application) reassignOrigin(w http.ResponseWriter, r *http.Request, year types.YearSlug, memberID types.MemberID) (types.TeamID, bool) {
	fromTeamID, err := app.models.Roster.TeamIDOf(r.Context(), year, memberID)
	if err != nil {
		if errors.Is(err, tables.ErrRecordNotFound) {
			app.NotFoundResponse(w, r)
			return "", false
		}
		app.ServerErrorResponse(w, r, err)
		return "", false
	}

	origin, err := app.models.Patrulje.GetByID(r.Context(), fromTeamID)
	if err != nil {
		// A roster row whose team is missing is a broken read model, not a client
		// error: the member exists and names a team that does not.
		app.ServerErrorResponse(w, r, err)
		return "", false
	}
	if origin.SignupStatus == types.SignupStatusStarted {
		app.FailedValidationResponse(w, r, map[string]string{
			"memberId": "patruljen er startet — brug nødtelefonens flytning i stedet",
		})
		return "", false
	}
	return fromTeamID, true
}

// reassignTarget validates a destination, answering the client itself when it is not one.
//
// Three separate messages rather than one generic refusal: "ikke optaget i løbet",
// "startet" and "fuld" are three different things for the operator to do something
// about, and a single 422 would leave them guessing which rule they hit.
func (app *application) reassignTarget(w http.ResponseWriter, r *http.Request, year types.YearSlug, teamID types.TeamID) bool {
	target, err := app.models.Patrulje.GetByID(r.Context(), teamID)
	if err != nil {
		if errors.Is(err, hqtables.ErrRecordNotFound) {
			app.FailedValidationResponse(w, r, map[string]string{"teamId": "ukendt patrulje"})
			return false
		}
		app.ServerErrorResponse(w, r, err)
		return false
	}
	if target.TeamNumber == "" {
		app.FailedValidationResponse(w, r, map[string]string{"teamId": "patruljen er ikke optaget i løbet"})
		return false
	}
	if target.SignupStatus == types.SignupStatusStarted {
		app.FailedValidationResponse(w, r, map[string]string{"teamId": "patruljen er startet"})
		return false
	}
	// Counted from the roster rather than the stored column: patrulje.memberCount is
	// frozen at who started and is 0 before that, so the stored value cannot answer
	// "is this team full yet" for a team that has not started — which is every team
	// this endpoint deals with.
	count, err := app.rosterCount(r.Context(), year, teamID)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return false
	}
	if count >= maxPatruljeMemberCount {
		app.FailedValidationResponse(w, r, map[string]string{"teamId": "patruljen er fuld"})
		return false
	}
	return true
}

// rosterCount is how many members a team has on the roster right now.
func (app *application) rosterCount(ctx context.Context, year types.YearSlug, teamID types.TeamID) (int, error) {
	teams, err := app.models.Patrulje.GetAll(ctx, patruljetable.Filter{YearSlug: year, TeamID: teamID})
	if err != nil {
		return 0, err
	}
	for _, t := range teams {
		if t.TeamID == teamID {
			return t.MemberCount, nil
		}
	}
	return 0, nil
}

// transferableLines is what would move with the member, and what it is worth.
//
// Reads through the same interface the transfer command uses, so the preview an
// operator confirms and the transfer that happens cannot disagree about which lines
// those are.
func (app *application) transferableLines(ctx context.Context, year types.YearSlug, teamID types.TeamID, memberID types.MemberID) ([]transferLine, int, error) {
	paid, err := app.models.Order.PaidLinesByMember(ctx, year, types.TeamTypePatrulje, string(teamID), string(memberID))
	if err != nil {
		return nil, 0, err
	}
	lines := make([]transferLine, 0, len(paid))
	total := 0
	for _, l := range paid {
		lines = append(lines, transferLine{
			ProductSKU:  l.ProductSKU,
			ProductName: l.ProductName,
			Quantity:    l.Quantity,
			UnitPrice:   l.UnitPrice,
			LineTotal:   l.LineTotal,
			Size:        lineSize(l.Line),
		})
		total += l.LineTotal
	}
	return lines, total, nil
}

// lineSize reads a line's size attribute, if it has one.
func lineSize(l order.Line) string {
	if l.Attributes == nil {
		return ""
	}
	if s, ok := l.Attributes["size"].(string); ok {
		return s
	}
	return ""
}

// transferActor is who performed the transfer, for the reassignment event.
//
// A third structurally identical actor struct, and the first that is *shared-go's own*
// rather than a local one — see the note on memberActor in actor.go, which says three
// is a pattern. If the local sos.Actor and spejderstatus.Actor ever converge on one
// type, this is the one to converge on.
func (app *application) transferActor(r *http.Request) messages.NathejkMemberActor {
	a := app.memberActor(r)
	return messages.NathejkMemberActor{UserID: a.UserID, Name: a.Name}
}

// transferCommandError maps the transfer command's refusals onto responses.
//
// ErrSameTeam is a 422 with a field message because it is something the operator can
// see and correct. ErrNotBalanced is a 500 on purpose: it means the credit and charge
// would not have netted to zero, nothing was published, and no wording of that belongs
// in front of an operator.
func (app *application) transferCommandError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, transfer.ErrSameTeam):
		app.FailedValidationResponse(w, r, map[string]string{"teamId": "spejderen er allerede p\u00e5 den patrulje"})
	case errors.Is(err, transfer.ErrMemberNotFound):
		app.NotFoundResponse(w, r)
	default:
		app.ServerErrorResponse(w, r, err)
	}
}
