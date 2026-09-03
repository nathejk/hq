package live

import (
	"errors"
	"testing"

	"github.com/jrgensen/cqrs"
)

// Subjects here are copied from this repo's consumers and commands rather than
// invented, so the parser is pinned against what is actually published.
func TestSignalFromSubject(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		want    Signal
	}{
		{
			name:    "entity event with id",
			subject: "NATHEJK.2026.patrulje.abc123.started",
			want:    Signal{Type: SignalEntityChanged, Entity: "patrulje", ID: "abc123", Year: "2026", Event: "started"},
		},
		{
			name: "colon separator is normalised by subject.FromStr",
			// Both forms are in active use in this codebase (83 "." vs 45 ":"),
			// and the normalisation happens before we see the subject. Pinned
			// here because it is invisible and load-bearing.
			subject: "NATHEJK:2026.order.o-1.lines.changed",
			want:    Signal{Type: SignalEntityChanged, Entity: "order", ID: "o-1", Year: "2026", Event: "lines.changed"},
		},
		{
			name:    "dotted event name survives whole",
			subject: "NATHEJK.2026.gøgler.g-9.status.changed",
			want:    Signal{Type: SignalEntityChanged, Entity: "gøgler", ID: "g-9", Year: "2026", Event: "status.changed"},
		},
		{
			name:    "deeply dotted event name",
			subject: "NATHEJK.2026.bandit.b-2.armNumber.assigned",
			want:    Signal{Type: SignalEntityChanged, Entity: "bandit", ID: "b-2", Year: "2026", Event: "armNumber.assigned"},
		},
		{
			name: "collection level event has no id",
			// table/checkgroup/consumer.go:25
			subject: "NATHEJK.2026.checkgroups.sorted",
			want:    Signal{Type: SignalEntityChanged, Entity: "checkgroups", Year: "2026", Event: "sorted"},
		},
		{
			name: "year level event reports the year entity",
			// table/year/consumer.go:21
			subject: "NATHEJK.2026.created",
			want:    Signal{Type: SignalEntityChanged, Entity: "year", ID: "2026", Year: "2026", Event: "created"},
		},
		{
			name:    "year level update",
			subject: "NATHEJK.2025.updated",
			want:    Signal{Type: SignalEntityChanged, Entity: "year", ID: "2025", Year: "2025", Event: "updated"},
		},
		{
			name:    "underscored event name",
			subject: "NATHEJK.2026.checkgroup.cg-1.checkpoints_sorted",
			want:    Signal{Type: SignalEntityChanged, Entity: "checkgroup", ID: "cg-1", Year: "2026", Event: "checkpoints_sorted"},
		},
		{
			name:    "payment received",
			subject: "NATHEJK.2026.payment.p-7.received",
			want:    Signal{Type: SignalEntityChanged, Entity: "payment", ID: "p-7", Year: "2026", Event: "received"},
		},
		{
			name:    "non-numeric year slug",
			subject: "NATHEJK.2026-forar.klan.k-1.signedup",
			want:    Signal{Type: SignalEntityChanged, Entity: "klan", ID: "k-1", Year: "2026-forar", Event: "signedup"},
		},
		{
			name:    "domain match is case insensitive",
			subject: "nathejk.2026.spejder.s-1.updated",
			want:    Signal{Type: SignalEntityChanged, Entity: "spejder", ID: "s-1", Year: "2026", Event: "updated"},
		},
		{
			name: "telemetry track reported",
			// PRD 011: the second stream hq consumes. The entity token is "track"
			// — not "position", not "telemetry" — and this is what every frontend
			// dependsOn keys off, so it is pinned here rather than assumed.
			subject: "TELEMETRY.2026.track.f30793d2-5393-4d90-bbfa-cf224bbc131b.reported",
			want: Signal{
				Type:   SignalEntityChanged,
				Entity: "track",
				ID:     "f30793d2-5393-4d90-bbfa-cf224bbc131b",
				Year:   "2026",
				Event:  "reported",
			},
		},
		{
			name:    "telemetry domain match is case insensitive too",
			subject: "telemetry.2026.track.p-1.reported",
			want:    Signal{Type: SignalEntityChanged, Entity: "track", ID: "p-1", Year: "2026", Event: "reported"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SignalFromSubject(cqrs.SubjectFromStr(tc.subject))
			if err != nil {
				t.Fatalf("SignalFromSubject(%q) returned error: %v", tc.subject, err)
			}
			if got != tc.want {
				t.Errorf("SignalFromSubject(%q)\n got: %+v\nwant: %+v", tc.subject, got, tc.want)
			}
		})
	}
}

// A subject we cannot express must be rejected rather than yielding a signal for
// entity "", which would invalidate nothing while appearing to work.
func TestSignalFromSubjectRejects(t *testing.T) {
	for _, subject := range []string{
		"",
		"NATHEJK",
		"NATHEJK.2026",
		"OTHERDOMAIN.2026.patrulje.p-1.started",
		"NATHEJK..patrulje.p-1.started",  // no year
		"NATHEJK.2026..p-1.started",      // no entity
		"NATHEJK.2026.patrulje..started", // no id where one is expected

		// The three-part "year entity changed" form is a NATHEJK convention. A
		// telemetry subject of that shape is not a statement about the year, and
		// reporting it as one would invalidate every year-dependent page for an
		// unrelated reason — so it is rejected rather than misread.
		"TELEMETRY.2026.reported",

		// Still a closed set: adding TELEMETRY must not turn the check into
		// "anything with enough parts".
		"TELEMETRYX.2026.track.p-1.reported",
		"NATS.2026.track.p-1.reported",
	} {
		t.Run(subject, func(t *testing.T) {
			_, err := SignalFromSubject(cqrs.SubjectFromStr(subject))
			if !errors.Is(err, ErrNotASignal) {
				t.Errorf("SignalFromSubject(%q) err = %v, want ErrNotASignal", subject, err)
			}
		})
	}
}

func TestSignalKeyCoalescesPerEntityAndID(t *testing.T) {
	a, err := SignalFromSubject(cqrs.SubjectFromStr("NATHEJK.2026.patrulje.p-1.started"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := SignalFromSubject(cqrs.SubjectFromStr("NATHEJK.2026.patrulje.p-1.numberassigned"))
	if err != nil {
		t.Fatal(err)
	}
	other, err := SignalFromSubject(cqrs.SubjectFromStr("NATHEJK.2026.patrulje.p-2.started"))
	if err != nil {
		t.Fatal(err)
	}

	// Two events about the same entity instance are interchangeable to a client:
	// both mean "refetch p-1". Different instances must not collapse.
	if a.Key() != b.Key() {
		t.Errorf("same instance should share a key: %q vs %q", a.Key(), b.Key())
	}
	if a.Key() == other.Key() {
		t.Errorf("different instances must not share a key: %q", a.Key())
	}
}

func TestResync(t *testing.T) {
	s := Resync()
	if s.Type != SignalResync {
		t.Errorf("Resync().Type = %q, want %q", s.Type, SignalResync)
	}
	if s.Entity != "" || s.ID != "" {
		t.Errorf("Resync() should name no entity, got %+v", s)
	}
}
