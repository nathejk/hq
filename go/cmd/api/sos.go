package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/nathejk/shared-go/tables"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/table/sos"
	"nathejk.dk/nathejk/table/spejderstatus"
)

// Handlers for the nødtelefon (PRD 001): SOS cases, their timeline and the
// patrols associated with them.
//
// These stay in hq permanently — only the projection, queries, commands and
// schema are lifted to shared-go later (PRD 001 §8). The year is always taken
// from X-YearSlug via app.YearSlug, never from the path or a query parameter,
// because the SPA's axios interceptor already sends it on every request.

// listSosHandler returns the year's cases, split into open and closed.
//
// Grouped server-side because both halves come from one query and the split is
// how the screen is laid out; the alternative is every client re-deriving it.
func (app *application) listSosHandler(w http.ResponseWriter, r *http.Request) {
	year := app.YearSlug(r)

	cases, err := app.models.Sos.GetAll(r.Context(), sos.Filter{YearSlug: year})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	open, closed := []*sos.Sos{}, []*sos.Sos{}
	for _, c := range cases {
		if c.Status == sos.StatusClosed {
			closed = append(closed, c)
			continue
		}
		open = append(open, c)
	}

	envelope := jsonapi.Envelope{
		"open":   open,
		"closed": closed,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// showSosHandler returns one case with its timeline and associated patrols.
//
// The patrols are enriched here with the identity and contact details an operator
// needs mid-call — number, name, group, korps, contact phone — read live from the
// patrulje projection rather than copied into sos_team, so a renamed patrol is
// never stale on a case. Deliberately no members: PRD 001 ships without them, and
// PRD 006 introduces them together with the status and actions that make them
// useful.
func (app *application) showSosHandler(w http.ResponseWriter, r *http.Request) {
	id := types.SosID(app.ReadNamedParam(r, "id"))

	c, err := app.models.Sos.GetByID(r.Context(), id)
	if err != nil {
		app.handleSosError(w, r, err)
		return
	}

	envelope := jsonapi.Envelope{
		"case":  c,
		"teams": app.sosTeams(r, c),
		// The lifecycle with Danish labels, so the card renders a status without a
		// label map of its own (PRD 006 §6).
		"memberStatuses": MemberStatuses(),
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// sosTeam is an associated patrol as the case screen needs it.
type sosTeam struct {
	TeamID       types.TeamID      `json:"teamId"`
	TeamNumber   string            `json:"teamNumber"`
	Name         string            `json:"name"`
	Group        string            `json:"group"`
	Korps        string            `json:"korps"`
	ContactName  string            `json:"contactName"`
	ContactPhone types.PhoneNumber `json:"contactPhone"`

	// ActiveMemberCount is the patrol's strength on the route, and MinMemberCount the
	// number it is expected to keep (3 for patruljer). Both are served so the card can
	// show "under styrke" without a second request and without hardcoding 3.
	//
	// Started matters for one reason and it is not obvious: a team that never started
	// also has zero racing members, so strength alone cannot tell *left the route* from
	// *never on it*. Without this the Udgået badge would appear on every patrol of a year
	// that has not raced yet — see PRD 006 §11.
	ActiveMemberCount int  `json:"activeMemberCount"`
	MinMemberCount    int  `json:"minMemberCount"`
	Started           bool `json:"started"`

	Members []sosMember `json:"members"`
}

// sosMember is one member of an associated patrol, as a **row** needs them: who they are
// and where they are in the lifecycle. Nothing more.
//
// Contact details, address, birthday and the status history are deliberately absent — the
// row is an index an operator scans down, and all of that belongs to one member at a time.
// `GET /api/member/:memberId` serves it when the detail modal opens, so a case with three
// patrols does not carry eighteen members' worth of detail nobody has asked to see.
type sosMember struct {
	MemberID types.MemberID     `json:"memberId"`
	Name     string             `json:"name"`
	Status   types.MemberStatus `json:"status"`

	// UpdatedAt is when the status last changed, so the modal can say "venter siden 21:40".
	// Nil for a member with no status row — one who has not started — rather than a zero
	// time, which the SPA would render as 1970.
	UpdatedAt *time.Time `json:"updatedAt"`
}

func (app *application) sosTeams(r *http.Request, c *sos.Sos) []sosTeam {
	teams := make([]sosTeam, 0, len(c.Teams))
	for _, t := range c.Teams {
		// 3 for patruljer. A per-team-type constant rather than configuration: nothing
		// branches on it, it is displayed so an operator can apply it, and no command
		// enforces it (PRD 006 §11, task 074).
		team := sosTeam{TeamID: t.TeamID, MinMemberCount: 3}
		// A patrol that cannot be found still shows as an associated row: losing the
		// association because the projection is behind would hide the fact that the
		// case is about somebody.
		if p, err := app.models.Patrulje.GetByID(r.Context(), t.TeamID); err == nil && p != nil {
			team.TeamNumber = p.TeamNumber
			team.Name = p.Name
			team.Group = p.Group
			team.Korps = p.Korps
			team.ContactName = p.ContactName
			team.ContactPhone = p.ContactPhone
			team.ActiveMemberCount = p.ActiveMemberCount
			team.Started = p.SignupStatus == types.SignupStatusStarted
		}
		team.Members = app.sosTeamMembers(r, t.TeamID)
		teams = append(teams, team)
	}
	return teams
}

// sosTeamMembers assembles a patrol's members with their lifecycle status.
//
// Two sources, joined here: the roster and contact details come from the spejder read
// model, the status and its timestamp from spejderstatus. They are separate reads rather
// than one join because the two entities are owned by different packages — and
// spejderstatus is keyed on *currentTeamId*, which is the point: a member moved into this
// patrol belongs on its card, and one moved away does not, even though the roster still
// lists them under the team they signed up with.
//
// Members with no status row are included with an empty status. A patrol whose case is
// opened before the race has members who simply have no lifecycle yet, and leaving them
// out would make the card look like it had lost people.
func (app *application) sosTeamMembers(r *http.Request, teamID types.TeamID) []sosMember {
	members := []sosMember{}

	statuses := map[types.MemberID]spejderstatus.SpejderStatus{}
	if rows, err := app.models.SpejderStatus.GetByTeam(r.Context(), spejderstatus.Filter{
		YearSlug: app.YearSlug(r),
		TeamID:   teamID,
	}); err == nil {
		for _, s := range rows {
			statuses[s.MemberID] = s
		}
	}

	roster, _, err := app.models.Members.GetSpejdere(data.Filters{TeamID: teamID})
	if err != nil {
		return members
	}
	seen := map[types.MemberID]bool{}
	for _, m := range roster {
		// Skip a member the roster still lists here but who has been moved to another
		// patrol: they are that patrol's business now.
		if s, ok := statuses[m.MemberID]; ok && s.CurrentTeamID != teamID {
			continue
		}
		seen[m.MemberID] = true
		member := sosMember{
			MemberID: m.MemberID,
			Name:     m.Name,
		}
		if s, ok := statuses[m.MemberID]; ok {
			member.Status = s.Status
			updated := s.UpdatedAt
			member.UpdatedAt = &updated
		}
		members = append(members, member)
	}
	// A member moved *into* this patrol is not on its roster, so they would otherwise be
	// missing from the card that now owns them — which is exactly the case the move
	// feature creates. Their name comes from their own team's roster.
	for id, s := range statuses {
		if seen[id] {
			continue
		}
		updated := s.UpdatedAt
		members = append(members, sosMember{
			MemberID:  id,
			Name:      app.memberName(id, s.InitialTeamID),
			Status:    s.Status,
			UpdatedAt: &updated,
		})
	}
	return members
}

// createSosHandler opens a case. Headline and description are both required.
func (app *application) createSosHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Headline    string `json:"headline"`
		Description string `json:"description"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	id, err := app.commands.Sos.Create(r.Context(), app.actor(r), app.YearSlug(r), input.Headline, input.Description)
	if err != nil {
		app.handleSosError(w, r, err)
		return
	}

	// The created case is read back rather than echoed from the input, so the client
	// gets the row as the projection recorded it, with its timestamps.
	//
	// It usually will not be there yet: the command publishes to JetStream and the
	// projection applies asynchronously, so this read loses the race more often than
	// it wins it. That is fine, but it must not be answered with just an id — the SPA
	// navigates to the new case immediately, and with nothing to render it would show
	// a not-found until the first signal arrived. So the fallback synthesises the
	// case from what was just published, with 202 to say "accepted, not yet
	// projected", and the client seeds its cache from it and revalidates when the
	// signal lands.
	if c, err := app.models.Sos.GetByID(r.Context(), id); err == nil {
		if err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"case": c}, nil); err != nil {
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	now := time.Now().UTC()
	pending := sos.Sos{
		ID:             id,
		YearSlug:       app.YearSlug(r),
		Headline:       strings.TrimSpace(input.Headline),
		Description:    strings.TrimSpace(input.Description),
		Status:         sos.StatusOpen,
		CreatedAt:      now,
		LastActivityAt: now,
		Timeline:       []sos.Activity{},
		Teams:          []sos.Team{},
	}
	if err := app.WriteJSON(w, http.StatusAccepted, jsonapi.Envelope{"case": pending}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// patchSosHandler carries every single-field update on a case.
//
// Pointer fields so "absent" is distinguishable from "set to empty" — the same
// pattern as updateYearHandler and patchKlanHandler. Close and reopen arrive here
// as a status field like any other, which is what makes them idempotent. The
// handler validates, hands the diff to the command, and answers 202 without
// reading anything back — see the comment at the end for why that matters.
func (app *application) patchSosHandler(w http.ResponseWriter, r *http.Request) {
	id := types.SosID(app.ReadNamedParam(r, "id"))

	var input struct {
		Headline            *string       `json:"headline"`
		Description         *string       `json:"description"`
		Severity            *sos.Severity `json:"severity"`
		AssigneeSectionSlug *types.Slug   `json:"assigneeSectionSlug"`
		Status              *sos.Status   `json:"status"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	// An assignee must be a section that has been marked assignable for the year,
	// so the API cannot be used to route a case somewhere the Organisation page
	// says it should not go. Clearing the assignee (empty slug) stays allowed.
	if input.AssigneeSectionSlug != nil && *input.AssigneeSectionSlug != "" {
		assignable, err := app.models.Sos.AssignableSections(r.Context(), app.YearSlug(r))
		if err != nil {
			app.ServerErrorResponse(w, r, err)
			return
		}
		if !containsSlug(assignable, *input.AssigneeSectionSlug) {
			app.BadRequestResponse(w, r, errFromString("section is not assignable"))
			return
		}
	}

	cmd := sos.PatchCommand{
		Headline:            input.Headline,
		Description:         input.Description,
		Severity:            input.Severity,
		AssigneeSectionSlug: input.AssigneeSectionSlug,
		Status:              input.Status,
	}
	if err := app.commands.Sos.Patch(r.Context(), app.actor(r), id, cmd); err != nil {
		app.handleSosError(w, r, err)
		return
	}

	// Deliberately **not** reading the case back.
	//
	// The command publishes to JetStream and the projection applies asynchronously,
	// so a read here almost always returns the row as it was *before* the patch. The
	// SPA applies the operator's change optimistically, so answering with pre-patch
	// values would make it flicker backwards to the old value and then forwards again
	// when the signal arrives — the "stale value that looks live" failure PRD 004 §12
	// spent the whole feature fighting.
	//
	// So: 202, echoing what was accepted. The client keeps its optimistic state and
	// the live signal delivers the authoritative row a moment later.
	envelope := jsonapi.Envelope{"accepted": cmd}
	if err := app.WriteJSON(w, http.StatusAccepted, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// deleteSosHandler soft-deletes a case created in error. Nothing is destroyed;
// the case simply stops resolving.
func (app *application) deleteSosHandler(w http.ResponseWriter, r *http.Request) {
	id := types.SosID(app.ReadNamedParam(r, "id"))

	if err := app.commands.Sos.Delete(r.Context(), app.actor(r), id); err != nil {
		app.handleSosError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": id}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// commentSosHandler adds a comment and returns its id, which the client needs to
// edit it later.
func (app *application) commentSosHandler(w http.ResponseWriter, r *http.Request) {
	id := types.SosID(app.ReadNamedParam(r, "id"))

	var input struct {
		Comment string `json:"comment"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	commentID, err := app.commands.Sos.Comment(r.Context(), app.actor(r), id, input.Comment)
	if err != nil {
		app.handleSosError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"commentId": commentID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updateSosCommentHandler amends a comment's text.
//
// Any operator may edit any comment (PRD 001 §11): there is no per-user identity
// to restrict it to the author. What makes that safe is that the edit is appended
// to the timeline rather than replacing the original.
func (app *application) updateSosCommentHandler(w http.ResponseWriter, r *http.Request) {
	id := types.SosID(app.ReadNamedParam(r, "id"))
	commentID := sos.CommentID(app.ReadNamedParam(r, "commentId"))

	var input struct {
		Comment string `json:"comment"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	if err := app.commands.Sos.UpdateComment(r.Context(), app.actor(r), id, commentID, input.Comment); err != nil {
		app.handleSosError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"commentId": commentID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// associateSosTeamHandler ties a patrol to a case.
//
// Patrols only: a case is about a patrol on the route. The check is here rather
// than in the domain because it is the patrulje read model that knows what a
// patrol is.
func (app *application) associateSosTeamHandler(w http.ResponseWriter, r *http.Request) {
	id := types.SosID(app.ReadNamedParam(r, "id"))
	teamID := types.TeamID(app.ReadNamedParam(r, "teamId"))

	if _, err := app.models.Patrulje.GetByID(r.Context(), teamID); err != nil {
		app.BadRequestResponse(w, r, errFromString("only patruljer can be associated with a case"))
		return
	}
	if err := app.commands.Sos.AssociateTeam(r.Context(), app.actor(r), id, teamID); err != nil {
		app.handleSosError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"teamId": teamID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// disassociateSosTeamHandler removes a patrol from a case.
//
// No patrol lookup: removing an association must keep working even if the patrol
// has since become unreadable, or a mistake could not be undone.
func (app *application) disassociateSosTeamHandler(w http.ResponseWriter, r *http.Request) {
	id := types.SosID(app.ReadNamedParam(r, "id"))
	teamID := types.TeamID(app.ReadNamedParam(r, "teamId"))

	if err := app.commands.Sos.DisassociateTeam(r.Context(), app.actor(r), id, teamID); err != nil {
		app.handleSosError(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"teamId": teamID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// handleSosError maps domain errors onto status codes in one place, so every
// handler answers a missing or deleted case the same way.
func (app *application) handleSosError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, tables.ErrRecordNotFound):
		// Covers the soft-deleted case too: it stops resolving, which is exactly
		// what an operator holding it open should be told.
		app.NotFoundResponse(w, r)
	case errors.Is(err, sos.ErrEmptyField), errors.Is(err, sos.ErrEmptyComment):
		app.BadRequestResponse(w, r, err)
	default:
		app.ServerErrorResponse(w, r, err)
	}
}

func containsSlug(slugs []types.Slug, want types.Slug) bool {
	for _, s := range slugs {
		if s == want {
			return true
		}
	}
	return false
}
