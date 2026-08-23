package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/shelter"
	"nathejk.dk/nathejk/table/sos"
	"nathejk.dk/nathejk/table/spejderstatus"
)

// Hønsegården: the shelter crew's screen (PRD 007).
//
// One endpoint serves the whole page, because the page is one question asked four ways —
// who is on their way, who is here, who is still out, and who has been handed on — and
// the crew reads all four at once. Four endpoints would be four round trips to render one
// screen and four chances for the counts to disagree with each other.

// shelterSection is one group of the screen, as it is rendered.
//
// The label travels with the data rather than living in the view, for the reason PRD 006 §6
// already settled for status labels: two places holding the same Danish copy is how one of
// them ends up saying something else to an operator at 3am. It also means the sections can be
// reordered or renamed without touching the SPA.
type shelterSection struct {
	Slug    string          `json:"slug"`
	Label   string          `json:"label"`
	Members []shelterMember `json:"members"`
}

// shelterMember is a scout as a row on the screen needs them.
//
// Contact details are limited to the two phone numbers, which is what the crew actually
// dials — the address, birthday and full history are one click away on the member modal
// (`GET /api/member/:memberId`) rather than carried for every row of a screen that stays
// open all night.
type shelterMember struct {
	MemberID types.MemberID     `json:"memberId"`
	Name     string             `json:"name"`
	Status   types.MemberStatus `json:"status"`

	// UpdatedAt is when the status last changed, so the screen can say "venter siden 21:40
	// (2t 14m)". Sent as a timestamp rather than a formatted duration: a duration
	// serialised by the server is wrong the moment it is rendered, and stops being wrong
	// only if the page refetches on a timer.
	UpdatedAt time.Time `json:"updatedAt"`

	Phone       string `json:"phone"`
	PhoneParent string `json:"phoneParent"`

	// Team is the patrol the scout is with now. Nil only for a member whose team cannot be
	// read at all, which is not worth failing the screen for — the row is still true and
	// still actionable.
	Team *memberTeamRef `json:"team"`

	// StartTeam is filled in only when it differs from Team, i.e. for a scout who was moved
	// to another patrol before dropping out. Omitted otherwise so the ordinary row does not
	// carry the same patrol twice, and so the SPA can treat its presence as the fact it is:
	// "this one did not start where you think".
	StartTeam *memberTeamRef `json:"startTeam,omitempty"`

	// Placement and PlacedAt are the shelter's own facts, empty and nil for anybody not in
	// the shelter. An accepted scout with no placering yet is the crew's next job, which is
	// why the empty string is a state rather than a missing value.
	Placement string     `json:"placement"`
	PlacedAt  *time.Time `json:"placedAt"`

	// SosID links to the open case, when there is one. There often is not: the shelter may
	// receive a scout nobody opened a case about, which is exactly why none of this
	// screen's write actions requires a case.
	SosID types.SosID `json:"sosId,omitempty"`
}

// shelterStatuses is the population of the screen: everybody who started and is no longer
// active.
//
// Split into the two halves that mean different things to the crew rather than returned as
// one list, because the sections are built from them and the counts are read separately:
// inCare is what the organisers are waiting on, closed is the record of the night.
//
// `finished`, `racing`, `registered` and `seated` are all absent, for two different reasons.
// The first two are the route's business — a scout on the trail, or one who walked it to the
// end, is not the shelter's problem. The last two never started at all, so including them
// would fill the screen with several hundred people who are at home in bed.
func shelterStatuses() (inCare []types.MemberStatus, closed []types.MemberStatus) {
	// Derived, not listed: the in-care set is shared-go's to define, and a fourth in-care
	// state added there must start appearing on this screen without anybody remembering to
	// edit a list here. That is the same argument spejderstatus.InOurCareStatuses() exists
	// for.
	inCare = spejderstatus.InOurCareStatuses()

	// These two cannot be derived — shared-go has no "ended in our care" predicate, and
	// adding one would be a change to a shared package for one screen's benefit. They are
	// the two endings available to a scout who left the route (`finished` is not one of
	// them, deliberately: see types.MemberStatusFinished), so the list is closed by the
	// domain rather than by taste, and shelterStatusesCoverTheLifecycle asserts it.
	closed = []types.MemberStatus{
		types.MemberStatusReunited,
		types.MemberStatusReleased,
	}
	return inCare, closed
}

