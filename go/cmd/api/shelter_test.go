package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/internal/data"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/shelter"
	"nathejk.dk/nathejk/table/sos"
	"nathejk.dk/nathejk/table/spejdernote"
	"nathejk.dk/nathejk/table/spejderstatus"
)

// The Hønsegården payload is assembled from five projections, and what is worth testing is
// the assembly: which scouts land in which section, whether a missing roster row still
// produces a usable row, and whether the placering and case link reach the right member.
// Fakes rather than a database, which is what the other tests in this package do.

// --- fakes ---

type fakeShelterStatus struct {
	rows []spejderstatus.SpejderStatus
	care *spejderstatus.Care
}

func (f *fakeShelterStatus) GetByStatuses(_ context.Context, _ types.YearSlug, statuses []types.MemberStatus) ([]spejderstatus.SpejderStatus, error) {
	wanted := map[types.MemberStatus]bool{}
	for _, s := range statuses {
		wanted[s] = true
	}
	out := []spejderstatus.SpejderStatus{}
	for _, row := range f.rows {
		if wanted[row.Status] {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeShelterStatus) InOurCare(context.Context, types.YearSlug) (*spejderstatus.Care, error) {
	if f.care == nil {
		return &spejderstatus.Care{}, nil
	}
	return f.care, nil
}

// Unused by the shelter handlers, present because spejderstatus.Queries is one interface per table
// (PRD 011 added TeamMemberships for the patrol track map).
func (f *fakeShelterStatus) TeamMemberships(context.Context, types.YearSlug, types.TeamID) ([]spejderstatus.Membership, error) {
	return nil, nil
}

func (f *fakeShelterStatus) GetByMemberID(context.Context, types.YearSlug, types.MemberID) (*spejderstatus.SpejderStatus, error) {
	return nil, spejderstatus.ErrRecordNotFound
}

func (f *fakeShelterStatus) GetByMemberIDs(context.Context, types.YearSlug, []types.MemberID) (map[types.MemberID]spejderstatus.SpejderStatus, error) {
	return map[types.MemberID]spejderstatus.SpejderStatus{}, nil
}

func (f *fakeShelterStatus) GetByTeam(context.Context, spejderstatus.Filter) ([]spejderstatus.SpejderStatus, error) {
	return nil, nil
}

func (f *fakeShelterStatus) GetHistory(context.Context, types.YearSlug, types.MemberID) ([]spejderstatus.StatusEvent, error) {
	return nil, nil
}

type fakeShelterPlacements struct {
	rows  map[types.MemberID]shelter.Placement
	zones []shelter.Zone
}

func (f *fakeShelterPlacements) GetByMemberIDs(_ context.Context, _ types.YearSlug, ids []types.MemberID) (map[types.MemberID]shelter.Placement, error) {
	out := map[types.MemberID]shelter.Placement{}
	for _, id := range ids {
		if p, ok := f.rows[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

func (f *fakeShelterPlacements) DistinctPlacements(context.Context, types.YearSlug) ([]shelter.Zone, error) {
	return f.zones, nil
}

type fakeShelterMembers struct {
	people []*data.Spejder
}

func (f *fakeShelterMembers) GetSpejdere(data.Filters) ([]*data.Spejder, data.Metadata, error) {
	return f.people, data.Metadata{}, nil
}

func (f *fakeShelterMembers) GetSeniore(data.Filters) ([]*data.Senior, data.Metadata, error) {
	return nil, data.Metadata{}, nil
}

type fakeShelterPatruljer struct {
	teams []patrulje.Patrulje
}

func (f *fakeShelterPatruljer) GetAll(context.Context, patrulje.Filter) ([]patrulje.Patrulje, error) {
	return f.teams, nil
}
func (f *fakeShelterPatruljer) GetByID(context.Context, types.TeamID) (*patrulje.Patrulje, error) {
	return nil, nil
}
func (f *fakeShelterPatruljer) GetStartedTeamIDs(context.Context, patrulje.Filter) ([]types.TeamID, error) {
	return nil, nil
}
func (f *fakeShelterPatruljer) GetStartedTeams(context.Context, patrulje.Filter) ([]patrulje.StartedTeam, error) {
	return nil, nil
}
func (f *fakeShelterPatruljer) GetDiscontinuedTeamIDs(context.Context, patrulje.Filter) ([]types.TeamID, error) {
	return nil, nil
}
func (f *fakeShelterPatruljer) AssignedNumbers(context.Context, types.YearSlug) (map[types.TeamID]string, error) {
	return nil, nil
}

type fakeShelterSos struct {
	byTeam map[types.TeamID][]*sos.Sos
	calls  int
}

func (f *fakeShelterSos) GetByTeam(_ context.Context, _ types.YearSlug, teamID types.TeamID) ([]*sos.Sos, error) {
	f.calls++
	return f.byTeam[teamID], nil
}
func (f *fakeShelterSos) GetAll(context.Context, sos.Filter) ([]*sos.Sos, error) { return nil, nil }
func (f *fakeShelterSos) GetByID(context.Context, types.SosID) (*sos.Sos, error) {
	return nil, nil
}
func (f *fakeShelterSos) AssignableSections(context.Context, types.YearSlug) ([]types.Slug, error) {
	return nil, nil
}

type fakeShelterNotes struct {
	rows map[types.MemberID]spejdernote.Summary
}

func (f *fakeShelterNotes) SummaryByMembers(_ context.Context, _ types.YearSlug, ids []types.MemberID) (map[types.MemberID]spejdernote.Summary, error) {
	out := map[types.MemberID]spejdernote.Summary{}
	for _, id := range ids {
		if s, ok := f.rows[id]; ok {
			out[id] = s
		}
	}
	return out, nil
}

func (f *fakeShelterNotes) GetByMember(context.Context, types.YearSlug, types.MemberID) ([]spejdernote.Note, error) {
	return nil, nil
}

func (f *fakeShelterNotes) GetByID(context.Context, types.YearSlug, spejdernote.NoteID) (*spejdernote.Note, error) {
	return nil, nil
}

// --- fixtures ---

type shelterResponse struct {
	Sections []struct {
		Slug    string `json:"slug"`
		Label   string `json:"label"`
		Members []struct {
			MemberID         types.MemberID     `json:"memberId"`
			Name             string             `json:"name"`
			Status           types.MemberStatus `json:"status"`
			Phone            string             `json:"phone"`
			PhoneParent      string             `json:"phoneParent"`
			Placement        string             `json:"placement"`
			PlacedAt         *time.Time         `json:"placedAt"`
			SosID            types.SosID        `json:"sosId"`
			TeamDiscontinued bool               `json:"teamDiscontinued"`
			NoteCount        int                `json:"noteCount"`
			LatestNote       string             `json:"latestNote"`
			LatestNoteAt     *time.Time         `json:"latestNoteAt"`
			Team             *struct {
				TeamID     types.TeamID `json:"teamId"`
				TeamNumber string       `json:"teamNumber"`
				Name       string       `json:"name"`
			} `json:"team"`
			StartTeam *struct {
				TeamID types.TeamID `json:"teamId"`
			} `json:"startTeam"`
		} `json:"members"`
	} `json:"sections"`
	Counts         map[string]int `json:"counts"`
	MemberStatuses []SlugLabel    `json:"memberStatuses"`
	Placements     []shelter.Zone `json:"placements"`
	Care           struct {
		Total int `json:"total"`
	} `json:"care"`
}

func status(id types.MemberID, team types.TeamID, s types.MemberStatus) spejderstatus.SpejderStatus {
	return spejderstatus.SpejderStatus{
		MemberID:      id,
		YearSlug:      "2026",
		InitialTeamID: team,
		CurrentTeamID: team,
		Status:        s,
		UpdatedAt:     time.Date(2026, 9, 26, 0, 42, 0, 0, time.UTC),
	}
}

func shelterApp(t *testing.T, st *fakeShelterStatus, pl *fakeShelterPlacements, me *fakeShelterMembers, pa *fakeShelterPatruljer, so *fakeShelterSos) *application {
	t.Helper()
	return &application{
		models: data.Models{
			SpejderStatus: st,
			Shelter:       pl,
			Members:       me,
			Patrulje:      pa,
			Sos:           so,
			// Notes default to none. The tests that care supply their own via shelterAppWithNotes.
			Note: &fakeShelterNotes{},
		},
	}
}

func shelterAppWithNotes(t *testing.T, st *fakeShelterStatus, notes *fakeShelterNotes) *application {
	t.Helper()
	app := shelterApp(t, st, &fakeShelterPlacements{}, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{})
	app.models.Note = notes
	return app
}

// The handler is invoked directly rather than through app.routes().
//
// Not a shortcut: routes() installs app.Metrics, which calls expvar.NewInt, and expvar
// panics on a duplicate name — so routes() can be built at most once per process and
// stream_test.go already builds it. That one call is also what covers this endpoint's
// *registration*, including the httprouter constraint that a static `/api/shelter` must not
// collide with the sibling wildcard `/api/member/:memberId`: a conflict there panics at
// construction, so stream_test would fail loudly.
func getShelter(t *testing.T, app *application) shelterResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/shelter", nil)
	req.Header.Set("X-YearSlug", "2026")
	rec := httptest.NewRecorder()

	app.showShelterHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var got shelterResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return got
}

// --- tests ---

// Every section carries a list, even when it is empty.
//
// Found by running the endpoint rather than by reading it: a nil slice marshals to `null`, so
// an empty section arrived as `"members": null` and the client's `section.members.length`
// would throw on the quiet night this screen is built for. "None" is a list of nothing.
func TestEverySectionCarriesAListEvenWhenEmpty(t *testing.T) {
	app := shelterApp(t, &fakeShelterStatus{}, &fakeShelterPlacements{}, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{})

	req := httptest.NewRequest(http.MethodGet, "/api/shelter", nil)
	req.Header.Set("X-YearSlug", "2026")
	rec := httptest.NewRecorder()
	app.showShelterHandler(rec, req)

	// Asserted on the raw JSON, because decoding into a []T turns both `null` and `[]` into a
	// nil slice and would pass either way — which is exactly how this got shipped to a browser
	// in the first place.
	if strings.Contains(rec.Body.String(), `"members":null`) {
		t.Errorf("a section serialised its members as null: %s", rec.Body.String())
	}
}

// A patrol with nobody left racing will not cross a finish line, so "genforenet med patruljen"
// is not available for its scouts. The fact is sent rather than a permission, so the screen can
// say why the button is disabled — and so the server is not deciding what the buttons are.
func TestTeamDiscontinuedIsReportedFromTheStrength(t *testing.T) {
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-alive", "team-1", types.MemberStatusSheltered),
		status("m-gone", "team-2", types.MemberStatusSheltered),
	}}
	pa := &fakeShelterPatruljer{teams: []patrulje.Patrulje{
		{TeamID: "team-1", Name: "Ulvene", ActiveMemberCount: 2},
		{TeamID: "team-2", Name: "Ravnene", ActiveMemberCount: 0},
	}}
	got := getShelter(t, shelterApp(t, st, &fakeShelterPlacements{}, &fakeShelterMembers{}, pa, &fakeShelterSos{}))

	for _, section := range got.Sections {
		for _, m := range section.Members {
			want := m.MemberID == "m-gone"
			if m.TeamDiscontinued != want {
				t.Errorf("%s: teamDiscontinued = %v, want %v", m.MemberID, m.TeamDiscontinued, want)
			}
		}
	}
}

// Notes on the rows (PRD 008): the count and a snippet, so the one scout with instructions is
// visible while scanning rather than hidden behind forty clicks.
func TestShelterRowsCarryTheNoteSummary(t *testing.T) {
	written := time.Date(2026, 9, 26, 1, 20, 0, 0, time.UTC)
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-noted", "team-1", types.MemberStatusSheltered),
		status("m-quiet", "team-1", types.MemberStatusSheltered),
	}}
	notes := &fakeShelterNotes{rows: map[types.MemberID]spejdernote.Summary{
		"m-noted": {Count: 3, LatestNote: "Ringet til mor. Hun henter kl. 06.", LatestAt: written},
	}}
	got := getShelter(t, shelterAppWithNotes(t, st, notes))

	for _, section := range got.Sections {
		for _, m := range section.Members {
			switch m.MemberID {
			case "m-noted":
				if m.NoteCount != 3 {
					t.Errorf("noteCount = %d, want 3", m.NoteCount)
				}
				if !strings.Contains(m.LatestNote, "Ringet til mor") {
					t.Errorf("latestNote = %q", m.LatestNote)
				}
				if m.LatestNoteAt == nil {
					t.Error("latestNoteAt is nil for a scout with notes")
				}
			case "m-quiet":
				// The one that would be a real incident: a note about one child appearing on
				// another child's row.
				if m.NoteCount != 0 || m.LatestNote != "" || m.LatestNoteAt != nil {
					t.Errorf("a scout with no notes inherited some: %+v", m)
				}
			}
		}
	}
}

