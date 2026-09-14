package searchperson_test

import (
	"testing"

	"github.com/jrgensen/cqrs/cqrstest"
	"nathejk.dk/internal/live"
	"nathejk.dk/nathejk/table/searchperson"
)

// The live contract, asserted rather than read off a boot log.
//
// A client declares what invalidates its resource by naming entity tokens, and a token no
// consumer can produce fails *silently*: the page looks live, never errors, and simply never
// updates. So the tokens this projection contributes are pinned here, and the search view's
// `dependsOn` is expected to be this list. If a source is added or removed, this test says so
// and the frontend's dependency list is updated in the same change.
func TestAdvertisedEntityTokens(t *testing.T) {
	table, err := searchperson.New(&cqrstest.Writer{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	set := live.EntitiesFrom(table)

	want := []string{
		"crew",       // a crew signup, which is what mints a crew member
		"crewmember", // and their later registration/edits
		"friend",
		"gøgler",
		"klan",     // the klan's contact person
		"patrulje", // the patrol's contact person
		"senior",
		"spejder",
	}

	if len(set.Entities) != len(want) {
		t.Fatalf("advertised %v, want %v", set.Entities, want)
	}
	for i, token := range want {
		if set.Entities[i] != token {
			t.Errorf("token %d: got %q, want %q (full set %v)", i, set.Entities[i], token, set.Entities)
		}
	}

	// There is deliberately no `personnel` token — that is the table's name, not an entity —
	// and no `bandit`, because a bandit is a senior with an arm number rather than a
	// population. Both mistakes have been made in this codebase before.
	for _, absent := range []string{"personnel", "bandit"} {
		if set.Has(absent) {
			t.Errorf("advertising %q, which is not an entity this projection indexes", absent)
		}
	}
}

// This projection must not be the reason the advertised set becomes open-ended.
//
// `Exhaustive` is false as soon as any consumer wildcards the *entity* position, as the signup
// projection does. That flag is what lets the SPA warn in dev about a dependency nothing can
// satisfy, so searchperson names its team types explicitly instead (task 176) even though
// mirroring signup would have been fewer lines.
func TestSearchpersonAloneIsExhaustive(t *testing.T) {
	table, err := searchperson.New(&cqrstest.Writer{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !live.EntitiesFrom(table).Exhaustive {
		t.Error("searchperson wildcards an entity position, making the advertised token set incomplete")
	}
}