// showShelterHandler serves the Hønsegården screen.
//
// @Summary     The shelter population for Hønsegården
// @Description Every scout of the active year who started and is no longer active, grouped as the screen renders them: in a car on the way, in the shelter, waiting to be collected, and handed on. Includes each scout's placering inside the shelter, the two phone numbers the crew dials, a link to the open SOS case where there is one, the in-our-care breakdown, the status label vocabulary, and the placeringer currently in use for the suggestion list. Year comes from the X-YearSlug header, or the current year.
// @Tags        shelter
// @Produce     json
// @Success     200 {object} map[string]interface{} "envelope with \"sections\", \"counts\", \"care\", \"memberStatuses\" and \"placements\""
// @Failure     500 {object} map[string]interface{}
// @Router      /api/shelter [get]
func (app *application) showShelterHandler(w http.ResponseWriter, r *http.Request) {
	year := app.YearSlug(r)
	ctx := r.Context()

	inCare, closed := shelterStatuses()
	rows, err := app.models.SpejderStatus.GetByStatuses(ctx, year, append(append([]types.MemberStatus{}, inCare...), closed...))
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	byStatus, err := app.shelterMembers(ctx, year, rows)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	// The order is the crew's working order, and it is the server's to decide: the arrivals
	// queue first because it is the only section with somebody standing in front of it, the
	// shelter itself second because that is what a parent at the door is answered from.
	sections := []shelterSection{
		{Slug: "transit", Label: "I bil — på vej hertil", Members: membersIn(byStatus, types.MemberStatusTransit)},
		{Slug: "sheltered", Label: "I Hønsegården", Members: membersIn(byStatus, types.MemberStatusSheltered)},
		{Slug: "waiting", Label: "Afventer afhentning", Members: membersIn(byStatus, types.MemberStatusWaiting)},
		{Slug: "closed", Label: "Afsluttet", Members: membersIn(byStatus, types.MemberStatusReunited, types.MemberStatusReleased)},
	}
	counts := map[string]int{}
	for _, s := range sections {
		counts[s.Slug] = len(s.Members)
	}

	// The same number the organisers are waiting on, read from the same query the dashboard
	// uses rather than counted from the rows above. Two independent counts of the children we
	// are responsible for is one more than the night can afford.
	care, err := app.models.SpejderStatus.InOurCare(ctx, year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	// The placering vocabulary in use tonight. There is no zone entity by design — the zones
	// are not known until race start — so the suggestions are whatever the crew has already
	// typed (PRD 007 §6).
	placements, err := app.models.Shelter.DistinctPlacements(ctx, year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	envelope := jsonapi.Envelope{
		"sections": sections,
		"counts":   counts,
		"care":     care,
		// The lifecycle with Danish labels, so this screen renders a status without a label
		// map of its own (PRD 006 §6).
		"memberStatuses": MemberStatuses(),
		"placements":     placements,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// membersIn collects the rows for one or more statuses into a section.
//
// Always a non-nil slice, which is the point and was found by running it rather than by
// reading it: a nil slice marshals to `null`, and an empty section is `"members": null` on the
// wire. Every client then has to defend against it — `section.members.length` throws on null,
// so the screen with nobody in a car would break precisely because nothing was wrong. "None"
// is a list of nothing, not the absence of a list, and saying so is the server's job.
func membersIn(byStatus map[types.MemberStatus][]shelterMember, statuses ...types.MemberStatus) []shelterMember {
	out := []shelterMember{}
	for _, s := range statuses {
		out = append(out, byStatus[s]...)
	}
	return out
}

// shelterMembers turns status rows into screen rows, grouped by status.
//
// Everything it needs is fetched in bulk before the loop: the year's roster for names and
// phones, the year's patrols for team names, the placeringer for the members in question. A
// query per member would be a query per row on a page the crew keeps open all night, and the
// population is tens of scouts drawn from tens of patrols.
//
// The one exception is the SOS case, which is looked up per *distinct patrol* and memoised —
// there is no bulk "cases for these teams" query, and adding one for a link in a row is not
// worth a new projection method. The bound is the number of affected patrols, not the number
// of scouts.
func (app *application) shelterMembers(ctx context.Context, year types.YearSlug, rows []spejderstatus.SpejderStatus) (map[types.MemberStatus][]shelterMember, error) {
	out := map[types.MemberStatus][]shelterMember{}
	if len(rows) == 0 {
		return out, nil
	}

	ids := make([]types.MemberID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.MemberID)
	}

	// The whole year's roster in one query, indexed by member. GetSpejdere is team-scoped or
	// year-scoped, and year-scoped is the cheaper shape here: one query returning a few
	// hundred rows beats one query per patrol, and this screen's members are spread thinly
	// across many patrols.
	roster, _, err := app.models.Members.GetSpejdere(data.Filters{Year: string(year)})
	if err != nil {
		return nil, err
	}
	people := make(map[types.MemberID]*data.Spejder, len(roster))
	for _, m := range roster {
		people[m.MemberID] = m
	}

	teams, err := app.models.Patrulje.GetAll(ctx, patrulje.Filter{YearSlug: year})
	if err != nil {
		return nil, err
	}
	patrols := make(map[types.TeamID]patrulje.Patrulje, len(teams))
	for _, t := range teams {
		patrols[t.TeamID] = t
	}

	placements, err := app.models.Shelter.GetByMemberIDs(ctx, year, ids)
	if err != nil {
		return nil, err
	}

	cases := map[types.TeamID]types.SosID{}
	for _, row := range rows {
		member := shelterMember{
			MemberID:  row.MemberID,
			Status:    row.Status,
			UpdatedAt: row.UpdatedAt,
			Team:      shelterTeamRef(patrols, row.CurrentTeamID),
		}
		if person, ok := people[row.MemberID]; ok {
			member.Name = person.Name
			member.Phone = person.Phone
			member.PhoneParent = person.PhoneParent
		}
		// A name is the one field the crew cannot work without, so a member missing from the
		// roster falls back to their id rather than rendering as a blank row. That happens
		// for a scout whose signup row was removed after they started — rare, and not a
		// reason to hide a child from the screen.
		if member.Name == "" {
			member.Name = string(row.MemberID)
		}
		// Only when it differs: for the ordinary scout the two are the same patrol, and
		// sending it twice would invite the SPA to render "startede i" on every row.
		if row.InitialTeamID != "" && row.InitialTeamID != row.CurrentTeamID {
			member.StartTeam = shelterTeamRef(patrols, row.InitialTeamID)
		}
		if p, ok := placements[row.MemberID]; ok {
			member.Placement = p.Placement
			member.PlacedAt = p.PlacedAt
		}
		if id, ok := cases[row.CurrentTeamID]; ok {
			member.SosID = id
		} else {
			id := app.openCaseFor(ctx, year, row.CurrentTeamID)
			cases[row.CurrentTeamID] = id
			member.SosID = id
		}
		out[row.Status] = append(out[row.Status], member)
	}
	return out, nil
}

// shelterTeamRef names a patrol from the map already fetched.
//
// Returns a ref with just the id when the patrol cannot be found, rather than nil: the id is
// still true and still links, and a crew member looking for a child needs the row more than
// they need its patrol's name.
func shelterTeamRef(patrols map[types.TeamID]patrulje.Patrulje, id types.TeamID) *memberTeamRef {
	if id == "" {
		return nil
	}
	ref := &memberTeamRef{TeamID: id}
	if team, ok := patrols[id]; ok {
		ref.Name = team.Name
		ref.TeamNumber = team.TeamNumber
	}
	return ref
}

// --- the write surface (PRD 007) ---
//
// None of these requires a `sosId`. The shelter may receive a scout nobody opened a case
// about, and the acceptance events are case-free by design — so they run under
// `caseOptional`, and no case is minted for them (see casePolicy in member.go).

// shelterRequest is the body all three write endpoints share.
//
// `to` and `placement` are each meaningful to one endpoint only. One struct rather than three
// because the SPA sends the same shape and the endpoints differ by verb, not by vocabulary.
type shelterRequest struct {
	// Placement is the placering. Optional on the acceptance — a crew member receiving
	// somebody at a run records the arrival now and the tent when they get back — and required
	// on the placering endpoint.
	Placement string `json:"placement"`

	// To is which ending a handover was: `released` (a guardian came) or `reunited` (their own
	// patrol finished and took them back). Never guessed from the hour: they are different
	// facts about who has the child.
	To types.MemberStatus `json:"to"`
}

// acceptIntoShelterHandler records that Hønsegården has received a scout.
//
// @Summary     Accept a scout into Hønsegården
// @Description Records that the shelter crew has taken charge of the scout, optionally with the placering they were put in. Valid from any started status — including `waiting` and `racing`, for the scout who arrived in a car nobody logged — because the receiving crew's word is the better evidence. Refused only for a member who never started. Requires no SOS case. A scout who is already sheltered is a no-op answered with 200; use the placering endpoint to move them.
// @Tags        shelter
// @Accept      json
// @Produce     json
// @Param       memberId path string true "Member id"
// @Param       body body shelterRequest false "Optional placering"
// @Success     200 {object} map[string]interface{} "envelope with \"change\" (null for a no-op) and \"sosId\""
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/member/{memberId}/shelter [put]
func (app *application) acceptIntoShelterHandler(w http.ResponseWriter, r *http.Request) {
	// The placering is read from a second decode of the body rather than being added to
	// memberRequest: it belongs to this screen, and widening the shared nødtelefon request
	// struct would offer a placering to every member endpoint.
	placement := strings.TrimSpace(app.shelterInput(r).Placement)

	app.memberStatusOperation(w, r, caseOptional, func(ctx memberContext) (*spejderstatus.Change, error) {
		if len([]rune(placement)) > shelter.MaxPlacementLength {
			return nil, shelter.ErrPlacementTooLong
		}
		return app.commands.Member.AcceptIntoShelter(ctx.r.Context(), ctx.actor, ctx.year, ctx.memberID, placement)
	})
}

// setPlacementHandler records where in the shelter a scout is, or moves them.
//
// @Summary     Set or change a sheltered scout's placering
// @Description Records where in Hønsegården the scout is. Free text by design — the zones are not known until race start, so the API suggests what is already in use (see GET /api/shelter) and enforces nothing beyond a 64-character limit. Valid only while the scout is sheltered; a blank placering is refused, because "nowhere" is not a fact about a child in our care. Setting the placering they already have is a no-op answered with 200. Requires no SOS case.
// @Tags        shelter
// @Accept      json
// @Produce     json
// @Param       memberId path string true "Member id"
// @Param       body body shelterRequest true "The placering"
// @Success     200 {object} map[string]interface{} "envelope with \"placement\""
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/member/{memberId}/placement [put]
func (app *application) setPlacementHandler(w http.ResponseWriter, r *http.Request) {
	memberID := types.MemberID(app.ReadNamedParam(r, "memberId"))
	if memberID == "" {
		app.NotFoundResponse(w, r)
		return
	}
	var input shelterRequest
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	// Not routed through memberStatusOperation: this changes no status, so there is no Change
	// to summarise and nothing for the strength arithmetic to do. Reusing that helper would
	// have meant inventing a status transition to describe a scout staying exactly where they
	// are in the lifecycle.
	if err := app.commands.Shelter.SetPlacement(r.Context(), app.memberActor(r), app.YearSlug(r), memberID, input.Placement); err != nil {
		app.shelterCommandError(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{"placement": strings.TrimSpace(input.Placement)}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// completeHandoverHandler records that somebody else has taken charge of a scout.
//
// @Summary     Hand a scout over and take them out of our care
// @Description Records the ending: `released` when a guardian collected them, `reunited` when their own patrol reached the finish and took them back. The two are not interchangeable and neither is `finished`, which no handover may confer — only walking the route unaided earns it. This is what takes the scout out of the in-our-care count. Does not require an arrival at HQ first, since a guardian can collect from the roadside. Requires no SOS case.
// @Tags        shelter
// @Accept      json
// @Produce     json
// @Param       memberId path string true "Member id"
// @Param       body body shelterRequest true "Which ending"
// @Success     200 {object} map[string]interface{} "envelope with \"change\" (null for a no-op) and \"sosId\""
// @Failure     400 {object} map[string]interface{}
// @Failure     404 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/member/{memberId}/handover [put]
func (app *application) completeHandoverHandler(w http.ResponseWriter, r *http.Request) {
	to := app.shelterInput(r).To

	app.memberStatusOperation(w, r, caseOptional, func(ctx memberContext) (*spejderstatus.Change, error) {
		return app.commands.Member.CompleteHandover(ctx.r.Context(), ctx.actor, ctx.year, ctx.memberID, to)
	})
}

// shelterInput reads this screen's fields from the request body without consuming it.
//
// memberStatusOperation decodes the body itself (into memberRequest), and a request body can
// only be read once — so the bytes are buffered and put back. The alternative was adding
// `placement` and `to` to memberRequest, which is the nødtelefon's shared shape: every member
// endpoint would then advertise a placering it has no use for.
//
// A body that will not parse is not reported here. memberStatusOperation decodes the same
// bytes a moment later and answers 400 properly; failing twice for one malformed body would
// mean two error paths for one mistake.
func (app *application) shelterInput(r *http.Request) shelterRequest {
	var input shelterRequest
	if r.Body == nil {
		return input
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return input
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	_ = json.Unmarshal(body, &input)
	return input
}

// shelterCommandError maps the placering command's errors onto responses.
//
// Each one is something the crew can act on, so each gets its own message rather than a
// generic 400: "scouten er ikke i Hønsegården" tells them to press Modtaget first, which is
// one button away.
func (app *application) shelterCommandError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, shelter.ErrNotSheltered):
		app.FailedValidationResponse(w, r, map[string]string{
			"placement": "spejderen er ikke i Hønsegården — modtag dem først",
		})
	case errors.Is(err, shelter.ErrEmptyPlacement):
		app.FailedValidationResponse(w, r, map[string]string{
			"placement": "placering skal angives",
		})
	case errors.Is(err, shelter.ErrPlacementTooLong):
		app.FailedValidationResponse(w, r, map[string]string{
			"placement": fmt.Sprintf("placering må højst være %d tegn", shelter.MaxPlacementLength),
		})
	case errors.Is(err, spejderstatus.ErrRecordNotFound):
		app.NotFoundResponse(w, r)
	default:
		app.ServerErrorResponse(w, r, err)
	}
}

// openCaseFor finds the patrol's open case, if it has one.
//
// The most recently created open case wins where there are several, which is the one an
// operator is working. A failure to read is swallowed on purpose: the case link is a
// convenience, and a screen that refused to show where children are because the SOS join
// failed would be trading the important thing for the incidental one.
func (app *application) openCaseFor(ctx context.Context, year types.YearSlug, teamID types.TeamID) types.SosID {
	if teamID == "" {
		return ""
	}
	found, err := app.models.Sos.GetByTeam(ctx, year, teamID)
	if err != nil {
		return ""
	}
	for _, c := range found {
		if c.Status == sos.StatusOpen {
			return c.ID
		}
	}
	return ""
}