// A note may be 2000 characters; a row shows a line and a half. Truncating on the server keeps a
// forty-scout screen from carrying 80KB of prose to render forty snippets.
func TestLatestNoteIsTruncatedServerSide(t *testing.T) {
	long := strings.Repeat("a", 500)
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-1", "team-1", types.MemberStatusSheltered),
	}}
	notes := &fakeShelterNotes{rows: map[types.MemberID]spejdernote.Summary{
		"m-1": {Count: 1, LatestNote: long},
	}}
	got := getShelter(t, shelterAppWithNotes(t, st, notes))

	for _, section := range got.Sections {
		for _, m := range section.Members {
			if len([]rune(m.LatestNote)) > noteSnippetLength+1 { // +1 for the ellipsis
				t.Errorf("snippet is %d runes, want at most %d", len([]rune(m.LatestNote)), noteSnippetLength+1)
			}
			if !strings.HasSuffix(m.LatestNote, "…") {
				t.Errorf("a truncated snippet should say so: %q", m.LatestNote)
			}
		}
	}
}

// Cutting UTF-8 by bytes produces a replacement glyph, and a snippet ending in ï¿½ reads as corrupted
// data rather than as an abbreviation. Danish makes this likely rather than theoretical: æ, ø and å
// are two bytes each.
func TestTruncateRunesDoesNotSplitCharacters(t *testing.T) {
	danish := strings.Repeat("æ", 200)

	got := truncateRunes(danish, 10)

	if got != strings.Repeat("æ", 10)+"…" {
		t.Errorf("truncateRunes = %q", got)
	}
	if strings.Contains(got, "\uFFFD") {
		t.Error("truncation split a character")
	}
}

