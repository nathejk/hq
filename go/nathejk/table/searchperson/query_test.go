package searchperson

import (
	"strings"
	"testing"
)

// The per-kind rules, tested as the pure function they are.
//
// This is where the domain lives: which of the several statuses in the join describes *this*
// person, and which of the several names describes where they belong. Deciding it in Go rather
// than in a SQL CASE is what makes this test possible.
func TestResultChoosesTheRightStatusPerKind(t *testing.T) {
	// One fully-populated join row, reused: every case sets a kind and asserts that only the
	// relevant fields are picked out of it. A rule that read the wrong column would show up
	// as another kind's value appearing here.
	base := row{
		id:   "id-1",
		year: "2026",
		name: "Anders Andersen",

		patruljeName:   "Ørnene",
		patruljeNumber: "42",
		patruljeStatus: "STARTED",
		klanName:       "Klan Nordvest",
		klanStatus:     "PAID",
		memberStatus:   "racing",
		personnelKlan:  "Gøglerklanen",
		personnelGroup: "Hvidovre",
		personnelStat:  "PAY",
		sectionLabel:   "Køkkenet",
		crewGroup:      "Crew-gruppen",
	}

	cases := []struct {
		kind           Kind
		wantStatus     string
		wantStatusKind StatusKind
		wantTeamName   string
		wantTeamNumber string
	}{{
		// The lifecycle status wins: it is the only one that describes the person rather
		// than their team.
		kind: KindSpejder, wantStatus: "racing", wantStatusKind: StatusKindMember,
		wantTeamName: "Ørnene", wantTeamNumber: "42",
	}, {
		kind: KindSenior, wantStatus: "PAID", wantStatusKind: StatusKindTeam,
		wantTeamName: "Klan Nordvest",
	}, {
		kind: KindPatruljeContact, wantStatus: "STARTED", wantStatusKind: StatusKindTeam,
		wantTeamName: "Ørnene", wantTeamNumber: "42",
	}, {
		kind: KindKlanContact, wantStatus: "PAID", wantStatusKind: StatusKindTeam,
		wantTeamName: "Klan Nordvest",
	}, {
		// The klan is the more specific of the two personnel fields, so it wins.
		kind: KindGoegler, wantStatus: "PAY", wantStatusKind: StatusKindTeam,
		wantTeamName: "Gøglerklanen",
	}, {
		kind: KindFriend, wantStatus: "PAY", wantStatusKind: StatusKindTeam,
		wantTeamName: "Gøglerklanen",
	}, {
		// Crew have a section and no status at all — neither their events nor their table
		// carry one, so it is reported as unknown rather than invented.
		kind: KindCrew, wantStatus: "", wantStatusKind: StatusKindUnknown,
		wantTeamName: "Køkkenet",
	}}

	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			r := base
			r.kind = tc.kind
			got := r.result(PhoneRoleOwn)

			if got.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tc.wantStatus)
			}
			if got.StatusKind != tc.wantStatusKind {
				t.Errorf("StatusKind = %q, want %q", got.StatusKind, tc.wantStatusKind)
			}
			if got.TeamName != tc.wantTeamName {
				t.Errorf("TeamName = %q, want %q", got.TeamName, tc.wantTeamName)
			}
			if got.TeamNumber != tc.wantTeamNumber {
				t.Errorf("TeamNumber = %q, want %q", got.TeamNumber, tc.wantTeamNumber)
			}
		})
	}
}

// A spejder with no status row is the ordinary pre-race state, not an error. It must not be
// dressed up as `registered` — an invented status is worse than an absent one, because an
// operator would act on it.
func TestSpejderWithoutStatusRowFallsBackToTheTeam(t *testing.T) {
	r := row{kind: KindSpejder, patruljeName: "Ørnene", patruljeStatus: "PAID"}
	got := r.result(PhoneRoleOwn)
	if got.Status != "PAID" || got.StatusKind != StatusKindTeam {
		t.Errorf("got %q/%q, want PAID/team", got.Status, got.StatusKind)
	}

	// And with nothing at all to fall back on, unknown.
	bare := row{kind: KindSpejder}.result(PhoneRoleOwn)
	if bare.Status != "" || bare.StatusKind != StatusKindUnknown {
		t.Errorf("got %q/%q, want an unknown status", bare.Status, bare.StatusKind)
	}
}

