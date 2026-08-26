package main

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
	"nathejk.dk/nathejk/table/personnel"
)

// assignablePerson is one person who may be put on a post, shaped for the picker.
//
// `id` and `name` are what the frontend binds to, and are kept exactly as the
// personnel projection used to serve them so nothing downstream had to change. The
// section fields are additive, and are what let the picker group people rather than
// present one flat list of everyone in the organisation.
type assignablePerson struct {
	ID           types.UserID `json:"id"`
	Name         string       `json:"name"`
	SectionSlug  string       `json:"sectionSlug"`
	SectionLabel string       `json:"sectionLabel"`
	// Priority marks the postmandskab, who are offered first.
	Priority bool `json:"priority"`
}

// Labels for the two groups that are not an organisation section.
const (
	sectionLabelNone    = "Uden sektion"
	sectionLabelFriends = "Tilmeldte hjælpere"
)

// isPostSection reports whether a section is the one that staffs posts.
//
// Matched on the slug prefix rather than a single hardcoded slug because there are
// already two in use — "postmandskab" and "postmand" — and a rule that recognised
// only one would quietly stop prioritising anybody the day an operator picked the
// other. Prioritising is a sort order and nothing more, so the cost of matching a
// section that merely starts with "postmand" is that it sorts early, which is
// almost certainly what was wanted anyway.
func isPostSection(slug string) bool {
	return strings.HasPrefix(slug, "postmand")
}

// assignablePersonnel returns everyone who may staff a checkpoint, postmandskab first.
//
// # Why this is not just the postmandskab
//
// A post that is unstaffed an hour before the patrols arrive gets whoever is
// standing there — a gøgler, somebody from HQ, a driver between runs. Restricting
// the picker to one section does not prevent that, it only prevents *recording*
// it, and an unrecorded stand-in is exactly the person the nødtelefon needs to
// reach at 3am. So every crew member is selectable and the postmandskab are merely
// first.
//
// # Why crew members at all
//
// This used to offer the personnel projection filtered to userType "friend". That
// silently became useless: there are no friend rows for 2026 at all, so the picker
// was empty and no post could be staffed from this screen. The people who run
// posts are crew members — they are managed on the Organisation page and hang off
// its sections — which is also what makes "prioritise the postmandskab" a thing
// that can be expressed.
//
// The friends are still included rather than dropped: they are real for the years
// that have them, and this is the only list that resolves an assigned person's
// name (PostList shows "Ukendt" for an id it cannot find).
func (app *application) assignablePersonnel(ctx context.Context, year types.YearSlug) []assignablePerson {
	// Section labels, so the picker can group by something an operator recognises
	// rather than by slug. A section that has since been deleted falls back to its
	// own slug below, which is ugly but findable — better than an empty group.
	labels := map[string]string{}
	if sections, err := app.models.Section.GetAll(ctx, section.Filter{YearSlug: year}); err == nil {
		for _, s := range sections {
			labels[string(s.Slug)] = s.Label
		}
	}

	out := []assignablePerson{}
	seen := map[types.UserID]bool{}

	if crew, err := app.models.CrewMember.GetAll(ctx, crewmember.Filter{YearSlug: year}); err == nil {
		for _, c := range crew {
			slug := string(c.SectionSlug)
			label := labels[slug]
			if label == "" {
				label = slug
				if slug == "" {
					label = sectionLabelNone
				}
			}
			name := c.Name
			if name == "" {
				// Unnameable but selectable: a crew member with no name is a data
				// problem, and hiding them from the picker would hide it too.
				name = string(c.Email)
			}
			seen[c.UserID] = true
			out = append(out, assignablePerson{
				ID:           c.UserID,
				Name:         name,
				SectionSlug:  slug,
				SectionLabel: label,
				Priority:     isPostSection(slug),
			})
		}
	}

	if friends, err := app.models.Personnel.GetAll(ctx, personnel.Filter{YearSlug: year, UserTypes: []string{"friend"}}); err == nil {
		for _, p := range friends {
			// A person who is both signed up as a helper and enrolled as crew is one
			// person; the crew row wins because it carries the section.
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			out = append(out, assignablePerson{ID: p.ID, Name: p.Name, SectionLabel: sectionLabelFriends})
		}
	}

	// Postmandskab first, then sections alphabetically, then the helpers, and by name
	// within each group. Sorted here rather than in the SPA so every consumer of the
	// list gets the same order — and so "prioritised" is one decision, in one place.
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Priority != b.Priority {
			return a.Priority
		}
		// The helpers are not a section; keep them last whatever they are called.
		aFriend := a.SectionLabel == sectionLabelFriends
		bFriend := b.SectionLabel == sectionLabelFriends
		if aFriend != bFriend {
			return bFriend
		}
		if a.SectionLabel != b.SectionLabel {
			return a.SectionLabel < b.SectionLabel
		}
		return a.Name < b.Name
	})
	return out
}

func (app *application) createCheckpersonnelHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CheckpointID string     `json:"checkpointId"`
		UserID       string     `json:"userId"`
		Start        *time.Time `json:"start"`
		End          *time.Time `json:"end"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	var tr *types.TimeRange
	if input.Start != nil && input.End != nil {
		tr = &types.TimeRange{
			Start: *input.Start,
			End:   *input.End,
		}
	}

	yearSlug := app.YearSlug(r)
	checkpersonnelID, err := app.commands.Checkpersonnel.Create(r.Context(), yearSlug, types.CheckpointID(input.CheckpointID), types.UserID(input.UserID), tr)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	envelope := jsonapi.Envelope{
		"checkpersonnelId": checkpersonnelID,
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) updateCheckpersonnelHandler(w http.ResponseWriter, r *http.Request) {
	id := types.CheckpersonnelID(app.ReadNamedParam(r, "id"))

	var input struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	}
	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	tr := types.TimeRange{
		Start: input.Start,
		End:   input.End,
	}
	if err := app.commands.Checkpersonnel.SetTimeRange(r.Context(), id, tr); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"updated": "ok"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) deleteCheckpersonnelHandler(w http.ResponseWriter, r *http.Request) {
	id := app.ReadNamedParam(r, "id")
	if err := app.commands.Checkpersonnel.Delete(r.Context(), types.CheckpersonnelID(id)); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"deleted": "ok"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