func TestTruncateRunesLeavesShortTextAlone(t *testing.T) {
	if got := truncateRunes("kort", 10); got != "kort" {
		t.Errorf("truncateRunes = %q, want the text unchanged and no ellipsis", got)
	}
}

// The sections, in the crew's working order. "På vej" first because those scouts are the ones
// something is about to happen to.
func TestShelterSectionsAreOrderedForTheCrew(t *testing.T) {
	app := shelterApp(t,
		&fakeShelterStatus{},
		&fakeShelterPlacements{},
		&fakeShelterMembers{},
		&fakeShelterPatruljer{},
		&fakeShelterSos{},
	)
	got := getShelter(t, app)

	want := []string{"onway", "sheltered", "closed"}
	if len(got.Sections) != len(want) {
		t.Fatalf("expected %d sections, got %d", len(want), len(got.Sections))
	}
	for i, slug := range want {
		if got.Sections[i].Slug != slug {
			t.Errorf("section %d = %q, want %q", i, got.Sections[i].Slug, slug)
		}
		if got.Sections[i].Label == "" {
			t.Errorf("section %q has no label; the SPA would have to invent Danish copy", slug)
		}
	}
}

// Who lands where, and — the load-bearing half — who does not appear at all. A scout still
// racing or already finished is the route's business, and one who never started is at home in
// bed; putting either on this screen would bury the handful of children who need something
// doing.
func TestShelterIncludesOnlyStartedAndNotActive(t *testing.T) {
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-transit", "team-1", types.MemberStatusTransit),
		status("m-sheltered", "team-1", types.MemberStatusSheltered),
		status("m-waiting", "team-2", types.MemberStatusWaiting),
		status("m-reunited", "team-2", types.MemberStatusReunited),
		status("m-released", "team-2", types.MemberStatusReleased),
		status("m-racing", "team-3", types.MemberStatusRacing),
		status("m-finished", "team-3", types.MemberStatusFinished),
		status("m-registered", "team-3", types.MemberStatusRegistered),
		status("m-seated", "team-3", types.MemberStatusSeated),
	}}
	app := shelterApp(t, st, &fakeShelterPlacements{}, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{})
	got := getShelter(t, app)

	wantCounts := map[string]int{"onway": 2, "sheltered": 1, "closed": 2}
	for slug, want := range wantCounts {
		if got.Counts[slug] != want {
			t.Errorf("counts[%q] = %d, want %d", slug, got.Counts[slug], want)
		}
	}
	for _, section := range got.Sections {
		for _, m := range section.Members {
			switch m.Status {
			case types.MemberStatusRacing, types.MemberStatusFinished,
				types.MemberStatusRegistered, types.MemberStatusSeated:
				t.Errorf("%s appears in section %q but does not belong on this screen", m.Status, section.Slug)
			}
		}
	}
}