// "Udmeldt" and "released" are different facts and must stay distinguishable: removed before
// the event is not the same as gone home during it. Collapsing them server-side would deny the
// UI the ability to say which, and an operator would ring a scout who left last night.
func TestRemovedIsSeparateFromStatus(t *testing.T) {
	r := row{kind: KindSpejder, deleted: true, memberStatus: "released"}
	got := r.result(PhoneRoleOwn)
	if !got.Removed {
		t.Error("Removed not reported")
	}
	if got.Status != "released" {
		t.Errorf("Status = %q, want the lifecycle status to survive alongside Removed", got.Status)
	}

	racing := row{kind: KindSpejder, deleted: true, memberStatus: "racing"}.result(PhoneRoleOwn)
	if !racing.Removed || racing.Status != "racing" {
		t.Error("the two axes are not independent")
	}
}

// Which number is shown follows which number matched. Showing the scout's own number on a hit
// that was found by their guardian's would tell the operator the wrong thing about who answers.
func TestParentRoleShowsTheParentNumber(t *testing.T) {
	r := row{kind: KindSpejder, phone: "11111111", phoneParent: "22222222"}

	own := r.result(PhoneRoleOwn)
	if own.Phone != "11111111" || own.PhoneRole != PhoneRoleOwn {
		t.Errorf("own: got %q/%q", own.Phone, own.PhoneRole)
	}

	parent := r.result(PhoneRoleParent)
	if parent.Phone != "22222222" || parent.PhoneRole != PhoneRoleParent {
		t.Errorf("parent: got %q/%q", parent.Phone, parent.PhoneRole)
	}
}

// Search must not become an export of the population. This is the first endpoint in hq that
// returns contact details for minors across the whole event, so what it *cannot* carry is worth
// asserting rather than trusting to review.
func TestResultCarriesNoSensitiveExtras(t *testing.T) {
	for _, forbidden := range []string{"address", "postalCode", "birthday", "birthDate", "note"} {
		if strings.Contains(strings.ToLower(selectResult), strings.ToLower(forbidden)) {
			t.Errorf("the search query selects %q; that belongs on the detail page, not in a result list", forbidden)
		}
	}
}

// Teams are joined on teamId alone. patrulje's own projection writes its year from
// msg.Time().Year() while search_person takes the subject's year token; they agree in practice,
// and teamId is unique, so there is no reason to depend on that agreement.
func TestTeamJoinsDoNotDependOnYearAgreement(t *testing.T) {
	for _, join := range []string{
		"LEFT JOIN patrulje p ON p.teamId = sp.teamId",
		"LEFT JOIN klan k ON k.teamId = sp.teamId",
	} {
		if !strings.Contains(selectResult, join) {
			t.Errorf("expected %q", join)
		}
	}
	if strings.Contains(selectResult, "p.year = sp.year") || strings.Contains(selectResult, "k.year = sp.year") {
		t.Error("a team join compares years, which the two projections derive differently")
	}
}

// Every join must be a LEFT JOIN. An inner join anywhere would silently drop people: a scout
// with no status row, a gøgler with no personnel row, a crew member never assigned a section.
// The failure would look like "this person is not in the system".
func TestEveryJoinIsOuter(t *testing.T) {
	if n := strings.Count(selectResult, "JOIN"); n != strings.Count(selectResult, "LEFT JOIN") {
		t.Errorf("%d JOINs but only %d are LEFT; an inner join drops people",
			n, strings.Count(selectResult, "LEFT JOIN"))
	}
}
