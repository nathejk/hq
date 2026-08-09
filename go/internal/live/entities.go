package live

import (
	"sort"

	"github.com/jrgensen/cqrs"
)

// EntitySet is the set of entity tokens the live stream can emit, as advertised to
// clients.
//
// It exists to catch a failure mode that is otherwise invisible. A client declares
// what invalidates a resource by naming entity tokens (`dependsOn: ['patrulje']`),
// and a token that no consumer can ever produce fails *silently*: the page looks
// live, never errors, and simply never updates. Task 037 got two of six tokens wrong
// — `scan` for what is really `qr`, and `personnel` for what is really
// `gøgler`/`friend`/`bandit` — and both were caught only by reading this package's
// callers. Advertising the real set turns that into a warning.
type EntitySet struct {
	// Entities is the sorted, deduplicated set of tokens derived from the wired
	// consumers.
	Entities []string `json:"entities"`

	// Exhaustive reports whether Entities is the complete set.
	//
	// It is false when some consumer subscribes with a wildcard in the entity
	// position — `NATHEJK.*.*.*.signedup` and the verification/mail patterns do
	// exactly that — because such a subscription matches events for entities that
	// are nowhere named. A token outside Entities may then still arrive.
	//
	// Reported rather than hidden: a client that treated a non-exhaustive set as
	// complete would reject valid dependencies, which is a worse bug than the one
	// this is meant to catch.
	Exhaustive bool `json:"exhaustive"`
}

// Has reports whether the set contains a token.
func (s EntitySet) Has(entity string) bool {
	for _, e := range s.Entities {
		if e == entity {
			return true
		}
	}
	return false
}

// wildcard is the token a subject pattern uses to match any single element. A
// consumer whose *entity* position is a wildcard makes the derived set incomplete.
const wildcard = "*"

// EntitiesFrom derives the advertised entity set from the consumers that are
// actually wired.
//
// Two deliberate choices, both about not letting this drift from reality:
//
//   - It reads each consumer's Consumes() at runtime rather than scanning source.
//     A grep for subject literals also finds commented-out patterns and fragments of
//     multi-line concatenations; the wiring cannot lie in the same way.
//   - It extracts the token with SignalFromSubject — the same function that turns an
//     event into a signal — rather than parsing subjects a second time. So the set
//     advertised is by construction the set that function can produce, including its
//     synthesised "year" entity, and a change to the parser updates the advertisement
//     for free.
//
// Pass the same slice given to NotifyAll.
func EntitiesFrom(consumers ...cqrs.Consumer) EntitySet {
	seen := make(map[string]struct{})
	exhaustive := true

	for _, c := range consumers {
		if c == nil {
			continue
		}
		for _, subj := range c.Consumes() {
			signal, err := SignalFromSubject(subj)
			if err != nil {
				// Off-convention subjects produce no signal, so they contribute
				// no token. Not a problem to report: the same subjects are
				// silently signal-less at runtime, by design.
				continue
			}
			if signal.Entity == wildcard || signal.Entity == "" {
				// The consumer matches any entity here, so the set it implies is
				// open-ended. Record that rather than pretending otherwise.
				exhaustive = false
				continue
			}
			seen[signal.Entity] = struct{}{}
		}
	}

	entities := make([]string, 0, len(seen))
	for e := range seen {
		entities = append(entities, e)
	}
	// Sorted so the advertisement is stable: it is compared in tests, and a set that
	// reordered between boots would show up as a spurious difference in the
	// blue/green comparison PRD 005 describes.
	sort.Strings(entities)

	return EntitySet{Entities: entities, Exhaustive: exhaustive}
}