// The two "not here yet" statuses share a section, and within it the arrivals come first:
// somebody getting out of a car is acted on within minutes, somebody by a road is not. The
// distinction is kept on the row — the status — rather than in the grouping.
func TestOnwaySectionPutsArrivalsBeforeTheTrailside(t *testing.T) {
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-waiting", "team-1", types.MemberStatusWaiting),
		status("m-transit", "team-1", types.MemberStatusTransit),
	}}
	got := getShelter(t, shelterApp(t, st, &fakeShelterPlacements{}, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{}))

	for _, section := range got.Sections {
		if section.Slug != "onway" {
			continue
		}
		if len(section.Members) != 2 {
			t.Fatalf("expected both scouts in one section, got %d", len(section.Members))
		}
		if section.Members[0].Status != types.MemberStatusTransit {
			t.Errorf("first row is %q, want transit — the rows that need doing belong on top", section.Members[0].Status)
		}
		if section.Members[1].Status != types.MemberStatusWaiting {
			t.Errorf("second row is %q, want waiting", section.Members[1].Status)
		}
	}
}

// Both endings share one section, because to the crew they are the same fact: this child is
// no longer ours. The distinction between them is still on the row, for the record.
func TestReunitedAndReleasedShareTheClosedSection(t *testing.T) {
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-1", "team-1", types.MemberStatusReunited),
		status("m-2", "team-1", types.MemberStatusReleased),
	}}
	app := shelterApp(t, st, &fakeShelterPlacements{}, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{})
	got := getShelter(t, app)

	var closed []types.MemberStatus
	for _, section := range got.Sections {
		if section.Slug != "closed" {
			continue
		}
		for _, m := range section.Members {
			closed = append(closed, m.Status)
		}
	}
	if len(closed) != 2 {
		t.Fatalf("expected both endings in the closed section, got %v", closed)
	}
	if closed[0] == closed[1] {
		t.Errorf("the two endings were collapsed into one status: %v", closed)
	}
}

