package live

import (
	"reflect"
	"testing"

	"github.com/jrgensen/cqrs"
)

// consumerOf builds a consumer declaring the given subject patterns, reusing the
// fakeConsumer from notify_test.go. Takes strings because every case here is about
// what a pattern parses to.
func consumerOf(patterns ...string) cqrs.Consumer {
	subjs := make([]cqrs.Subject, 0, len(patterns))
	for _, p := range patterns {
		subjs = append(subjs, cqrs.SubjectFromStr(p))
	}
	return &fakeConsumer{subjects: subjs}
}

// Patterns are copied verbatim from this repo's consumers, not invented, so the
// derivation is pinned against what is actually wired.
func TestEntitiesFrom(t *testing.T) {
	tests := []struct {
		name           string
		consumers      []cqrs.Consumer
		want           []string
		wantExhaustive bool
	}{
		{
			name: "entity with id, and the colon form",
			consumers: []cqrs.Consumer{
				// table/checkpoint/consumer.go
				consumerOf("NATHEJK.*.checkpoint.*.created"),
				// shared-go/tables/order/consumer.go uses the colon form
				consumerOf("NATHEJK:*.order.*.created"),
			},
			want:           []string{"checkpoint", "order"},
			wantExhaustive: true,
		},
		{
			name: "collection level pattern contributes its plural token",
			// table/checkgroup/consumer.go:25 — `checkgroups` is a legitimate token
			// that is not any table's name, so it must survive.
			consumers: []cqrs.Consumer{
				consumerOf("NATHEJK.*.checkgroups.sorted"),
			},
			want:           []string{"checkgroups"},
			wantExhaustive: true,
		},
		{
			name: "year level pattern contributes the synthesised year token",
			// table/year/consumer.go — `year` never appears in a subject; the parser
			// invents it. A client may depend on it, so it must be advertised.
			consumers: []cqrs.Consumer{
				consumerOf("NATHEJK.*.created", "NATHEJK.*.updated"),
			},
			want:           []string{"year"},
			wantExhaustive: true,
		},
		{
			name: "non-ASCII token survives byte for byte",
			// table/personnel/consumer.go — the token the SPA has to match on.
			consumers: []cqrs.Consumer{
				consumerOf("NATHEJK.*.gøgler.*.signedup"),
			},
			want:           []string{"gøgler"},
			wantExhaustive: true,
		},
		{
			name: "wildcard entity makes the set non-exhaustive without polluting it",
			// table/patruljestatus.go:40 subscribes to every entity's signedup.
			consumers: []cqrs.Consumer{
				consumerOf("NATHEJK:*.*.*.signedup",
					"NATHEJK.*.klan.*.updated"),
			},
			want:           []string{"klan"},
			wantExhaustive: false,
		},
		{
			name: "off-convention subjects contribute nothing and do not spoil the set",
			// Both appear in this repo (commented out today): they produce no signal
			// at runtime either, so they must not appear and must not be mistaken for
			// a wildcard.
			consumers: []cqrs.Consumer{
				consumerOf("monolith:nathejk_team",
					"nathejk",
					"NATHEJK.*.lok.*.updated"),
			},
			want:           []string{"lok"},
			wantExhaustive: true,
		},
		{
			name: "duplicates across consumers collapse",
			// Several projections consume checkgroup.deleted; the token appears once.
			consumers: []cqrs.Consumer{
				consumerOf("NATHEJK.*.checkgroup.*.deleted"),
				consumerOf("NATHEJK.*.checkgroup.*.deleted"),
				consumerOf("NATHEJK.*.checkgroup.*.updated"),
			},
			want:           []string{"checkgroup"},
			wantExhaustive: true,
		},
		{
			name: "a consumer that declares no subjects is harmless",
			// table/spejderstatus.go returns an empty slice today.
			consumers: []cqrs.Consumer{
				consumerOf(),
			},
			want:           []string{},
			wantExhaustive: true,
		},
		{
			name:           "no consumers at all",
			consumers:      nil,
			want:           []string{},
			wantExhaustive: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EntitiesFrom(tc.consumers...)
			if !reflect.DeepEqual(got.Entities, tc.want) {
				t.Errorf("entities = %q, want %q", got.Entities, tc.want)
			}
			if got.Exhaustive != tc.wantExhaustive {
				t.Errorf("exhaustive = %v, want %v", got.Exhaustive, tc.wantExhaustive)
			}
		})
	}
}

// The set must be stable across calls: it is compared in tests, and PRD 005's
// blue/green comparison would report a reordered set as a difference.
func TestEntitiesFromIsSorted(t *testing.T) {
	c := consumerOf("NATHEJK.*.qr.*.scanned",
		"NATHEJK.*.checkgroup.*.created",
		"NATHEJK.*.bandit.*.armNumber.assigned",
		"NATHEJK.*.senior.*.updated")

	want := []string{"bandit", "checkgroup", "qr", "senior"}
	for i := 0; i < 3; i++ {
		if got := EntitiesFrom(c).Entities; !reflect.DeepEqual(got, want) {
			t.Fatalf("call %d: entities = %q, want %q", i, got, want)
		}
	}
}

// Guards the two tokens task 037 actually got wrong, so a future rename cannot
// quietly reintroduce the confusion.
func TestEntitiesFromNamesSubjectsNotProjections(t *testing.T) {
	set := EntitiesFrom(consumerOf(
		// The scan projection consumes qr events.
		"NATHEJK.*.qr.*.scanned",
		// The personnel projection consumes gøgler/friend/bandit events.
		"NATHEJK.*.gøgler.*.signedup",
		"NATHEJK.*.friend.*.signedup",
		"NATHEJK.*.bandit.*.armNumber.assigned"))

	for _, want := range []string{"qr", "gøgler", "friend", "bandit"} {
		if !set.Has(want) {
			t.Errorf("set should contain %q; has %q", want, set.Entities)
		}
	}
	// The projections' own names are not tokens, which is the whole trap.
	for _, unwanted := range []string{"scan", "personnel"} {
		if set.Has(unwanted) {
			t.Errorf("set should not contain the projection name %q", unwanted)
		}
	}
}

func TestEntitySetHas(t *testing.T) {
	set := EntitySet{Entities: []string{"klan", "patrulje"}}
	if !set.Has("klan") {
		t.Error("Has(klan) = false")
	}
	if set.Has("klaner") {
		t.Error("Has(klaner) = true; must not match by prefix or plural")
	}
	if (EntitySet{}).Has("klan") {
		t.Error("empty set claims to have klan")
	}
}
