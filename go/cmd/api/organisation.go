package main

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/crewmember"
	"nathejk.dk/nathejk/table/section"
)

// showOrganisationHandler returns the sections for the current year plus the
// crew members in that year, so the frontend has everything needed to render
// the org tree and the "unassigned" side panel. It also returns the list of
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

	envelope := jsonapi.Envelope{
		"year":                  year,
		"sections":              sections,
		"crewMembers":           members,
		"availableYearsForCopy": otherYears,
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