// The row carries what the crew dials and where the scout is. The placering belongs to one
// member and must not leak onto another sharing the section.
func TestShelterRowCarriesContactAndPlacering(t *testing.T) {
	placedAt := time.Date(2026, 9, 26, 0, 51, 0, 0, time.UTC)
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-1", "team-1", types.MemberStatusSheltered),
		status("m-2", "team-1", types.MemberStatusSheltered),
	}}
	pl := &fakeShelterPlacements{rows: map[types.MemberID]shelter.Placement{
		"m-1": {MemberID: "m-1", Placement: "Telt 4", PlacedAt: &placedAt},
	}}
	me := &fakeShelterMembers{people: []*data.Spejder{
		{MemberID: "m-1", Name: "Ida", Phone: "12345678", PhoneParent: "87654321"},
		{MemberID: "m-2", Name: "Noah"},
	}}
	pa := &fakeShelterPatruljer{teams: []patrulje.Patrulje{
		{TeamID: "team-1", TeamNumber: "42", Name: "Ulvene"},
	}}
	got := getShelter(t, shelterApp(t, st, pl, me, pa, &fakeShelterSos{}))

	rows := map[types.MemberID]struct {
		name, phone, parent, placement string
		placed                         bool
		team                           string
	}{}
	for _, section := range got.Sections {
		for _, m := range section.Members {
			entry := rows[m.MemberID]
			entry.name, entry.phone, entry.parent = m.Name, m.Phone, m.PhoneParent
			entry.placement, entry.placed = m.Placement, m.PlacedAt != nil
			if m.Team != nil {
				entry.team = m.Team.Name + "/" + m.Team.TeamNumber
			}
			rows[m.MemberID] = entry
		}
	}

	if rows["m-1"].name != "Ida" || rows["m-1"].phone != "12345678" || rows["m-1"].parent != "87654321" {
		t.Errorf("m-1 lost its contact details: %+v", rows["m-1"])
	}
	if rows["m-1"].placement != "Telt 4" || !rows["m-1"].placed {
		t.Errorf("m-1 lost its placering: %+v", rows["m-1"])
	}
	if rows["m-1"].team != "Ulvene/42" {
		t.Errorf("m-1 lost its patrol: %+v", rows["m-1"])
	}
	// The one that would be a real incident: a scout who has not been placed must not inherit
	// somebody else's tent, or the crew looks for a child where another child is.
	if rows["m-2"].placement != "" || rows["m-2"].placed {
		t.Errorf("m-2 inherited a placering it does not have: %+v", rows["m-2"])
	}
}

