package main

import (
	"context"
	"testing"

	"github.com/nathejk/shared-go/tables/crewmember"
	"github.com/nathejk/shared-go/tables/section"
	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/table/personnel"
)

// --- fakes ---

type fakeSectionQueries struct {
	section.Queries
	sections []section.Section
}

func (f *fakeSectionQueries) GetAll(context.Context, section.Filter) ([]section.Section, error) {
	return f.sections, nil
}

type fakeCrewQueries struct {
	crewmember.Queries
	crew []crewmember.CrewMember
}

func (f *fakeCrewQueries) GetAll(context.Context, crewmember.Filter) ([]crewmember.CrewMember, error) {
	return f.crew, nil
}

type fakePersonnelQueries struct {
	people []*personnel.Person
}

func (f *fakePersonnelQueries) GetAll(context.Context, personnel.Filter) ([]*personnel.Person, error) {
	return f.people, nil
}
func (f *fakePersonnelQueries) GetByID(context.Context, types.UserID) (*personnel.Person, error) {
	return nil, nil
}

func staffingApp(sections []section.Section, crew []crewmember.CrewMember, friends []*personnel.Person) *application {
	return &application{
		models: data.Models{
			Section:    &fakeSectionQueries{sections: sections},
			CrewMember: &fakeCrewQueries{crew: crew},
			Personnel:  &fakePersonnelQueries{people: friends},
		},
	}
}

func names(people []assignablePerson) []string {
	out := make([]string, 0, len(people))
	for _, p := range people {
		out = append(out, p.Name)
	}
	return out
}

// --- tests ---

// The point of the change: the postmandskab come first, and everybody else is still
// there. A post left unstaffed gets whoever is standing nearby, and a picker that
// refuses to record that does not prevent the arrangement — only the record of it.
func TestAssignablePersonnelPutsPostmandskabFirstWithoutExcludingAnyone(t *testing.T) {
	app := staffingApp(
		[]section.Section{
			{Slug: "postmandskab", Label: "Postmandskab"},
			{Slug: "hq", Label: "HQ"},
			{Slug: "goeglerledelse", Label: "Gøglerledelse"},
		},
		[]crewmember.CrewMember{
			{UserID: "u-hq", Name: "Bo", SectionSlug: "hq"},
			{UserID: "u-post", Name: "Alma", SectionSlug: "postmandskab"},
			{UserID: "u-goegl", Name: "Cecilie", SectionSlug: "goeglerledelse"},
		},
		nil,
	)

	got := app.assignablePersonnel(context.Background(), "2026")

	if len(got) != 3 {
		t.Fatalf("offered %d people, want all 3: %v", len(got), names(got))
	}
	if got[0].Name != "Alma" {
		t.Errorf("first offered = %q, want the postmandskab member Alma; order: %v", got[0].Name, names(got))
	}
	if !got[0].Priority {
		t.Errorf("the postmandskab member is not marked priority")
	}
	// Gøglerledelse before HQ: sections sort by label once priority is settled.
	if want := []string{"Alma", "Cecilie", "Bo"}; !equal(names(got), want) {
		t.Errorf("order = %v, want %v", names(got), want)
	}
}

// Two slugs are already in use ("postmandskab" and "postmand"), so a rule keyed to
// one exact slug would silently stop prioritising anybody the day an operator used
// the other.
func TestAssignablePersonnelRecognisesEitherPostmandSection(t *testing.T) {
	app := staffingApp(
		[]section.Section{{Slug: "postmand", Label: "postmand3"}, {Slug: "team", Label: "Team"}},
		[]crewmember.CrewMember{
			{UserID: "u-team", Name: "Bo", SectionSlug: "team"},
			{UserID: "u-post", Name: "Alma", SectionSlug: "postmand"},
		},
		nil,
	)

	got := app.assignablePersonnel(context.Background(), "2026")

	if !got[0].Priority || got[0].Name != "Alma" {
		t.Errorf("expected the postmand-section member first, got %v", names(got))
	}
}

// The signed-up helpers are real for the years that have them, and this list is also
// what resolves an assigned person's name on the post list — so they are kept, and
// kept last.
func TestAssignablePersonnelKeepsHelpersLast(t *testing.T) {
	app := staffingApp(
		[]section.Section{{Slug: "team", Label: "Team"}},
		[]crewmember.CrewMember{{UserID: "u-team", Name: "Zenia", SectionSlug: "team"}},
		[]*personnel.Person{{ID: "u-friend", Name: "Alma"}},
	)

	got := app.assignablePersonnel(context.Background(), "2026")

	if len(got) != 2 {
		t.Fatalf("offered %v, want both the crew member and the helper", names(got))
	}
	// Alphabetically Alma would win; the grouping is what must decide.
	if got[0].Name != "Zenia" || got[1].Name != "Alma" {
		t.Errorf("order = %v, want the crew member before the helper", names(got))
	}
	if got[1].SectionLabel != sectionLabelFriends {
		t.Errorf("helper group = %q, want %q", got[1].SectionLabel, sectionLabelFriends)
	}
}

// One human enrolled twice must be offered once, or an operator picks a name and
// cannot tell which of the two identical entries they chose.
func TestAssignablePersonnelDeduplicatesAcrossSources(t *testing.T) {
	app := staffingApp(
		[]section.Section{{Slug: "team", Label: "Team"}},
		[]crewmember.CrewMember{{UserID: "same", Name: "Alma", SectionSlug: "team"}},
		[]*personnel.Person{{ID: "same", Name: "Alma"}},
	)

	got := app.assignablePersonnel(context.Background(), "2026")

	if len(got) != 1 {
		t.Fatalf("offered %d entries for one person: %v", len(got), names(got))
	}
	// The crew row wins: it is the one carrying the section.
	if got[0].SectionLabel != "Team" {
		t.Errorf("kept the helper row (%q); the crew row carries the section", got[0].SectionLabel)
	}
}

// A crew member assigned to nothing is still assignable — most of the organisation
// starts out that way — and must not land in a group labelled with an empty string.
func TestAssignablePersonnelLabelsTheUnsectioned(t *testing.T) {
	app := staffingApp(nil, []crewmember.CrewMember{{UserID: "u", Name: "Alma"}}, nil)

	got := app.assignablePersonnel(context.Background(), "2026")

	if len(got) != 1 {
		t.Fatalf("offered %d, want 1", len(got))
	}
	if got[0].SectionLabel != sectionLabelNone {
		t.Errorf("label = %q, want %q", got[0].SectionLabel, sectionLabelNone)
	}
}

// A section deleted after somebody was assigned to it leaves the slug behind. Showing
// the slug is ugly; showing an empty group heading is unusable.
func TestAssignablePersonnelFallsBackToTheSlugForAnUnknownSection(t *testing.T) {
	app := staffingApp(nil, []crewmember.CrewMember{{UserID: "u", Name: "Alma", SectionSlug: "en-sektion-ingen-har-klassificeret"}}, nil)

	got := app.assignablePersonnel(context.Background(), "2026")

	if got[0].SectionLabel != "en-sektion-ingen-har-klassificeret" {
		t.Errorf("label = %q, want the slug as a fallback", got[0].SectionLabel)
	}
}

// The list is iterated by the SPA; an empty organisation is an empty array.
func TestAssignablePersonnelIsNeverNil(t *testing.T) {
	if got := staffingApp(nil, nil, nil).assignablePersonnel(context.Background(), "2026"); got == nil {
		t.Errorf("assignablePersonnel returned nil, want an empty slice")
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
