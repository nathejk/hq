package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/tables/vehicle"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
)

// showOrganisationHandler returns the sections for the current year plus the
// crew members and vehicles in that year, so the frontend has everything needed
// to render the org tree and the "unassigned" side panel. It also returns the list of
// prior years that have sections so the UI can offer a "copy from year" flow
// when the current year is empty.
func (app *application) showOrganisationHandler(w http.ResponseWriter, r *http.Request) {
	year := app.YearSlug(r)

	sections, err := app.models.Section.GetAll(r.Context(), section.Filter{YearSlug: year})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	members, err := app.models.CrewMember.GetAll(r.Context(), crewmember.Filter{YearSlug: year})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	// Vehicles hang off the same sections as crew members, so they travel with
	// the organisation payload rather than through an endpoint of their own —
	// one round trip is what lets the tree render both in a single pass.
	vehicles, err := app.models.Vehicle.GetAll(r.Context(), vehicle.Filter{YearSlug: year})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	availableYears, err := app.models.Section.ListYearsWithSections(r.Context())
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	// Filter out the current year from the copy-source list.
	otherYears := make([]types.YearSlug, 0, len(availableYears))
	for _, y := range availableYears {
		if y != year {
			otherYears = append(otherYears, y)
		}
	}

	// Which sections may be assigned an SOS case (PRD 001). Returned as a list of
	// slugs alongside the sections rather than merged into each section object: the
	// section belongs to shared-go and knows nothing about the nødtelefon, and
	// keeping the two apart here is what keeps that true.
	assignable, err := app.models.Sos.AssignableSections(r.Context(), year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	// Which sections are dispatch units (PRD 009): a subsection of logistics holding a
	// vehicle, a driver and possibly a co-driver, that tours may be assigned to. Beside
	// the sections for the same reason as the sos flag above.
	dispatchable, err := app.models.Dispatch.DispatchableSections(r.Context(), year)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	// Which crew members came through the public signup form, as ready-made links
	// to the page they filled in.
	//
	// A map keyed by userId rather than a field on each member: CrewMember belongs
	// to shared-go, and a signup URL is this screen's concern. Built here rather
	// than in the SPA for two reasons — the host is configuration (BASEURL, which
	// differs in dev and stage), and a crew member registered by an HQ operator has
	// no signup at all, so a link assembled from the id alone would send the
	// operator to a page that does not exist.
	signups, err := app.models.Signup.TeamIDsByType(r.Context(), year, types.TeamTypeCrew)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	signupUrls := map[types.UserID]string{}
	for _, m := range members {
		// A crew signup mints the crew member with userId == teamId, which is what
		// makes this lookup possible at all.
		if signups[types.TeamID(m.UserID)] {
			signupUrls[m.UserID] = fmt.Sprintf("%s/crew/%s", strings.TrimSuffix(app.config.baseurl, "/"), m.UserID)
		}
	}

	envelope := jsonapi.Envelope{
		"year":                  year,
		"sections":              sections,
		"crewMembers":           members,
		"vehicles":              vehicles,
		"availableYearsForCopy": otherYears,
		"sosAssignableSections": assignable,
		"dispatchableSections":  dispatchable,
		"crewSignupUrls":        signupUrls,
		// The canonical korps list, so the edit form offers the same eight options
		// the signup form does and stores the same slugs. Sent from here rather than
		// duplicated in the SPA: the update handler casts whatever it is given to a
		// CorpsSlug, so a free-text field would happily invent korps nobody can
		// filter on.
		"corpsOptions": types.CorpsSlugs.AsObjects(),
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// createSectionHandler adds a new section (root or child).
func (app *application) createSectionHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Slug              string `json:"slug"`
		Label             string `json:"label"`
		ParentSectionSlug string `json:"parentSectionSlug"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	input.Label = strings.TrimSpace(input.Label)
	if input.Label == "" {
		app.BadRequestResponse(w, r, errFromString("label is required"))
		return
	}
	slug := types.Slug(input.Slug)
	if slug == "" {
		slug = slugify(input.Label)
	}
	if !slug.Valid() {
		app.BadRequestResponse(w, r, errFromString("invalid slug; use lowercase letters, digits and single hyphens"))
		return
	}

	year := app.YearSlug(r)
	err := app.commands.Section.Add(r.Context(), year, slug, types.Slug(input.ParentSectionSlug), input.Label)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"slug":              slug,
		"label":             input.Label,
		"parentSectionSlug": input.ParentSectionSlug,
	}
	if err := app.WriteJSON(w, http.StatusCreated, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// moveSectionHandler reparents an existing section.
func (app *application) moveSectionHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ParentSectionSlug string `json:"parentSectionSlug"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	year := app.YearSlug(r)
	slug := types.Slug(app.ReadNamedParam(r, "slug"))
	if !slug.Valid() {
		app.BadRequestResponse(w, r, errFromString("invalid section slug"))
		return
	}
	newParent := types.Slug(input.ParentSectionSlug)
	if newParent != "" && !newParent.Valid() {
		app.BadRequestResponse(w, r, errFromString("invalid parent slug"))
		return
	}
	if err := app.commands.Section.Move(r.Context(), year, slug, newParent); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"slug":              slug,
		"parentSectionSlug": newParent,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// sortSectionsHandler applies a new sibling order under a single parent.
func (app *application) sortSectionsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ParentSectionSlug string   `json:"parentSectionSlug"`
		SortedSlugs       []string `json:"sortedSlugs"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if len(input.SortedSlugs) == 0 {
		app.BadRequestResponse(w, r, errFromString("sortedSlugs must contain at least one slug"))
		return
	}
	sorted := make([]types.Slug, len(input.SortedSlugs))
	for i, s := range input.SortedSlugs {
		sorted[i] = types.Slug(s)
	}
	year := app.YearSlug(r)
	if err := app.commands.Section.Sort(r.Context(), year, types.Slug(input.ParentSectionSlug), sorted); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"sorted": "ok"}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updateSectionHandler renames a section (keeps slug and parent). Because
// shared-go does not yet define NathejkSectionRenamed / NathejkSectionMoved,
// this endpoint currently only supports label changes; the underlying
// command reuses NathejkSectionAdded as an idempotent upsert.
func (app *application) updateSectionHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Label string `json:"label"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	input.Label = strings.TrimSpace(input.Label)
	if input.Label == "" {
		app.BadRequestResponse(w, r, errFromString("label is required"))
		return
	}
	year := app.YearSlug(r)
	slug := types.Slug(app.ReadNamedParam(r, "slug"))
	if !slug.Valid() {
		app.BadRequestResponse(w, r, errFromString("invalid section slug"))
		return
	}
	if err := app.commands.Section.Rename(r.Context(), year, slug, input.Label); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"slug":  slug,
		"label": input.Label,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// deleteSectionHandler deletes an empty section (no child sections, no
// assigned crew members).
func (app *application) deleteSectionHandler(w http.ResponseWriter, r *http.Request) {
	year := app.YearSlug(r)
	slug := types.Slug(app.ReadNamedParam(r, "slug"))
	if !slug.Valid() {
		app.BadRequestResponse(w, r, errFromString("invalid section slug"))
		return
	}
	// Refuse when crew members are still assigned. The section commander only
	// knows about child sections; this check adds the crew-member half.
	assigned, err := app.models.CrewMember.GetAll(r.Context(), crewmember.Filter{YearSlug: year, SectionSlug: slug})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	if len(assigned) > 0 {
		app.BadRequestResponse(w, r, errFromString("cannot delete section: crew members are still assigned"))
		return
	}
	// Same for vehicles: deleting the section would leave them pointing at a slug
	// nothing renders, so they would drop out of the tree without turning up in the
	// unassigned panel either.
	parked, err := app.models.Vehicle.GetAll(r.Context(), vehicle.Filter{YearSlug: year, SectionSlug: slug})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}
	if len(parked) > 0 {
		app.BadRequestResponse(w, r, errFromString("cannot delete section: vehicles are still assigned"))
		return
	}

	if err := app.commands.Section.Delete(r.Context(), year, slug); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": "ok"}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// copySectionsFromYearHandler republishes SectionAdded events from a source
// year into the current year. Refuses if the current year already has any
// sections.
func (app *application) copySectionsFromYearHandler(w http.ResponseWriter, r *http.Request) {
	dest := app.YearSlug(r)
	source := types.YearSlug(app.ReadNamedParam(r, "sourceYear"))
	if source == "" {
		app.BadRequestResponse(w, r, errFromString("sourceYear is required"))
		return
	}
	n, err := app.commands.Section.CopyFromYear(r.Context(), source, dest)
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"copied":     n,
		"sourceYear": source,
		"destYear":   dest,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// registerCrewMemberHandler creates a new crew member.
func (app *application) registerCrewMemberHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
		Email string `json:"email"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		app.BadRequestResponse(w, r, errFromString("name is required"))
		return
	}
	year := app.YearSlug(r)
	userID, err := app.commands.CrewMember.Register(r.Context(), year, input.Name, types.PhoneNumber(input.Phone), types.EmailAddress(input.Email))
	if err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"userId": userID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// updateCrewMemberHandler edits a crew member's details.
//
// The fields are optional pointers, but the underlying event is a full replace
// rather than a delta, so anything the request leaves out is filled in from the
// current read model. Without that, saving the name from a small form would
// silently blank out medlemsnummer, gruppe, korps and kost.
func (app *application) updateCrewMemberHandler(w http.ResponseWriter, r *http.Request) {
	userID := types.UserID(app.ReadNamedParam(r, "userId"))
	if userID == "" {
		app.BadRequestResponse(w, r, errFromString("userId is required"))
		return
	}
	var input struct {
		Name     *string `json:"name"`
		Phone    *string `json:"phone"`
		Email    *string `json:"email"`
		MedlemNr *string `json:"medlemnr"`
		Group    *string `json:"group"`
		Corps    *string `json:"corps"`
		Diet     *string `json:"diet"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	current, err := app.models.CrewMember.GetByID(r.Context(), userID)
	if err != nil {
		app.NotFoundResponse(w, r)
		return
	}
	fields := crewmember.UpdateFields{
		Name:     current.Name,
		Phone:    current.Phone,
		Email:    current.Email,
		MedlemNr: current.MedlemNr,
		Group:    current.Group,
		Corps:    current.Corps,
		Diet:     current.Diet,
	}
	// Additionals is stored as a JSON document but carried as a map, so it has to
	// be decoded to be handed back unchanged.
	if current.Additionals != "" {
		var additionals map[string]any
		if err := json.Unmarshal([]byte(current.Additionals), &additionals); err == nil {
			fields.Additionals = additionals
		}
	}
	if input.Name != nil {
		fields.Name = strings.TrimSpace(*input.Name)
	}
	if fields.Name == "" {
		app.BadRequestResponse(w, r, errFromString("name is required"))
		return
	}
	if input.Phone != nil {
		fields.Phone = types.PhoneNumber(strings.TrimSpace(*input.Phone))
	}
	if input.Email != nil {
		fields.Email = types.EmailAddress(strings.TrimSpace(*input.Email))
	}
	if input.MedlemNr != nil {
		fields.MedlemNr = strings.TrimSpace(*input.MedlemNr)
	}
	if input.Group != nil {
		fields.Group = strings.TrimSpace(*input.Group)
	}
	if input.Corps != nil {
		fields.Corps = types.CorpsSlug(strings.TrimSpace(*input.Corps))
	}
	if input.Diet != nil {
		fields.Diet = strings.TrimSpace(*input.Diet)
	}
	if err := app.commands.CrewMember.Update(r.Context(), app.YearSlug(r), userID, fields); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"userId": userID}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// deleteCrewMemberHandler withdraws a crew member (soft delete in the read
// model).
func (app *application) deleteCrewMemberHandler(w http.ResponseWriter, r *http.Request) {
	userID := types.UserID(app.ReadNamedParam(r, "userId"))
	if userID == "" {
		app.BadRequestResponse(w, r, errFromString("userId is required"))
		return
	}
	if err := app.commands.CrewMember.Delete(r.Context(), app.YearSlug(r), userID); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": "ok"}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// assignCrewMemberSectionHandler assigns (or unassigns, when sectionSlug is
// empty) a crew member to a section. Assignment silently unassigns from the
// previous section.
func (app *application) assignCrewMemberSectionHandler(w http.ResponseWriter, r *http.Request) {
	userID := types.UserID(app.ReadNamedParam(r, "userId"))
	if userID == "" {
		app.BadRequestResponse(w, r, errFromString("userId is required"))
		return
	}
	var input struct {
		SectionSlug string `json:"sectionSlug"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	year := app.YearSlug(r)
	sectionSlug := types.Slug(input.SectionSlug)
	if sectionSlug != "" {
		if _, err := app.models.Section.GetBySlug(r.Context(), year, sectionSlug); err != nil {
			app.BadRequestResponse(w, r, errFromString("section not found in current year"))
			return
		}
	}
	if err := app.commands.CrewMember.AssignSection(r.Context(), year, userID, sectionSlug); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"userId":      userID,
		"sectionSlug": sectionSlug,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// setSectionSosAssignableHandler toggles whether a section may be assigned SOS
// cases (PRD 001 §6).
//
// The flag is owned by the SOS domain rather than by the section itself: "can be
// assigned nødråb" is a fact about the nødtelefon, and a section does not become a
// different thing because the emergency phone can route to it. The route lives
// under /api/section/ anyway, because that is the screen an operator sets it from.
func (app *application) setSectionSosAssignableHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Assignable bool `json:"assignable"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	year := app.YearSlug(r)
	slug := types.Slug(app.ReadNamedParam(r, "slug"))
	if !slug.Valid() {
		app.BadRequestResponse(w, r, errFromString("invalid section slug"))
		return
	}
	// The section must exist for the year, or a typo would create an assignable
	// entry for a section nobody can see — invisible in the UI and impossible to
	// turn off from it.
	if _, err := app.models.Section.GetBySlug(r.Context(), year, slug); err != nil {
		app.NotFoundResponse(w, r)
		return
	}
	if err := app.commands.Sos.SetSectionAssignable(r.Context(), app.actor(r), year, slug, input.Assignable); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"slug":       slug,
		"assignable": input.Assignable,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// setSectionDispatchableHandler toggles whether a section is a dispatch unit (PRD 009 §6).
//
// PUT /api/section/:slug/dispatchable
//
//	request:  {"dispatchable": true|false}
//	response: 200 {"slug": "bil-2", "dispatchable": true}
//	          400 invalid slug or body, 404 no such section for the year
//
// Owned by the dispatch domain rather than by the section itself: "holds a car and can be
// sent out" is a fact about kørsel, and a section does not become a different thing because
// logistics can dispatch it. The route lives under /api/section/ anyway, because the
// Organisation page is the screen an operator sets it from — exactly as the sos flag does.
func (app *application) setSectionDispatchableHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Dispatchable bool `json:"dispatchable"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	year := app.YearSlug(r)
	slug := types.Slug(app.ReadNamedParam(r, "slug"))
	if !slug.Valid() {
		app.BadRequestResponse(w, r, errFromString("invalid section slug"))
		return
	}
	// The section must exist for the year, or a typo would mark a section nobody can see
	// as a dispatch unit — invisible in the UI and impossible to turn off from it.
	if _, err := app.models.Section.GetBySlug(r.Context(), year, slug); err != nil {
		app.NotFoundResponse(w, r)
		return
	}
	if err := app.commands.Dispatch.SetSectionDispatchable(r.Context(), app.dispatchActor(r), year, slug, input.Dispatchable); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	envelope := jsonapi.Envelope{
		"slug":         slug,
		"dispatchable": input.Dispatchable,
	}
	if err := app.WriteJSON(w, http.StatusOK, envelope, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// -- helpers --------------------------------------------------------------

// errFromString wraps a message in an error without pulling in errors.New at
// every call site.
type stringError string

func (s stringError) Error() string { return string(s) }
func errFromString(s string) error  { return stringError(s) }

// slugify produces a types.Slug-compatible identifier from a human label. It
// lowercases, replaces runs of non [a-z0-9] with a single hyphen, and trims
// leading/trailing hyphens. If nothing valid remains it returns "".
var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(label string) types.Slug {
	s := strings.ToLower(strings.TrimSpace(label))
	// Rough transliteration for the common Danish characters so labels like
	// "Gøgl & Badut" produce "goegl-badut" rather than "-".
	replacer := strings.NewReplacer(
		"æ", "ae",
		"ø", "oe",
		"å", "aa",
	)
	s = replacer.Replace(s)
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return types.Slug(s)
}