// A scout accepted but not yet bedded down: on the screen, with an empty placering and no
// placedAt. That combination is the crew's next job, so it must survive to the client rather
// than being normalised into something that looks finished.
func TestAcceptedButUnplacedIsVisible(t *testing.T) {
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-1", "team-1", types.MemberStatusSheltered),
	}}
	pl := &fakeShelterPlacements{rows: map[types.MemberID]shelter.Placement{
		"m-1": {MemberID: "m-1", Placement: ""},
	}}
	got := getShelter(t, shelterApp(t, st, pl, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{}))

	for _, section := range got.Sections {
		if section.Slug != "sheltered" {
			continue
		}
		if len(section.Members) != 1 {
			t.Fatalf("expected the unplaced scout in the shelter section, got %d rows", len(section.Members))
		}
		if m := section.Members[0]; m.Placement != "" || m.PlacedAt != nil {
			t.Errorf("expected an empty placering and no placedAt, got %q / %v", m.Placement, m.PlacedAt)
		}
	}
}

// A member missing from the roster still gets a row.
//
// It happens when a signup row is removed after the scout started. A blank row would be worse
// than useless — the crew cannot act on a child with no name — so the id is the fallback, and
// the row stays on the screen.
func TestMemberMissingFromRosterFallsBackToID(t *testing.T) {
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-ghost", "team-1", types.MemberStatusSheltered),
	}}
	got := getShelter(t, shelterApp(t, st, &fakeShelterPlacements{}, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{}))

	for _, section := range got.Sections {
		for _, m := range section.Members {
			if m.MemberID == "m-ghost" && m.Name != "m-ghost" {
				t.Errorf("name = %q, want the id as a fallback so the row is still actionable", m.Name)
			}
		}
	}
}

// A scout moved to another patrol before dropping out: both patrols on the row, and only
// then. The ordinary row must not carry the same patrol twice or every row would render
// "startede i".
func TestStartTeamOnlyWhenItDiffers(t *testing.T) {
	moved := status("m-moved", "team-2", types.MemberStatusSheltered)
	moved.InitialTeamID = "team-1"
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		moved,
		status("m-stayed", "team-1", types.MemberStatusSheltered),
	}}
	got := getShelter(t, shelterApp(t, st, &fakeShelterPlacements{}, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{}))

	for _, section := range got.Sections {
		for _, m := range section.Members {
			switch m.MemberID {
			case "m-moved":
				if m.StartTeam == nil || m.StartTeam.TeamID != "team-1" {
					t.Errorf("a moved scout lost the patrol they started with: %+v", m.StartTeam)
				}
			case "m-stayed":
				if m.StartTeam != nil {
					t.Errorf("an unmoved scout carries a redundant startTeam: %+v", m.StartTeam)
				}
			}
		}
	}
}

