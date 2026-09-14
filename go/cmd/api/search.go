package main

import (
	"errors"
	"net/http"

	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/searchperson"
)

// errSearchUnavailable is returned when the projection is not wired.
//
// Its own error so the 500 says something useful in the log. The alternative — answering 200
// with no results — would read to an operator as "this person does not exist", which is the one
// answer this feature must never give wrongly.
var errSearchUnavailable = errors.New("search: the person index is not wired")

// Person search (PRD 014).
//
// One endpoint, read-only, so it can be called from any screen. Notably it does *not* carry the
// "modtag i Hønsegården" action that task 096 wants on its results: that write belongs to PRD
// 007, and keeping search actionless is what makes it safe to reach from everywhere.
//
// # Why /api/search/person and not /api/search
//
// So that a later search over poster, kort or dispatch tasks is a sibling rather than a
// breaking change to this one's response shape.

// searchPersonHandler finds people by phone number or by name.
//
// @Summary     Find a person by phone number or name
// @Description Searches every population HQ knows about — spejdere, seniorer, gøglere, friends, crew, and the contact persons of patruljer and klaner — and returns each match with the number that matched, whose number it is, where they belong and how they currently stand.
// @Description
// @Description **Interpretation.** A query reduced to 8 or more digits is an exact number lookup; 4–7 digits is a prefix match, for a number read back badly over the phone. A query containing letters is also matched against names, mid-string and case-insensitively (so `Koch` finds `Rakel A. Koch`). Both happen at once when a query holds letters *and* a number, which is what makes a pasted `tlf. 12 34 56 78` work. Below 4 digits or 3 letters nothing is searched and `tooShort` is true — which is deliberately distinct from an empty `results`, because "keep typing" and "ingen match" are different answers.
// @Description
// @Description **Numbers.** Both a person's own number and their guardian's are searched, since it is very often the parent who rings; `phoneRole` says which matched (`own`, `parent`, `contact`, or empty when the person has no number at all). Formats are normalised, so `+45 12 34 56 78`, `12 34 56 78` and `12345678` are one number.
// @Description
// @Description **Departed people are included, not hidden.** Somebody removed from a roster is returned with `removed` true — rendered "udmeldt" — because "ingen match" for a person the event has actually met is an answer an operator cannot distinguish from "never involved". `removed` and `status` are separate axes: udmeldt (never started) is not `released` (went home during the night). `statusKind` says which vocabulary `status` is drawn from — `member` for a scout's lifecycle status, `team` for a team's signup status, empty when unknown, which is ordinary before the race.
// @Description
// @Description **Year.** The active year only, from the X-YearSlug header. Set `includeOtherYears=true` to search every year; the server never widens the scope on its own, not even on an empty result. Cross-year rows carry their `year`.
// @Description
// @Description **Cap.** At most 50 results, and `truncated` says when that bit — narrow the query rather than assuming the list is complete.
// @Description
// @Description Deliberately carries no address, birthday or notes: this is a way to find a person, not to export the population.
// @Tags        search
// @Produce     json
// @Param       q query string true "Phone number or name"
// @Param       includeOtherYears query boolean false "Search all years rather than the active one (default false)"
// @Success     200 {object} map[string]interface{} "envelope with \"search\" holding results, truncated, tooShort, matchedPhone and matchedName"
// @Failure     500 {object} map[string]interface{}
// @Router      /api/search/person [get]
func (app *application) searchPersonHandler(w http.ResponseWriter, r *http.Request) {
	if app.models.SearchPerson == nil {
		// Configuration, not a request problem: the projection is wired in main.go, and a nil
		// here means the API is running without it. Better a 500 than an empty result set,
		// which would read as "this person does not exist".
		app.ServerErrorResponse(w, r, errSearchUnavailable)
		return
	}

	qs := r.URL.Query()

	// The flag is read as an explicit opt-in and nothing else. A malformed value ("yes", "1 ",
	// "TRUE!") leaves the scope at the active year, because the failure mode of the alternative
	// is a query silently reaching across every edition of the event — and this endpoint
	// returns minors' contact details.
	includeOtherYears := app.ReadString(qs, "includeOtherYears", "") == "true"

	results, err := app.models.SearchPerson.Search(r.Context(), searchperson.Query{
		Text:              app.ReadString(qs, "q", ""),
		Year:              string(app.YearSlug(r)),
		IncludeOtherYears: includeOtherYears,
	})
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"search": results}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