// The case link, and the reason it is memoised: the lookup is per patrol, so three scouts off
// one patrol must cost one query rather than three. This is a page the crew keeps open all
// night.
func TestOpenCaseIsLinkedOncePerPatrol(t *testing.T) {
	st := &fakeShelterStatus{rows: []spejderstatus.SpejderStatus{
		status("m-1", "team-1", types.MemberStatusTransit),
		status("m-2", "team-1", types.MemberStatusTransit),
		status("m-3", "team-1", types.MemberStatusTransit),
	}}
	so := &fakeShelterSos{byTeam: map[types.TeamID][]*sos.Sos{
		"team-1": {
			{ID: "sos-closed", Status: sos.StatusClosed},
			{ID: "sos-open", Status: sos.StatusOpen},
		},
	}}
	got := getShelter(t, shelterApp(t, st, &fakeShelterPlacements{}, &fakeShelterMembers{}, &fakeShelterPatruljer{}, so))

	for _, section := range got.Sections {
		for _, m := range section.Members {
			// The closed case must not win: an operator following the link needs the call
			// somebody is working, not one that was dealt with hours ago.
			if m.SosID != "sos-open" {
				t.Errorf("%s linked to %q, want the open case", m.MemberID, m.SosID)
			}
		}
	}
	if so.calls != 1 {
		t.Errorf("the case lookup ran %d times for one patrol; it must be memoised", so.calls)
	}
}

// The screen's vocabulary comes from the server: status labels, so no second Danish label map
// can drift from the first (PRD 006 §6), and the placeringer in use, which is the entire zone
// model — the zones are not known until race start, so they are whatever the crew has typed.
func TestShelterServesLabelsAndPlaceringSuggestions(t *testing.T) {
	pl := &fakeShelterPlacements{zones: []shelter.Zone{{Placement: "Telt 4", Count: 3}}}
	got := getShelter(t, shelterApp(t, &fakeShelterStatus{}, pl, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{}))

	if len(got.MemberStatuses) == 0 {
		t.Error("no status labels; the view would need a label map of its own")
	}
	if len(got.Placements) != 1 || got.Placements[0].Placement != "Telt 4" {
		t.Errorf("placering suggestions = %+v, want the zones in use", got.Placements)
	}
}

// The in-our-care total comes from the same query the dashboard uses, not from counting the
// rows on this screen. Two independent counts of the children we are responsible for is one
// more than the night can afford.
func TestCareTotalComesFromTheCareQuery(t *testing.T) {
	st := &fakeShelterStatus{
		rows: []spejderstatus.SpejderStatus{status("m-1", "team-1", types.MemberStatusSheltered)},
		care: &spejderstatus.Care{Total: 7},
	}
	got := getShelter(t, shelterApp(t, st, &fakeShelterPlacements{}, &fakeShelterMembers{}, &fakeShelterPatruljer{}, &fakeShelterSos{}))

	if got.Care.Total != 7 {
		t.Errorf("care total = %d, want the care query's answer (7), not a count of rows", got.Care.Total)
	}
}

// The population is derived where it can be, so a fourth in-care state added to shared-go
// starts appearing here without anybody editing this screen — and nothing that belongs to the
// route can sneak in.
func TestShelterStatusesAreDerivedAndExcludeTheRoute(t *testing.T) {
	inCare, closed := shelterStatuses()

	if len(inCare) != len(spejderstatus.InOurCareStatuses()) {
		t.Errorf("the in-care half is not derived from shared-go's set: %v", inCare)
	}
	for _, s := range append(append([]types.MemberStatus{}, inCare...), closed...) {
		if !s.Valid() {
			t.Errorf("%q is not a status the lifecycle knows", s)
		}
		if s.CanFinish() {
			t.Errorf("%q can still finish, so the scout is on the route and not the shelter's business", s)
		}
		if s == types.MemberStatusFinished {
			t.Error("finished belongs to the route, not to the shelter")
		}
	}
}
